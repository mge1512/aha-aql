// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e
// tests by: claude-opus-5-5

// Tests for spec version 0.24.0: the platforms of the tests. A test that
// needs FakeAha runs on Linux only (newEnv skips it elsewhere); a test that
// sends no request to Aha! runs on every platform (newLocalEnv), and it
// fails when the server opens a tunnel all the same.
package aqltests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// The start of the MCP protocol sends no request to Aha!: initialize and
// server/discover give the server instructions and the capability prompts
// on every platform.
func TestStartOfTheMCPProtocol(t *testing.T) {
	t.Parallel()
	e := newLocalEnv(t)
	vars := map[string]string{"AHA_DEFAULT_PRODUCT": "demo"}

	s := e.stdioRaw(vars)
	m := s.request("initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "aqltests", "version": "0"},
	}, 30*time.Second)
	if got := instructionsOf(t, m); got != serverInstructions {
		t.Fatalf("instructions\n got: %q\nwant: %q", got, serverInstructions)
	}
	var caps struct {
		Capabilities map[string]json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(m.Result, &caps); err != nil || caps.Capabilities["prompts"] == nil || caps.Capabilities["tools"] == nil {
		t.Fatalf("capabilities tools and prompts are not declared: %s", m.Result)
	}

	d := e.stdioRaw(vars)
	dm := d.request("server/discover", map[string]any{
		"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
			"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "aqltests", "version": "0"},
		},
	}, 30*time.Second)
	if got := instructionsOf(t, dm); got != serverInstructions {
		t.Fatalf("server/discover instructions\n got: %q\nwant: %q", got, serverInstructions)
	}
	e.wantNoRequest()
}

// On Windows the binary has the name mcp-server-aha-aql.exe; elsewhere it
// has no suffix.
func TestBinaryNameOfThePlatform(t *testing.T) {
	want := "mcp-server-aha-aql"
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if os.Getenv("AHA_AQL_TEST_BINARY") != "" {
		t.Skip("the binary under test is given by AHA_AQL_TEST_BINARY")
	}
	if got := filepath.Base(binPath); got != want {
		t.Fatalf("binary name %q, want %q", got, want)
	}
}

// A test that needs FakeAha is skipped on macOS and Windows, with a reason
// that names the platform; on Linux it runs.
func TestFakeAhaTestsRunOnLinuxOnly(t *testing.T) {
	var skipped bool
	ok := t.Run("needs-fakeaha", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()
		e := newEnv(t)
		r := e.run("", nil, "query", "query=SELECT email FROM me")
		if r.code != 0 || r.stdout != `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`+"\n" {
			t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
		}
		e.wantCalls("GET /me")
	})
	if runtime.GOOS == "linux" {
		if skipped || !ok {
			t.Fatalf("on linux a test with FakeAha must run and pass (skipped=%v ok=%v)", skipped, ok)
		}
	} else if !skipped {
		t.Fatalf("on %s a test with FakeAha must be skipped", runtime.GOOS)
	}
}
