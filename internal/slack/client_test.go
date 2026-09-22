package slack

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r, ContentLength: int64(len(body))}
}
func client(t *testing.T, fn transport) *Client {
	t.Helper()
	return New("fixture-token", Options{HTTP: &http.Client{Transport: fn}, Wait: func(context.Context, time.Duration) error { return nil }, MaxFileSize: 1024})
}
func authResponse(r *http.Request) *http.Response {
	return response(r, 200, `{"ok":true,"bot_id":"B1","team_id":"T1"}`)
}
func TestIdentityAndLazyResolution(t *testing.T) {
	calls := []string{}
	c := client(t, func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatal("missing credential")
		}
		switch r.URL.Path {
		case "/api/auth.test":
			return authResponse(r), nil
		case "/api/chat.postMessage":
			return response(r, 200, `{"ok":true,"ts":"1.000001","channel":"C1"}`), nil
		}
		t.Fatalf("unexpected API %s", r.URL.Path)
		return nil, nil
	})
	if len(calls) > 0 {
		t.Fatal("constructor did I/O")
	}
	for i := 0; i < 2; i++ {
		if _, e := c.Post(context.Background(), PostOptions{Channel: "C1", Payload: map[string]any{"text": "hi"}}); e != nil {
			t.Fatal(e)
		}
	}
	if len(calls) != 3 {
		t.Fatal(calls)
	}
}
func TestRejectNonBotIdentity(t *testing.T) {
	for _, body := range []string{`{"ok":true,"user_id":"U1","team_id":"T1"}`, `{"ok":true,"bot_id":"B1"}`, `{"ok":false,"error":"invalid_auth"}`} {
		calls := 0
		c := client(t, func(r *http.Request) (*http.Response, error) { calls++; return response(r, 200, body), nil })
		if _, e := c.Post(context.Background(), PostOptions{Channel: "C1"}); e == nil || calls != 1 {
			t.Fatal("nonbot accepted", e, calls)
		}
	}
}
func TestRetryReplaysBodyAndHonorsCancellation(t *testing.T) {
	var bodies []string
	var waits []time.Duration
	c := New("fixture", Options{HTTP: &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "auth.test") {
			return authResponse(r), nil
		}
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) < 3 {
			res := response(r, 429, "")
			res.Header.Set("Retry-After", "2")
			return res, nil
		}
		return response(r, 200, `{"ok":true,"ts":"1.0","channel":"C1"}`), nil
	})}, Wait: func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil }})
	if _, e := c.Post(context.Background(), PostOptions{Channel: "C1", Payload: map[string]any{"text": "hello", "blocks": []any{map[string]string{"type": "divider"}}}}); e != nil {
		t.Fatal(e)
	}
	if len(bodies) != 3 || bodies[0] != bodies[1] || bodies[0] != bodies[2] || len(waits) != 2 || waits[0] != 2*time.Second {
		t.Fatal(bodies, waits)
	}
	c.wait = func(context.Context, time.Duration) error { return context.Canceled }
	bodies = nil
	if _, e := c.Post(context.Background(), PostOptions{Channel: "C1"}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
func TestRetryDelayValidation(t *testing.T) {
	for _, s := range []string{"-1", "9223372036854775807", "9999999999999999999999"} {
		if _, e := retryDelay(s); e == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"", "not-a-number"} {
		if d, e := retryDelay(s); e != nil || d != time.Second {
			t.Fatal(s, d, e)
		}
	}
}
func TestAmbiguousMutationNotRetriedAndErrorRedacted(t *testing.T) {
	calls := 0
	c := client(t, func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "auth.test") {
			return authResponse(r), nil
		}
		calls++
		return nil, errors.New("signed-url?token=PRIVATE")
	})
	_, e := c.Post(context.Background(), PostOptions{Channel: "C1"})
	if e == nil || strings.Contains(e.Error(), "PRIVATE") || calls != 1 {
		t.Fatal(e, calls)
	}
}
func TestPaginationShortPagesAndInvalidCursor(t *testing.T) {
	for _, mode := range []string{"short", "missing", "repeat"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			c := client(t, func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "auth.test") {
					return authResponse(r), nil
				}
				calls++
				r.ParseForm()
				if calls == 1 {
					switch mode {
					case "missing":
						return response(r, 200, `{"ok":true,"has_more":true}`), nil
					default:
						return response(r, 200, `{"ok":true,"messages":[{"ts":"1.0"}],"response_metadata":{"next_cursor":"next"}}`), nil
					}
				}
				if r.Form.Get("cursor") != "next" {
					t.Fatal("cursor lost")
				}
				if mode == "repeat" {
					return response(r, 200, `{"ok":true,"response_metadata":{"next_cursor":"next"}}`), nil
				}
				return response(r, 200, `{"ok":true,"messages":[{"ts":"2.0"}]}`), nil
			})
			v, e := c.Messages(context.Background(), "C1", "", "", "")
			if mode == "short" {
				if e != nil || len(v) != 2 || calls != 2 {
					t.Fatal(v, e, calls)
				}
			} else if e == nil {
				t.Fatal("bad cursor accepted")
			}
		})
	}
}
func TestPostJoinReplayAndPayload(t *testing.T) {
	bodies := []string{}
	c := client(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/auth.test":
			return authResponse(r), nil
		case "/api/conversations.info":
			return response(r, 200, `{"ok":true,"channel":{"id":"C1","is_private":false}}`), nil
		case "/api/conversations.join":
			return response(r, 200, `{"ok":true}`), nil
		case "/api/chat.postMessage":
			b, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(b))
			if len(bodies) == 1 {
				return response(r, 200, `{"ok":false,"error":"not_in_channel"}`), nil
			}
			return response(r, 200, `{"ok":true,"channel":"C1","ts":"1.0"}`), nil
		}
		t.Fatal(r.URL.Path)
		return nil, nil
	})
	p, e := ParseContent(`{"channel":"EVIL","thread_ts":"9.0","text":"fallback","blocks":[{"type":"divider"}],"attachments":[{"text":"hello"}],"unfurl_links":false}`, "payload")
	if e != nil {
		t.Fatal(e)
	}
	_, e = c.Post(context.Background(), PostOptions{Channel: "C1", Thread: "1.000001", Payload: p})
	if e != nil || len(bodies) != 2 || bodies[0] != bodies[1] || strings.Contains(bodies[0], "EVIL") {
		t.Fatal(bodies, e)
	}
	var b map[string]any
	json.Unmarshal([]byte(bodies[0]), &b)
	if b["thread_ts"] != "1.000001" || b["text"] != "fallback" {
		t.Fatal(b)
	}
}
func TestPayloadValidation(t *testing.T) {
	for _, p := range []string{`null`, `[]`, `{}`, `{"text":null}`, `{"blocks":{}}`, `{"unfurl_links":"false"}`, `{"blocks":[]}`} {
		if _, e := ParseContent(p, "payload"); e == nil {
			t.Fatal(p)
		}
	}
	for _, p := range []string{`null`, `{}`, `[]`, `{"blocks":"bad"}`} {
		if _, e := ParseContent(p, "blocks"); e == nil {
			t.Fatal(p)
		}
	}
}
func TestCreateReportsPartialFailure(t *testing.T) {
	calls := []string{}
	c := client(t, func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/api/auth.test":
			return authResponse(r), nil
		case "/api/conversations.create":
			return response(r, 200, `{"ok":true,"channel":{"id":"CNEW","name":"new"}}`), nil
		case "/api/conversations.setTopic":
			return response(r, 200, `{"ok":false,"error":"missing_scope","needed":"channels:manage"}`), nil
		}
		t.Fatal(r.URL.Path)
		return nil, nil
	})
	v, e := c.Create(context.Background(), CreateOptions{Name: "new", Topic: "topic", Description: "later"})
	if e == nil || !strings.Contains(e.Error(), "CNEW") || v.ID != "CNEW" || len(calls) != 3 {
		t.Fatal(v, e, calls)
	}
}
func TestCreatePrivateChannelRequestsPrivate(t *testing.T) {
	body := ""
	c := client(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/auth.test":
			return authResponse(r), nil
		case "/api/conversations.create":
			b, _ := io.ReadAll(r.Body)
			body = string(b)
			return response(r, 200, `{"ok":true,"channel":{"id":"CNEW","name":"new"}}`), nil
		}
		t.Fatal(r.URL.Path)
		return nil, nil
	})
	v, e := c.Create(context.Background(), CreateOptions{Name: "new", Private: true})
	if e != nil || v.ID != "CNEW" || !strings.Contains(body, "is_private=true") {
		t.Fatal(v, e, body)
	}
}
func TestAmbiguousUserAndDirectGroup(t *testing.T) {
	c := client(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/auth.test":
			return authResponse(r), nil
		case "/api/users.list":
			return response(r, 200, `{"ok":true,"members":[{"id":"U1","name":"same"},{"id":"U2","profile":{"display_name":"same"}}]}`), nil
		case "/api/usergroups.users.list":
			return response(r, 200, `{"ok":true,"users":["U1","U2","U1"]}`), nil
		}
		t.Fatal(r.URL.Path)
		return nil, nil
	})
	if _, e := c.ResolveUser(context.Background(), "same"); e == nil {
		t.Fatal("ambiguous name accepted")
	}
	ids, e := c.Invitees(context.Background(), []string{"S1", "U1"})
	if e != nil || strings.Join(ids, ",") != "U1,U2" {
		t.Fatal(ids, e)
	}
}

func TestPostUnknownPrivacyDoesNotJoin(t *testing.T) {
	c := client(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/auth.test":
			return authResponse(r), nil
		case "/api/chat.postMessage":
			return response(r, 200, `{"ok":false,"error":"not_in_channel"}`), nil
		case "/api/conversations.info":
			return response(r, 200, `{"ok":true,"channel":{"id":"C1"}}`), nil
		}
		t.Fatal("joined unverified public channel", r.URL.Path)
		return nil, nil
	})
	if _, err := c.Post(context.Background(), PostOptions{Channel: "C1", Payload: map[string]any{"text": "hello"}}); err == nil {
		t.Fatal("unknown privacy accepted")
	}
}
