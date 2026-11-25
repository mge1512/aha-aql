// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268
// tests by: claude-opus-5-5

// Tests for spec version 0.23.0: server instructions, server/discover,
// the prompt start, http mode without MCP sessions, the dry run of an
// insert, lower-case Boolean settings, sorted A02 names and connections
// that stay open.
package aqltests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const serverInstructions = `This server reads and changes records of one Aha! account with AQL, a small SQL dialect.
The tool query reads (SELECT, DESCRIBE). The tool execute, when the tool list has it, changes one record (INSERT, UPDATE, DELETE).
Start with SELECT key, name FROM products, then DESCRIBE an entity for its field names.
Most error messages show what to write instead; then send the corrected statement.
`

func startPromptText(key string) string {
	return strings.ReplaceAll(`Use the tool query of the Aha! server, with one AQL statement in each call.
1. SELECT key, name FROM products
2. DESCRIBE features lists the fields of features; DESCRIBE alone lists the entities.
3. SELECT id, name, status FROM features WHERE product = '{KEY}'
Search by word: SELECT id, name FROM features WHERE product = '{KEY}' AND name CONTAINS 'word'
A workspace or project is a product; a milestone or version is a release; an objective is a goal.
Most error messages show what to write instead; then send the corrected statement.
`, "{KEY}", key)
}

func promptNames(t *testing.T, m rpcMsg) []string {
	t.Helper()
	if m.Error != nil {
		t.Fatalf("prompts/list: %v", m.Error.Message)
	}
	var r struct {
		Prompts []struct {
			Name string `json:"name"`
		} `json:"prompts"`
	}
	if err := json.Unmarshal(m.Result, &r); err != nil {
		t.Fatalf("prompts/list result: %v", err)
	}
	out := []string{}
	for _, p := range r.Prompts {
		out = append(out, p.Name)
	}
	return out
}

func instructionsOf(t *testing.T, m rpcMsg) string {
	t.Helper()
	if m.Error != nil {
		t.Fatalf("answer is a JSON-RPC error: %d %s", m.Error.Code, m.Error.Message)
	}
	var r struct {
		Instructions *string `json:"instructions"`
	}
	if err := json.Unmarshal(m.Result, &r); err != nil || r.Instructions == nil {
		t.Fatalf("no instructions in %s", m.Result)
	}
	return *r.Instructions
}

func TestServerInstructionsAndEntryPoint(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vars := map[string]string{"AHA_DEFAULT_PRODUCT": "demo"}

	// initialize
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
	if err := json.Unmarshal(m.Result, &caps); err != nil || caps.Capabilities["prompts"] == nil {
		t.Fatalf("capability prompts is not declared: %s", m.Result)
	}
	e.wantNoRequest()

	// server/discover, from a client of 2026-07-28 that sends no initialize
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

	// the tool list
	s.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	var q toolInfo
	for _, tl := range s.listTools() {
		if tl.Name == "query" {
			q = tl
		}
	}
	ls := lines(q.Description)
	if len(ls) < 2 || ls[1] != `Start with: SELECT key, name FROM products. Most other lists need a product key.` {
		t.Fatalf("second line: %q", q.Description)
	}
	for _, want := range []string{
		`Search by word: SELECT id, name FROM features WHERE product = 'DEMO' AND name CONTAINS 'word'`,
		`In name and description: SELECT id, name FROM features WHERE release = 'DEMO-R-3' AND (name CONTAINS 'word' OR description CONTAINS 'word')`,
	} {
		if !hasLine(q.Description, want) {
			t.Fatalf("description has no line %q:\n%s", want, q.Description)
		}
	}
	if q.Description != queryDescription("DEMO") {
		t.Fatalf("query description\n got: %q\nwant: %q", q.Description, queryDescription("DEMO"))
	}
}

func TestStartPrompt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(map[string]string{"AHA_DEFAULT_PRODUCT": "demo"})
	m := s.request("prompts/list", map[string]any{}, 30*time.Second)
	if m.Error != nil {
		t.Fatalf("prompts/list: %v", m.Error.Message)
	}
	var pl struct {
		Prompts []struct {
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Arguments   []json.RawMessage `json:"arguments"`
		} `json:"prompts"`
	}
	if err := json.Unmarshal(m.Result, &pl); err != nil {
		t.Fatal(err)
	}
	if len(pl.Prompts) != 1 || pl.Prompts[0].Name != "start" || pl.Prompts[0].Description != "How to start with AQL" || len(pl.Prompts[0].Arguments) != 0 {
		t.Fatalf("prompts: %s", m.Result)
	}
	g := s.request("prompts/get", map[string]any{"name": "start"}, 120*time.Second)
	if g.Error != nil {
		t.Fatalf("prompts/get: %v", g.Error.Message)
	}
	var pg struct {
		Messages []struct {
			Role    string `json:"role"`
			Content struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(g.Result, &pg); err != nil {
		t.Fatal(err)
	}
	if len(pg.Messages) != 1 || pg.Messages[0].Role != "user" || pg.Messages[0].Content.Type != "text" {
		t.Fatalf("messages: %s", g.Result)
	}
	text := pg.Messages[0].Content.Text
	if text != startPromptText("DEMO") {
		t.Fatalf("prompt text\n got: %q\nwant: %q", text, startPromptText("DEMO"))
	}
	found := false
	for _, l := range lines(text) {
		if strings.HasPrefix(l, "3.") {
			found = l == `3. SELECT id, name, status FROM features WHERE product = 'DEMO'`
		}
	}
	if !found {
		t.Fatalf("line 3.: %q", text)
	}
}

func TestHTTPModeKeepsNoSession(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.http(map[string]string{"AHA_API_KEY": ""}, "subdomain=example")
	c := s.client(map[string]string{"Aha-Api-Key": "fixture-key"})
	a := c.initialize()
	if a.status != 200 {
		t.Fatalf("initialize: %d %s", a.status, a.raw)
	}
	if _, ok := a.header["Mcp-Session-Id"]; ok {
		t.Fatalf("initialize answer has Mcp-Session-Id: %v", a.header)
	}
	c2 := s.client(map[string]string{"Aha-Api-Key": "fixture-key", "Mcp-Session-Id": "not-a-session"})
	b := c2.request("tools/call", map[string]any{"name": "query", "arguments": map[string]any{"query": "SELECT key FROM products"}}, 120*time.Second)
	if b.status != 200 {
		t.Fatalf("tools/call: %d %s", b.status, b.raw)
	}
	wantResult(t, parseToolResult(t, b.msg), `{"entity":"products","scope":"all products","count":2,"records":[{"key":"DEMO"},{"key":"OTHER"}]}`)
	if _, ok := b.header["Mcp-Session-Id"]; ok {
		t.Fatalf("tools/call answer has Mcp-Session-Id: %v", b.header)
	}
}

func TestDryRunOfAnInsert(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "execute", "dry-run=true", "statement=INSERT INTO features (parent, name) VALUES ('demo-r-3', 'Export to PDF')")
	if r.code != 0 || r.stdout != `{"name":"Export to PDF","message":"would create feature under DEMO-R-3"}`+"\n" || r.stderr != "" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	e.wantNoRequest()
}

func TestBooleanSettingsAreLowerCase(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.startAndStop(map[string]string{"AHA_READ_ONLY": "TRUE"})
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: AHA_READ_ONLY must be true or false` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.startAndStop(nil, "read-only=True")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: read-only must be true or false` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.run("", nil, "execute", "dry-run=FALSE", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	if r.code != 2 || strings.TrimSpace(r.stderr) != `error: dry-run must be true or false` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestUnknownParameterNamesAreSorted(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.call("query", map[string]any{"query": "SELECT key FROM products", "beta": 1, "Zeta": 1, "alpha": 1}, 60*time.Second),
		`tool query has one parameter: query (one AQL statement). It does not accept: Zeta, alpha, beta.`)
	e.wantNoCall()
}

func TestConnectionIsKeptOpen(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	for i := 0; i < 3; i++ {
		wantResult(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-3'`),
			`{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`)
	}
	if n := len(e.fake.Calls()); n != 3 {
		t.Fatalf("requests inside the tunnel: %d: %s", n, e.fake.describeCalls())
	}
	// Three requests one after the other: a new tunnel for every request
	// would give three CONNECT requests.
	if n := len(e.fake.Connects()); n > 1 {
		t.Fatalf("CONNECT requests for three sequential requests: %d", n)
	}
}
