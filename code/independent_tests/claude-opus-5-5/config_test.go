// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// load-configuration, manual-mode command line, and the template EXAMPLEs
// for the command line (unknown transport, version flag).

import (
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
)

// EXAMPLE: unknown_first_word, template EXAMPLE unknown_first_word_with_command_words
func TestUnknownFirstWord(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "", "websocket")
	expectLine(t, "stderr", r.stderr, "error: unknown transport 'websocket'. Valid: stdio, http, query, execute")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	r = run(t, sb.env(f, fixtureEnv...), "")
	expectLine(t, "stderr", r.stderr, "error: unknown transport ''. Valid: stdio, http, query, execute")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	expectNoCalls(t, f)
	if len(f.Connects()) != 0 {
		t.Errorf("the proxy got a connection")
	}
}

// MILESTONE 0.0.0 acceptance: --version shows the spec hash.
func TestVersionFlag(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	r := run(t, sb.env(nil), "", "--version")
	if r.code != 0 {
		t.Fatalf("exit code %d, stderr %s", r.code, r.stderr)
	}
	if !regexp.MustCompile(`[0-9a-f]{64}`).MatchString(r.stdout) {
		t.Errorf("--version output has no spec hash: %q", r.stdout)
	}
	if !strings.Contains(r.stdout, "9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262") {
		t.Errorf("--version output has not the hash of this spec: %q", r.stdout)
	}
}

// EXAMPLE: account_from_the_environment
func TestAccountFromTheEnvironment(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, "AHA_SUBDOMAIN=Example", "AHA_API_KEY=fixture-key"))
	s.initialize()
	if len(f.Calls()) != 0 {
		t.Errorf("a request was sent before any tool call: %s", describeCalls(f.Calls()))
	}
	tools := s.listTools()
	if len(tools) != 2 {
		t.Errorf("read_only must be false: tools %v", tools)
	}
	for _, tl := range tools {
		if tl.Name == "query" && !strings.Contains(tl.Description, "for example 'PROD-23'.") {
			t.Errorf("default_product must have no value; description:\n%s", tl.Description)
		}
	}
	f.Reset()
	expectResult(t, s.query("SELECT email FROM me"), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	for _, c := range f.Calls() {
		if c.Host != "example.aha.io" || c.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("request with host %q and Authorization %q", c.Host, c.Header.Get("Authorization"))
		}
	}
}

// EXAMPLE: arguments_take_precedence
func TestArgumentsTakePrecedence(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	keyFile := sb.writeFile("aha-key", "fixture-key\n", 0o600)
	s := startStdio(t, sb.env(f, "AHA_SUBDOMAIN=other", "AHA_API_KEY=environment-key", "AHA_DEFAULT_PRODUCT=OTHER", "AHA_READ_ONLY=false"),
		"subdomain=example", "api-key-file="+keyFile, "default-product=DEMO", "read-only=true")
	s.initialize()
	tools := s.listTools()
	if len(tools) != 1 || tools[0].Name != "query" {
		t.Fatalf("read_only must be true: tools %v", tools)
	}
	if !strings.Contains(tools[0].Description, "id is the reference number of a record, for example 'DEMO-23'.\n") {
		t.Errorf("default_product must be DEMO; description:\n%s", tools[0].Description)
	}
	f.Reset()
	expectResult(t, s.query("SELECT email FROM me"), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	for _, c := range f.Calls() {
		if c.Host != "example.aha.io" || c.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("request with host %q and Authorization %q", c.Host, c.Header.Get("Authorization"))
		}
	}
}

// EXAMPLE: settings_from_the_environment
func TestSettingsFromTheEnvironment(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, "AHA_SUBDOMAIN=example", "AHA_API_KEY=fixture-key", "AHA_DEFAULT_PRODUCT=demo", "AHA_READ_ONLY=true"))
	s.initialize()
	tools := s.listTools()
	if len(tools) != 1 || tools[0].Name != "query" {
		t.Fatalf("tool list must be query alone: %v", tools)
	}
	expectLine(t, "description of query", tools[0].Description, "id is the reference number of a record, for example 'DEMO-23'.")
}

// EXAMPLE: read_only_variable_of_the_wrong_form
func TestReadOnlyVariableOfTheWrongForm(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, "AHA_SUBDOMAIN=example", "AHA_API_KEY=fixture-key", "AHA_READ_ONLY=1"), "", "stdio")
	expectLine(t, "stderr", r.stderr, "error: AHA_READ_ONLY must be true or false")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	expectNoCalls(t, f)
	// MILESTONE 0.1.0 acceptance: AHA_READ_ONLY=maybe
	r = run(t, sb.env(nil, "AHA_READ_ONLY=maybe"), "", "stdio", "subdomain=example")
	expectLine(t, "stderr", r.stderr, "error: AHA_READ_ONLY must be true or false")
}

// EXAMPLE: start_without_an_account
func TestStartWithoutAnAccount(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f), "", "stdio")
	expectLine(t, "stderr", r.stderr, "error: no Aha! account. Use subdomain=NAME or set AHA_SUBDOMAIN")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	r = run(t, sb.env(f), "", "stdio", "subdomain=example")
	expectLine(t, "stderr", r.stderr, "error: no credential. Use api-key-file=PATH or set AHA_API_KEY")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	expectNoCalls(t, f)
}

// EXAMPLE: key_file_that_other_users_can_read
func TestKeyFileThatOtherUsersCanRead(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	keyFile := sb.writeFile("aha-key", "fixture-key", 0o644)
	r := run(t, sb.env(nil), "", "stdio", "subdomain=example", "api-key-file="+keyFile)
	expectLine(t, "stderr", r.stderr, "error: "+keyFile+" can be read by other users. Run: chmod 600 "+keyFile)
	if strings.Contains(r.stderr, "fixture-key") {
		t.Errorf("stderr has the credential")
	}
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
}

// EXAMPLE: credential_is_not_an_option
func TestCredentialIsNotAnOption(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	r := run(t, sb.env(nil, "AHA_SUBDOMAIN=example"), "", "stdio", "api-key=fixture-key")
	expectLine(t, "stderr", r.stderr, "error: unknown option 'api-key' for stdio. Options: subdomain, api-key-file, default-product, read-only")
	if strings.Contains(r.stderr, "fixture-key") {
		t.Errorf("stderr has the credential")
	}
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
}

// EXAMPLE: values_of_the_wrong_form
func TestValuesOfTheWrongForm(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	r := run(t, sb.env(nil, "AHA_API_KEY=fixture-key"), "", "stdio", "subdomain=https://example.aha.io")
	expectLine(t, "stderr", r.stderr, "error: subdomain must be the account name before .aha.io, for example mycompany")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	r = run(t, sb.env(nil, "AHA_API_KEY=fixture-key"), "", "stdio", "subdomain=example", "read-only=yes")
	expectLine(t, "stderr", r.stderr, "error: read-only must be true or false")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
}

// Start messages G01, G03, G04 (variable), G07 of load-configuration.
func TestStartMessages(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	cases := []struct {
		env  []string
		args []string
		line string
	}{
		{[]string{"AHA_API_KEY=fixture-key"}, []string{"stdio", "subdomain=example", "readonly"}, "error: argument 2 is not key=value"},
		{[]string{"AHA_API_KEY=fixture-key"}, []string{"stdio", "=x"}, "error: argument 1 is not key=value"},
		{[]string{"AHA_API_KEY=fixture-key"}, []string{"stdio", "subdomain=example", "subdomain=example"}, "error: option 'subdomain' is given twice"},
		{[]string{"AHA_API_KEY=fixture-key", "AHA_SUBDOMAIN=my company"}, []string{"stdio"}, "error: AHA_SUBDOMAIN must be the account name before .aha.io, for example mycompany"},
		{[]string{"AHA_API_KEY=fixture-key"}, []string{"stdio", "subdomain=example", "default-product="}, "error: default-product must be a product key, id or name"},
		{[]string{"AHA_API_KEY=fixture-key"}, []string{"stdio", "subdomain=example", "api-key-file="}, "error: api-key-file must be a file path"},
		{nil, []string{"stdio", "subdomain=example", "api-key-file=" + sb.dir + "/missing"}, "error: no usable credential in " + sb.dir + "/missing: no such file"},
		{nil, []string{"stdio", "subdomain=example", "api-key-file=" + sb.dir}, "error: no usable credential in " + sb.dir + ": not a regular file"},
		{[]string{"AHA_API_KEY=two words"}, []string{"stdio", "subdomain=example"}, "error: no usable credential in AHA_API_KEY: not a single value of at most 4096 printable characters"},
		{[]string{"AHA_API_KEY=x"}, []string{"query", "subdomain=example", "dry-run=true", "query=SELECT email FROM me"}, "error: unknown option 'dry-run' for query. Options: subdomain, api-key-file, default-product, read-only, query"},
		{[]string{"AHA_API_KEY=x"}, []string{"execute", "subdomain=example", "query=x"}, "error: unknown option 'query' for execute. Options: subdomain, api-key-file, default-product, read-only, dry-run, statement"},
		{nil, []string{"http", "statement=x"}, "error: unknown option 'statement' for http. Options: subdomain, default-product, read-only, listen"},
	}
	empty := sb.writeFile("empty-key", "  \n", 0o600)
	cases = append(cases, struct {
		env  []string
		args []string
		line string
	}{nil, []string{"stdio", "subdomain=example", "api-key-file=" + empty}, "error: no usable credential in " + empty + ": empty"})
	for _, c := range cases {
		r := run(t, sb.env(nil, c.env...), "", c.args...)
		expectLine(t, "stderr of "+strings.Join(c.args, " "), r.stderr, c.line)
		if r.code != 2 {
			t.Errorf("%v: exit code %d, want 2", c.args, r.code)
		}
		if strings.Contains(r.stderr, "fixture-key") || strings.Contains(r.stderr, "two words") {
			t.Errorf("%v: stderr has the credential", c.args)
		}
	}
}

// MILESTONE 0.1.0 acceptance criterion with mktemp + chmod 644.
func TestKeyFilePermissionAcceptance(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	p := sb.writeFile("k", "k\n", 0o644)
	r := run(t, sb.env(nil), "", "stdio", "subdomain=example", "api-key-file="+p)
	if !strings.Contains(r.stderr, "can be read by other users. Run: chmod 600") {
		t.Errorf("stderr: %s", r.stderr)
	}
}

// EXAMPLE: http_mode_uses_no_credential_of_the_process
func TestHTTPModeUsesNoCredentialOfTheProcess(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	h := startHTTP(t, sb.env(f, fixtureEnv...))
	time.Sleep(300 * time.Millisecond)
	errText := h.stderr.String()
	if !strings.Contains(errText, "AHA_API_KEY") {
		t.Errorf("no warning about AHA_API_KEY; stderr:\n%s", errText)
	}
	if strings.Contains(errText, "fixture-key") {
		t.Errorf("the warning has the credential")
	}
	expectNoCalls(t, f)

	r := run(t, sb.env(f, fixtureEnv...), "", "http", "api-key-file="+sb.dir+"/aha-key")
	expectLine(t, "stderr", r.stderr, "error: http mode has no option api-key-file. Each request brings its credential in the header Aha-Api-Key")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}

	h2 := startHTTP(t, sb.env(f))
	time.Sleep(300 * time.Millisecond)
	if strings.Contains(strings.ToLower(h2.stderr.String()), "warning") {
		t.Errorf("a warning without AHA variables:\n%s", h2.stderr.String())
	}
	expectNoCalls(t, f)
}

// Template EXAMPLE: http_bind_error
func TestHTTPBindError(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	r := run(t, sb.env(nil), "", "http", "listen="+l.Addr().String(), "subdomain=example")
	if r.code != 1 {
		t.Errorf("exit code %d, want 1; stderr %s", r.code, r.stderr)
	}
	if strings.TrimSpace(r.stderr) == "" {
		t.Errorf("no diagnostic on stderr")
	}
}

// Template EXAMPLE: graceful_shutdown_http
func TestGracefulShutdownHTTP(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	h := startHTTP(t, sb.env(nil), "subdomain=example")
	_ = h.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-h.done:
	case <-time.After(35 * time.Second):
		t.Fatal("the server did not end after SIGTERM")
	}
	if h.code != 0 {
		t.Errorf("exit code %d after SIGTERM, want 0", h.code)
	}
}

// Template: listen on a non-loopback address writes a warning (load-configuration 6 d).
func TestListenWarning(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	addr := freePort(t)
	port := addr[strings.LastIndex(addr, ":")+1:]
	cmd := startHTTPAt(t, sb, "0.0.0.0:"+port)
	if !strings.Contains(strings.ToLower(cmd), "tls") {
		t.Errorf("no warning about TLS for a non-loopback address:\n%s", cmd)
	}
}

func startHTTPAt(t *testing.T, sb *sandbox, listen string) string {
	t.Helper()
	h := &httpServer{t: t, stderr: &syncBuf{}, done: make(chan struct{})}
	h.cmd = exec.Command(binaryPath, "http", "listen="+listen, "subdomain=example")
	h.cmd.Env = sb.env(nil)
	h.cmd.Stderr = h.stderr
	if err := h.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	_ = h.cmd.Process.Signal(syscall.SIGTERM)
	_ = h.cmd.Wait()
	return h.stderr.String()
}

// stdio: end of standard input ends the server with exit code 0
// (template EXAMPLE minimal_tool_server_stdio, exit_code = 0 on EOF).
func TestStdioEndsOnEOF(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT email FROM me"), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	if code := s.closeAndWait(); code != 0 {
		t.Errorf("exit code %d on EOF, want 0", code)
	}
}

// Template EXAMPLE tool_error_handling: protocol errors are JSON-RPC errors,
// and the server goes on.
func TestProtocolErrorsAreJSONRPCErrors(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	r := s.rpc("tools/call", map[string]any{"name": "bogus", "arguments": map[string]any{}})
	if r.Error == nil {
		t.Errorf("an unknown tool must give a JSON-RPC error")
	}
	s.sendRaw("{not json")
	r2 := s.waitFor("null")
	if r2.Error == nil || r2.Error.Code != -32700 {
		t.Errorf("a parse error must give the JSON-RPC error -32700: %+v", r2.Error)
	}
	expectResult(t, s.query("SELECT email FROM me"), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
}

var _ = os.Getpid
