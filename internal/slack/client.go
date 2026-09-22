// Package slack implements the bot-only Slack boundary. Constructors perform no I/O.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	HTTP        *http.Client
	Now         func() time.Time
	Wait        func(context.Context, time.Duration) error
	Warn        func(string)
	MaxFileSize int64
}
type Client struct {
	token       string
	http        *http.Client
	now         func() time.Time
	wait        func(context.Context, time.Duration) error
	warn        func(string)
	maxFileSize int64
	identity    *Identity
	names       map[string]string
}
type Identity struct {
	BotID  string `json:"bot_id"`
	TeamID string `json:"team_id"`
}

func New(token string, o Options) *Client {
	h := o.HTTP
	if h == nil {
		h = &http.Client{}
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	wait := o.Wait
	if wait == nil {
		wait = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	warn := o.Warn
	if warn == nil {
		warn = func(string) {}
	}
	return &Client{token: token, http: h, now: now, wait: wait, warn: warn, maxFileSize: o.MaxFileSize, names: map[string]string{}}
}

type APIError struct{ Method, Code, Needed string }

func (e *APIError) Error() string {
	s := e.Method + ": " + e.Code
	if e.Needed != "" {
		s += " (needed scopes: " + e.Needed + ")"
	}
	return s
}
func IsCode(err error, code string) bool {
	var e *APIError
	return errors.As(err, &e) && e.Code == code
}

var safeAPIValue = regexp.MustCompile(`^[A-Za-z0-9_:, .-]{1,256}$`)

func diagnostic(s string) string {
	if !safeAPIValue.MatchString(s) {
		return "unrecognized response"
	}
	return s
}
func transportError(ctx context.Context, stage string) error {
	if ctx.Err() != nil {
		return fmt.Errorf("%s: %w", stage, ctx.Err())
	}
	return fmt.Errorf("%s: transport failed; outcome may be unknown", stage)
}

// Authenticate is cached only by this invocation's client, never across credentials.
func (c *Client) Authenticate(ctx context.Context) error {
	if c.identity != nil {
		return nil
	}
	if c.token == "" {
		return errors.New("bot token is required; configure a profile or SCAT_TOKEN")
	}
	if strings.HasPrefix(c.token, "xoxp-") || strings.HasPrefix(c.token, "xapp-") {
		return errors.New("scat requires bot credentials, not user or app-level credentials")
	}
	var v Identity
	if err := c.call(ctx, "auth.test", nil, &v, true); err != nil {
		return err
	}
	if v.BotID == "" || v.TeamID == "" {
		return errors.New("auth.test: credentials do not identify a workspace bot")
	}
	c.identity = &v
	return nil
}
func (c *Client) api(ctx context.Context, method string, body any, out any) error {
	if err := c.Authenticate(ctx); err != nil {
		return err
	}
	return c.call(ctx, method, body, out, method != "files.completeUploadExternal")
}
func (c *Client) call(ctx context.Context, method string, body any, out any, retry bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	contentType := "application/json; charset=utf-8"
	if method != "chat.postMessage" && method != "files.completeUploadExternal" {
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(data, &fields); err != nil {
			return err
		}
		form := url.Values{}
		for k, v := range fields {
			var str string
			if json.Unmarshal(v, &str) == nil {
				form.Set(k, str)
			} else {
				form.Set(k, string(v))
			}
		}
		data = []byte(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	}
	attempts := 1
	if retry {
		attempts = 3
	}
	for n := 0; n < attempts; n++ {
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		req, err := http.NewRequestWithContext(callCtx, http.MethodPost, "https://slack.com/api/"+method, bytes.NewReader(data))
		if err != nil {
			cancel()
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", contentType)
		h := *c.http
		h.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		h.Jar = nil
		resp, err := h.Do(req)
		if err != nil {
			e := transportError(callCtx, method)
			cancel()
			return e
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024+1))
		closeErr := resp.Body.Close()
		cancel()
		if resp.StatusCode == 429 && retry && n+1 < attempts {
			d, err := retryDelay(resp.Header.Get("Retry-After"))
			if err != nil {
				return fmt.Errorf("%s: %w", method, err)
			}
			if err = c.wait(ctx, d); err != nil {
				return err
			}
			continue
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("%s: HTTP %d; operation not confirmed", method, resp.StatusCode)
		}
		if readErr != nil || closeErr != nil || len(raw) > 32*1024*1024 {
			return fmt.Errorf("%s: incomplete or oversized API response", method)
		}
		var status struct {
			OK     bool   `json:"ok"`
			Error  string `json:"error"`
			Needed string `json:"needed"`
		}
		if json.Unmarshal(raw, &status) != nil {
			return fmt.Errorf("%s: invalid API response", method)
		}
		if !status.OK {
			return &APIError{method, diagnostic(strings.ReplaceAll(status.Error, c.token, "[redacted]")), diagnosticOptional(strings.ReplaceAll(status.Needed, c.token, "[redacted]"))}
		}
		if out != nil && json.Unmarshal(raw, out) != nil {
			return fmt.Errorf("%s: invalid response fields", method)
		}
		return nil
	}
	return fmt.Errorf("%s: rate limit exhausted", method)
}
func diagnosticOptional(s string) string {
	if s == "" {
		return ""
	}
	return diagnostic(s)
}
func retryDelay(s string) (time.Duration, error) {
	if s == "" {
		return time.Second, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		var ne *strconv.NumError
		if errors.As(err, &ne) && errors.Is(ne.Err, strconv.ErrRange) {
			return 0, errors.New("Retry-After overflow")
		}
		return time.Second, nil
	}
	if n < 0 || n > int64((1<<63-1)/int64(time.Second)) {
		return 0, errors.New("invalid Retry-After duration")
	}
	return time.Duration(n) * time.Second, nil
}

type metadata struct {
	NextCursor string `json:"next_cursor"`
}

func nextCursor(m metadata, more bool, seen map[string]bool) (string, error) {
	s := strings.TrimSpace(m.NextCursor)
	if s == "" {
		if more {
			return "", errors.New("pagination has_more without next_cursor")
		}
		return "", nil
	}
	if seen[s] {
		return "", errors.New("pagination repeated cursor")
	}
	seen[s] = true
	return s, nil
}
func trustedURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return nil, errors.New("invalid Slack transfer URL")
	}
	if !trustedHost(u) {
		return nil, errors.New("untrusted Slack transfer URL")
	}
	return u, nil
}
func trustedHost(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	return u.Scheme == "https" && u.User == nil && (u.Port() == "" || u.Port() == "443") && (host == "slack.com" || strings.HasSuffix(host, ".slack.com"))
}
