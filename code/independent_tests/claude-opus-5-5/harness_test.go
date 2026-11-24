// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// Harness: builds the binary once into a temporary directory, creates a
// test certificate authority, and starts the binary in its three modes.
// Every test runs the binary with a clean environment (as `env -i`), with
// HOME in its own temporary directory, and with the proxy of FakeAha.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var (
	binaryPath string
	testPKI    struct {
		caPEM []byte
		leaf  tls.Certificate
	}
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	dir, err := os.MkdirTemp("", "aha-aql-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkdtemp:", err)
		return 1
	}
	defer os.RemoveAll(dir)
	binaryPath = filepath.Join(dir, "mcp-server-aha-aql")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = filepath.Join("..", "..")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build of the binary under test failed:", err)
		return 1
	}
	if err := makePKI(); err != nil {
		fmt.Fprintln(os.Stderr, "test PKI:", err)
		return 1
	}
	return m.Run()
}

func makePKI() error {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "FakeAha test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	caCert, _ := x509.ParseCertificate(caDER)
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "*.aha.io"},
		DNSNames:     []string{"*.aha.io"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	testPKI.caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	testPKI.leaf = tls.Certificate{Certificate: [][]byte{leafDER, caDER}, PrivateKey: leafKey}
	return nil
}

// sandbox is the temporary directory of one test.
type sandbox struct {
	t    *testing.T
	dir  string
	home string
	ca   string
	cdir string
}

func newSandbox(t *testing.T) *sandbox {
	t.Helper()
	dir := t.TempDir()
	sb := &sandbox{t: t, dir: dir, home: filepath.Join(dir, "home"), ca: filepath.Join(dir, "ca.pem"), cdir: filepath.Join(dir, "certs")}
	for _, d := range []string{sb.home, sb.cdir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	sb.writeFile("ca.pem", string(testPKI.caPEM), 0o644)
	return sb
}

// writeFile creates a file inside the sandbox and refuses any other place.
func (sb *sandbox) writeFile(name, content string, mode os.FileMode) string {
	sb.t.Helper()
	p := filepath.Join(sb.dir, name)
	rel, err := filepath.Rel(sb.dir, p)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		sb.t.Fatalf("refusing to write outside the sandbox: %s", p)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		sb.t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		sb.t.Fatal(err)
	}
	return p
}

// env is the complete environment of the binary: HOME, the trust store, the
// proxy of FakeAha (when given) and the variables of the test.
func (sb *sandbox) env(f *FakeAha, vars ...string) []string {
	e := []string{"HOME=" + sb.home, "SSL_CERT_FILE=" + sb.ca, "SSL_CERT_DIR=" + sb.cdir}
	if f != nil {
		e = append(e, "HTTPS_PROXY="+f.ProxyURL())
	}
	return append(e, vars...)
}

var fixtureEnv = []string{"AHA_SUBDOMAIN=example", "AHA_API_KEY=fixture-key"}

// ---- runs that end: manual mode and start errors ----

type runResult struct {
	stdout, stderr string
	code           int
}

func run(t *testing.T, env []string, stdin string, args ...string) runResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Env = env
	cmd.Dir = t.TempDir()
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return runResult{stdout: out.String(), stderr: errb.String(), code: code}
}

func hasLine(text, line string) bool {
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimRight(l, "\r") == line {
			return true
		}
	}
	return false
}

func expectLine(t *testing.T, what, text, line string) {
	t.Helper()
	if !hasLine(text, line) {
		t.Errorf("%s has no line\n  %q\ngot:\n%s", what, line, text)
	}
}

// ---- stdio mode ----

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type stdioServer struct {
	t      *testing.T
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan string
	stderr *syncBuf
	nextID int
	done   chan struct{}
	code   int
}

func startStdio(t *testing.T, env []string, args ...string) *stdioServer {
	t.Helper()
	cmd := exec.Command(binaryPath, append([]string{"stdio"}, args...)...)
	cmd.Env = env
	cmd.Dir = t.TempDir()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	s := &stdioServer{t: t, cmd: cmd, stdin: stdin, lines: make(chan string, 64), stderr: &syncBuf{}, done: make(chan struct{})}
	cmd.Stderr = s.stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			s.lines <- sc.Text()
		}
		close(s.lines)
	}()
	go func() {
		err := cmd.Wait()
		if ee, ok := err.(*exec.ExitError); ok {
			s.code = ee.ExitCode()
		} else if err != nil {
			s.code = -1
		}
		close(s.done)
	}()
	t.Cleanup(func() {
		_ = s.stdin.Close()
		select {
		case <-s.done:
		case <-time.After(10 * time.Second):
			_ = s.cmd.Process.Kill()
			<-s.done
		}
	})
	return s
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func (s *stdioServer) send(v any) {
	s.t.Helper()
	b, _ := json.Marshal(v)
	if _, err := s.stdin.Write(append(b, '\n')); err != nil {
		s.t.Fatalf("write to the server: %v", err)
	}
}

func (s *stdioServer) sendRaw(line string) {
	s.t.Helper()
	if _, err := io.WriteString(s.stdin, line+"\n"); err != nil {
		s.t.Fatalf("write to the server: %v", err)
	}
}

func (s *stdioServer) waitFor(id string) rpcResponse {
	s.t.Helper()
	timeout := time.After(150 * time.Second)
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				s.t.Fatalf("the server ended before the answer %s; stderr:\n%s", id, s.stderr.String())
			}
			var r rpcResponse
			if json.Unmarshal([]byte(line), &r) != nil {
				s.t.Fatalf("standard output has a line that is not JSON: %q", line)
			}
			if string(r.ID) == id {
				return r
			}
		case <-timeout:
			s.t.Fatalf("no answer %s; stderr:\n%s", id, s.stderr.String())
		}
	}
}

func (s *stdioServer) rpc(method string, params any) rpcResponse {
	s.t.Helper()
	s.nextID++
	msg := map[string]any{"jsonrpc": "2.0", "id": s.nextID, "method": method}
	if params != nil {
		msg["params"] = params
	}
	s.send(msg)
	return s.waitFor(fmt.Sprint(s.nextID))
}

func (s *stdioServer) initialize() {
	s.t.Helper()
	r := s.rpc("initialize", map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "independent-tests", "version": "0"}})
	if r.Error != nil {
		s.t.Fatalf("initialize: %v", r.Error.Message)
	}
	s.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
}

type toolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	} `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
}

func (s *stdioServer) listTools() []toolInfo {
	s.t.Helper()
	r := s.rpc("tools/list", map[string]any{})
	if r.Error != nil {
		s.t.Fatalf("tools/list: %v", r.Error.Message)
	}
	var res struct {
		Tools []toolInfo `json:"tools"`
	}
	if err := json.Unmarshal(r.Result, &res); err != nil {
		s.t.Fatalf("tools/list result: %v", err)
	}
	return res.Tools
}

type toolResult struct {
	Text    string
	IsError bool
}

func parseToolResult(t *testing.T, raw json.RawMessage) toolResult {
	t.Helper()
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("tool result: %v: %s", err, raw)
	}
	if len(res.Content) != 1 || res.Content[0].Type != "text" {
		t.Fatalf("the tool result must have exactly one text: %s", raw)
	}
	return toolResult{Text: res.Content[0].Text, IsError: res.IsError}
}

func (s *stdioServer) call(tool string, args map[string]any) toolResult {
	s.t.Helper()
	r := s.rpc("tools/call", map[string]any{"name": tool, "arguments": args})
	if r.Error != nil {
		s.t.Fatalf("tools/call %s answered with a JSON-RPC error: %d %s", tool, r.Error.Code, r.Error.Message)
	}
	return parseToolResult(s.t, r.Result)
}

func (s *stdioServer) query(q string) toolResult {
	s.t.Helper()
	return s.call("query", map[string]any{"query": q})
}

func (s *stdioServer) execute(st string) toolResult {
	s.t.Helper()
	return s.call("execute", map[string]any{"statement": st})
}

// closeAndWait ends standard input and returns the exit code.
func (s *stdioServer) closeAndWait() int {
	s.t.Helper()
	_ = s.stdin.Close()
	select {
	case <-s.done:
	case <-time.After(20 * time.Second):
		s.t.Fatalf("the server did not end after the end of standard input")
	}
	return s.code
}

// fixtureServer starts the server in stdio mode for the fixture account,
// lists the tools (so that the ExampleKey is chosen), and resets the record
// of FakeAha: the calls that an example counts are the calls of its statement.
func fixtureServer(t *testing.T, f *FakeAha, extraEnv ...string) *stdioServer {
	t.Helper()
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, append(append([]string{}, fixtureEnv...), extraEnv...)...))
	s.initialize()
	s.listTools()
	f.Reset()
	return s
}

func expectResult(t *testing.T, got toolResult, want string) {
	t.Helper()
	if got.IsError {
		t.Errorf("tool result is an error: %s\nwant: %s", got.Text, want)
		return
	}
	if got.Text != want {
		t.Errorf("tool result\n got: %s\nwant: %s", got.Text, want)
	}
}

func expectError(t *testing.T, got toolResult, want string) {
	t.Helper()
	if !got.IsError {
		t.Errorf("tool result is not marked as an error: %s\nwant error: %s", got.Text, want)
		return
	}
	if got.Text != want {
		t.Errorf("error message\n got: %s\nwant: %s", got.Text, want)
	}
}

func expectErrorPrefix(t *testing.T, got toolResult, prefix string) {
	t.Helper()
	if !got.IsError {
		t.Errorf("tool result is not marked as an error: %s", got.Text)
		return
	}
	if !strings.HasPrefix(got.Text, prefix) {
		t.Errorf("error message\n got: %s\nwant prefix: %s", got.Text, prefix)
	}
}

func expectNoCalls(t *testing.T, f *FakeAha) {
	t.Helper()
	if c := f.Calls(); len(c) != 0 {
		t.Errorf("FakeAha recorded %d calls, want none: %s", len(c), describeCalls(c))
	}
}

func expectNoWrite(t *testing.T, f *FakeAha) {
	t.Helper()
	if c := f.WriteCalls(); len(c) != 0 {
		t.Errorf("FakeAha recorded write requests: %s", describeCalls(c))
	}
}

func describeCalls(cs []Call) string {
	var parts []string
	for _, c := range cs {
		parts = append(parts, c.Method+" "+c.Path)
	}
	return strings.Join(parts, ", ")
}

// expectCalls checks the exact list of calls "METHOD /route" in order.
func expectCalls(t *testing.T, f *FakeAha, want ...string) {
	t.Helper()
	var got []string
	for _, c := range f.Calls() {
		got = append(got, c.Method+" "+strings.TrimPrefix(c.Path, "/api/v1"))
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("FakeAha calls\n got: %v\nwant: %v", got, want)
	}
}

func countCalls(f *FakeAha, method, route string) int {
	return len(f.CallsTo(method, "/api/v1"+route))
}

// decodeOrdered decodes a JSON object and returns its member names in order.
func memberNames(t *testing.T, raw string) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(raw))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		t.Fatalf("not a JSON object: %s", raw)
	}
	var names []string
	for dec.More() {
		k, _ := dec.Token()
		names = append(names, k.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatalf("bad JSON: %s", raw)
		}
	}
	return names
}

// firstRecord returns the raw text of records[0] of a query result.
func records(t *testing.T, text string) []json.RawMessage {
	t.Helper()
	var res struct {
		Records []json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal([]byte(text), &res); err != nil {
		t.Fatalf("result is not JSON: %s", text)
	}
	return res.Records
}

// ---- http mode ----

type httpServer struct {
	t      *testing.T
	cmd    *exec.Cmd
	addr   string
	stderr *syncBuf
	done   chan struct{}
	code   int
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func startHTTP(t *testing.T, env []string, args ...string) *httpServer {
	t.Helper()
	addr := freePort(t)
	cmd := exec.Command(binaryPath, append([]string{"http", "listen=" + addr}, args...)...)
	cmd.Env = env
	cmd.Dir = t.TempDir()
	h := &httpServer{t: t, cmd: cmd, addr: addr, stderr: &syncBuf{}, done: make(chan struct{})}
	cmd.Stderr = h.stderr
	cmd.Stdout = h.stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		err := cmd.Wait()
		if ee, ok := err.(*exec.ExitError); ok {
			h.code = ee.ExitCode()
		} else if err != nil {
			h.code = -1
		}
		close(h.done)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-h.done:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-h.done
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			c.Close()
			return h
		}
		select {
		case <-h.done:
			t.Fatalf("http server ended: code %d, stderr:\n%s", h.code, h.stderr.String())
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatalf("http server does not accept connections; stderr:\n%s", h.stderr.String())
	return nil
}

// post sends one JSON-RPC message to /mcp and returns the status and the
// JSON-RPC message of the answer (from a JSON body or an SSE stream).
func (h *httpServer) post(headers map[string]string, msg any) (int, rpcResponse, string) {
	h.t.Helper()
	b, _ := json.Marshal(msg)
	req, _ := http.NewRequest(http.MethodPost, "http://"+h.addr+"/mcp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := (&http.Client{Timeout: 150 * time.Second}).Do(req)
	if err != nil {
		h.t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	var r rpcResponse
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		for _, l := range strings.Split(text, "\n") {
			if strings.HasPrefix(l, "data:") {
				var x rpcResponse
				if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(l, "data:"))), &x) == nil && (x.Result != nil || x.Error != nil) {
					r = x
				}
			}
		}
	} else if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &r)
	}
	return resp.StatusCode, r, text
}

func initMsg() map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "t", "version": "0"}}}
}

func callMsg(tool string, args map[string]any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args}}
}
