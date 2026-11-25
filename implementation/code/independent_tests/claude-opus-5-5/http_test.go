// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30
// tests by: claude-opus-5-5

package aqltests

import (
	"encoding/json"
	"strings"
	"syscall"
	"testing"
	"time"
)

func wantRefused(t *testing.T, a httpAnswer, status int, msg string) {
	t.Helper()
	if a.status != status {
		t.Fatalf("status %d, want %d: %s", a.status, status, a.raw)
	}
	var body struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   *rpcError       `json:"error"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(a.raw), &body); err != nil {
		t.Fatalf("body is not JSON-RPC: %q", a.raw)
	}
	if body.Error == nil || body.Error.Code != -32600 || body.Error.Message != msg || string(body.ID) != "null" || body.Result != nil {
		t.Fatalf("body: %s", a.raw)
	}
}

func TestHTTPRequestBringsItsAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_SUBDOMAIN": "", "AHA_API_KEY": ""})
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key", "Aha-Subdomain": "example"})
	if a := c.initialize(); a.status != 200 {
		t.Fatalf("initialize: %d %s", a.status, a.raw)
	}
	e.fake.Reset()
	wantResult(t, c.query(`SELECT email FROM me`), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	calls := e.fake.Calls()
	if len(calls) != 1 || calls[0].Method != "GET" || calls[0].Path != "/me" || calls[0].Host != "example.aha.io" ||
		calls[0].Header.Get("Authorization") != "Bearer fixture-key" {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestHTTPRequestWithoutACredential(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	c := s.client(nil)
	a := c.request("tools/list", map[string]any{}, 30*time.Second)
	wantRefused(t, a, 400, `no credential. Send the header Aha-Api-Key with an Aha! API key`)
	if strings.Contains(a.raw, "tools") {
		t.Fatalf("answer has a tool list: %s", a.raw)
	}
	wantRefused(t, c.initialize(), 400, `no credential. Send the header Aha-Api-Key with an Aha! API key`)
	bad := s.client(map[string]string{"Aha-Api-Key": "two words"})
	wantRefused(t, bad.request("tools/list", map[string]any{}, 30*time.Second), 400, `no credential. Send the header Aha-Api-Key with an Aha! API key`)
	e.wantNoRequest()
}

func TestHTTPRequestForAnotherAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key", "Aha-Subdomain": "other"})
	a := c.request("tools/call", map[string]any{"name": "query", "arguments": map[string]any{"query": "SELECT email FROM me"}}, 30*time.Second)
	wantRefused(t, a, 403, `this server works with one Aha! account only`)
	c2 := s.client(map[string]string{"Aha-Api-Key": "fixture-key", "Aha-Subdomain": "Not An Account"})
	wantRefused(t, c2.request("tools/list", map[string]any{}, 30*time.Second), 400, `Aha-Subdomain must be the account name before .aha.io`)
	e.wantNoRequest()
}

func TestHTTPRequestWithoutAnAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_SUBDOMAIN": "", "AHA_API_KEY": ""})
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key"})
	wantRefused(t, c.request("tools/list", map[string]any{}, 30*time.Second), 400, `no Aha! account. Send the header Aha-Subdomain`)
	e.wantNoRequest()
}

func TestHTTPRequestFromAWebPage(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	c := s.client(map[string]string{"Origin": "https://pages.example", "Aha-Api-Key": "fixture-key"})
	wantRefused(t, c.request("tools/list", map[string]any{}, 30*time.Second), 403, `requests from a web page are not accepted`)
	e.wantNoRequest()
}

func TestHTTPRequestsShareNothing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	c1 := s.client(map[string]string{"Aha-Api-Key": "fixture-key"})
	c1.initialize()
	e.fake.Reset()
	wantResult(t, c1.query(`SELECT key FROM products`), `{"entity":"products","scope":"all products","count":2,"records":[{"key":"DEMO"},{"key":"OTHER"}]}`)
	if n := e.fake.count("GET", "/products"); n != 1 {
		t.Fatalf("first request read the product list %d times", n)
	}
	c2 := s.client(map[string]string{"Aha-Api-Key": "other-key"})
	c2.initialize()
	e.fake.Reset()
	r := c2.query(`SELECT key FROM products`)
	wantError(t, r, `Aha! refused the request (status 403): the API key is not valid for the account example, or it has no permission for this record. Repeating the statement does not help.`)
	if n := e.fake.count("GET", "/products"); n != 1 {
		t.Fatalf("second request read the product list %d times", n)
	}
	if strings.Contains(r.text, "other-key") {
		t.Fatal("message has the credential")
	}
	for _, c := range e.fake.Calls() {
		if c.Header.Get("Authorization") != "Bearer other-key" {
			t.Fatalf("second request used another credential: %s", c.Header.Get("Authorization"))
		}
	}
}

func TestDefinitionsAreNotKeptInHTTPMode(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key"})
	c.initialize()
	e.fake.Reset()
	want := `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`
	wantResult(t, c.query(`SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`), want)
	wantResult(t, c.query(`SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`), want)
	if n := e.fake.count("GET", "/custom_field_definitions"); n != 2 {
		t.Fatalf("definitions read %d times", n)
	}
}

func TestHTTPToolListAndExampleKeyPerRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""}, "default-product=demo")
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key"})
	c.initialize()
	a := c.request("tools/list", map[string]any{}, 60*time.Second)
	if a.status != 200 {
		t.Fatalf("tools/list: %d %s", a.status, a.raw)
	}
	tools := parseTools(t, a.msg)
	if len(tools) != 2 || !strings.Contains(tools[0].Description+tools[1].Description, "'DEMO-23'") {
		t.Fatalf("tools: %+v", tools)
	}
	e.fake.Reset()
	c.request("tools/list", map[string]any{}, 60*time.Second)
	if n := e.fake.count("GET", "/products"); n != 1 {
		t.Fatalf("each tool list request chooses its key: %d", n)
	}
}

func TestHTTPExecuteWritesALogLine(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key"})
	c.initialize()
	a := c.request("tools/call", map[string]any{"name": "execute", "arguments": map[string]any{"statement": "DELETE FROM features WHERE id = 'demo-6'"}}, 60*time.Second)
	wantResult(t, parseToolResult(t, a.msg), `{"id":"DEMO-6","message":"Feature DEMO-6 deleted"}`)
	if !waitFor(func() bool {
		for _, l := range lines(s.stderr.String()) {
			if m := logLine.FindStringSubmatch(l); m != nil && m[3] == "delete" && m[5] == "DEMO-6" && m[6] == "ok" && m[2] == "example" {
				return true
			}
		}
		return false
	}, 3*time.Second) {
		t.Fatalf("stderr: %q", s.stderr.String())
	}
}

func TestGracefulShutdownHTTP(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key"})
	c.initialize()
	wantResult(t, c.query(`SELECT email FROM me`), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-s.exited:
		if err != nil {
			t.Fatalf("exit after SIGTERM: %v", err)
		}
		s.exited <- nil
	case <-time.After(40 * time.Second):
		t.Fatal("server did not end after SIGTERM")
	}
}

func TestGracefulShutdownStdio(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-s.exited:
		if err != nil {
			t.Fatalf("exit after SIGTERM: %v", err)
		}
		s.exited <- nil
	case <-time.After(20 * time.Second):
		t.Fatal("server did not end after SIGTERM")
	}
}
