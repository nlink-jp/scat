package slack

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nlink-jp/scat/internal/input"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// copyBounded never adds one to an untrusted int64 limit (which could overflow).
func copyBounded(ctx context.Context, dst io.Writer, src io.Reader, limit int64) (int64, error) {
	r := input.Reader{Context: ctx, Source: src}
	if limit == 0 {
		return io.Copy(dst, r)
	}
	n, err := io.Copy(dst, io.LimitReader(r, limit))
	if err != nil {
		return n, err
	}
	var b [1]byte
	m, err := io.ReadFull(r, b[:])
	if m > 0 {
		return n, errors.New("file exceeds configured size limit")
	}
	if err == io.EOF {
		return n, nil
	}
	return n, err
}

type Snapshot struct {
	File *os.File
	Size int64
	dir  string
}

func Stage(ctx context.Context, r io.Reader, limit int64) (*Snapshot, error) {
	if limit < 0 {
		return nil, errors.New("negative file limit")
	}
	dir, err := os.MkdirTemp("", "scat-upload-")
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, "input-")
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(dir))
	}
	s := &Snapshot{File: f, dir: dir}
	s.Size, err = copyBounded(ctx, f, r, limit)
	if err == nil {
		_, err = f.Seek(0, io.SeekStart)
	}
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return s, nil
}
func (s *Snapshot) Close() error {
	e := s.File.Close()
	e2 := os.RemoveAll(s.dir)
	return errors.Join(e, e2)
}

type UploadOptions struct {
	Channel, User, Filename, Comment, Thread string
	Source                                   *Snapshot
}
type FileID struct {
	ID string `json:"id"`
}
type UploadResult struct {
	Files   []FileID `json:"files"`
	Channel string   `json:"channel"`
}

func (c *Client) Upload(ctx context.Context, o UploadOptions) (UploadResult, error) {
	result := UploadResult{Files: []FileID{}}
	if err := ValidateThread(o.Thread); err != nil {
		return result, err
	}
	if o.Source == nil || o.Filename == "" {
		return result, errors.New("upload source and filename are required")
	}
	id, err := c.UploadDestination(ctx, o.Channel, o.User)
	if err != nil {
		return result, err
	}
	var ticket struct {
		URL string `json:"upload_url"`
		ID  string `json:"file_id"`
	}
	if err = c.api(ctx, "files.getUploadURLExternal", map[string]any{"filename": o.Filename, "length": o.Source.Size}, &ticket); err != nil {
		return result, err
	}
	if ticket.ID == "" {
		return result, errors.New("upload allocation: missing file ID")
	}
	u, err := trustedURL(ticket.URL)
	if err != nil {
		return result, fmt.Errorf("file %s allocation: %w", ticket.ID, err)
	}
	transferCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if _, err = o.Source.File.Seek(0, io.SeekStart); err != nil {
		return result, err
	}
	req, err := http.NewRequestWithContext(transferCtx, http.MethodPost, u.String(), io.NopCloser(o.Source.File))
	if err != nil {
		return result, errors.New("invalid upload request")
	}
	req.ContentLength = o.Source.Size
	if o.Source.Size == 0 {
		req.Body = http.NoBody
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	h := *c.http
	h.Jar = nil
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := h.Do(req)
	if err != nil {
		return result, fmt.Errorf("file %s byte transfer: %w", ticket.ID, transportError(transferCtx, u.Hostname()))
	}
	_, readErr := io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
	closeErr := resp.Body.Close()
	if resp.StatusCode != 200 || readErr != nil || closeErr != nil {
		return result, fmt.Errorf("file %s byte transfer: HTTP %d or incomplete response; not finalized", ticket.ID, resp.StatusCode)
	}
	body := map[string]any{"files": []FileID{{ticket.ID}}, "channel_id": id}
	if o.Comment != "" {
		body["initial_comment"] = o.Comment
	}
	if o.Thread != "" {
		body["thread_ts"] = o.Thread
	}
	var completed struct {
		Files []FileID `json:"files"`
	}
	if err = c.api(ctx, "files.completeUploadExternal", body, &completed); err != nil {
		return result, fmt.Errorf("file %s completion not confirmed; do not replay automatically: %w", ticket.ID, err)
	}
	found := false
	for _, f := range completed.Files {
		if f.ID == ticket.ID {
			found = true
		}
	}
	if !found {
		return result, fmt.Errorf("file %s completion: missing file confirmation; outcome unknown", ticket.ID)
	}
	result.Channel = id
	result.Files = completed.Files
	return result, nil
}
func safeComponent(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, "/\\:\x00\r\n") && filepath.Base(s) == s
}

// Download keeps the directory anchored and only replaces a target after verified bytes close successfully.
func (c *Client) Download(ctx context.Context, f File, dir string) (string, error) {
	if !safeComponent(f.ID) || !safeComponent(f.Name) {
		return "", errors.New("unsafe attachment filename")
	}
	if c.maxFileSize > 0 && f.Size > c.maxFileSize {
		return "", errors.New("attachment exceeds configured file limit")
	}
	raw := f.URL
	if raw == "" {
		raw = f.PrivateURL
	}
	origin, err := trustedURL(raw)
	if err != nil {
		return "", err
	}
	if err = c.Authenticate(ctx); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(abs, 0700); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return "", err
	}
	defer root.Close()
	name := f.ID + "_" + f.Name
	if info, e := root.Lstat(name); e == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("attachment target is a symlink")
	} else if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	var random [16]byte
	if _, err = rand.Read(random[:]); err != nil {
		return "", err
	}
	temp := ".scat-" + hex.EncodeToString(random[:])
	dst, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	defer root.Remove(temp)
	defer dst.Close()
	transferCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(transferCtx, http.MethodGet, origin.String(), nil)
	if err != nil {
		return "", errors.New("invalid download request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	withheld := false
	h := *c.http
	h.Jar = nil
	h.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many download redirects")
		}
		if r.URL.Scheme != "https" || r.URL.User != nil || (r.URL.Port() != "" && r.URL.Port() != "443") {
			return errors.New("unsafe download redirect")
		}
		r.Header.Del("Authorization")
		r.Header.Del("Cookie")
		if trustedHost(via[0].URL) && trustedHost(r.URL) {
			// G119 is intentionally retained: Slack sibling hosts need Bearer.
			// Both origin and target must be HTTPS under slack.com, judged against
			// via[0]. TestDownloadRedirectsHTMLAndBinary covers sibling (required),
			// foreign (forbidden), login HTML and downgrade cases.
			r.Header.Set("Authorization", "Bearer "+c.token)
		} else {
			withheld = true
		}
		return nil
	}
	resp, err := h.Do(req)
	if err != nil {
		return "", transportError(transferCtx, "download "+origin.Hostname())
	}
	defer resp.Body.Close()
	host := resp.Request.URL.Hostname()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download %s: HTTP %d", host, resp.StatusCode)
	}
	if c.maxFileSize > 0 && resp.ContentLength > c.maxFileSize {
		return "", errors.New("download exceeds configured file limit")
	}
	reader := bufio.NewReader(resp.Body)
	prefix, _ := reader.Peek(4096)
	if err = checkFileResponse(prefix, resp.Header.Get("Content-Type"), f.Mimetype, f.Size, withheld); err != nil {
		return "", fmt.Errorf("download %s: %w", host, err)
	}
	n, err := copyBounded(transferCtx, dst, reader, c.maxFileSize)
	if err != nil {
		return "", fmt.Errorf("download %s: incomplete transfer", host)
	}
	if (resp.ContentLength >= 0 && n != resp.ContentLength) || (f.Size > 0 && n != f.Size) {
		return "", errors.New("download length mismatch")
	}
	if err = resp.Body.Close(); err != nil {
		return "", errors.New("download response close failed")
	}
	if err = dst.Close(); err != nil {
		return "", err
	}
	if err = root.Rename(temp, name); err != nil {
		return "", err
	}
	return filepath.Join(abs, name), nil
}
func checkFileResponse(prefix []byte, contentType, expected string, expectedSize int64, withheld bool) error {
	var status struct {
		OK    *bool  `json:"ok"`
		Error string `json:"error"`
	}
	if json.Unmarshal(prefix, &status) == nil && status.OK != nil && !*status.OK && status.Error != "" {
		// A JSON attachment can itself be a saved Slack error. Require the
		// expected JSON metadata and exact recorded size before accepting it.
		mime := strings.ToLower(strings.TrimSpace(strings.Split(expected, ";")[0]))
		if withheld || !(mime == "application/json" || strings.HasSuffix(mime, "+json")) || expectedSize <= 0 || int64(len(prefix)) != expectedSize {
			return errors.New("file endpoint returned an API error or ambiguous JSON")
		}
	}
	s := strings.ToLower(strings.TrimSpace(string(prefix)))
	html := strings.Contains(strings.ToLower(contentType), "text/html") || strings.HasPrefix(s, "<!doctype html") || strings.HasPrefix(s, "<html")
	if !html {
		return nil
	}
	login := strings.Contains(s, "slack.com/signin") || strings.Contains(s, "sign in to slack") || strings.Contains(s, "slack.com/checkcookie") || strings.Contains(s, "name=\"signin\"") || strings.Contains(s, "id=\"signin_form\"")
	if withheld || login || !strings.HasPrefix(strings.ToLower(expected), "text/html") {
		return errors.New("HTML login or ambiguous file response; check files:read and file access")
	}
	return nil
}
