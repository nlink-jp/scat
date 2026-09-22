package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadStagesAndOneShotCompletion(t *testing.T) {
	for _, mode := range []string{"success", "transfer-failure", "complete-timeout", "complete-rate-limit", "complete-not-ok", "membership-failure"} {
		t.Run(mode, func(t *testing.T) {
			data := []byte{0, 255, 2, 3, 4}
			s, e := Stage(context.Background(), bytes.NewReader(data), 100)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			alloc, transfer, complete := 0, 0, 0
			c := client(t, func(r *http.Request) (*http.Response, error) {
				switch r.URL.Path {
				case "/api/auth.test":
					return authResponse(r), nil
				case "/api/conversations.info":
					if mode == "membership-failure" {
						return response(r, 200, `{"ok":false,"error":"missing_scope"}`), nil
					}
					return response(r, 200, `{"ok":true,"channel":{"id":"C1","is_member":true}}`), nil
				case "/api/files.getUploadURLExternal":
					alloc++
					r.ParseForm()
					if r.Form.Get("length") != "5" || r.Form.Get("filename") != "bytes.bin" {
						t.Fatal(r.Form)
					}
					return response(r, 200, `{"ok":true,"upload_url":"https://files.slack.com/upload","file_id":"F1"}`), nil
				case "/upload":
					transfer++
					b, _ := io.ReadAll(r.Body)
					if r.Method != "POST" || r.ContentLength != 5 || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || !bytes.Equal(b, data) {
						t.Fatal("incorrect byte transfer")
					}
					if mode == "transfer-failure" {
						return response(r, 500, "failed"), nil
					}
					return response(r, 200, "OK - 5"), nil
				case "/api/files.completeUploadExternal":
					complete++
					b, _ := io.ReadAll(r.Body)
					var payload map[string]any
					json.Unmarshal(b, &payload)
					if payload["channel_id"] != "C1" || payload["thread_ts"] != "1.000001" {
						t.Fatal(payload)
					}
					switch mode {
					case "complete-timeout":
						return nil, errors.New("unknown")
					case "complete-rate-limit":
						return response(r, 429, "rate limited"), nil
					case "complete-not-ok":
						return response(r, 200, `{"ok":false,"error":"not_in_channel"}`), nil
					}
					return response(r, 200, `{"ok":true,"files":[{"id":"F1"}]}`), nil
				}
				t.Fatal(r.URL.Path)
				return nil, nil
			})
			result, e := c.Upload(context.Background(), UploadOptions{Channel: "C1", Filename: "bytes.bin", Thread: "1.000001", Source: s})
			if mode == "success" {
				if e != nil || result.Channel != "C1" || len(result.Files) != 1 {
					t.Fatal(result, e)
				}
			} else {
				if e == nil || len(result.Files) != 0 {
					t.Fatal("false success", result, e)
				}
			}
			if mode == "membership-failure" {
				if alloc != 0 || transfer != 0 || complete != 0 {
					t.Fatal("allocated before membership")
				}
			} else if mode == "transfer-failure" {
				if complete != 0 {
					t.Fatal("finalized failed transfer")
				}
			} else if complete != 1 {
				t.Fatal("completion replay", complete)
			}
		})
	}
}
func TestUploadURLCredentialAndRedirectGuard(t *testing.T) {
	for _, u := range []string{"http://files.slack.com/a", "https://slack.com.evil.example/a", "https://evilslack.com/a", "https://x@files.slack.com/a", "https://files.slack.com:444/a"} {
		if _, e := trustedURL(u); e == nil {
			t.Fatal(u)
		}
	}
}
func TestStageLimitsAndSnapshot(t *testing.T) {
	if s, e := Stage(context.Background(), strings.NewReader("12345"), 4); e == nil {
		s.Close()
		t.Fatal("limit bypass")
	}
	s, e := Stage(context.Background(), strings.NewReader("12345"), 0)
	if e != nil {
		t.Fatal(e)
	}
	path := s.File.Name()
	info, e := s.File.Stat()
	if e != nil || info.Mode().Perm() != 0600 || s.Size != 5 {
		t.Fatal(info, e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("snapshot leaked")
	}
}
func TestDownloadRedirectsHTMLAndBinary(t *testing.T) {
	for _, mode := range []string{"sibling", "foreign", "foreign-html", "missing-scope", "legitimate-html", "ambiguous-html", "oversize", "broken-body", "downgrade"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "F1_file.bin")
			os.WriteFile(path, []byte("existing"), 0600)
			expectedMIME := "application/octet-stream"
			if mode == "legitimate-html" || mode == "missing-scope" {
				expectedMIME = "text/html"
			}
			data := string([]byte{0, 255, 1, 2})
			if mode == "legitimate-html" {
				data = "<!doctype html><html><body>report</body></html>"
			}
			var finalHost string
			c := client(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/api/auth.test" {
					return authResponse(r), nil
				}
				if r.URL.Path == "/start" {
					res := response(r, 302, "")
					target := "https://other.slack.com/final"
					if strings.HasPrefix(mode, "foreign") {
						target = "https://outside.example/final"
					}
					if mode == "downgrade" {
						target = "http://other.slack.com/final"
					}
					res.Header.Set("Location", target)
					return res, nil
				}
				finalHost = r.URL.Hostname()
				if finalHost == "outside.example" {
					if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
						t.Fatal("credential leaked")
					}
				} else if r.Header.Get("Authorization") != "Bearer fixture-token" {
					t.Fatal("sibling lost credentials")
				}
				res := response(r, 200, data)
				switch mode {
				case "missing-scope", "foreign-html":
					res = response(r, 200, `<!doctype html><html>Sign in to Slack</html>`)
					res.Header.Set("Content-Type", "text/html")
				case "ambiguous-html":
					res = response(r, 200, "<html>Unknown page</html>")
					res.Header.Set("Content-Type", "text/html")
				case "legitimate-html":
					res.Header.Set("Content-Type", "text/html")
				case "oversize":
					res = response(r, 200, strings.Repeat("x", 1025))
					res.ContentLength = -1
				case "broken-body":
					res.Body = io.NopCloser(failedReader{})
					res.ContentLength = -1
				}
				return res, nil
			})
			got, e := c.Download(context.Background(), File{ID: "F1", Name: "file.bin", Mimetype: expectedMIME, URL: "https://files.slack.com/start"}, dir)
			success := mode == "sibling" || mode == "foreign" || mode == "legitimate-html"
			actual, _ := os.ReadFile(path)
			if success {
				if e != nil || got != path || string(actual) != data {
					t.Fatal(got, e, actual)
				}
			} else {
				if e == nil || got != "" || string(actual) != "existing" {
					t.Fatal("bad download replaced destination", got, e)
				}
				if mode == "downgrade" && finalHost != "" {
					t.Fatal("downgraded request sent")
				}
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 {
				t.Fatal("temporary file leak", entries)
			}
		})
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, errors.New("copy failed") }
func TestDownloadUnsafeTargets(t *testing.T) {
	calls := 0
	c := client(t, func(r *http.Request) (*http.Response, error) { calls++; return authResponse(r), nil })
	dir := t.TempDir()
	for _, name := range []string{"../escape", "foo/bar", "foo\\bar", ".", ".."} {
		if _, e := c.Download(context.Background(), File{ID: "F1", Name: name, URL: "https://files.slack.com/file"}, dir); e == nil {
			t.Fatal(name)
		}
	}
	if calls != 0 {
		t.Fatal("unsafe name made network request")
	}
	target := filepath.Join(t.TempDir(), "outside")
	os.WriteFile(target, []byte("untouched"), 0600)
	if e := os.Symlink(target, filepath.Join(dir, "F1_link")); e != nil {
		t.Skip(e)
	}
	if _, e := c.Download(context.Background(), File{ID: "F1", Name: "link", URL: "https://files.slack.com/file"}, dir); e == nil {
		t.Fatal("followed symlink")
	}
	b, _ := os.ReadFile(target)
	if string(b) != "untouched" {
		t.Fatal("changed symlink target")
	}
}

func TestUploadUnknownMembershipDoesNotAllocateOrJoin(t *testing.T) {
	for _, channel := range []string{`{"id":"C1","is_private":false}`, `{"id":"C1","is_member":false}`} {
		c := client(t, func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/api/auth.test":
				return authResponse(r), nil
			case "/api/conversations.info":
				return response(r, 200, `{"ok":true,"channel":`+channel+`}`), nil
			}
			t.Fatal("unexpected allocation/join", r.URL.Path)
			return nil, nil
		})
		if _, err := c.UploadDestination(context.Background(), "C1", ""); err == nil {
			t.Fatal("unknown membership/visibility accepted")
		}
	}
}

func TestLegitimateErrorJSONAttachment(t *testing.T) {
	data := `{"ok":false,"error":"missing_scope"}`
	c := client(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/auth.test" {
			return authResponse(r), nil
		}
		res := response(r, 200, data)
		res.Header.Set("Content-Type", "application/json")
		return res, nil
	})
	file := File{ID: "F1", Name: "error.json", Mimetype: "application/json", Size: int64(len(data)), URL: "https://files.slack.com/file"}
	path, err := c.Download(context.Background(), file, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != data {
		t.Fatal(string(b))
	}
	file.Size++
	if _, err = c.Download(context.Background(), file, t.TempDir()); err == nil {
		t.Fatal("ambiguous API error accepted")
	}
}

type cancelOnClose struct {
	io.Reader
	cancel context.CancelFunc
}

func (c cancelOnClose) Close() error { c.cancel(); return nil }
func TestDownloadCanceledAtClosePreservesDestination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	path := filepath.Join(dir, "F1_file")
	os.WriteFile(path, []byte("old"), 0600)
	c := client(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/auth.test" {
			return authResponse(r), nil
		}
		res := response(r, 200, "new")
		res.Body = cancelOnClose{strings.NewReader("new"), cancel}
		return res, nil
	})
	if _, err := c.Download(ctx, File{ID: "F1", Name: "file", URL: "https://files.slack.com/file"}, dir); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "old" {
		t.Fatal("canceled download replaced destination")
	}
}

// Real Slack serves externally uploaded HTML/JSON as text/plain metadata plus
// application/force-download and Content-Disposition: attachment. Preserve the
// bytes without allowing login pages, mismatched identities or foreign hosts.
func TestDownloadSlackForcedAttachment(t *testing.T) {
	for _, body := range []string{"<!doctype html><html><body>report</body></html>", `{"ok":false,"error":"fixture_content"}`} {
		for _, mode := range []string{"valid", "large", "wrong-name", "no-disposition", "no-size", "wrong-size", "login", "foreign", "wrong-type"} {
			t.Run(fmt.Sprintf("%s/%t", mode, strings.HasPrefix(body, "<")), func(t *testing.T) {
				data := body
				if mode == "large" {
					data += strings.Repeat(" ", 8192)
				}
				if mode == "login" {
					data = "<!doctype html><html>Sign in to Slack</html>"
				}
				size := int64(len(data))
				if mode == "wrong-size" {
					size++
				}
				if mode == "no-size" {
					size = 0
				}
				dir := t.TempDir()
				target := filepath.Join(dir, "F1_fixture.txt")
				if err := os.WriteFile(target, []byte("previous"), 0600); err != nil {
					t.Fatal(err)
				}
				c := client(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/api/auth.test" {
						return authResponse(r), nil
					}
					if r.URL.Path == "/start" {
						res := response(r, 302, "")
						host := "other.slack.com"
						if mode == "foreign" {
							host = "outside.example"
						}
						res.Header.Set("Location", "https://"+host+"/file")
						return res, nil
					}
					if mode == "foreign" && r.Header.Get("Authorization") != "" {
						t.Fatal("credential leaked")
					}
					res := response(r, 200, data)
					res.Header.Set("Content-Type", "application/force-download")
					res.Header.Set("Content-Disposition", `attachment; filename="fixture.txt"`)
					if mode == "wrong-name" {
						res.Header.Set("Content-Disposition", `attachment; filename="other.txt"`)
					}
					if mode == "no-disposition" {
						res.Header.Del("Content-Disposition")
					}
					if mode == "wrong-type" {
						res.Header.Set("Content-Type", "text/html")
					}
					return res, nil
				})
				c.maxFileSize = 16 << 10
				path, err := c.Download(context.Background(), File{ID: "F1", Name: "fixture.txt", Mimetype: "text/plain", Size: size, URL: "https://files.slack.com/start"}, dir)
				got, readErr := os.ReadFile(target)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if mode == "valid" || mode == "large" {
					if err != nil || path != target || string(got) != data {
						t.Fatalf("valid attachment rejected: %v", err)
					}
				} else if err == nil || path != "" || string(got) != "previous" {
					t.Fatalf("untrusted response accepted: %v", err)
				}
			})
		}
	}
}
