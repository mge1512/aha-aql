// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268
// tests by: claude-opus-5-5

package aqltests

import (
	"strings"
	"testing"
	"time"
)

func queryDescription(key string) string {
	return strings.ReplaceAll(`Read from Aha! with one AQL statement.
Start with: SELECT key, name FROM products. Most other lists need a product key.
DESCRIBE lists the entities; DESCRIBE features lists the fields of features.
A workspace or project is a product; a milestone or version is a release; an objective is a goal.
Search by word: SELECT id, name FROM features WHERE product = '{KEY}' AND name CONTAINS 'word'
In name and description: SELECT id, name FROM features WHERE release = '{KEY}-R-3' AND (name CONTAINS 'word' OR description CONTAINS 'word')
SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]
DESCRIBE entity
Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me.
id is the reference number of a record, for example '{KEY}-23'.
Filter by parent: product = '{KEY}', release = '{KEY}-R-3', feature = '{KEY}-23', initiative = '{KEY}-S-1'.
Other conditions: !=, <, >, CONTAINS 'text', IS NULL. Count with COUNT(*).
Custom fields are written custom.key. DESCRIBE entity lists them.
Text goes in single quotes. Write an apostrophe inside text twice: 'Year''s end'.
Examples:
SELECT id, name, status FROM features WHERE release = '{KEY}-R-3'
SELECT status, COUNT(*) FROM features WHERE product = '{KEY}' GROUP BY status
SELECT id, name, release_date FROM releases WHERE product = '{KEY}' AND parking_lot = false ORDER BY release_date
SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features'
SELECT email FROM me
`, "{KEY}", key)
}

func executeDescription(key string) string {
	return strings.ReplaceAll(`Change ONE record in Aha! with one AQL statement. Three forms:
INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)
UPDATE entity SET field = 'value', ... WHERE id = 'id'
DELETE FROM entity WHERE id = 'id'
parent is the record the new one belongs to: a release for features and epics, a product for releases, initiatives, goals and ideas, a feature for requirements, any record for comments.
Text and apostrophes as in query. DESCRIBE entity in query lists the field names.
Examples:
INSERT INTO features (parent, name) VALUES ('{KEY}-R-3', 'Export to CSV')
UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = '{KEY}-23'
INSERT INTO comments (parent, body) VALUES ('{KEY}-23', 'It''s ready.')
`, "{KEY}", key)
}

func TestToolsWithoutReadOnlyMode(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdioRaw(nil)
	s.initialize()
	tools := s.listTools()
	if len(tools) != 2 {
		t.Fatalf("tools: %d", len(tools))
	}
	byName := map[string]toolInfo{}
	for _, tl := range tools {
		byName[tl.Name] = tl
	}
	q, x := byName["query"], byName["execute"]
	if q.Name == "" || x.Name == "" {
		t.Fatalf("tools: %+v", tools)
	}
	if len(q.InputSchema.Properties) != 1 || q.InputSchema.Properties["query"].Type != "string" ||
		q.InputSchema.Properties["query"].Description != "One AQL statement" || !eqStrings(q.InputSchema.Required, []string{"query"}) {
		t.Fatalf("query schema: %+v", q.InputSchema)
	}
	if len(x.InputSchema.Properties) != 1 || x.InputSchema.Properties["statement"].Type != "string" ||
		x.InputSchema.Properties["statement"].Description != "One AQL statement" || !eqStrings(x.InputSchema.Required, []string{"statement"}) {
		t.Fatalf("execute schema: %+v", x.InputSchema)
	}
	for k, v := range map[string]bool{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": true} {
		if q.Annotations[k] != v {
			t.Fatalf("query %s = %v", k, q.Annotations[k])
		}
	}
	for k, v := range map[string]bool{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": false, "openWorldHint": true} {
		if x.Annotations[k] != v {
			t.Fatalf("execute %s = %v", k, x.Annotations[k])
		}
	}
	if q.Description != queryDescription("PROD") {
		t.Fatalf("query description\n got: %q\nwant: %q", q.Description, queryDescription("PROD"))
	}
	if x.Description != executeDescription("PROD") {
		t.Fatalf("execute description\n got: %q\nwant: %q", x.Description, executeDescription("PROD"))
	}
	// No resource; the prompt list is start alone.
	a := s.request("resources/list", map[string]any{}, 30*time.Second)
	if a.Error == nil && strings.Contains(string(a.Result), `"name"`) {
		t.Fatalf("resources/list: %s", a.Result)
	}
	if names := promptNames(t, s.request("prompts/list", map[string]any{}, 30*time.Second)); !eqStrings(names, []string{"start"}) {
		t.Fatalf("prompts: %v", names)
	}
}

func TestToolsInReadOnlyMode(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdioRaw(nil, "read-only=true")
	s.initialize()
	tools := s.listTools()
	if len(tools) != 1 || tools[0].Name != "query" {
		t.Fatalf("tools: %+v", tools)
	}
	a := s.request("tools/call", map[string]any{"name": "execute", "arguments": map[string]any{"statement": "DELETE FROM features WHERE id = 'DEMO-6'"}}, 30*time.Second)
	if a.Error == nil && !strings.Contains(string(a.Result), `"isError":true`) {
		t.Fatalf("execute was callable: %s", a.Result)
	}
	e.wantNoWrite()
}

func TestUnknownToolIsAProtocolError(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	a := s.request("tools/call", map[string]any{"name": "bogus", "arguments": map[string]any{}}, 30*time.Second)
	if a.Error == nil {
		t.Fatalf("no JSON-RPC error: %s", a.Result)
	}
	// The server goes on after a line that is not JSON.
	if _, err := s.stdin.Write([]byte("{not json\n")); err != nil {
		t.Fatal(err)
	}
	wantResult(t, s.query(`SELECT email FROM me`), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
	if code := s.closeAndWait(); code != 0 {
		t.Fatalf("exit code on EOF: %d", code)
	}
}

func TestQueryRejectsProductParameter(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.call("query", map[string]any{"query": "SELECT id FROM features", "product": "DEMO"}, 60*time.Second),
		`tool query has one parameter: query (one AQL statement). It does not accept: product. Name the product inside the statement: ... WHERE product = 'PROD'`)
	e.wantNoCall()
}

func TestQueryRequiresAStatement(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	msg := `query parameter is required: one AQL statement. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
	wantError(t, s.call("query", map[string]any{}, 60*time.Second), msg)
	wantError(t, s.query("   "), msg)
	wantError(t, s.call("query", map[string]any{"query": 42}, 60*time.Second), msg)
}

func TestQueryRefusesWriteStatements(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`UPDATE features SET name = 'X' WHERE id = 'DEMO-1'`), `query only reads. Use the execute tool for INSERT, UPDATE and DELETE.`)
	e.wantNoCall()
}

func TestExampleKeyIsTheDefaultProduct(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdioRaw(nil, "default-product=demo")
	s.initialize()
	tools := s.listTools()
	all := ""
	for _, tl := range tools {
		all += tl.Description
	}
	if !hasLine(all, "id is the reference number of a record, for example 'DEMO-23'.") || strings.Contains(all, "PROD") || strings.Contains(all, "OTHER") {
		t.Fatalf("descriptions: %s", all)
	}
}

func TestExampleKeyWithoutADefaultProduct(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdioRaw(nil)
	s.initialize()
	all := ""
	for _, tl := range s.listTools() {
		all += tl.Description
	}
	if !hasLine(all, "Filter by parent: product = 'PROD', release = 'PROD-R-3', feature = 'PROD-23', initiative = 'PROD-S-1'.") ||
		strings.Contains(all, "DEMO") || strings.Contains(all, "OTHER") {
		t.Fatalf("descriptions: %s", all)
	}
}

func TestExampleKeyIsNotAProductOfTheAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddProduct("PROD", "7000000000000000006", 0, "")
	s := e.stdioRaw(nil)
	s.initialize()
	var exec string
	all := ""
	for _, tl := range s.listTools() {
		all += tl.Description
		if tl.Name == "execute" {
			exec = tl.Description
		}
	}
	if !hasLine(exec, "INSERT INTO features (parent, name) VALUES ('PRODX-R-3', 'Export to CSV')") || strings.Contains(all, "'PROD-") {
		t.Fatalf("descriptions: %s", all)
	}
}

func TestDefaultProductThatTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdioRaw(nil, "default-product=nope")
	s.initialize()
	all := ""
	for _, tl := range s.listTools() {
		all += tl.Description
	}
	if !hasLine(all, "id is the reference number of a record, for example 'PROD-23'.") {
		t.Fatalf("descriptions: %s", all)
	}
	if !waitFor(func() bool {
		return hasLine(s.stderr.String(), "warning: default-product names no product of the account. Examples use the key PROD")
	}, 3*time.Second) {
		t.Fatalf("stderr: %q", s.stderr.String())
	}
	if strings.Contains(s.stderr.String(), "nope") {
		t.Fatalf("warning has the value of an option: %q", s.stderr.String())
	}
}

func TestExampleKeyWhenTheProductListCannotBeRead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.NoAnswer("GET", "/products")
	s := e.stdioRaw(nil, "default-product=demo")
	s.initialize()
	m := s.request("tools/list", map[string]any{}, 150*time.Second)
	tools := parseTools(t, m)
	if len(tools) != 2 {
		t.Fatalf("tools: %+v", tools)
	}
	all := ""
	for _, tl := range tools {
		all += tl.Description
	}
	if !hasLine(all, "id is the reference number of a record, for example 'PROD-23'.") {
		t.Fatalf("descriptions: %s", all)
	}
	if !waitFor(func() bool {
		return hasLine(s.stderr.String(), "warning: the product list could not be read. Examples use the key PROD")
	}, 3*time.Second) || strings.Contains(s.stderr.String(), "fixture-key") {
		t.Fatalf("stderr: %q", s.stderr.String())
	}
	if strings.Contains(s.stderr.String(), "default-product names no product") {
		t.Fatalf("G15 written as well: %q", s.stderr.String())
	}
}

// The ExampleKey is chosen once for a stdio process: a second tool list
// reads no product list.
func TestExampleKeyIsKeptForTheProcess(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	s.listTools()
	wantError(t, s.query(`SELECT * FROM tickets`),
		`unknown entity 'tickets'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`)
	e.wantNoCall()
}
