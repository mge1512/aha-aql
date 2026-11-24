// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// tool-registration, query, execute envelopes and log lines.

import (
	"regexp"
	"strings"
	"testing"
)

const queryDescription = `Read from Aha! with one AQL statement.
SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]
DESCRIBE entity
Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me.
id is the reference number of a record, for example 'PROD-23'.
Filter by parent: product = 'PROD', release = 'PROD-R-3', feature = 'PROD-23', initiative = 'PROD-S-1'.
Other conditions: !=, <, >, CONTAINS 'text', IS NULL. Count with COUNT(*).
Custom fields are written custom.key. DESCRIBE entity lists them.
Text goes in single quotes. Write an apostrophe inside text twice: 'Year''s end'.
Examples:
SELECT id, name, status FROM features WHERE release = 'PROD-R-3'
SELECT status, COUNT(*) FROM features WHERE product = 'PROD' GROUP BY status
SELECT id, name, release_date FROM releases WHERE product = 'PROD' AND parking_lot = false ORDER BY release_date
SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'
SELECT email FROM me
`

const executeDescription = `Change ONE record in Aha! with one AQL statement. Three forms:
INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)
UPDATE entity SET field = 'value', ... WHERE id = 'id'
DELETE FROM entity WHERE id = 'id'
parent is the record the new one belongs to: a release for features and epics, a product for releases, initiatives, goals and ideas, a feature for requirements, any record for comments.
Text and apostrophes as in query. DESCRIBE entity in query lists the field names.
Examples:
INSERT INTO features (parent, name) VALUES ('PROD-R-3', 'Export to CSV')
UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = 'PROD-23'
INSERT INTO comments (parent, body) VALUES ('PROD-23', 'It''s ready.')
`

// EXAMPLE: tools_without_read_only_mode
func TestToolsWithoutReadOnlyMode(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, fixtureEnv...))
	s.initialize()
	tools := s.listTools()
	if len(tools) != 2 {
		t.Fatalf("tool list has %d tools, want 2", len(tools))
	}
	byName := map[string]toolInfo{}
	for _, tl := range tools {
		byName[tl.Name] = tl
	}
	q, ok := byName["query"]
	if !ok {
		t.Fatal("no tool query")
	}
	e, ok := byName["execute"]
	if !ok {
		t.Fatal("no tool execute")
	}
	if len(q.InputSchema.Properties) != 1 || q.InputSchema.Properties["query"].Type != "string" {
		t.Errorf("query must have the one parameter query: %+v", q.InputSchema.Properties)
	}
	if q.InputSchema.Properties["query"].Description != "One AQL statement" || strings.Join(q.InputSchema.Required, ",") != "query" {
		t.Errorf("parameter query: %+v required %v", q.InputSchema.Properties["query"], q.InputSchema.Required)
	}
	if len(e.InputSchema.Properties) != 1 || e.InputSchema.Properties["statement"].Type != "string" ||
		e.InputSchema.Properties["statement"].Description != "One AQL statement" || strings.Join(e.InputSchema.Required, ",") != "statement" {
		t.Errorf("execute must have the one parameter statement: %+v", e.InputSchema.Properties)
	}
	if q.Annotations["readOnlyHint"] != true || q.Annotations["openWorldHint"] != true {
		t.Errorf("query annotations: %v", q.Annotations)
	}
	if e.Annotations["readOnlyHint"] != false || e.Annotations["destructiveHint"] != true ||
		e.Annotations["idempotentHint"] != false || e.Annotations["openWorldHint"] != true {
		t.Errorf("execute annotations: %v", e.Annotations)
	}
	if q.Description != queryDescription {
		t.Errorf("description of query\n got: %q\nwant: %q", q.Description, queryDescription)
	}
	if e.Description != executeDescription {
		t.Errorf("description of execute\n got: %q\nwant: %q", e.Description, executeDescription)
	}
	lines := strings.Split(q.Description, "\n")
	idx := func(prefix string) int {
		for i, l := range lines {
			if strings.HasPrefix(l, prefix) {
				return i
			}
		}
		return -1
	}
	c := idx("Custom fields are written custom.key. DESCRIBE entity lists them.")
	if !(idx("Other conditions:") < c && c < idx("Text goes in single quotes.")) {
		t.Errorf("line order of the description of query")
	}
	// no MCP resource and no MCP prompt
	if r := s.rpc("resources/list", map[string]any{}); r.Error == nil && strings.Contains(string(r.Result), `"uri"`) {
		t.Errorf("the server offers resources: %s", r.Result)
	}
	if r := s.rpc("prompts/list", map[string]any{}); r.Error == nil && strings.Contains(string(r.Result), `"name"`) {
		t.Errorf("the server offers prompts: %s", r.Result)
	}
}

// EXAMPLE: tools_in_read_only_mode; MILESTONE 0.5.0 acceptance (tools/list)
func TestToolsInReadOnlyMode(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, append(fixtureEnv, "AHA_READ_ONLY=true")...))
	s.initialize()
	tools := s.listTools()
	if len(tools) != 1 || tools[0].Name != "query" {
		t.Errorf("tool list %v, want query alone", tools)
	}
	for _, tl := range tools {
		if strings.Contains(tl.Description, "Change ONE record in Aha!") {
			t.Errorf("read-only tool list has the description of execute")
		}
	}
	if !strings.Contains(tools[0].Description, "Read from Aha! with one AQL statement") {
		t.Errorf("description of query: %s", tools[0].Description)
	}
	// the tool execute is not offered: a call is a protocol error
	r := s.rpc("tools/call", map[string]any{"name": "execute", "arguments": map[string]any{"statement": "DELETE FROM features WHERE id = 'DEMO-6'"}})
	if r.Error == nil && !parseToolResult(t, r.Result).IsError {
		t.Errorf("execute must not run in read-only mode")
	}
	expectNoWrite(t, f)
}

// EXAMPLE: query_result_envelope
func TestQueryResultEnvelope(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'"),
		`{"entity":"features","scope":"release DEMO-R-3","count":3,"records":[{"id":"DEMO-3","name":"Live dashboard","status":"In development"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration"},{"id":"DEMO-5","name":"Rider's handbook","status":"Shipped"}]}`)
	expectCalls(t, f, "GET /releases/DEMO-R-3/features")
	if c := f.Calls(); len(c) == 1 && c[0].Query.Get("page") != "1" {
		t.Errorf("page %q, want 1", c[0].Query.Get("page"))
	}
}

// EXAMPLE: query_rejects_product_parameter
func TestQueryRejectsProductParameter(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.call("query", map[string]any{"query": "SELECT id FROM features", "product": "DEMO"}),
		"tool query has one parameter: query (one AQL statement). It does not accept: product. Name the product inside the statement: ... WHERE product = 'PROD'")
	expectNoCalls(t, f)
}

// EXAMPLE: query_requires_a_statement
func TestQueryRequiresAStatement(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	msg := "query parameter is required: one AQL statement. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'"
	expectError(t, s.call("query", map[string]any{}), msg)
	expectError(t, s.call("query", map[string]any{"query": "   "}), msg)
	expectError(t, s.call("query", map[string]any{"query": 5}), msg)
	expectNoCalls(t, f)
}

// EXAMPLE: query_refuses_write_statements
func TestQueryRefusesWriteStatements(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("UPDATE features SET name = 'X' WHERE id = 'DEMO-1'"), "query only reads. Use the execute tool for INSERT, UPDATE and DELETE.")
	expectNoCalls(t, f)
}

// EXAMPLE: execute_result_envelope
func TestExecuteResultEnvelope(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.execute("INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'Export to PDF')"),
		`{"id":"DEMO-7","name":"Export to PDF","parent":"DEMO-R-3","message":"Feature DEMO-7 created"}`)
	expectCalls(t, f, "POST /releases/DEMO-R-3/features")
	if c := f.Calls(); len(c) == 1 && c[0].Body != `{"feature":{"name":"Export to PDF"}}` {
		t.Errorf("body %s", c[0].Body)
	}
}

// EXAMPLE: execute_rejects_unknown_parameter
func TestExecuteRejectsUnknownParameter(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.call("execute", map[string]any{"query": "DELETE FROM features WHERE id = 'DEMO-6'"}),
		"tool execute has one parameter: statement (one AQL statement). It does not accept: query. Pass the statement as statement.")
	expectNoCalls(t, f)
	expectError(t, s.call("execute", map[string]any{}), "statement parameter is required: one INSERT, UPDATE or DELETE statement")
}

// EXAMPLE: execute_refuses_read_statements
func TestExecuteRefusesReadStatements(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("SELECT id FROM features WHERE product = 'DEMO'"), "execute only changes data. Use the query tool for SELECT and DESCRIBE.")
	expectNoCalls(t, f)
}

var logTime = `[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z`

// EXAMPLE: execute_writes_a_log_line (the clock cannot be replaced from
// outside the binary: the time is checked by its form)
func TestExecuteWritesALogLine(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.execute("UPDATE features SET status = 'shipped' WHERE id = 'demo-1'"),
		`{"id":"DEMO-1","name":"CSV export","status":"Shipped","message":"Feature DEMO-1 status changed to 'Shipped'"}`)
	s.closeAndWait()
	re := regexp.MustCompile(`(?m)^` + logTime + ` execute account=example op=update entity=features id=DEMO-1 result=ok$`)
	if !re.MatchString(s.stderr.String()) {
		t.Errorf("no log line; stderr:\n%s", s.stderr.String())
	}
	if n := len(regexp.MustCompile(`(?m) execute account=`).FindAllString(s.stderr.String(), -1)); n != 1 {
		t.Errorf("%d log lines, want 1", n)
	}
}

// EXAMPLE: execute_logs_a_refused_statement
func TestExecuteLogsARefusedStatement(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectErrorPrefix(t, s.execute("DELETE FROM features WHERE name = 'CSV export'"), "one statement changes one record: the WHERE must start with id = '...'.")
	s.closeAndWait()
	errText := s.stderr.String()
	re := regexp.MustCompile(`(?m)^` + logTime + ` execute account=example op=none entity=none id=none result=syntax$`)
	if !re.MatchString(errText) {
		t.Errorf("no log line; stderr:\n%s", errText)
	}
	if strings.Contains(errText, "CSV export") || strings.Contains(errText, "fixture-key") {
		t.Errorf("the diagnostic output has a value or the credential:\n%s", errText)
	}
	expectNoCalls(t, f)
}

// execute log line for an error after parsing: the id and the ErrorKind.
func TestExecuteLogsTheErrorKind(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("DELETE FROM features WHERE id = 'demo-99'"), "there is no feature with the id 'DEMO-99'")
	s.closeAndWait()
	re := regexp.MustCompile(`(?m)^` + logTime + ` execute account=example op=delete entity=features id=DEMO-99 result=not_found$`)
	if !re.MatchString(s.stderr.String()) {
		t.Errorf("no log line; stderr:\n%s", s.stderr.String())
	}
}

// manual mode writes no log line of execute.
func TestManualModeWritesNoLogLine(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "", "execute", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	if r.stdout != `{"id":"DEMO-6","message":"Feature DEMO-6 deleted"}`+"\n" || r.stderr != "" || r.code != 0 {
		t.Errorf("stdout %q stderr %q code %d", r.stdout, r.stderr, r.code)
	}
}
