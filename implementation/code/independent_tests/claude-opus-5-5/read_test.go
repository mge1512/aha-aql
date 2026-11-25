// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30
// tests by: claude-opus-5-5

package aqltests

import (
	"encoding/json"
	"strings"
	"testing"
)

var featureFields = []string{"id", "name", "status", "release", "initiative", "epic", "assigned_to", "product",
	"start_date", "due_date", "tags", "score", "progress", "description", "created_at", "updated_at", "url"}

func eqStrings(a, b []string) bool { return strings.Join(a, "\x00") == strings.Join(b, "\x00") }

func TestQueryResultEnvelope(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'`),
		`{"entity":"features","scope":"release DEMO-R-3","count":3,"records":[{"id":"DEMO-3","name":"Live dashboard","status":"In development"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration"},{"id":"DEMO-5","name":"Rider's handbook","status":"Shipped"}]}`)
	e.wantCalls("GET /releases/DEMO-R-3/features")
	c := e.fake.Calls()[0]
	if c.Query.Get("page") != "1" {
		t.Fatalf("page: %q", c.Query.Get("page"))
	}
}

func TestCountPerStatus(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status`),
		`{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"In design","COUNT(*)":1},{"status":"Under consideration","COUNT(*)":3},{"status":"In development","COUNT(*)":1},{"status":"Shipped","COUNT(*)":1}]}`)
}

func TestStarShowsDefaultFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT * FROM features WHERE product = 'DEMO' AND status = 'under consideration'`),
		`{"entity":"features","scope":"product DEMO","count":3,"records":[{"id":"DEMO-2","name":"Offline maps","status":"Under consideration","release":"DEMO-R-2"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration","release":"DEMO-R-3"},{"id":"DEMO-6","name":"Route planner","status":"Under consideration","release":"DEMO-R-1"}],"note":"* shows id, name, status, release. Other fields of features: initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url."}`)
}

func TestRecordByIDHasAllFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`SELECT * FROM features WHERE id = 'demo-3'`)
	if r.isError {
		t.Fatal(r.text)
	}
	top := orderedObject(t, r.text)
	if string(memberOf(t, top, "scope")) != `"feature DEMO-3"` || string(memberOf(t, top, "count")) != "1" {
		t.Fatalf("scope/count: %s", r.text)
	}
	if hasMember(top, "note") {
		t.Fatalf("result has a note: %s", r.text)
	}
	row := rows(t, r.text)[0]
	want := append(append([]string{}, featureFields...), "custom.customer", "custom.effort_days", "custom.go_live", "custom.markets", "custom.priority", "custom.reviewers")
	if !eqStrings(keysOf(row), want) {
		t.Fatalf("labels\n got: %v\nwant: %v", keysOf(row), want)
	}
	for k, v := range map[string]string{"id": `"DEMO-3"`, "name": `"Live dashboard"`, "status": `"In development"`,
		"release": `"DEMO-R-3"`, "initiative": `"DEMO-S-1"`, "product": `"DEMO"`, "epic": "null", "assigned_to": "null"} {
		if got := string(memberOf(t, row, k)); got != v {
			t.Fatalf("%s = %s, want %s", k, got, v)
		}
	}
	e.wantCallSet("GET /features/DEMO-3", "GET /products", "GET /custom_field_definitions")
}

func TestFieldWithoutValueIsNull(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, assigned_to FROM features WHERE release = 'DEMO-R-2'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","assigned_to":"test.user@example.com"},{"id":"DEMO-2","assigned_to":null}]}`)
}

func TestLabelIsTheWordOfTheStatement(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT ref_num, title FROM features WHERE release_id = 'DEMO-R-2'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"ref_num":"DEMO-1","title":"CSV export"},{"ref_num":"DEMO-2","title":"Offline maps"}]}`)
}

func TestNameEqualityAndSearch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND name = 'live'`),
		`{"entity":"features","scope":"product DEMO","count":0,"records":[],"note":"No feature has exactly the name 'live'. name = compares the whole name; to search inside names use: name CONTAINS 'live'"}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND name CONTAINS 'live'`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND name = 'offline MAPS'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-2"}]}`)
}

func TestUnknownIDGivesAnEmptyResult(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT * FROM features WHERE id = 'DEMO-99'`),
		`{"entity":"features","scope":"feature DEMO-99","count":0,"records":[],"note":"There is no feature with the id 'DEMO-99'."}`)
	if n := e.fake.count("GET", "/custom_field_definitions"); n != 0 {
		t.Fatalf("definitions were read %d times", n)
	}
}

func TestRecordThatExistsAndDoesNotMatch(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	note := `{"entity":"features","scope":"feature DEMO-1","count":0,"records":[],"note":"The feature DEMO-1 exists, and another condition of the statement is not true for it. To see the record: SELECT * FROM features WHERE id = 'DEMO-1'"}`
	wantResult(t, s.query(`SELECT id FROM features WHERE id = 'demo-1' AND status = 'Shipped'`), note)
	wantResult(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-1' AND id = 'DEMO-2'`), note)
	wantResult(t, s.query(`SELECT id FROM features WHERE id = 'DEMO-1' LIMIT 0`),
		`{"entity":"features","scope":"feature DEMO-1","count":0,"total":1,"records":[]}`)
}

func TestStatusesOfAnEntityThatHasNone(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'comments'`),
		`{"entity":"statuses","scope":"product DEMO","count":0,"records":[],"note":"DEMO has statuses for: releases, features, epics, requirements, initiatives, goals, ideas."}`)
}

func TestDefaultLimitIsReported(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddProduct("BULK", "7000000000000000003", 60, "BULK-R-1")
	s := e.stdio(nil)
	r := s.query(`SELECT id, name FROM features WHERE product = 'BULK'`)
	if r.isError {
		t.Fatal(r.text)
	}
	top := orderedObject(t, r.text)
	if string(memberOf(t, top, "count")) != "50" || string(memberOf(t, top, "total")) != "60" || string(memberOf(t, top, "truncated")) != "true" {
		t.Fatalf("count/total/truncated: %s", r.text)
	}
	rs := rows(t, r.text)
	if len(rs) != 50 || string(memberOf(t, rs[0], "id")) != `"BULK-1"` || string(memberOf(t, rs[49], "id")) != `"BULK-50"` {
		t.Fatalf("rows: %d", len(rs))
	}
	if got := string(memberOf(t, top, "note")); got != `"Showing 50 of 60 records. Narrow the result with WHERE, raise LIMIT (up to 200), or count with COUNT(*)."` {
		t.Fatalf("note: %s", got)
	}
}

func TestReleasesNeedAProduct(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id, name FROM releases`),
		`releases are listed per product: add product = '...' to WHERE. Example: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`)
}

func TestStarWithAnIDShowsCustomValues(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`SELECT * FROM features WHERE id = 'DEMO-1'`)
	if r.isError {
		t.Fatal(r.text)
	}
	row := rows(t, r.text)[0]
	if !eqStrings(keysOf(row)[:17], featureFields) {
		t.Fatalf("catalog labels: %v", keysOf(row))
	}
	extra := row[17:]
	want := []string{`custom.customer="Acme"`, `custom.effort_days=12.5`, `custom.go_live="2026-04-15"`, `custom.markets="EMEA, APAC"`,
		`custom.priority="High"`, `custom.release_notes="Adds the export.\nSee the handbook."`, `custom.reviewers="second.user@example.com"`, `custom.value_score=31`}
	var got []string
	for _, m := range extra {
		got = append(got, m.key+"="+string(m.val))
	}
	if !eqStrings(got, want) {
		t.Fatalf("custom labels\n got: %v\nwant: %v", got, want)
	}
	if n := e.fake.count("GET", "/custom_field_definitions"); n != 1 {
		t.Fatalf("definitions read %d times", n)
	}

	r = s.query(`SELECT * FROM features WHERE id = 'DEMO-2'`)
	row = rows(t, r.text)[0]
	got = nil
	for _, m := range row[17:] {
		got = append(got, m.key+"="+string(m.val))
	}
	if !eqStrings(got, []string{"custom.effort_days=8", `custom.priority="Medium"`}) {
		t.Fatalf("DEMO-2 custom labels: %v", got)
	}

	r = s.query(`SELECT *, custom.platforms FROM features WHERE id = 'DEMO-2'`)
	row = rows(t, r.text)[0]
	got = nil
	for _, m := range row[17:] {
		got = append(got, m.key+"="+string(m.val))
	}
	if !eqStrings(got, []string{"custom.platforms=null", "custom.effort_days=8", `custom.priority="Medium"`}) {
		t.Fatalf("labels after *, custom.platforms: %v", got)
	}

	wantResult(t, s.query(`SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`)
}

func TestStarWithoutAnIDHasCatalogFieldsOnly(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`SELECT * FROM features WHERE release = 'DEMO-R-2'`)
	for _, row := range rows(t, r.text) {
		if !eqStrings(keysOf(row), []string{"id", "name", "status", "release"}) {
			t.Fatalf("labels: %v", keysOf(row))
		}
	}
	note := string(memberOf(t, orderedObject(t, r.text), "note"))
	if !strings.HasPrefix(note, `"* shows id, name, status, release.`) {
		t.Fatalf("note: %s", note)
	}
	if e.fake.count("GET", "/custom_field_definitions") != 0 {
		t.Fatal("definitions were read")
	}
	for _, c := range e.fake.Calls() {
		if strings.Contains(c.Query.Get("fields"), "custom_fields") {
			t.Fatalf("fields names custom_fields: %s", c.Query.Get("fields"))
		}
	}
	e.fake.Reset()
	r = s.query(`SELECT * FROM products`)
	for _, row := range rows(t, r.text) {
		if !eqStrings(keysOf(row), []string{"key", "name"}) {
			t.Fatalf("product labels: %v", keysOf(row))
		}
	}
	e.wantCalls("GET /products")
}

var productFields = []string{"key", "name", "product_line", "parent", "description", "created_at", "url"}

func TestStarShowsTheCustomValuesOfAProduct(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`SELECT * FROM products WHERE key = 'DEMO'`)
	row := rows(t, r.text)[0]
	if !eqStrings(keysOf(row), append(append([]string{}, productFields...), "custom.cost_center")) {
		t.Fatalf("labels: %v (%s)", keysOf(row), r.text)
	}
	if string(memberOf(t, row, "custom.cost_center")) != `"CC-100"` {
		t.Fatal(r.text)
	}
	if e.fake.count("GET", "/custom_field_definitions") != 1 || e.fake.count("GET", "/products/"+idDEMO) != 1 {
		t.Fatalf("calls: %s", e.fake.describeCalls())
	}
	r = s.query(`SELECT * FROM products WHERE key = 'OTHER'`)
	if !eqStrings(keysOf(rows(t, r.text)[0]), productFields) {
		t.Fatalf("labels: %s", r.text)
	}
	e.fake.Answer("GET", "/custom_field_definitions", 403, -1, nil, `{"error":"Forbidden"}`)
	e.fake.Reset()
	r = s.query(`SELECT * FROM products WHERE key = 'DEMO'`)
	if !eqStrings(keysOf(rows(t, r.text)[0]), productFields) {
		t.Fatalf("labels with 403: %s", r.text)
	}
	for _, c := range e.fake.Calls() {
		if c.Method == "GET" && strings.HasPrefix(c.Path, "/products/") {
			t.Fatalf("product was read: %s", c.Path)
		}
	}
}

func TestCustomFieldsInGroupByAndOrderBy(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT custom.priority, COUNT(*) FROM features WHERE product = 'DEMO' AND custom.priority IS NOT NULL GROUP BY custom.priority`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"custom.priority":"High","COUNT(*)":2},{"custom.priority":"Medium","COUNT(*)":1}]}`)
	wantResult(t, s.query(`SELECT SUM(custom.effort_days) FROM features WHERE product = 'DEMO'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"SUM(custom.effort_days)":23.5}]}`)
	wantResult(t, s.query(`SELECT id, custom.effort_days FROM features WHERE product = 'DEMO' ORDER BY custom.effort_days DESC LIMIT 2`),
		`{"entity":"features","scope":"product DEMO","count":2,"total":6,"records":[{"id":"DEMO-1","custom.effort_days":12.5},{"id":"DEMO-2","custom.effort_days":8}]}`)
}

func TestDoubledApostropheInText(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND name = 'Rider''s handbook'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-5"}]}`)
}

func TestDateWithoutQuotes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' AND release_date > 2026-06-01`),
		`{"entity":"releases","scope":"product DEMO","count":2,"records":[{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`)
}

func TestTextThatIsNotClosed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE name = 'Offline maps`),
		`the text that starts with the apostrophe at position 38 is not closed. Text goes in single quotes: 'like this'.`)
	e.wantNoCall()
}

func TestUnexpectedCharacter(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE name ~ 'x'`), `unexpected character ~ at position 36.`)
	e.wantNoCall()
}

func TestApostropheInsideAWordIsReadAsText(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' AND name = 'Rider's cup release'`),
		`{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-4"}]}`)
}

func TestStrictReadingIsKept(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE release='DEMO-R-3'AND status='Shipped'`),
		`{"entity":"features","scope":"release DEMO-R-3","count":1,"records":[{"id":"DEMO-5"}]}`)
}

func TestStatementsWithoutSelect(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`FROM releases WHERE product = 'DEMO'`),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","name":"What's next","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","name":"Spring release","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","name":"Summer release","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","name":"Rider's cup release","release_date":"2026-12-15","parking_lot":false}],"note":"* shows id, name, release_date, parking_lot. Other fields of releases: start_date, status, released, product, progress, description, created_at, url."}`)
	wantResult(t, s.query(`COUNT(*) FROM features WHERE product = 'DEMO'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`)
}

func TestTextAfterTheStatement(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantErrorPrefix(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' LIMIT 5 please`),
		`cannot read the statement at position 56 (please): expected the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order.`)
	e.wantNoCall()
}

func TestJoinIsRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT f.name FROM features f JOIN releases r ON f.release = r.id`),
		`JOIN is not supported. Read one entity per statement and filter by its parent, for example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`)
}

func TestIDOrEntityAlone(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`DEMO-3`), `DEMO-3 is an id, not a statement. To read that record: SELECT * FROM features WHERE id = 'DEMO-3'`)
	wantError(t, s.query(`features`), `features is an entity, not a statement. To list its records: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'. To see its fields: DESCRIBE features`)
}

func TestApostropheHintInAnError(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`SELECT id FROM features WHERE name = 'it's`)
	if !r.isError || !strings.Contains(r.text, `The text contains an apostrophe (it's). Inside text, write an apostrophe twice: it''s`) {
		t.Fatalf("message: %s", r.text)
	}
}

func TestLabelOfACustomField(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT f.custom.priority, f.custom.customer AS customer FROM features f WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"custom.priority":"High","customer":"Acme"}]}`)
}

func TestLabelsOfSelectItems(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT COUNT(1) FROM features WHERE product = 'DEMO'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`)
	wantResult(t, s.query(`SELECT id AS 'feature id' FROM features WHERE release = 'DEMO-R-2'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"feature id":"DEMO-1"},{"feature id":"DEMO-2"}]}`)
}

func TestKeysAndArgumentsThatAreRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantErrorPrefix(t, s.query(`SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status ORDER BY name`),
		`cannot read the statement at position 87 (name): expected a field of GROUP BY, a selected field or an aggregate after ORDER BY.`)
	e.wantNoCall()
	wantErrorPrefix(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' ORDER BY COUNT(*)`),
		`cannot read the statement at position 57 (COUNT(*)): expected a field name.`)
	wantErrorPrefix(t, s.query(`SELECT SUM(*) FROM features WHERE product = 'DEMO'`),
		`cannot read the statement at position 12 (*) FROM features WHERE p): expected a field inside SUM( ).`)
}

func TestARowHasEachLabelOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, id, f.name, name FROM features f WHERE release = 'DEMO-R-2'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","name":"CSV export"},{"id":"DEMO-2","name":"Offline maps"}]}`)
	wantResult(t, s.query(`SELECT COUNT(*), COUNT(1) FROM features WHERE product = 'DEMO'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`)
	wantResult(t, s.query(`SELECT COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status, workflow_status`),
		`{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"In design","COUNT(*)":1},{"status":"Under consideration","COUNT(*)":3},{"status":"In development","COUNT(*)":1},{"status":"Shipped","COUNT(*)":1}]}`)
	r := s.query(`SELECT *, name FROM features WHERE release = 'DEMO-R-2'`)
	rs := rows(t, r.text)
	if len(rs) != 2 {
		t.Fatal(r.text)
	}
	for _, row := range rs {
		if !eqStrings(keysOf(row), []string{"id", "name", "status", "release"}) {
			t.Fatalf("labels: %v", keysOf(row))
		}
	}
	e.fake.Reset()
	wantError(t, s.query(`SELECT id, name AS id FROM features WHERE release = 'DEMO-R-2'`),
		`two select items have the label 'id'. Give one of them another label, in the form: field AS label`)
	e.wantNoCall()
	wantError(t, s.query(`SELECT COUNT(*) AS status FROM features WHERE product = 'DEMO' GROUP BY status`),
		`the label 'status' is also the name of a field of features. Choose another label, in the form: field AS label`)
	wantError(t, s.query(`SELECT *, name AS id FROM features WHERE release = 'DEMO-R-2'`),
		`the label 'id' is also the name of a field of features. Choose another label, in the form: field AS label`)
}

func TestValuesThatDoNotFitTheType(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND created_at > '01.02.2026'`),
		`created_at is a timestamp: write the value as 'YYYY-MM-DD' or 'YYYY-MM-DDThh:mm:ssZ'. Got '01.02.2026'.`)
	e.wantNoCall()
	wantError(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' AND release_date = '2026-02-30'`),
		`release_date is a date: write the value as 'YYYY-MM-DD'. Got '2026-02-30'.`)
	wantError(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND score > 'abc'`),
		`score is a number: write the value as 5 or 2.5. Got 'abc'.`)
	wantError(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' AND released > 0`),
		`released is true or false: compare it with = or !=, for example released = TRUE`)
	e.wantNoCall()
}

func TestDescribeAndReadHaveTheSameFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`DESCRIBE releases`)
	var d struct {
		Fields []struct {
			Name string `json:"name"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(r.text), &d); err != nil {
		t.Fatal(r.text)
	}
	var names []string
	for _, f := range d.Fields {
		names = append(names, f.Name)
	}
	want := []string{"id", "name", "release_date", "start_date", "status", "parking_lot", "released", "product", "progress", "description", "created_at", "url"}
	if !eqStrings(names, want) {
		t.Fatalf("fields: %v", names)
	}
	r = s.query(`SELECT * FROM releases WHERE id = 'DEMO-R-2'`)
	if !eqStrings(keysOf(rows(t, r.text)[0]), want) {
		t.Fatalf("labels: %s", r.text)
	}
}

func TestGoalStatusFromTheSuccessMetric(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, success_metric, status FROM goals WHERE product = 'DEMO'`),
		`{"entity":"goals","scope":"product DEMO","count":1,"records":[{"id":"DEMO-G-1","success_metric":"Monthly viewers","status":"On track"}]}`)
	for _, c := range e.fake.Calls() {
		if c.Path == "/products/"+idDEMO+"/goals" {
			names := strings.Split(c.Query.Get("fields"), ",")
			n := 0
			for _, f := range names {
				if f == "success_metric" {
					n++
				}
				if f == "workflow_status" {
					t.Fatalf("fields names workflow_status: %s", c.Query.Get("fields"))
				}
			}
			if n != 1 {
				t.Fatalf("fields names success_metric %d times", n)
			}
			return
		}
	}
	t.Fatalf("no goal list: %s", e.fake.describeCalls())
}

func TestReleaseOfAnEpic(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, release FROM epics WHERE product = 'DEMO'`),
		`{"entity":"epics","scope":"product DEMO","count":1,"records":[{"id":"DEMO-E-1","release":"DEMO-R-3"}]}`)
	found := false
	for _, c := range e.fake.Calls() {
		if c.Path == "/products/"+idDEMO+"/epics" {
			found = true
			names := map[string]bool{}
			for _, f := range strings.Split(c.Query.Get("fields"), ",") {
				names[f] = true
			}
			if !names["release"] || names["release_reference_num"] || names["color"] {
				t.Fatalf("fields: %s", c.Query.Get("fields"))
			}
		}
	}
	if !found {
		t.Fatal(e.fake.describeCalls())
	}
	e.fake.Reset()
	wantResult(t, s.query(`SELECT name FROM epics WHERE release = 'demo-r-3'`),
		`{"entity":"epics","scope":"release DEMO-R-3","count":1,"records":[{"name":"Race day"}]}`)
	if e.fake.count("GET", "/releases/DEMO-R-3/epics") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestStatusesOfAnEntity(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	six := `{"entity":"statuses","scope":"product DEMO","count":6,"records":[{"name":"Under consideration"},{"name":"In design"},{"name":"Ready to develop"},{"name":"In development"},{"name":"Shipped"},{"name":"Will not implement"}]}`
	wantResult(t, s.query(`SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'features'`), six)
	if e.fake.count("GET", "/products/"+idDEMO) != 1 || e.fake.count("GET", "/products/"+idDEMO+"/workflows") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
	wantResult(t, s.query(`SELECT name FROM statuses WHERE product = 'demo' AND entity = 'feature'`), six)
	wantResult(t, s.query(`SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'epics'`), six)
	wantResult(t, s.query(`SELECT entity FROM statuses WHERE product = 'DEMO' AND name = 'Shipped'`),
		`{"entity":"statuses","scope":"product DEMO","count":5,"records":[{"entity":"releases"},{"entity":"features"},{"entity":"epics"},{"entity":"requirements"},{"entity":"ideas"}]}`)
}

func TestCurrentUser(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT email FROM me`),
		`{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
}

func TestCommentsOfARecord(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	want := `{"entity":"comments","scope":"feature DEMO-3","count":1,"records":[{"body":"Verified with the provider.","user":"Test User"}]}`
	wantResult(t, s.query(`SELECT body, user FROM comments WHERE parent = 'DEMO-3'`), want)
	wantResult(t, s.query(`SELECT body, user FROM comments WHERE feature = 'demo-3'`), want)
}

func TestCustomFieldsAreReadByName(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, custom.priority, custom.go_live FROM features WHERE release = 'DEMO-R-2'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","custom.priority":"High","custom.go_live":"2026-04-15"},{"id":"DEMO-2","custom.priority":"Medium","custom.go_live":null}]}`)
	e.wantCallSet("GET /custom_field_definitions", "GET /releases/DEMO-R-2/features")
}

func TestUnknownCustomFieldListsTheCustomFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id, custom.severity FROM features WHERE product = 'DEMO'`),
		`unknown custom field 'custom.severity' on features. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score. To see their types: DESCRIBE features`)
	e.wantCalls("GET /custom_field_definitions")
}

func TestCustomFieldOfATypeThatIsNotSupported(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id, custom.business_case FROM features WHERE id = 'DEMO-1'`),
		`custom.business_case has the type EquationField, and the server cannot read or set custom fields of that type. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score.`)
	if e.fake.count("GET", "/features/DEMO-1") != 0 {
		t.Fatal("record was read")
	}
}

func TestEntityWithoutCustomFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT body, custom.priority FROM comments WHERE parent = 'DEMO-3'`),
		`comments have no custom fields that can be used here. Fields of comments: id, body, user, parent, created_at, url. Example: SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'`)
	e.wantNoCall()
	wantError(t, s.query(`SELECT id, custom.priority FROM goals WHERE product = 'DEMO'`),
		`goals have no custom fields that can be used here. Fields of goals: id, name, success_metric, status, product, progress, value, effort, description, created_at, updated_at, url. Example: SELECT id, name FROM goals WHERE product = 'PROD'`)
}

func TestCustomFieldsThatTheKeyCannotRead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.Answer("GET", "/custom_field_definitions", 403, -1, nil, `{"error":"Forbidden"}`)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id, custom.priority FROM features WHERE product = 'DEMO'`),
		`the custom fields of the account cannot be read with this API key (status 403). Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`)
	wantError(t, s.query(`SELECT id, severity FROM features WHERE product = 'DEMO'`),
		`unknown field 'severity' on features. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`)
	r := s.query(`DESCRIBE features`)
	if r.isError {
		t.Fatal(r.text)
	}
	d := orderedObject(t, r.text)
	var fields []any
	_ = json.Unmarshal(memberOf(t, d, "fields"), &fields)
	if len(fields) != 17 || hasMember(d, "custom_fields") {
		t.Fatalf("describe: %s", r.text)
	}
	r = s.query(`SELECT * FROM features WHERE id = 'DEMO-1'`)
	if r.isError || !eqStrings(keysOf(rows(t, r.text)[0]), featureFields) {
		t.Fatalf("star: %s", r.text)
	}
	wantResult(t, s.query(`SELECT id, name FROM features WHERE release = 'DEMO-R-2'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","name":"CSV export"},{"id":"DEMO-2","name":"Offline maps"}]}`)
}

func TestCustomFieldThatWasAddedLater(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`)
	e.fake.AddDefinition("7000000000000000820", "risk", "Risk", "CustomFieldDefinitions::TextField", "Feature")
	e.fake.Reset()
	wantResult(t, s.query(`SELECT id, custom.risk FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.risk":null}]}`)
	if e.fake.count("GET", "/custom_field_definitions") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestOwnNameOfTheEntity(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT feature_id, feature_name FROM features WHERE feature = 'DEMO-5'`),
		`{"entity":"features","scope":"feature DEMO-5","count":1,"records":[{"feature_id":"DEMO-5","feature_name":"Rider's handbook"}]}`)
	e.wantCalls("GET /features/DEMO-5")
}

func TestUnknownFieldListsTheFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id, severity FROM features WHERE product = 'DEMO'`),
		`unknown field 'severity' on features. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`)
	e.wantCalls("GET /custom_field_definitions")
}

func TestUnknownEntityListsTheEntities(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT * FROM tickets`),
		`unknown entity 'tickets'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`)
}

func TestUsersCannotBeFilteredByProduct(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT name FROM users WHERE product = 'DEMO'`),
		`users cannot be filtered by product. Fields of users: id, name, email, created_at. Example: SELECT name, email FROM users`)
}

func TestBareKeyOfACustomField(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, priority FROM features WHERE id = 'demo-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","priority":"High"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE release = 'DEMO-R-2' AND Custom.Priority = 'medium'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":1,"records":[{"id":"DEMO-2"}]}`)
}

func TestCatalogNameWinsOverACustomKey(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddDefinition("7000000000000000821", "status", "Review status", "CustomFieldDefinitions::TextField", "Feature")
	e.fake.SetCustomValue("DEMO-1", "status", "Checked")
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT status, custom.status FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"status":"In design","custom.status":"Checked"}]}`)
}

func TestProductInThreeSpellings(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	want := `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1"},{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'demo'`), want)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = '7000000000000000001'`), want)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'Demo Product'`), want)
}

func TestTheProductListIsNotKept(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	want := `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1"},{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO'`), want)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO'`), want)
	n := 0
	for _, c := range e.fake.Calls() {
		if c.Method == "GET" && c.Path == "/products" {
			n++
			if strings.Contains(c.Query.Get("fields"), "description") {
				t.Fatalf("fields names description: %s", c.Query.Get("fields"))
			}
		}
	}
	if n != 2 {
		t.Fatalf("product list read %d times", n)
	}
}

func TestProductsAreNotMixed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, name FROM goals WHERE product = 'other'`),
		`{"entity":"goals","scope":"product OTHER","count":1,"records":[{"id":"OTHER-G-1","name":"Other goal"}]}`)
}

func TestUnknownProductListsTheKeys(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM releases WHERE product = 'DEMOX'`),
		`unknown product 'DEMOX'. Product keys: DEMO, OTHER. To see their names: SELECT key, name FROM products`)
}

func TestReleaseIDWhereAProductIsNeeded(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE product = 'DEMO-R-3'`),
		`'DEMO-R-3' is a release, not a product. Use: WHERE release = 'DEMO-R-3'`)
}

func TestInListReadsEachParent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, release FROM features WHERE release IN ('DEMO-R-1', 'DEMO-R-2')`),
		`{"entity":"features","scope":"release DEMO-R-1, release DEMO-R-2","count":3,"records":[{"id":"DEMO-6","release":"DEMO-R-1"},{"id":"DEMO-1","release":"DEMO-R-2"},{"id":"DEMO-2","release":"DEMO-R-2"}]}`)
	e.wantCalls("GET /releases/DEMO-R-1/features", "GET /releases/DEMO-R-2/features")
}

func TestCountNeedsOneListRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddProduct("BULK", "7000000000000000003", 60, "BULK-R-1")
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT COUNT(*) FROM features WHERE product = 'BULK'`),
		`{"entity":"features","scope":"product BULK","count":1,"records":[{"COUNT(*)":60}]}`)
	if n := e.fake.count("GET", "/products/7000000000000000003/features"); n != 1 {
		t.Fatalf("feature list read %d times: %s", n, e.fake.describeCalls())
	}
}

func TestEveryPageIsReadOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddProduct("BIG", "7000000000000000004", 450, "")
	s := e.stdio(nil)
	r := s.query(`SELECT id FROM features WHERE product = 'BIG' ORDER BY id`)
	var pages []string
	for _, c := range e.fake.Calls() {
		if c.Path == "/products/7000000000000000004/features" {
			pages = append(pages, c.Query.Get("page"))
			if c.Query.Get("per_page") != "200" {
				t.Fatalf("per_page %s", c.Query.Get("per_page"))
			}
		}
	}
	if !eqStrings(pages, []string{"1", "2", "3"}) {
		t.Fatalf("pages: %v", pages)
	}
	top := orderedObject(t, r.text)
	if string(memberOf(t, top, "count")) != "50" || string(memberOf(t, top, "total")) != "450" || string(memberOf(t, top, "truncated")) != "true" {
		t.Fatalf("result: %.300s", r.text)
	}
}

func TestTooManyPages(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddProduct("HUGE", "7000000000000000005", 3100, "")
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id, name FROM features WHERE product = 'HUGE' ORDER BY name`),
		`this would read 3100 records. Narrow it down, for example with release = 'PROD-R-3'`)
	path := "/products/7000000000000000005/features"
	if n := e.fake.count("GET", path); n != 1 {
		t.Fatalf("list read %d times", n)
	}
	r := s.query(`SELECT id, name FROM features WHERE product = 'HUGE'`)
	top := orderedObject(t, r.text)
	if string(memberOf(t, top, "count")) != "50" || string(memberOf(t, top, "total")) != "3100" || string(memberOf(t, top, "truncated")) != "true" {
		t.Fatalf("result: %.300s", r.text)
	}
	if n := e.fake.count("GET", path); n != 2 {
		t.Fatalf("list read %d times in total", n)
	}
}

func TestReleaseByNameListsTheReleases(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND release = 'Summer release'`),
		`release = needs the id of the release, not its name. The releases of DEMO: DEMO-R-1 (What's next), DEMO-R-2 (Spring release), DEMO-R-3 (Summer release), DEMO-R-4 (Rider's cup release)`)
}

func TestParentThatDoesNotExist(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE release = 'DEMO-R-99'`),
		`there is no release with the id 'DEMO-R-99'. To find it: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`)
}

func TestReleaseIDWhereAFeatureIDIsNeeded(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT * FROM features WHERE id = 'DEMO-R-3'`),
		`'DEMO-R-3' is the id of a release, not of a feature. For the features of that release: SELECT * FROM features WHERE release = 'DEMO-R-3'`)
	for _, c := range e.fake.Calls() {
		if c.Method == "GET" && strings.Count(c.Path, "/") == 2 && !strings.HasPrefix(c.Path, "/products") {
			t.Fatalf("ApiGet call: %s", c.Path)
		}
	}
}

func TestCommentsNeedAParent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT body FROM comments`),
		`comments are listed per record: add parent = '...' to WHERE. Example: SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'`)
}

func TestAnswersInTheJSONFormOfAha(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, status FROM features WHERE release = 'DEMO-R-2'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","status":"In design"},{"id":"DEMO-2","status":"Under consideration"}]}`)
	c := e.fake.Calls()[0]
	names := map[string]bool{}
	for _, f := range strings.Split(c.Query.Get("fields"), ",") {
		names[f] = true
		if strings.Contains(f, ".") {
			t.Fatalf("fields has a dot: %s", c.Query.Get("fields"))
		}
	}
	if !names["reference_num"] || !names["workflow_status"] {
		t.Fatalf("fields: %s", c.Query.Get("fields"))
	}
	wantResult(t, s.query(`SELECT id, status FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","status":"In design"}]}`)
}

func TestAnswerWithAnUnknownForm(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.Answer("GET", "/releases/DEMO-R-2/features", 200, -1, nil, `{"items":[]}`)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE release = 'DEMO-R-2'`),
		`aha: API error (status 200) on GET /releases/DEMO-R-2/features: the answer has an unknown form`)
}

func TestCustomFieldsAreRequestedOnlyWhenUsed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	s.query(`SELECT id, name FROM features WHERE release = 'DEMO-R-2'`)
	e.wantCalls("GET /releases/DEMO-R-2/features")
	if strings.Contains(e.fake.Calls()[0].Query.Get("fields"), "custom_fields") {
		t.Fatal("fields names custom_fields")
	}
	e.fake.Reset()
	wantResult(t, s.query(`SELECT id, custom.effort_days FROM features WHERE release = 'DEMO-R-2' ORDER BY custom.effort_days`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-2","custom.effort_days":8},{"id":"DEMO-1","custom.effort_days":12.5}]}`)
	for _, c := range e.fake.Calls() {
		if c.Path == "/releases/DEMO-R-2/features" && !strings.Contains(c.Query.Get("fields"), "custom_fields") {
			t.Fatalf("fields does not name custom_fields: %s", c.Query.Get("fields"))
		}
		if strings.HasPrefix(c.Path, "/features/") {
			t.Fatalf("ApiGet on a feature: %s", c.Path)
		}
	}
}

func TestListWithoutCustomFieldsIsAnError(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.OmitCustomFieldsInLists(true)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id, custom.priority FROM features WHERE release = 'DEMO-R-2'`),
		`Aha! returned this list of features without custom fields. A record that is read by its id has them: add id = '...' to WHERE.`)
	for _, c := range e.fake.Calls() {
		if strings.HasPrefix(c.Path, "/features/") {
			t.Fatalf("ApiGet on a feature: %s", c.Path)
		}
	}
	wantResult(t, s.query(`SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`)
}

func TestInListIsCheckedBeforeTheFirstRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT id FROM features WHERE id IN ('DEMO-1', 'DEMO-R-2')`),
		`'DEMO-R-2' is the id of a release, not of a feature. For the features of that release: SELECT * FROM features WHERE release = 'DEMO-R-2'`)
	e.wantNoCall()
	wantResult(t, s.query(`SELECT id FROM features WHERE id IN ('DEMO-1', 'demo-1', 'DEMO-2')`),
		`{"entity":"features","scope":"feature DEMO-1, feature DEMO-2","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`)
	e.wantCalls("GET /features/DEMO-1", "GET /features/DEMO-2")
}

func TestDescriptionOfAProductIsRequestedWhenItIsUsed(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	s.query(`SELECT key, name FROM products`)
	e.wantCalls("GET /products")
	if strings.Contains(e.fake.Calls()[0].Query.Get("fields"), "description") {
		t.Fatal("fields names description")
	}
	e.fake.Reset()
	r := s.query(`SELECT key, description FROM products`)
	e.wantCalls("GET /products")
	if !strings.Contains(e.fake.Calls()[0].Query.Get("fields"), "description") {
		t.Fatal("fields does not name description")
	}
	rs := rows(t, r.text)
	if len(rs) != 2 {
		t.Fatal(r.text)
	}
	for _, row := range rs {
		if !eqStrings(keysOf(row), []string{"key", "description"}) {
			t.Fatalf("labels: %v", keysOf(row))
		}
	}
}

func TestCustomFieldOfAProduct(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT key, custom.cost_center FROM products`),
		`{"entity":"products","scope":"all products","count":2,"records":[{"key":"DEMO","custom.cost_center":"CC-100"},{"key":"OTHER","custom.cost_center":null}]}`)
	ok := false
	for _, c := range e.fake.Calls() {
		if c.Path == "/products" && strings.Contains(c.Query.Get("fields"), "custom_fields") {
			ok = true
		}
	}
	if !ok {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestParkingLotAndMissingDate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, release_date, parking_lot FROM releases WHERE product = 'DEMO'`),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","release_date":"2026-12-15","parking_lot":false}]}`)
}

func TestDescriptionIsPlainText(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT description FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"description":"Export all columns.\nKeep the order."}]}`)
}

func TestCustomValuesByKind(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT custom.effort_days, custom.markets, custom.reviewers, custom.release_notes, custom.value_score FROM features WHERE id = 'DEMO-1'`),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"custom.effort_days":12.5,"custom.markets":"EMEA, APAC","custom.reviewers":"second.user@example.com","custom.release_notes":"Adds the export.\nSee the handbook.","custom.value_score":31}]}`)
	wantResult(t, s.query(`SELECT custom.effort_days, custom.markets, custom.go_live FROM features WHERE id = 'DEMO-2'`),
		`{"entity":"features","scope":"feature DEMO-2","count":1,"records":[{"custom.effort_days":8,"custom.markets":null,"custom.go_live":null}]}`)
}

func TestOrAndNotAreEvaluated(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND (status = 'Shipped' OR status = 'In design')`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-5"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE release = 'DEMO-R-3' AND NOT status = 'Shipped'`),
		`{"entity":"features","scope":"release DEMO-R-3","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`)
}

func TestMeAsAValue(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND assigned_to = 'me'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`)
}

func TestFieldsWithoutAValueInConditions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' AND release_date IS NULL`),
		`{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-1"}]}`)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' AND parking_lot = false`),
		`{"entity":"releases","scope":"product DEMO","count":3,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE release = 'DEMO-R-2' AND assigned_to != 'test.user@example.com'`),
		`{"entity":"features","scope":"release DEMO-R-2","count":1,"records":[{"id":"DEMO-2"}]}`)
}

func TestConditionsOnCustomFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.priority = 'high'`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-3"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.effort_days > 5`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.go_live < '2026-06-01'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers CONTAINS 'me'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers CONTAINS 'second user'`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-3"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.priority IS NULL`),
		`{"entity":"features","scope":"product DEMO","count":3,"records":[{"id":"DEMO-4"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`)
}

func TestFieldsWithSeveralValues(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, custom.markets FROM features WHERE product = 'DEMO' AND custom.markets = 'emea'`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1","custom.markets":"EMEA, APAC"},{"id":"DEMO-3","custom.markets":"EMEA"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.markets = 'APAC, EMEA'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.markets != 'APAC'`),
		`{"entity":"features","scope":"product DEMO","count":5,"records":[{"id":"DEMO-2"},{"id":"DEMO-3"},{"id":"DEMO-4"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers = 'me'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`)
	e.fake.Reset()
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers = 'second.user@example.com, me'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`)
	if n := e.fake.count("GET", "/me"); n != 1 {
		t.Fatalf("/me read %d times", n)
	}
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers IN ('Second User', 'nobody@example.com')`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-3"}]}`)
	wantResult(t, s.query(`SELECT id, tags FROM features WHERE product = 'DEMO' AND tags = 'LIVE'`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3","tags":"live, dashboard"},{"id":"DEMO-4","tags":"live"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND tags = 'dashboard, live'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND tags CONTAINS 'board'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`)
}

func TestTimestampsArePointsInTime(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	one := `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND updated_at = '2026-02-02T08:00:00Z'`), one)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND updated_at <= '2026-02-02T09:00:00+01:00'`), one)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND updated_at > '2026-02-04T00:15:00+01:00'`),
		`{"entity":"features","scope":"product DEMO","count":5,"records":[{"id":"DEMO-2"},{"id":"DEMO-3"},{"id":"DEMO-4"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND updated_at = '2026-02-02'`), one)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' AND release_date >= '2026-06-30T00:00:00Z'`),
		`{"entity":"releases","scope":"product DEMO","count":2,"records":[{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`)
}

func TestEmptyTextAndNullStandForNoValue(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND initiative = ''`),
		`{"entity":"features","scope":"product DEMO","count":4,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' AND initiative != ''`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`)
	e.fake.Reset()
	r := s.query(`SELECT id FROM features WHERE product = 'DEMO' AND initiative IN ('DEMO-S-1', NULL)`)
	top := orderedObject(t, r.text)
	if string(memberOf(t, top, "scope")) != `"product DEMO"` || string(memberOf(t, top, "count")) != "6" {
		t.Fatalf("result: %s", r.text)
	}
	if e.fake.count("GET", "/initiatives/DEMO-S-1/features") != 0 {
		t.Fatal("initiative list was read")
	}
}

func TestGroupingWithoutGroupBy(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT release, COUNT(*) FROM features WHERE product = 'DEMO'`),
		`{"entity":"features","scope":"product DEMO","count":3,"records":[{"release":"DEMO-R-2","COUNT(*)":2},{"release":"DEMO-R-3","COUNT(*)":3},{"release":"DEMO-R-1","COUNT(*)":1}]}`)
}

func TestRowsWithoutAValueSortLast(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date`),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"},{"id":"DEMO-R-1"}]}`)
	wantResult(t, s.query(`SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date DESC`),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-4"},{"id":"DEMO-R-3"},{"id":"DEMO-R-2"},{"id":"DEMO-R-1"}]}`)
}

func TestCountOfNothingIsZero(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT COUNT(*) FROM features WHERE product = 'DEMO' AND status = 'Will not implement'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":0}]}`)
}

func TestLimitOfTheStatementKeepsTheTotal(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE product = 'DEMO' LIMIT 2`),
		`{"entity":"features","scope":"product DEMO","count":2,"total":6,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`)
}

func TestIDsAreSortedByTheirNumbers(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddProduct("BULK", "7000000000000000003", 60, "BULK-R-1")
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id FROM features WHERE release = 'BULK-R-1' ORDER BY id DESC LIMIT 2`),
		`{"entity":"features","scope":"release BULK-R-1","count":2,"total":60,"records":[{"id":"BULK-60"},{"id":"BULK-59"}]}`)
	wantResult(t, s.query(`SELECT MIN(id), MAX(id) FROM features WHERE release = 'BULK-R-1'`),
		`{"entity":"features","scope":"release BULK-R-1","count":1,"records":[{"MIN(id)":"BULK-1","MAX(id)":"BULK-60"}]}`)
	wantResult(t, s.query(`SELECT id FROM features WHERE release = 'BULK-R-1' AND id > 'BULK-58' ORDER BY id`),
		`{"entity":"features","scope":"release BULK-R-1","count":2,"records":[{"id":"BULK-59"},{"id":"BULK-60"}]}`)
}

func TestOrderByALabelAndByAnAggregate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT name AS n FROM features WHERE release = 'DEMO-R-2' ORDER BY n DESC`),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"n":"Offline maps"},{"n":"CSV export"}]}`)
	wantResult(t, s.query(`SELECT status FROM features WHERE product = 'DEMO' GROUP BY status ORDER BY COUNT(*) DESC, status`),
		`{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"Under consideration"},{"status":"In design"},{"status":"In development"},{"status":"Shipped"}]}`)
}

func TestRecordsWithoutAValueAreOneGroup(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT initiative, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY initiative`),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"initiative":null,"COUNT(*)":4},{"initiative":"DEMO-S-1","COUNT(*)":2}]}`)
}

func TestArticlesAndIDsInMessages(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`SELECT * FROM features WHERE epic = 'DEMO'`), `'DEMO' is a product, not an epic. Use: WHERE product = 'DEMO'`)
	wantError(t, s.query(`SELECT * FROM epics WHERE id = 'DEMO-S-1'`),
		`'DEMO-S-1' is the id of an initiative, not of an epic. For the epics of that initiative: SELECT * FROM epics WHERE initiative = 'DEMO-S-1'`)
	wantError(t, s.execute(`INSERT INTO ideas (parent) VALUES ('DEMO')`), `name is required to create an idea`)
	wantResult(t, s.query(`SELECT * FROM products WHERE key = 'NOPE'`),
		`{"entity":"products","scope":"all products","count":0,"records":[],"note":"There is no product with the key 'NOPE'."}`)
	wantError(t, s.query(`demo-3`), `DEMO-3 is an id, not a statement. To read that record: SELECT * FROM features WHERE id = 'DEMO-3'`)
	r := s.query(`SELECT * FROM me WHERE name = 'x'`)
	note := ""
	_ = json.Unmarshal(memberOf(t, orderedObject(t, r.text), "note"), &note)
	if !strings.HasSuffix(note, `No user has exactly the name 'x'. name = compares the whole name; to search inside names use: name CONTAINS 'x'`) {
		t.Fatalf("note: %s", note)
	}
	wantError(t, s.execute(`INSERT INTO comments (parent, body) VALUES ('DEMO-1-9', 'c')`),
		`there is no requirement with the id 'DEMO-1-9'. To find the id, use the query tool: SELECT id, name FROM requirements WHERE feature = 'DEMO-1'`)
}

// Every statement that the spec shows in messages and descriptions is
// accepted: the query tool reads it without a syntax error or a name error.
func TestStatementsInMessagesCanBeSent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.AddProduct("PROD", "7000000000000000006", 0, "")
	s := e.stdio(map[string]string{"AHA_DEFAULT_PRODUCT": "PROD"})
	reads := []string{
		`SELECT key, name FROM products`,
		`SELECT id, name, release_date FROM releases WHERE product = 'PROD'`,
		`SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`,
		`SELECT id, name, status FROM epics WHERE product = 'PROD'`,
		`SELECT id, name, status FROM requirements WHERE feature = 'PROD-23'`,
		`SELECT id, name, status FROM initiatives WHERE product = 'PROD'`,
		`SELECT id, name FROM goals WHERE product = 'PROD'`,
		`SELECT id, name, votes FROM ideas WHERE product = 'PROD' ORDER BY votes DESC LIMIT 5`,
		`SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'`,
		`SELECT name, email FROM users`,
		`SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'`,
		`SELECT name, email FROM me`,
		`SELECT status, COUNT(*) FROM features WHERE product = 'PROD' GROUP BY status`,
		`SELECT id, name, release_date FROM releases WHERE product = 'PROD' AND parking_lot = false ORDER BY release_date`,
		`SELECT email FROM me`,
		`SELECT COUNT(*) FROM features WHERE product = 'PROD'`,
		`SELECT id, name FROM releases WHERE product = 'PROD'`,
		`SELECT id, name FROM requirements WHERE feature = 'PROD-23'`,
		`SELECT id, body FROM comments WHERE parent = 'PROD-23'`,
		`DESCRIBE features`,
		`SELECT id, custom.customer FROM features WHERE id = 'PROD-23'`,
	}
	for _, q := range reads {
		r := s.query(q)
		if r.isError && (strings.HasPrefix(r.text, "cannot read") || strings.HasPrefix(r.text, "unknown") || strings.HasPrefix(r.text, "a statement starts")) {
			t.Fatalf("%s: %s", q, r.text)
		}
	}
	// The write examples: dry-run in manual mode checks the form without a write.
	for _, w := range []string{
		`UPDATE features SET custom.priority = '...' WHERE id = '...'`,
		`INSERT INTO features (parent, name) VALUES ('PROD-R-3', 'Export to CSV')`,
		`UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = 'PROD-23'`,
		`INSERT INTO comments (parent, body) VALUES ('PROD-23', 'It''s ready.')`,
	} {
		res := e.run("", nil, "execute", "dry-run=true", "statement="+w)
		if strings.Contains(res.stderr, "error: cannot read") || strings.Contains(res.stderr, "Forms:") || strings.Contains(res.stderr, "unknown field") {
			t.Fatalf("%s: %s", w, res.stderr)
		}
	}
	e.wantNoWrite()
}
