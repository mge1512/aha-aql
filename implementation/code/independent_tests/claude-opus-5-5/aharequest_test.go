// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268
// tests by: claude-opus-5-5

package aqltests

import (
	"strings"
	"testing"
	"time"
)

func TestRequestAddressAndHeaders(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE id = 'demo-3'`),
		`{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`)
	calls := e.fake.Calls()
	if len(calls) != 1 {
		t.Fatal(e.fake.describeCalls())
	}
	c := calls[0]
	if c.Method != "GET" || c.Host != "example.aha.io" || c.Path != "/features/DEMO-3" {
		t.Fatalf("request: %s %s %s", c.Method, c.Host, c.Path)
	}
	if c.Header.Get("Authorization") != "Bearer fixture-key" || c.Header.Get("Accept") != "application/json" {
		t.Fatalf("headers: %v", c.Header)
	}
	ua := c.Header.Get("User-Agent")
	if !strings.HasPrefix(ua, "mcp-server-aha-aql/") || !strings.HasSuffix(ua, "(https://github.com/mge1512/aha-aql)") {
		t.Fatalf("User-Agent: %q", ua)
	}
}

func TestWriteRequestHasAContentType(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	s.execute(`UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'`)
	for _, c := range e.fake.Calls() {
		if c.Method == "PUT" {
			if c.Header.Get("Content-Type") != "application/json" || c.Header.Get("Authorization") != "Bearer fixture-key" {
				t.Fatalf("headers: %v", c.Header)
			}
			return
		}
	}
	t.Fatal(e.fake.describeCalls())
}

func TestRequestThroughTheProxyOfTheEnvironment(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	// No tools/list before the query: a tunnel that the tool list opened
	// stays open (aha-request, Connections), and the CONNECT of the query
	// would then lie before the reset of the recorded requests.
	s := e.stdioRaw(nil)
	s.initialize()
	wantResult(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-3'`),
		`{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`)
	cs := e.fake.Connects()
	if len(cs) < 1 {
		t.Fatalf("no CONNECT request")
	}
	for _, c := range cs {
		if c.Host != "example.aha.io:443" {
			t.Fatalf("connects: %+v", cs)
		}
		for k, vs := range c.Header {
			for _, v := range vs {
				if strings.Contains(v, "fixture-key") {
					t.Fatalf("CONNECT header %s has the credential", k)
				}
			}
		}
	}
	e.wantCalls("GET /features/DEMO-3")
}

func TestCertificateThatTheTrustStoreDoesNotHave(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	_, _, otherPEM := newCA(t, "another CA")
	other := writeSandboxFile(t, e.dir, "other-ca.pem", otherPEM, 0o600)
	s := e.stdio(map[string]string{"SSL_CERT_FILE": other})
	wantError(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-3'`), `cannot reach Aha!: the certificate of example.aha.io could not be verified.`)
	e.wantNoCall()
}

func TestProxyDoesNotOpenTheTunnel(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	// FakeAha refuses CONNECT from the start (GIVEN). A tunnel that a tool
	// list opened before would stay open and be used (aha-request,
	// Connections), so the server gets no tools/list first.
	e.fake.RefuseConnect(true)
	s := e.stdioRaw(nil)
	s.initialize()
	r := s.execute(`UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'`)
	wantError(t, r, `cannot reach Aha!: the proxy did not open a connection to example.aha.io.`)
	if strings.Contains(r.text, strings.TrimPrefix(e.fake.ProxyURL, "http://")) {
		t.Fatal("message has the address of the proxy")
	}
	e.wantNoCall()
}

func TestTooManyRequestsIsSentAgain(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AnswerRateLimited("GET", "/features/DEMO-3", 1, 2)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-3'`),
		`{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`)
	var times []time.Time
	for _, c := range e.fake.Calls() {
		if c.Path == "/features/DEMO-3" {
			times = append(times, c.Time)
		}
	}
	if len(times) != 2 {
		t.Fatalf("calls: %s", e.fake.describeCalls())
	}
	if d := times[1].Sub(times[0]); d < time.Second || d >= 5*time.Second {
		t.Fatalf("second call after %v", d)
	}
}

func TestTooManyRequestsThreeTimes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.Answer("GET", "/features/DEMO-3", 429, -1, nil, "")
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-3'`), `aha: API error (status 429): too many requests`)
	var times []time.Time
	for _, c := range e.fake.Calls() {
		if c.Path == "/features/DEMO-3" {
			times = append(times, c.Time)
		}
	}
	if len(times) != 3 {
		t.Fatalf("calls: %s", e.fake.describeCalls())
	}
	if times[1].Sub(times[0]) < 5*time.Second || times[2].Sub(times[1]) < 5*time.Second {
		t.Fatalf("gaps: %v %v", times[1].Sub(times[0]), times[2].Sub(times[1]))
	}
}

func TestRedirectIsNotFollowed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.Answer("GET", "/features/DEMO-3", 302, -1, map[string]string{"Location": "https://other.example/api/v1/features/DEMO-3"}, "")
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-3'`), `aha: API error (status 302) on GET /features/DEMO-3: (no text)`)
	calls := e.fake.Calls()
	if len(calls) != 1 || calls[0].Host != "example.aha.io" {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestCredentialThatAhaDoesNotAccept(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(map[string]string{"AHA_API_KEY": "wrong-key"})
	r := s.query(`SELECT email FROM me`)
	wantError(t, r, `Aha! refused the request (status 403): the API key is not valid for the account example, or it has no permission for this record. Repeating the statement does not help.`)
	if strings.Contains(r.text, "wrong-key") || strings.Contains(s.stderr.String(), "wrong-key") {
		t.Fatal("credential in the output")
	}
}

func TestOtherStatusIsAnAPIError(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.Answer("GET", "/features/DEMO-3", 500, -1, nil, `{"error":"Internal failure"}`)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-3'`), `aha: API error (status 500) on GET /features/DEMO-3: {"error":"Internal failure"}`)
	e.fake.Answer("GET", "/features/DEMO-2", 200, -1, nil, `not json`)
	wantError(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-2'`), `aha: API error (status 200) on GET /features/DEMO-2: the answer is not JSON`)
}

func TestWriteRequestWithoutAnAnswer(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.NoAnswer("PUT", "/features/DEMO-1")
	s := e.stdio(nil)
	r := s.call("execute", map[string]any{"statement": `UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'`}, 150*time.Second)
	wantError(t, r, `cannot reach Aha!: no answer within 60 seconds. The change may have been made. Read the record before you send the statement again.`)
	if n := e.fake.count("PUT", "/features/DEMO-1"); n != 1 {
		t.Fatalf("ApiUpdate calls: %d", n)
	}
}

func TestWriteRequestThatAhaRejects(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.Answer("PUT", "/features/DEMO-1", 422, -1, nil, `{"error":"Name is too long"}`)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'`),
		`aha: API error (status 422) on PUT /features/DEMO-1: {"error":"Name is too long"}`)
}
