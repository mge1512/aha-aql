// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268
// tests by: claude-opus-5-5

package aqltests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// startAndStop starts the server in stdio mode with no MCP message and
// returns its exit code and diagnostic output after EOF.
func (e *env) startAndStop(extra map[string]string, args ...string) runResult {
	e.t.Helper()
	return runBin(e.t, e.dir, e.vars(extra), "", 30*time.Second, append([]string{"stdio"}, args...)...)
}

func (e *env) wantNoRequest() {
	e.t.Helper()
	if len(e.fake.Calls()) != 0 || len(e.fake.Connects()) != 0 {
		e.t.Fatalf("a request was sent: %s (connects %d)", e.fake.describeCalls(), len(e.fake.Connects()))
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "--version")
	if r.code != 0 || !strings.Contains(r.stdout+r.stderr, "spec:"+specSHA) || !regexp.MustCompile(`[0-9a-f]{64}`).MatchString(r.stdout) {
		t.Fatalf("version: code %d out %q err %q", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "mcp-server-aha-aql") {
		t.Fatalf("version: %q", r.stdout)
	}
	e.wantNoRequest()
}

func TestUnknownFirstWord(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "websocket")
	if r.code != 2 || !hasLine(r.stderr, `error: unknown transport 'websocket'. Valid: stdio, http, query, execute`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.run("", nil)
	if r.code != 2 || !hasLine(r.stderr, `error: unknown transport ''. Valid: stdio, http, query, execute`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.run("", nil, "bogus")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: unknown transport 'bogus'. Valid: stdio, http, query, execute` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestAccountFromTheEnvironment(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdioRaw(map[string]string{"AHA_SUBDOMAIN": "Example"})
	time.Sleep(300 * time.Millisecond)
	e.wantNoRequest()
	s.initialize()
	tools := s.listTools()
	if len(tools) != 2 {
		t.Fatalf("read_only must be false: %d tools", len(tools))
	}
	if !strings.Contains(tools[0].Description+tools[1].Description, "'PROD-23'") {
		t.Fatal("default_product must have no value")
	}
	e.fake.Reset()
	wantResult(t, s.query(`SELECT email FROM me`), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	for _, c := range e.fake.Connects() {
		if c.Host != "example.aha.io:443" {
			t.Fatalf("CONNECT %s", c.Host)
		}
	}
	if c := e.fake.Calls(); len(c) != 1 || c[0].Header.Get("Authorization") != "Bearer fixture-key" {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestArgumentsTakePrecedence(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	keyFile := writeSandboxFile(t, e.dir, "aha-key", []byte("fixture-key\n"), 0o600)
	extra := map[string]string{"AHA_SUBDOMAIN": "other", "AHA_API_KEY": "environment-key", "AHA_DEFAULT_PRODUCT": "OTHER", "AHA_READ_ONLY": "false"}
	s := e.stdioRaw(extra, "subdomain=example", "api-key-file="+keyFile, "default-product=DEMO", "read-only=true")
	s.initialize()
	tools := s.listTools()
	if len(tools) != 1 || tools[0].Name != "query" {
		t.Fatalf("read_only must be true: %+v", tools)
	}
	if !strings.Contains(tools[0].Description, "id is the reference number of a record, for example 'DEMO-23'.") {
		t.Fatalf("default_product must be DEMO: %s", tools[0].Description)
	}
	e.fake.Reset()
	wantResult(t, s.query(`SELECT email FROM me`), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	if c := e.fake.Calls(); len(c) != 1 || c[0].Header.Get("Authorization") != "Bearer fixture-key" {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestSettingsFromTheEnvironment(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdioRaw(map[string]string{"AHA_DEFAULT_PRODUCT": "demo", "AHA_READ_ONLY": "true"})
	s.initialize()
	tools := s.listTools()
	if len(tools) != 1 || tools[0].Name != "query" {
		t.Fatalf("tools: %+v", tools)
	}
	if !hasLine(tools[0].Description, "id is the reference number of a record, for example 'DEMO-23'.") {
		t.Fatalf("description: %s", tools[0].Description)
	}
}

func TestReadOnlyVariableOfTheWrongForm(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.startAndStop(map[string]string{"AHA_READ_ONLY": "1"})
	if r.code != 2 || !hasLine(r.stderr, `error: AHA_READ_ONLY must be true or false`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(map[string]string{"AHA_READ_ONLY": "maybe"}, "subdomain=example")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: AHA_READ_ONLY must be true or false` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestStartWithoutAnAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	none := map[string]string{"AHA_SUBDOMAIN": "", "AHA_API_KEY": ""}
	r := e.startAndStop(none)
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: no Aha! account. Use subdomain=NAME or set AHA_SUBDOMAIN` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(none, "subdomain=example")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: no credential. Use api-key-file=PATH or set AHA_API_KEY` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestKeyFileThatOtherUsersCanRead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	keyFile := writeSandboxFile(t, e.dir, "aha-key", []byte("fixture-key"), 0o644)
	r := e.startAndStop(map[string]string{"AHA_API_KEY": ""}, "subdomain=example", "api-key-file="+keyFile)
	want := "error: " + keyFile + " can be read by other users. Run: chmod 600 " + keyFile
	if r.code != 2 || !hasLine(r.stderr, want) || strings.Contains(r.stderr, "fixture-key") {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestKeyFileThatCannotBeUsed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	missing := filepath.Join(e.dir, "missing-key")
	r := e.startAndStop(nil, "api-key-file="+missing)
	if r.code != 2 || strings.TrimSpace(r.stderr) != "error: no usable credential in "+missing+": no such file" {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	empty := writeSandboxFile(t, e.dir, "empty-key", []byte(" \n"), 0o600)
	r = e.startAndStop(nil, "api-key-file="+empty)
	if r.code != 2 || strings.TrimSpace(r.stderr) != "error: no usable credential in "+empty+": empty" {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	sub := filepath.Join(e.dir, "keydir")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	r = e.startAndStop(nil, "api-key-file="+sub)
	if r.code != 2 || strings.TrimSpace(r.stderr) != "error: no usable credential in "+sub+": not a regular file" {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	spaced := writeSandboxFile(t, e.dir, "spaced-key", []byte("two words\n"), 0o600)
	r = e.startAndStop(nil, "api-key-file="+spaced)
	if r.code != 2 || strings.TrimSpace(r.stderr) != "error: no usable credential in "+spaced+": not a single value of at most 4096 printable characters" {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(map[string]string{"AHA_API_KEY": "bad key"})
	if r.code != 2 || strings.TrimSpace(r.stderr) != "error: no usable credential in AHA_API_KEY: not a single value of at most 4096 printable characters" ||
		strings.Contains(r.stderr, "bad key") {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestCredentialIsNotAnOption(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.startAndStop(map[string]string{"AHA_API_KEY": ""}, "api-key=fixture-key")
	if r.code != 2 || !hasLine(r.stderr, `error: unknown option 'api-key' for stdio. Options: subdomain, api-key-file, default-product, read-only`) ||
		strings.Contains(r.stderr, "fixture-key") {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestValuesOfTheWrongForm(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.startAndStop(map[string]string{"AHA_SUBDOMAIN": ""}, "subdomain=https://example.aha.io")
	if r.code != 2 || !hasLine(r.stderr, `error: subdomain must be the account name before .aha.io, for example mycompany`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(nil, "subdomain=example", "read-only=yes")
	if r.code != 2 || !hasLine(r.stderr, `error: read-only must be true or false`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(map[string]string{"AHA_SUBDOMAIN": "not valid"})
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: AHA_SUBDOMAIN must be the account name before .aha.io, for example mycompany` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(nil, "default-product=")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: default-product must be a product key, id or name` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(nil, "api-key-file=")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: api-key-file must be a file path` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestArgumentsOfTheWrongForm(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.startAndStop(nil, "subdomain=example", "verbose")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: argument 2 is not key=value` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(nil, "=x")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: argument 1 is not key=value` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(nil, "subdomain=example", "subdomain=other")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: option 'subdomain' is given twice` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(nil, "bogus=1")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: unknown option 'bogus' for stdio. Options: subdomain, api-key-file, default-product, read-only` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.run("", nil, "query", "listen=127.0.0.1:1")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: unknown option 'listen' for query. Options: subdomain, api-key-file, default-product, read-only, query` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.run("", nil, "execute", "bogus=1")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: unknown option 'bogus' for execute. Options: subdomain, api-key-file, default-product, read-only, dry-run, statement` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.run("", nil, "http", "bogus=1")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: unknown option 'bogus' for http. Options: subdomain, default-product, read-only, listen` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestHTTPModeUsesNoCredentialOfTheProcess(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(nil)
	time.Sleep(300 * time.Millisecond)
	if !hasLine(s.stderr.String(), `warning: AHA_API_KEY is not read in http mode. Each request brings its credential in the header Aha-Api-Key`) ||
		strings.Contains(s.stderr.String(), "fixture-key") {
		t.Fatalf("stderr: %q", s.stderr.String())
	}
	e.wantNoRequest()
	r := e.run("", nil, "http", "api-key-file="+filepath.Join(e.dir, "aha-key"))
	if r.code != 2 || !hasLine(r.stderr, `error: http mode has no option api-key-file. Each request brings its credential in the header Aha-Api-Key`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	s2 := e.http(map[string]string{"AHA_SUBDOMAIN": "", "AHA_API_KEY": ""})
	time.Sleep(300 * time.Millisecond)
	if strings.Contains(s2.stderr.String(), "warning") {
		t.Fatalf("stderr: %q", s2.stderr.String())
	}
}

func TestListenAddressThatIsNotLoopback(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := runBin(t, e.dir, e.vars(map[string]string{"AHA_API_KEY": ""}), "", 30*time.Second, "http", "listen=192.0.2.1:8080")
	if !hasLine(r.stderr, `warning: the listen address is not a loopback address. Every request has a credential: put a TLS endpoint in front of the server`) {
		t.Fatalf("stderr %q", r.stderr)
	}
	if r.code != 1 {
		t.Fatalf("bind of an address the machine does not have: code %d", r.code)
	}
	e.wantNoRequest()
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	time.Sleep(300 * time.Millisecond)
	for _, l := range lines(s.stderr.String()) {
		if strings.HasPrefix(l, "warning:") {
			t.Fatalf("warning: %q", l)
		}
	}
}

func TestHTTPBindError(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""})
	addr := strings.TrimSuffix(strings.TrimPrefix(s.base, "http://"), "/mcp")
	r := runBin(t, e.dir, e.vars(map[string]string{"AHA_API_KEY": ""}), "", 30*time.Second, "http", "listen="+addr)
	if r.code != 1 || strings.TrimSpace(r.stderr) == "" {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
}
