// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268
// tests by: claude-opus-5-5

package aqltests

import (
	"strings"
	"testing"
)

func TestManualQueryPrintsTheToolResult(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	want := `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","name":"CSV export"}]}`
	r := e.run("", nil, "query", "query=SELECT id, name FROM features WHERE id = 'demo-1'")
	if r.code != 0 || r.stdout != want+"\n" || r.stderr != "" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	e.wantCalls("GET /features/DEMO-1")
	s := e.stdio(nil)
	wantResult(t, s.query(`SELECT id, name FROM features WHERE id = 'demo-1'`), want)
}

func TestManualStatementFromStandardInput(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("SELECT email FROM me\n", nil, "query", "query=-")
	if r.code != 0 || r.stdout != `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`+"\n" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
}

func TestManualDryRunSendsNoWriteRequest(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "execute", "dry-run=true", "statement=UPDATE features SET status = 'shipped' WHERE id = 'DEMO-1'")
	if r.code != 0 || r.stdout != `{"id":"DEMO-1","status":"Shipped","message":"would update feature DEMO-1"}`+"\n" || r.stderr != "" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	if e.fake.count("GET", "/features/DEMO-1") != 1 {
		t.Fatal(e.fake.describeCalls())
	}
	e.wantNoWrite()
	r = e.run("", nil, "execute", "dry-run=true", "statement=UPDATE features SET status = 'done' WHERE id = 'DEMO-1'")
	if r.code != 1 || r.stdout != "" || !hasLine(r.stderr, `error: workflow status 'done' not found for feature; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement`) {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	e.wantNoWrite()
}

func TestManualDryRunOfInsertAndDelete(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "execute", "dry-run=true", "statement=INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'Export to PDF')")
	if r.code != 0 || r.stdout != `{"name":"Export to PDF","message":"would create feature under DEMO-R-3"}`+"\n" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	r = e.run("", nil, "execute", "dry-run=true", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	if r.code != 0 || r.stdout != `{"id":"DEMO-6","message":"would delete feature DEMO-6"}`+"\n" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	e.wantNoWrite()
}

func TestManualErrorGoesToStandardError(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "query", "query=SELECT id, name FROM releases")
	if r.code != 1 || r.stdout != "" || !hasLine(r.stderr, `error: releases are listed per product: add product = '...' to WHERE. Example: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`) {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
}

func TestManualStatementArgumentIsMissing(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "query")
	if r.code != 2 || !hasLine(r.stderr, `error: query needs query=STATEMENT, or query=- to read the statement from standard input`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	r = e.run("", nil, "execute")
	if r.code != 2 || !hasLine(r.stderr, `error: execute needs statement=STATEMENT, or statement=- to read the statement from standard input`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestManualExecuteIsRefusedInReadOnly(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	msg := `error: execute is not available, because read-only is true (option read-only or variable AHA_READ_ONLY)`
	r := e.run("", nil, "execute", "read-only=true", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	if r.code != 2 || !hasLine(r.stderr, msg) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
	r = e.run("", map[string]string{"AHA_READ_ONLY": "true"}, "execute", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	if r.code != 2 || !hasLine(r.stderr, msg) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoRequest()
}

func TestManualModeWritesNoLogLine(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "execute", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	if r.code != 0 || r.stdout != `{"id":"DEMO-6","message":"Feature DEMO-6 deleted"}`+"\n" || r.stderr != "" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
}

func TestManualQueryRefusesWrites(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("DELETE FROM features WHERE id = 'DEMO-1'\n", nil, "query", "query=-")
	if r.code != 1 || r.stdout != "" || strings.TrimSpace(r.stderr) != `error: query only reads. Use the execute tool for INSERT, UPDATE and DELETE.` {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	r = e.run("", nil, "execute", "statement=SELECT id FROM features")
	if r.code != 1 || strings.TrimSpace(r.stderr) != `error: execute only changes data. Use the query tool for SELECT and DESCRIBE.` {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantNoCall()
}

func TestDryRunShowsAListAsText(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "execute", "dry-run=true", "statement=UPDATE features SET custom.markets = 'EMEA, LATAM' WHERE id = 'DEMO-1'")
	if r.code != 0 || r.stdout != `{"id":"DEMO-1","custom.markets":"EMEA, LATAM","message":"would update feature DEMO-1"}`+"\n" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	e.wantNoWrite()
}

func TestExampleKeyIsChosenWhenItIsNeeded(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", nil, "query", "query=SELECT id FROM features WHERE id = 'DEMO-3'")
	if r.code != 0 || r.stdout != `{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`+"\n" {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	if e.fake.count("GET", "/products") != 0 {
		t.Fatal("product list was read")
	}
	e.fake.Reset()
	r = e.run("", nil, "query", "query=SELECT id FROM bugs")
	if r.code != 1 || !hasLine(r.stderr, `error: unknown entity 'bugs'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`) {
		t.Fatalf("code %d stderr %q", r.code, r.stderr)
	}
	e.wantCalls("GET /products")
}

func TestManualProxyThatCannotBeReached(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	r := e.run("", map[string]string{"HTTPS_PROXY": "http://127.0.0.1:1", "AHA_API_KEY": "x"}, "query", "query=SELECT id FROM features WHERE id = 'DEMO-1'")
	if r.code != 1 || r.stdout != "" || strings.TrimSpace(r.stderr) != `error: cannot reach Aha!: the proxy did not open a connection to example.aha.io.` {
		t.Fatalf("code %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
}
