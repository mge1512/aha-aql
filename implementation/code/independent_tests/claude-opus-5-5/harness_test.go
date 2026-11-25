// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e
// tests by: claude-opus-5-5

// The harness starts the binary mcp-server-aha-aql as a user starts it and
// talks to it through its documented interface only: MCP over standard
// input and output, MCP over HTTP, and the command words query and
// execute. It never imports the code of the server.
package aqltests

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const specSHA = "caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e"

var binPath string

// TestMain builds the binary from the project root (two directories up)
// into a temporary directory and removes it after the run.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "aha-aql-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binPath = filepath.Join(dir, binaryName())
	if p := os.Getenv("AHA_AQL_TEST_BINARY"); p != "" {
		binPath = p
	} else {
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Dir = filepath.Join("..", "..")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "build failed: %v\n%s", err, out)
			os.RemoveAll(dir)
			os.Exit(1)
		}
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// ---- the sandbox of one test ----

type env struct {
	t    *testing.T
	dir  string
	fake *FakeAha
}

func newEnv(t *testing.T) *env {
	t.Helper()
	skipWithoutFakeAha(t)
	dir := t.TempDir()
	return &env{t: t, dir: dir, fake: NewFakeAha(t, dir)}
}

// binaryName is the name of the binary on the platform of the test. On
// Windows it is mcp-server-aha-aql.exe.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "mcp-server-aha-aql.exe"
	}
	return "mcp-server-aha-aql"
}

// skipWithoutFakeAha skips a test that needs FakeAha on a platform where
// the server does not take its trust store from SSL_CERT_FILE. Only Linux
// does; on macOS and Windows the server checks the certificate of FakeAha
// against the trust store of the system, which does not have the
// certificate authority of the test (spec: "Platforms of the tests").
func skipWithoutFakeAha(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skipf("needs FakeAha: on %s the server ignores SSL_CERT_FILE and uses the trust store of the system", runtime.GOOS)
	}
}

// newLocalEnv is the sandbox of a test that sends no request to Aha!: the
// command line, the configuration, the version and the start of the MCP
// protocol. Such a test runs on every platform. FakeAha is still the proxy
// of the environment, so that a request that should not happen is seen:
// when the server opens a tunnel, the test fails.
func newLocalEnv(t *testing.T) *env {
	t.Helper()
	dir := t.TempDir()
	e := &env{t: t, dir: dir, fake: NewFakeAha(t, dir)}
	t.Cleanup(func() {
		if n := e.fake.Tunnels(); n > 0 {
			t.Errorf("a test that runs on every platform sent %d request(s) to Aha!; it must use newEnv", n)
		}
	})
	return e
}

// vars gives the environment of the server process: HOME in the sandbox,
// the proxy and the trust store of the test, and the fixture account.
// A value "" in extra removes a variable.
func (e *env) vars(extra map[string]string) []string {
	m := map[string]string{
		"HOME":          e.dir,
		"HTTPS_PROXY":   e.fake.ProxyURL,
		"SSL_CERT_FILE": e.fake.CAFile,
		"AHA_SUBDOMAIN": "example",
		"AHA_API_KEY":   "fixture-key",
	}
	for k, v := range extra {
		if v == "" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	var out []string
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// ---- manual mode ----

type runResult struct {
	stdout, stderr string
	code           int
}

func (e *env) run(stdin string, extra map[string]string, args ...string) runResult {
	e.t.Helper()
	return runBin(e.t, e.dir, e.vars(extra), stdin, 150*time.Second, args...)
}

func runBin(t *testing.T, dir string, vars []string, stdin string, timeout time.Duration, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	cmd.Env = vars
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("command did not end: %v", args)
	}
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("wait: %v", err)
	}
	return runResult{stdout: out.String(), stderr: errb.String(), code: code}
}

// ---- MCP over stdio ----

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcMsg struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type stdioSrv struct {
	t      *testing.T
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan string
	stderr *syncBuf
	nextID int
	exited chan error
	mu     sync.Mutex
}

// stdio starts the server in stdio mode, initializes the session and
// answers the tool list once, as the INTERFACES section assumes. The
// recorded calls are then forgotten.
func (e *env) stdio(extra map[string]string, args ...string) *stdioSrv {
	e.t.Helper()
	s := e.stdioRaw(extra, args...)
	s.initialize()
	s.listTools()
	e.fake.Reset()
	return s
}

func (e *env) stdioRaw(extra map[string]string, args ...string) *stdioSrv {
	e.t.Helper()
	cmd := exec.Command(binPath, append([]string{"stdio"}, args...)...)
	cmd.Dir = e.dir
	cmd.Env = e.vars(extra)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		e.t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.t.Fatal(err)
	}
	s := &stdioSrv{t: e.t, cmd: cmd, stdin: stdin, lines: make(chan string, 100), stderr: &syncBuf{}, exited: make(chan error, 1)}
	cmd.Stderr = s.stderr
	if err := cmd.Start(); err != nil {
		e.t.Fatal(err)
	}
	go readLines(stdout, s.lines)
	go func() { s.exited <- cmd.Wait() }()
	e.t.Cleanup(func() {
		_ = s.stdin.Close()
		select {
		case <-s.exited:
		case <-time.After(5 * time.Second):
			_ = s.cmd.Process.Kill()
		}
	})
	return s
}

func (s *stdioSrv) send(v any) {
	s.t.Helper()
	b, _ := json.Marshal(v)
	if _, err := s.stdin.Write(append(b, '\n')); err != nil {
		s.t.Fatalf("write to server: %v", err)
	}
}

func (s *stdioSrv) request(method string, params any, timeout time.Duration) rpcMsg {
	s.t.Helper()
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.mu.Unlock()
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	s.send(msg)
	deadline := time.After(timeout)
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				s.t.Fatalf("server closed stdout; stderr: %s", s.stderr.String())
			}
			var m rpcMsg
			if json.Unmarshal([]byte(line), &m) != nil {
				s.t.Fatalf("stdout line is not JSON-RPC: %q", line)
			}
			if string(m.ID) == fmt.Sprint(id) {
				return m
			}
		case <-deadline:
			s.t.Fatalf("no answer to %s within %v; stderr: %s", method, timeout, s.stderr.String())
		}
	}
}

func (s *stdioSrv) initialize() {
	s.t.Helper()
	m := s.request("initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "aqltests", "version": "0"},
	}, 30*time.Second)
	if m.Error != nil {
		s.t.Fatalf("initialize: %v", m.Error.Message)
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

func (s *stdioSrv) listTools() []toolInfo {
	s.t.Helper()
	m := s.request("tools/list", map[string]any{}, 120*time.Second)
	return parseTools(s.t, m)
}

func parseTools(t *testing.T, m rpcMsg) []toolInfo {
	t.Helper()
	if m.Error != nil {
		t.Fatalf("tools/list: %v", m.Error.Message)
	}
	var r struct {
		Tools []toolInfo `json:"tools"`
	}
	if err := json.Unmarshal(m.Result, &r); err != nil {
		t.Fatalf("tools/list result: %v", err)
	}
	return r.Tools
}

type toolResult struct {
	text    string
	isError bool
}

func parseToolResult(t *testing.T, m rpcMsg) toolResult {
	t.Helper()
	if m.Error != nil {
		t.Fatalf("tools/call answered with a JSON-RPC error: %d %s", m.Error.Code, m.Error.Message)
	}
	var r struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(m.Result, &r); err != nil {
		t.Fatalf("tool result: %v", err)
	}
	if len(r.Content) != 1 || r.Content[0].Type != "text" {
		t.Fatalf("tool result must have exactly one text: %s", m.Result)
	}
	return toolResult{text: r.Content[0].Text, isError: r.IsError}
}

func (s *stdioSrv) call(tool string, args map[string]any, timeout time.Duration) toolResult {
	s.t.Helper()
	m := s.request("tools/call", map[string]any{"name": tool, "arguments": args}, timeout)
	return parseToolResult(s.t, m)
}

func (s *stdioSrv) query(stmt string) toolResult {
	s.t.Helper()
	return s.call("query", map[string]any{"query": stmt}, 120*time.Second)
}

func (s *stdioSrv) execute(stmt string) toolResult {
	s.t.Helper()
	return s.call("execute", map[string]any{"statement": stmt}, 120*time.Second)
}

// closeAndWait closes standard input and gives the exit code.
func (s *stdioSrv) closeAndWait() int {
	s.t.Helper()
	_ = s.stdin.Close()
	select {
	case err := <-s.exited:
		s.exited <- err
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		if err != nil {
			s.t.Fatalf("wait: %v", err)
		}
		return 0
	case <-time.After(20 * time.Second):
		s.t.Fatalf("server did not end after EOF")
	}
	return -1
}

// ---- MCP over HTTP ----

type httpSrv struct {
	t      *testing.T
	base   string
	cmd    *exec.Cmd
	stderr *syncBuf
	exited chan error
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

func (e *env) http(extra map[string]string, args ...string) *httpSrv {
	e.t.Helper()
	addr := freePort(e.t)
	cmd := exec.Command(binPath, append([]string{"http", "listen=" + addr}, args...)...)
	cmd.Dir = e.dir
	cmd.Env = e.vars(extra)
	s := &httpSrv{t: e.t, base: "http://" + addr + "/mcp", cmd: cmd, stderr: &syncBuf{}, exited: make(chan error, 1)}
	cmd.Stderr = s.stderr
	cmd.Stdout = s.stderr
	if err := cmd.Start(); err != nil {
		e.t.Fatal(err)
	}
	go func() { s.exited <- cmd.Wait() }()
	e.t.Cleanup(func() {
		// Windows cannot send SIGTERM to a process; the process is ended
		// there at once.
		if runtime.GOOS == "windows" {
			_ = cmd.Process.Kill()
		} else {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
		select {
		case <-s.exited:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
		}
	})
	deadline := time.Now().Add(15 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("http server did not start; stderr: %s", s.stderr.String())
		}
		select {
		case err := <-s.exited:
			e.t.Fatalf("http server ended: %v; stderr: %s", err, s.stderr.String())
		case <-time.After(50 * time.Millisecond):
		}
	}
	return s
}

type httpAnswer struct {
	status int
	msg    rpcMsg
	raw    string
	header http.Header
}

type httpClient struct {
	s       *httpSrv
	headers map[string]string
	session string
	nextID  int
}

func (s *httpSrv) client(headers map[string]string) *httpClient {
	return &httpClient{s: s, headers: headers}
}

func (c *httpClient) post(body any, timeout time.Duration) httpAnswer {
	t := c.s.t
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.s.base, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	if c.session != "" {
		req.Header.Set("Mcp-Session-Id", c.session)
	}
	hc := &http.Client{Timeout: timeout}
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatalf("POST /mcp: %v", err)
	}
	defer resp.Body.Close()
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.session = sid
	}
	data, _ := io.ReadAll(resp.Body)
	a := httpAnswer{status: resp.StatusCode, raw: string(data), header: resp.Header.Clone()}
	payload := data
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		sc := bufio.NewScanner(bytes.NewReader(data))
		sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
		payload = nil
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "data:") {
				d := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				var m rpcMsg
				if json.Unmarshal([]byte(d), &m) == nil && len(m.ID) > 0 {
					payload = []byte(d)
				}
			}
		}
	}
	if len(bytes.TrimSpace(payload)) > 0 {
		_ = json.Unmarshal(payload, &a.msg)
	}
	return a
}

func (c *httpClient) request(method string, params any, timeout time.Duration) httpAnswer {
	c.nextID++
	msg := map[string]any{"jsonrpc": "2.0", "id": c.nextID, "method": method}
	if params != nil {
		msg["params"] = params
	}
	return c.post(msg, timeout)
}

func (c *httpClient) initialize() httpAnswer {
	a := c.request("initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "aqltests", "version": "0"},
	}, 30*time.Second)
	if a.status == 200 {
		c.post(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}, 30*time.Second)
	}
	return a
}

func (c *httpClient) query(stmt string) toolResult {
	c.s.t.Helper()
	a := c.request("tools/call", map[string]any{"name": "query", "arguments": map[string]any{"query": stmt}}, 120*time.Second)
	if a.status != 200 {
		c.s.t.Fatalf("tools/call: status %d: %s", a.status, a.raw)
	}
	return parseToolResult(c.s.t, a.msg)
}

// ---- assertions ----

func wantResult(t *testing.T, got toolResult, want string) {
	t.Helper()
	if got.isError {
		t.Fatalf("tool result is an error: %s\nwant: %s", got.text, want)
	}
	if got.text != want {
		t.Fatalf("tool result\n got: %s\nwant: %s", got.text, want)
	}
}

func wantError(t *testing.T, got toolResult, want string) {
	t.Helper()
	if !got.isError {
		t.Fatalf("tool result is not an error: %s\nwant error: %s", got.text, want)
	}
	if got.text != want {
		t.Fatalf("error message\n got: %s\nwant: %s", got.text, want)
	}
}

func wantErrorPrefix(t *testing.T, got toolResult, prefix string) {
	t.Helper()
	if !got.isError {
		t.Fatalf("tool result is not an error: %s", got.text)
	}
	if !strings.HasPrefix(got.text, prefix) {
		t.Fatalf("error message\n got: %s\nwant prefix: %s", got.text, prefix)
	}
}

func (e *env) wantNoCall() {
	e.t.Helper()
	if c := e.fake.Calls(); len(c) != 0 {
		e.t.Fatalf("FakeAha recorded calls: %s", e.fake.describeCalls())
	}
}

func (e *env) wantNoWrite() {
	e.t.Helper()
	if w := e.fake.writeCalls(); len(w) != 0 {
		e.t.Fatalf("FakeAha recorded a write request: %s %s", w[0].Method, w[0].Path)
	}
}

// wantCalls checks that the recorded calls are exactly these, each
// "METHOD /path", in this order.
func (e *env) wantCalls(want ...string) {
	e.t.Helper()
	var got []string
	for _, c := range e.fake.Calls() {
		got = append(got, c.Method+" "+c.Path)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		e.t.Fatalf("calls\n got: %v\nwant: %v", got, want)
	}
}

// wantCallSet checks the calls without regard to their order.
func (e *env) wantCallSet(want ...string) {
	e.t.Helper()
	count := map[string]int{}
	for _, c := range e.fake.Calls() {
		count[c.Method+" "+c.Path]++
	}
	for _, w := range want {
		count[w]--
	}
	for k, n := range count {
		if n != 0 {
			e.t.Fatalf("calls differ at %s (%+d); got: %s", k, n, e.fake.describeCalls())
		}
	}
}

func (e *env) lastBody(method, path string) string {
	e.t.Helper()
	var body string
	found := false
	for _, c := range e.fake.Calls() {
		if c.Method == method && c.Path == path {
			body = string(c.Body)
			found = true
		}
	}
	if !found {
		e.t.Fatalf("no %s call on %s; calls: %s", method, path, e.fake.describeCalls())
	}
	return body
}

func (e *env) wantBody(method, path, want string) {
	e.t.Helper()
	if got := e.lastBody(method, path); got != want {
		e.t.Fatalf("body of %s %s\n got: %s\nwant: %s", method, path, got, want)
	}
}

// orderedObject reads a JSON object and keeps the order of its members.
type member struct {
	key string
	val json.RawMessage
}

func orderedObject(t *testing.T, text string) []member {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(text))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		t.Fatalf("not a JSON object: %s", text)
	}
	var out []member
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			t.Fatalf("bad JSON object: %v", err)
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("bad JSON value: %v", err)
		}
		out = append(out, member{key: k.(string), val: v})
	}
	return out
}

func keysOf(ms []member) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.key)
	}
	return out
}

func memberOf(t *testing.T, ms []member, key string) json.RawMessage {
	t.Helper()
	for _, m := range ms {
		if m.key == key {
			return m.val
		}
	}
	t.Fatalf("no member %q in %v", key, keysOf(ms))
	return nil
}

func hasMember(ms []member, key string) bool {
	for _, m := range ms {
		if m.key == key {
			return true
		}
	}
	return false
}

// rows gives the rows of a read result, each with the order of its labels.
func rows(t *testing.T, text string) [][]member {
	t.Helper()
	ms := orderedObject(t, text)
	var raw []json.RawMessage
	if err := json.Unmarshal(memberOf(t, ms, "records"), &raw); err != nil {
		t.Fatal(err)
	}
	var out [][]member
	for _, r := range raw {
		out = append(out, orderedObject(t, string(r)))
	}
	return out
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func hasLine(s, line string) bool {
	for _, l := range strings.Split(s, "\n") {
		if l == line {
			return true
		}
	}
	return false
}

func waitFor(cond func() bool, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}
