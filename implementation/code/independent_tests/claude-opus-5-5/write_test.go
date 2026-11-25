// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e
// tests by: claude-opus-5-5

package aqltests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

var logLine = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z) execute account=(\S+) op=(\S+) entity=(\S+) id=(\S+) result=(\S+)$`)

func TestExecuteResultEnvelope(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'Export to PDF')`),
		`{"id":"DEMO-7","name":"Export to PDF","parent":"DEMO-R-3","message":"Feature DEMO-7 created"}`)
	if w := e.fake.writeCalls(); len(w) != 1 || w[0].Method != "POST" || w[0].Path != "/releases/DEMO-R-3/features" {
		t.Fatalf("writes: %s", e.fake.describeCalls())
	}
	e.wantBody("POST", "/releases/DEMO-R-3/features", `{"feature":{"name":"Export to PDF"}}`)
}

func TestExecuteRejectsUnknownParameter(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.call("execute", map[string]any{"query": `DELETE FROM features WHERE id = 'DEMO-6'`}, 60*time.Second),
		`tool execute has one parameter: statement (one AQL statement). It does not accept: query. Pass the statement as statement.`)
	e.wantNoCall()
}

func TestExecuteRefusesReadStatements(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`SELECT id FROM features WHERE product = 'DEMO'`), `execute only changes data. Use the query tool for SELECT and DESCRIBE.`)
	e.wantNoCall()
}

func TestExecuteRequiresAStatement(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.call("execute", map[string]any{}, 60*time.Second), `statement parameter is required: one INSERT, UPDATE or DELETE statement`)
	wantError(t, s.execute("  "), `statement parameter is required: one INSERT, UPDATE or DELETE statement`)
	wantError(t, s.call("execute", map[string]any{"statement": 5}, 60*time.Second), `statement parameter is required: one INSERT, UPDATE or DELETE statement`)
	e.wantNoCall()
}

func TestExecuteWritesALogLine(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	before := len(lines(s.stderr.String()))
	start := time.Now().UTC().Truncate(time.Second)
	wantResult(t, s.execute(`UPDATE features SET status = 'shipped' WHERE id = 'demo-1'`),
		`{"id":"DEMO-1","name":"CSV export","status":"Shipped","message":"Feature DEMO-1 status changed to 'Shipped'"}`)
	end := time.Now().UTC()
	var newLines []string
	waitFor(func() bool { newLines = lines(s.stderr.String())[before:]; return len(newLines) >= 1 }, 2*time.Second)
	time.Sleep(100 * time.Millisecond)
	newLines = lines(s.stderr.String())[before:]
	if len(newLines) != 1 {
		t.Fatalf("diagnostic lines: %q", newLines)
	}
	m := logLine.FindStringSubmatch(newLines[0])
	if m == nil || m[2] != "example" || m[3] != "update" || m[4] != "features" || m[5] != "DEMO-1" || m[6] != "ok" {
		t.Fatalf("log line: %q", newLines[0])
	}
	ts, err := time.Parse("2006-01-02T15:04:05Z", m[1])
	if err != nil || ts.Before(start) || ts.After(end) {
		t.Fatalf("time %s not between %s and %s", m[1], start, end)
	}
}

func TestExecuteLogsARefusedStatement(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	before := len(lines(s.stderr.String()))
	wantErrorPrefix(t, s.execute(`DELETE FROM features WHERE name = 'CSV export'`),
		`one statement changes one record: the WHERE must start with id = '...'.`)
	time.Sleep(200 * time.Millisecond)
	nl := lines(s.stderr.String())[before:]
	if len(nl) != 1 {
		t.Fatalf("diagnostic lines: %q", nl)
	}
	m := logLine.FindStringSubmatch(nl[0])
	if m == nil || m[2] != "example" || m[3] != "none" || m[4] != "none" || m[5] != "none" || m[6] != "syntax" {
		t.Fatalf("log line: %q", nl[0])
	}
	all := s.stderr.String()
	if strings.Contains(all, "CSV export") || strings.Contains(all, "fixture-key") {
		t.Fatalf("diagnostic output has a value: %s", all)
	}
	e.wantNoCall()
}

func TestInsertWithOtherSpellings(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`INSERT INTO features (release_id, title, assignee) VALUES ('DEMO-R-2', 'Dark mode', 'me')`),
		`{"id":"DEMO-7","name":"Dark mode","parent":"DEMO-R-2","message":"Feature DEMO-7 created"}`)
	e.wantBody("POST", "/releases/DEMO-R-2/features", `{"feature":{"name":"Dark mode","assigned_to_user":"test.user@example.com"}}`)
	if e.fake.count("POST", "/releases/DEMO-R-2/features") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestInsertCommentWithApostrophe(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`INSERT INTO comments (parent, body) VALUES ('DEMO-3', 'It''s ready.')`)
	if r.isError {
		t.Fatal(r.text)
	}
	e.wantBody("POST", "/features/DEMO-3/comments", `{"comment":{"body":"It's ready."}}`)
	m := orderedObject(t, r.text)
	if !eqStrings(keysOf(m), []string{"id", "parent", "body", "created_at", "message"}) {
		t.Fatalf("members: %v", keysOf(m))
	}
	var id, created string
	_ = json.Unmarshal(memberOf(t, m, "id"), &id)
	_ = json.Unmarshal(memberOf(t, m, "created_at"), &created)
	if !regexp.MustCompile(`^[0-9]{6,}$`).MatchString(id) || !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T`).MatchString(created) {
		t.Fatal(r.text)
	}
	if string(memberOf(t, m, "parent")) != `"DEMO-3"` || string(memberOf(t, m, "body")) != `"It's ready."` ||
		string(memberOf(t, m, "message")) != `"Comment added to DEMO-3"` {
		t.Fatal(r.text)
	}
}

func TestInsertWithoutParent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`INSERT INTO features (name) VALUES ('Dark mode')`),
		`parent is required: INSERT needs the column parent with the id of a release. Form: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)`)
	e.wantNoWrite()
}

func TestInsertWithAProductAsRelease(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`INSERT INTO features (parent, name) VALUES ('DEMO', 'Dark mode')`),
		`'DEMO' is a product, not a release. Use the id of one of its releases: DEMO-R-1 (What's next), DEMO-R-2 (Spring release), DEMO-R-3 (Summer release), DEMO-R-4 (Rider's cup release)`)
	e.wantNoWrite()
}

func TestInsertUnderAnUnknownRelease(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`INSERT INTO features (parent, name) VALUES ('DEMO-R-99', 'Dark mode')`),
		`there is no release with the id 'DEMO-R-99'. To find the id, use the query tool: SELECT id, name FROM releases WHERE product = 'DEMO'`)
	r := s.query(`SELECT COUNT(*) FROM features WHERE product = 'DEMO'`)
	wantResult(t, r, `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`)
}

func TestInsertGoalWithoutASuccessMetric(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`INSERT INTO goals (parent, name) VALUES ('DEMO', 'Win the cup')`), `success_metric is required to create a goal`)
	wantError(t, s.execute(`INSERT INTO goals (parent, name, success_metric) VALUES ('DEMO', 'Win the cup', '')`), `success_metric is required to create a goal`)
	for _, c := range e.fake.Calls() {
		if c.Method == "POST" {
			t.Fatalf("ApiCreate: %s", c.Path)
		}
	}
}

func TestUpdateStatusInAnyCase(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`UPDATE features SET status = 'in design' WHERE id = 'demo-2'`),
		`{"id":"DEMO-2","name":"Offline maps","status":"In design","message":"Feature DEMO-2 status changed to 'In design'"}`)
	if e.fake.count("GET", "/features/DEMO-2") != 1 || e.fake.count("PUT", "/features/DEMO-2") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
	e.wantBody("PUT", "/features/DEMO-2", `{"feature":{"workflow_status":"In design"}}`)
	for _, c := range e.fake.Calls() {
		if c.Method == "GET" && strings.HasSuffix(c.Path, "/features") {
			t.Fatalf("feature list: %s", c.Path)
		}
	}
}

func TestUpdateAssignedToMe(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-2'`),
		`{"id":"DEMO-2","name":"Offline maps","assigned_to":"test.user@example.com","message":"Feature DEMO-2 updated"}`)
}

func TestUpdateWithATrueGuard(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-3' AND status = 'in development'`),
		`{"id":"DEMO-3","name":"Live dashboard","status":"Shipped","message":"Feature DEMO-3 status changed to 'Shipped'"}`)
}

func TestUpdateWithAFalseGuard(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-2' AND name = 'CSV export'`),
		`nothing was changed: feature DEMO-2 has name = 'Offline maps', not 'CSV export'`)
	if e.fake.count("GET", "/features/DEMO-2") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
	e.wantNoWrite()
}

func TestUpdateWithAGuardForNoValue(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-2' AND assigned_to = ''`),
		`{"id":"DEMO-2","name":"Offline maps","assigned_to":"test.user@example.com","message":"Feature DEMO-2 updated"}`)
	e.fake.ResetData()
	e.fake.Reset()
	wantError(t, s.execute(`UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-1' AND assigned_to = ''`),
		`nothing was changed: feature DEMO-1 has assigned_to = 'test.user@example.com', not ''`)
	e.wantNoWrite()
}

func TestUpdateOfAnUnknownRecord(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET name = 'X' WHERE id = 'DEMO-99'`), `there is no feature with the id 'DEMO-99'`)
	e.wantNoWrite()
}

func TestUpdateSendsEmptyText(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE features SET description = '' WHERE id = 'DEMO-1'`)
	e.wantBody("PUT", "/features/DEMO-1", `{"feature":{"description":""}}`)
	wantResult(t, r, `{"id":"DEMO-1","name":"CSV export","description":null,"message":"Feature DEMO-1 updated"}`)
}

func TestUpdateWithADateWithoutQuotes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`UPDATE features SET due_date = 2026-12-01 WHERE id = 'DEMO-1'`),
		`{"id":"DEMO-1","name":"CSV export","due_date":"2026-12-01","message":"Feature DEMO-1 updated"}`)
}

func TestUpdateOfTheNameShowsItOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE features SET name = 'CSV export 2', due_date = '2026-05-01' WHERE id = 'DEMO-1'`)
	e.wantBody("PUT", "/features/DEMO-1", `{"feature":{"name":"CSV export 2","due_date":"2026-05-01"}}`)
	wantResult(t, r, `{"id":"DEMO-1","name":"CSV export 2","due_date":"2026-05-01","message":"Feature DEMO-1 updated"}`)
}

func TestUpdateSetsACustomField(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`UPDATE features SET custom.priority = 'low' WHERE id = 'demo-2'`),
		`{"id":"DEMO-2","name":"Offline maps","custom.priority":"Low","message":"Feature DEMO-2 updated"}`)
	e.wantCallSet("GET /custom_field_definitions", "GET /features/DEMO-2", "GET /custom_field_definitions/7000000000000000801/options", "PUT /features/DEMO-2")
	e.wantBody("PUT", "/features/DEMO-2", `{"feature":{"custom_fields":{"priority":"Low"}}}`)
}

func TestUpdateWithACustomGuard(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-1' AND custom.priority = 'Low'`),
		`nothing was changed: feature DEMO-1 has custom.priority = 'High', not 'Low'`)
	e.wantNoWrite()
	wantResult(t, s.execute(`UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-1' AND custom.priority = 'high'`),
		`{"id":"DEMO-1","name":"CSV export","status":"Shipped","message":"Feature DEMO-1 status changed to 'Shipped'"}`)
}

func TestDeleteOneFeature(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`DELETE FROM features WHERE id = 'DEMO-6'`), `{"id":"DEMO-6","message":"Feature DEMO-6 deleted"}`)
	e.wantCalls("GET /features/DEMO-6", "DELETE /features/DEMO-6")
}

func TestDeleteAReleaseThroughItsProduct(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantResult(t, s.execute(`DELETE FROM releases WHERE id = 'DEMO-R-4'`), `{"id":"DEMO-R-4","message":"Release DEMO-R-4 deleted"}`)
	if e.fake.count("DELETE", "/products/"+idDEMO+"/releases/DEMO-R-4") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
}

func TestDeleteWithAFalseGuard(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`DELETE FROM features WHERE id = 'DEMO-6' AND status = 'Shipped'`),
		`nothing was changed: feature DEMO-6 has status = 'Under consideration', not 'Shipped'`)
	e.wantNoWrite()
}

func TestDeleteOfAnUnknownRecord(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`DELETE FROM features WHERE id = 'DEMO-99'`), `there is no feature with the id 'DEMO-99'`)
	e.wantNoWrite()
}

func TestDeleteWithACustomGuard(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`DELETE FROM features WHERE id = 'DEMO-2' AND custom.priority = 'High'`),
		`nothing was changed: feature DEMO-2 has custom.priority = 'Medium', not 'High'`)
	e.wantNoWrite()
	wantResult(t, s.execute(`DELETE FROM features WHERE id = 'DEMO-2' AND custom.priority = 'medium'`), `{"id":"DEMO-2","message":"Feature DEMO-2 deleted"}`)
}

func TestWriteWithoutAnIDIsRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	msg := `one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record.`
	wantError(t, s.execute(`UPDATE features SET name = 'X' WHERE status = 'Shipped'`), msg)
	e.wantNoCall()
	wantError(t, s.execute(`UPDATE features SET name = 'X' WHERE id = 'DEMO-1' OR id = 'DEMO-2'`), msg)
	e.wantNoCall()
}

func TestWriteWithoutWhereIsRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET name = 'X'`),
		`UPDATE needs WHERE id = '...' with the id of the one record to change. Forms: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...) | UPDATE entity SET field = 'value', ... WHERE id = 'id' | DELETE FROM entity WHERE id = 'id'`)
	e.wantNoCall()
	wantErrorPrefix(t, s.execute(`DELETE FROM features`), `DELETE needs WHERE id = '...' with the id of the one record to change.`)
	e.wantNoCall()
}

func TestInsertWithTwoRowsIsRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'A'), ('DEMO-R-3', 'B')`),
		`one INSERT creates one record. Send one statement per record.`)
	e.wantNoWrite()
}

const featureWritable = `Fields of features that can be set: name, status, release, initiative, assigned_to, start_date, due_date, tags, description, and parent in INSERT. Custom fields, with UPDATE only: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers`

func TestInsertWithAnUnknownColumnIsRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`INSERT INTO features (parent, name, bogus) VALUES ('DEMO-R-3', 'A', 'x')`),
		`unknown field 'bogus' on features. `+featureWritable)
	e.wantNoWrite()
}

func TestUpdateOfAFieldThatIsSetInInsertOnly(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE goals SET success_metric = 'Weekly viewers' WHERE id = 'DEMO-G-1'`),
		`the field success_metric of goals can be set in INSERT only. Fields of goals that can be set: name, success_metric (INSERT only), status, progress, description, and parent in INSERT`)
	if e.fake.count("GET", "/goals/DEMO-G-1") != 0 {
		t.Fatal("goal was read")
	}
	e.wantNoWrite()
}

func TestAssignmentOfAWordOrAnExpression(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET name = name WHERE id = 'DEMO-1'`), `text goes in single quotes: write 'name' instead of name`)
	e.wantNoCall()
	wantError(t, s.execute(`UPDATE features SET name = name + ' (old)' WHERE id = 'DEMO-1'`), `unexpected character + at position 33.`)
	e.wantNoCall()
}

func TestNullCannotBeWritten(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET due_date = NULL WHERE id = 'DEMO-1'`), `NULL cannot be written. Leave the field out.`)
	e.wantNoCall()
}

func TestWriteToAReadOnlyEntity(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE users SET name = 'X' WHERE id = '7000000000000000901'`),
		`users cannot be changed. Entities that can be changed: releases, features, epics, requirements, initiatives, goals, ideas, comments`)
	e.wantNoCall()
}

func TestProductsCannotBeChanged(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`INSERT INTO products (name, key) VALUES ('New product', 'NEW')`),
		`products cannot be changed. Entities that can be changed: releases, features, epics, requirements, initiatives, goals, ideas, comments`)
	e.wantNoCall()
}

func TestAFieldThatIsGivenTwice(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET status = 'Shipped', status = 'In design' WHERE id = 'DEMO-2'`),
		`the field status is given twice. A statement sets each field once.`)
	e.wantNoCall()
	wantError(t, s.execute(`UPDATE features SET release = 'DEMO-R-3', parent = 'DEMO-R-1' WHERE id = 'DEMO-2'`),
		`the field release is given twice. A statement sets each field once.`)
	wantError(t, s.execute(`INSERT INTO comments (parent, body, text) VALUES ('DEMO-3', 'one', 'two')`),
		`the field body is given twice. A statement sets each field once.`)
	e.wantNoCall()
	wantError(t, s.execute(`UPDATE features SET custom.customer = 'Acme', custom.CUSTOMER = 'Globex' WHERE id = 'DEMO-2'`),
		`the field custom.customer is given twice. A statement sets each field once.`)
	e.wantNoWrite()
}

func TestInsertWithACustomFieldIsRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	msg := `custom fields cannot be set in INSERT. Leave out custom.priority, create the record, and then send: UPDATE features SET custom.priority = '...' WHERE id = '...'`
	wantError(t, s.execute(`INSERT INTO features (parent, name, custom.priority) VALUES ('DEMO-R-3', 'A', 'High')`), msg)
	e.wantNoWrite()
	wantError(t, s.execute(`INSERT INTO features (parent, name, priority) VALUES ('DEMO-R-3', 'A', 'High')`), msg)
	e.wantNoWrite()
}

func TestCustomFieldThatCannotBeSet(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET custom.value_score = 40 WHERE id = 'DEMO-1'`),
		`the field custom.value_score of features cannot be set. `+featureWritable)
	if e.fake.count("GET", "/features/DEMO-1") != 0 {
		t.Fatal("record was read")
	}
	e.wantNoWrite()
}

func TestTagsAreSplitAtCommas(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE features SET tags = 'export, csv ,' WHERE id = 'DEMO-1'`)
	e.wantBody("PUT", "/features/DEMO-1", `{"feature":{"tags":"export,csv"}}`)
	wantResult(t, r, `{"id":"DEMO-1","name":"CSV export","tags":"export, csv","message":"Feature DEMO-1 updated"}`)
}

func TestUnknownStatusListsTheStatuses(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET status = 'Done' WHERE id = 'DEMO-2'`),
		`workflow status 'Done' not found for feature; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement`)
	e.wantNoWrite()
}

func TestStatusOfAnEpic(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE epics SET status = 'in development' WHERE id = 'DEMO-E-1'`)
	e.wantBody("PUT", "/epics/DEMO-E-1", `{"epic":{"workflow_status":"In development"}}`)
	wantResult(t, r, `{"id":"DEMO-E-1","name":"Race day","status":"In development","message":"Epic DEMO-E-1 status changed to 'In development'"}`)
	e.fake.ResetData()
	e.fake.Reset()
	wantError(t, s.execute(`UPDATE epics SET status = 'On track' WHERE id = 'DEMO-E-1'`),
		`workflow status 'On track' not found for epic; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement`)
	e.wantNoWrite()
}

func TestDateThatIsNotADate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET due_date = 'next friday' WHERE id = 'DEMO-1'`), `due_date: invalid date 'next friday', use YYYY-MM-DD`)
	e.wantNoWrite()
}

func TestUserByName(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET assigned_to = 'Second User' WHERE id = 'DEMO-2'`),
		`assigned_to: there is no user 'Second User'. Use an e-mail address, or 'me' for yourself`)
	e.wantNoWrite()
}

func TestCustomValuesAreCheckedByKind(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE features SET custom.effort_days = '4.5', custom.go_live = 2026-09-01, custom.customer = 'Initech', custom.platforms = 'ios, web' WHERE id = 'DEMO-2'`)
	e.wantBody("PUT", "/features/DEMO-2", `{"feature":{"custom_fields":{"effort_days":4.5,"go_live":"2026-09-01","customer":"Initech","platforms":["iOS","Web"]}}}`)
	wantResult(t, r, `{"id":"DEMO-2","name":"Offline maps","custom.effort_days":4.5,"custom.go_live":"2026-09-01","custom.customer":"Initech","custom.platforms":"iOS, Web","message":"Feature DEMO-2 updated"}`)
}

func TestOptionThatIsNotInTheList(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.execute(`UPDATE features SET custom.priority = 'Urgent' WHERE id = 'DEMO-2'`), `custom.priority: there is no option 'Urgent'. Options: High, Medium, Low`)
	wantError(t, s.execute(`UPDATE features SET custom.platforms = 'web, Linux' WHERE id = 'DEMO-2'`), `custom.platforms: there is no option 'Linux'. Options: Web, iOS, Android`)
	wantError(t, s.execute(`UPDATE features SET custom.effort_days = 'many' WHERE id = 'DEMO-2'`), `custom.effort_days must be a number`)
	wantError(t, s.execute(`UPDATE features SET custom.reviewers = 'Second User' WHERE id = 'DEMO-2'`),
		`custom.reviewers: there is no user 'Second User'. Use an e-mail address, or 'me' for yourself`)
	e.wantNoWrite()
}

func TestBodyOfAnUpdateRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE features SET status = 'in design', assigned_to = 'me', due_date = '2026-07-01' WHERE id = 'DEMO-2'`)
	if r.isError {
		t.Fatal(r.text)
	}
	e.wantBody("PUT", "/features/DEMO-2", `{"feature":{"workflow_status":"In design","assigned_to_user":"test.user@example.com","due_date":"2026-07-01"}}`)
}

func TestBodyOfACreateRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`INSERT INTO releases (parent, name, release_date, description) VALUES ('demo', 'Winter release', '2027-01-31', 'Maintenance only')`)
	if r.isError {
		t.Fatal(r.text)
	}
	e.wantBody("POST", "/products/"+idDEMO+"/releases", `{"release":{"name":"Winter release","release_date":"2027-01-31","theme":"Maintenance only"}}`)
}

func TestBodyWithCustomFields(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE features SET due_date = '2026-05-01', custom.markets = 'EMEA, LATAM ,', custom.reviewers = 'me, second.user@example.com', start_date = '2026-04-01', custom.release_notes = 'Ready.' WHERE id = 'DEMO-1'`)
	e.wantBody("PUT", "/features/DEMO-1", `{"feature":{"due_date":"2026-05-01","custom_fields":{"markets":["EMEA","LATAM"],"reviewers":["7000000000000000900","second.user@example.com"],"release_notes":"Ready."},"start_date":"2026-04-01"}}`)
	wantResult(t, r, `{"id":"DEMO-1","name":"CSV export","due_date":"2026-05-01","custom.markets":"EMEA, LATAM","custom.reviewers":"test.user@example.com, second.user@example.com","start_date":"2026-04-01","custom.release_notes":"Ready.","message":"Feature DEMO-1 updated"}`)
}

func TestBodiesForAGoal(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`INSERT INTO goals (parent, name, success_metric, status) VALUES ('DEMO', 'Win the cup', 'Podium places', 'at risk')`)
	e.wantBody("POST", "/products/"+idDEMO+"/goals", `{"goal":{"name":"Win the cup","success_metric":{"name":"Podium places","workflow_status":"At risk"}}}`)
	if string(memberOf(t, orderedObject(t, r.text), "message")) != `"Goal DEMO-G-2 created"` {
		t.Fatal(r.text)
	}
	e.fake.ResetData()
	e.fake.Reset()
	r = s.execute(`UPDATE goals SET status = 'achieved' WHERE id = 'DEMO-G-1'`)
	e.wantBody("PUT", "/goals/DEMO-G-1", `{"goal":{"workflow_status":"Achieved"}}`)
	wantResult(t, r, `{"id":"DEMO-G-1","name":"Grow the audience","status":"Achieved","message":"Goal DEMO-G-1 status changed to 'Achieved'"}`)
}

// INVARIANT: no JSON object of a body has two members with the same name,
// and an accepted write sends every assigned field.
func TestEveryAssignedFieldIsSent(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.execute(`UPDATE initiatives SET name = 'Coverage', end_date = '2026-09-30', value = 3, presented = true WHERE id = 'DEMO-S-1'`)
	if r.isError {
		t.Fatal(r.text)
	}
	e.wantBody("PUT", "/initiatives/DEMO-S-1", `{"initiative":{"name":"Coverage","end_date":"2026-09-30","value":3,"presented":true}}`)
}
