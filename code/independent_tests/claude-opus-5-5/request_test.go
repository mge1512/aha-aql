// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// aha-request and request-account (http mode).

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

// EXAMPLE: request_address_and_headers
func TestRequestAddressAndHeaders(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id FROM features WHERE id = 'demo-3'"),
		`{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`)
	cs := f.Calls()
	if len(cs) != 1 {
		t.Fatalf("calls %s", describeCalls(cs))
	}
	c := cs[0]
	if c.Method != "GET" || c.Host != "example.aha.io" || c.Path != "/api/v1/features/DEMO-3" {
		t.Errorf("request %s https://%s%s", c.Method, c.Host, c.Path)
	}
	if c.Header.Get("Authorization") != "Bearer fixture-key" || c.Header.Get("Accept") != "application/json" {
		t.Errorf("headers %v", c.Header)
	}
	ua := c.Header.Get("User-Agent")
	if !strings.HasPrefix(ua, "mcp-server-aha-aql/") || !strings.HasSuffix(ua, "(https://github.com/mge1512/aha-aql)") {
		t.Errorf("User-Agent %q", ua)
	}
	for _, h := range f.Connects() {
		if h != "example.aha.io:443" {
			t.Errorf("CONNECT %v", h)
		}
	}
}

// EXAMPLE: request_through_the_proxy_of_the_environment
// FakeAha is the proxy of the environment; its address takes the place of
// http://10.200.0.1:3128.
func TestRequestThroughTheProxyOfTheEnvironment(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id FROM features WHERE id = 'DEMO-3'"),
		`{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`)
	cs := f.Calls()
	if len(cs) != 1 || cs[0].Proxy != f.ProxyURL() {
		t.Errorf("calls %s", describeCalls(cs))
	}
	// https_proxy in lower case works in the same way
	f2 := newFakeAha(t)
	sb := newSandbox(t)
	env := []string{"HOME=" + sb.home, "SSL_CERT_FILE=" + sb.ca, "SSL_CERT_DIR=" + sb.cdir, "https_proxy=" + f2.ProxyURL()}
	r := run(t, append(env, fixtureEnv...), "", "query", "query=SELECT email FROM me")
	if r.stdout != `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`+"\n" {
		t.Errorf("stdout %q stderr %q", r.stdout, r.stderr)
	}
}

// EXAMPLE: host_of_aha_without_the_proxy
// The server connects to example.aha.io directly. The test does not know
// whether that host can be reached from the test machine; the trust store has
// only the test CA, so the request cannot succeed. FakeAha must see nothing.
func TestHostOfAhaWithoutTheProxy(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f, "NO_PROXY=.aha.io")
	res := s.query("SELECT id FROM features WHERE id = 'DEMO-3'")
	expectErrorPrefix(t, res, "cannot reach Aha!: ")
	if len(f.Calls()) != 0 || len(f.Connects()) != 0 {
		t.Errorf("the proxy was used: %v", f.Connects())
	}
}

// EXAMPLE: proxy_does_not_open_the_tunnel
func TestProxyDoesNotOpenTheTunnel(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.RefuseTunnel()
	s := fixtureServer(t, f)
	res := s.execute("UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'")
	expectError(t, res, "cannot reach Aha!: the proxy did not open a connection to example.aha.io.")
	if strings.Contains(res.Text, "127.0.0.1") {
		t.Errorf("the message has the address of the proxy")
	}
	expectNoWrite(t, f)
}

// EXAMPLE: too_many_requests_is_sent_again
// The clock cannot be replaced from outside: X-Ratelimit-Reset is two
// seconds from now, and the test checks that the server waited.
func TestTooManyRequestsIsSentAgain(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	reset := strconv.FormatInt(time.Now().Add(2*time.Second).Unix(), 10)
	f.Answer("GET", "/features/DEMO-3", 429, map[string]string{"X-Ratelimit-Reset": reset}, `{"error":"Too many requests"}`, 1)
	start := time.Now()
	expectResult(t, s.query("SELECT id FROM features WHERE id = 'DEMO-3'"),
		`{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`)
	if el := time.Since(start); el < 900*time.Millisecond || el > 20*time.Second {
		t.Errorf("waited %v, want about 1 to 3 seconds", el)
	}
	if n := countCalls(f, "GET", "/features/DEMO-3"); n != 2 {
		t.Errorf("%d calls, want 2", n)
	}
}

// EXAMPLE: too_many_requests_three_times (expected-long: 2 x 5 seconds)
func TestTooManyRequestsThreeTimes(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	f.Answer("GET", "/features/DEMO-3", 429, nil, `{"error":"Too many requests"}`, 0)
	start := time.Now()
	expectError(t, s.query("SELECT id FROM features WHERE id = 'DEMO-3'"), "aha: API error (status 429): too many requests")
	if el := time.Since(start); el < 9*time.Second {
		t.Errorf("waited %v, want 2 x 5 seconds", el)
	}
	if n := countCalls(f, "GET", "/features/DEMO-3"); n != 3 {
		t.Errorf("%d calls, want 3", n)
	}
}

// EXAMPLE: redirect_is_not_followed
func TestRedirectIsNotFollowed(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	f.Answer("GET", "/features/DEMO-3", 302, map[string]string{"Location": "https://other.example/api/v1/features/DEMO-3"}, "", 0)
	expectError(t, s.query("SELECT id FROM features WHERE id = 'DEMO-3'"), "aha: API error (status 302) on GET /features/DEMO-3: (no text)")
	cs := f.Calls()
	if len(cs) != 1 || cs[0].Host != "example.aha.io" {
		t.Errorf("calls %s", describeCalls(cs))
	}
}

// EXAMPLE: credential_that_aha_does_not_accept
func TestCredentialThatAhaDoesNotAccept(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, "AHA_SUBDOMAIN=example", "AHA_API_KEY=wrong-key"))
	s.initialize()
	s.listTools()
	res := s.query("SELECT email FROM me")
	expectError(t, res, "Aha! refused the request (status 403): the API key is not valid for the account example, or it has no permission for this record. Repeating the statement does not help.")
	if strings.Contains(res.Text, "wrong-key") {
		t.Errorf("the result has the credential")
	}
	s.closeAndWait()
	if strings.Contains(s.stderr.String(), "wrong-key") {
		t.Errorf("the diagnostic output has the credential")
	}
}

// EXAMPLE: write_request_without_an_answer (expected-long: 60 seconds)
func TestWriteRequestWithoutAnAnswer(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.NoAnswer("PUT", "/features/DEMO-1")
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'"),
		"cannot reach Aha!: no answer within 60 seconds. The change may have been made. Read the record before you send the statement again.")
	if n := countCalls(f, "PUT", "/features/DEMO-1"); n != 1 {
		t.Errorf("%d ApiUpdate calls, want 1", n)
	}
}

// ---- request-account ----

func httpFixture(t *testing.T, f *FakeAha, args ...string) *httpServer {
	t.Helper()
	sb := newSandbox(t)
	return startHTTP(t, sb.env(f), args...)
}

func httpCall(t *testing.T, h *httpServer, headers map[string]string, tool string, args map[string]any) (int, toolResult, rpcResponse, string) {
	t.Helper()
	if headers["Origin"] == "" {
		st, r, body := h.post(headers, initMsg())
		if st != 200 || r.Error != nil {
			return st, toolResult{}, r, body
		}
	}
	st, r, body := h.post(headers, callMsg(tool, args))
	if st != 200 || r.Error != nil || r.Result == nil {
		return st, toolResult{}, r, body
	}
	return st, parseToolResult(t, r.Result), r, body
}

// EXAMPLE: http_request_brings_its_account
func TestHTTPRequestBringsItsAccount(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f)
	st, res, _, body := httpCall(t, h, map[string]string{"Aha-Api-Key": "fixture-key", "Aha-Subdomain": "example"}, "query", map[string]any{"query": "SELECT email FROM me"})
	if st != 200 {
		t.Fatalf("status %d body %s", st, body)
	}
	expectResult(t, res, `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	cs := f.Calls()
	if len(cs) != 1 || cs[0].Method != "GET" || cs[0].Host != "example.aha.io" || cs[0].Path != "/api/v1/me" || cs[0].Header.Get("Authorization") != "Bearer fixture-key" {
		t.Errorf("calls %s", describeCalls(cs))
	}
}

func expectRefused(t *testing.T, st int, r rpcResponse, body string, wantStatus int, msg string) {
	t.Helper()
	if st != wantStatus {
		t.Errorf("status %d, want %d; body %s", st, wantStatus, body)
	}
	var m struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   *rpcError       `json:"error"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("body is not a JSON-RPC error: %s", body)
	}
	if m.Error == nil || m.Error.Code != -32600 || m.Error.Message != msg || string(m.ID) != "null" || m.Result != nil {
		t.Errorf("body %s, want the JSON-RPC error -32600 %q with id null", body, msg)
	}
	if strings.Contains(body, "fixture-key") {
		t.Errorf("the answer has the credential")
	}
}

// EXAMPLE: http_request_without_a_credential
func TestHTTPRequestWithoutACredential(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f, "subdomain=example")
	st, r, body := h.post(nil, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/list"})
	expectRefused(t, st, r, body, 400, "no credential. Send the header Aha-Api-Key with an Aha! API key")
	if strings.Contains(body, `"tools"`) {
		t.Errorf("the answer has a tool list")
	}
	st, r, body = h.post(map[string]string{"Aha-Api-Key": "two words"}, initMsg())
	expectRefused(t, st, r, body, 400, "no credential. Send the header Aha-Api-Key with an Aha! API key")
	expectNoCalls(t, f)
}

// EXAMPLE: http_request_for_another_account
func TestHTTPRequestForAnotherAccount(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f, "subdomain=example")
	st, r, body := h.post(map[string]string{"Aha-Api-Key": "fixture-key", "Aha-Subdomain": "other"}, callMsg("query", map[string]any{"query": "SELECT email FROM me"}))
	expectRefused(t, st, r, body, 403, "this server works with one Aha! account only")
	st, r, body = h.post(map[string]string{"Aha-Api-Key": "fixture-key", "Aha-Subdomain": "Not valid!"}, initMsg())
	expectRefused(t, st, r, body, 400, "Aha-Subdomain must be the account name before .aha.io")
	expectNoCalls(t, f)
	// the subdomain in another case is the same account
	st2, res, _, b2 := httpCall(t, h, map[string]string{"Aha-Api-Key": "fixture-key", "Aha-Subdomain": "EXAMPLE"}, "query", map[string]any{"query": "SELECT email FROM me"})
	if st2 != 200 || res.IsError {
		t.Errorf("status %d body %s", st2, b2)
	}
}

// request-account step 7: no subdomain anywhere.
func TestHTTPRequestWithoutAnAccount(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f)
	st, r, body := h.post(map[string]string{"Aha-Api-Key": "fixture-key"}, initMsg())
	expectRefused(t, st, r, body, 400, "no Aha! account. Send the header Aha-Subdomain")
	expectNoCalls(t, f)
}

// EXAMPLE: http_request_from_a_web_page
func TestHTTPRequestFromAWebPage(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f, "subdomain=example")
	st, r, body := h.post(map[string]string{"Origin": "https://pages.example", "Aha-Api-Key": "fixture-key"}, initMsg())
	expectRefused(t, st, r, body, 403, "requests from a web page are not accepted")
	expectNoCalls(t, f)
}

// EXAMPLE: http_requests_share_nothing
func TestHTTPRequestsShareNothing(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f, "subdomain=example")
	st, res, _, body := httpCall(t, h, map[string]string{"Aha-Api-Key": "fixture-key"}, "query", map[string]any{"query": "SELECT key FROM products"})
	if st != 200 {
		t.Fatalf("status %d body %s", st, body)
	}
	expectResult(t, res, `{"entity":"products","scope":"all products","count":2,"records":[{"key":"DEMO"},{"key":"OTHER"}]}`)
	st, res, _, body = httpCall(t, h, map[string]string{"Aha-Api-Key": "other-key"}, "query", map[string]any{"query": "SELECT key FROM products"})
	if st != 200 {
		t.Fatalf("status %d body %s", st, body)
	}
	expectError(t, res, "Aha! refused the request (status 403): the API key is not valid for the account example, or it has no permission for this record. Repeating the statement does not help.")
	var byKey = map[string]int{}
	for _, c := range f.CallsTo("GET", "/api/v1/products") {
		byKey[c.Header.Get("Authorization")]++
	}
	if byKey["Bearer fixture-key"] < 1 || byKey["Bearer other-key"] < 1 {
		t.Errorf("ApiList on /products per credential: %v", byKey)
	}
}

// EXAMPLE: definitions_are_not_kept_in_http_mode
func TestDefinitionsAreNotKeptInHTTPMode(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f, "subdomain=example")
	want := `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`
	for i := 0; i < 2; i++ {
		st, res, _, body := httpCall(t, h, map[string]string{"Aha-Api-Key": "fixture-key"}, "query", map[string]any{"query": "SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'"})
		if st != 200 {
			t.Fatalf("status %d body %s", st, body)
		}
		expectResult(t, res, want)
	}
	if n := countCalls(f, "GET", "/custom_field_definitions"); n != 2 {
		t.Errorf("%d ApiList calls on /custom_field_definitions, want 2", n)
	}
}

// Template EXAMPLE minimal_tool_server_http and declared_tool_execution_error
// in http mode: the tool list per request has the ExampleKey of the request,
// and an AqlError is a tool result in an HTTP 200 answer.
func TestHTTPToolListAndToolError(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	h := httpFixture(t, f, "subdomain=example", "default-product=DEMO")
	headers := map[string]string{"Aha-Api-Key": "fixture-key"}
	st, r, body := h.post(headers, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "tools/list"})
	if st != 200 || r.Error != nil {
		t.Fatalf("status %d body %s", st, body)
	}
	if !strings.Contains(string(r.Result), "for example 'DEMO-23'.") || strings.Contains(string(r.Result), "PROD") {
		t.Errorf("tool list %s", r.Result)
	}
	st, res, _, body := httpCall(t, h, headers, "query", map[string]any{"query": "SELECT id FROM tickets"})
	if st != 200 || !res.IsError || !strings.HasPrefix(res.Text, "unknown entity 'tickets'.") {
		t.Errorf("status %d result %+v body %s", st, res, body)
	}
	// the server remains running and accepts further requests
	st, res, _, _ = httpCall(t, h, headers, "query", map[string]any{"query": "SELECT email FROM me"})
	if st != 200 || res.IsError {
		t.Errorf("second request: status %d %+v", st, res)
	}
}
