// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// manual-mode, choose-example-key, and the MILESTONE 0.5.0 acceptance lines.

import (
	"strings"
	"testing"
)

// EXAMPLE: manual_query_prints_the_tool_result
func TestManualQueryPrintsTheToolResult(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	stmt := "SELECT id, name FROM features WHERE id = 'demo-1'"
	r := run(t, sb.env(f, fixtureEnv...), "", "query", "query="+stmt)
	want := `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","name":"CSV export"}]}`
	if r.stdout != want+"\n" {
		t.Errorf("stdout\n got: %q\nwant: %q", r.stdout, want+"\n")
	}
	if r.stderr != "" {
		t.Errorf("stderr is not empty: %q", r.stderr)
	}
	if r.code != 0 {
		t.Errorf("exit code %d", r.code)
	}
	expectCalls(t, f, "GET /features/DEMO-1")
	// the tool query returns the same text in stdio mode
	s := fixtureServer(t, f)
	expectResult(t, s.query(stmt), want)
}

// EXAMPLE: manual_statement_from_standard_input
func TestManualStatementFromStandardInput(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "SELECT email FROM me\n", "query", "query=-")
	want := `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`
	if r.stdout != want+"\n" || r.code != 0 {
		t.Errorf("stdout %q code %d stderr %q", r.stdout, r.code, r.stderr)
	}
}

// EXAMPLE: manual_dry_run_sends_no_write_request
func TestManualDryRunSendsNoWriteRequest(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "", "execute", "dry-run=true", "statement=UPDATE features SET status = 'shipped' WHERE id = 'DEMO-1'")
	want := `{"id":"DEMO-1","status":"Shipped","message":"would update feature DEMO-1"}`
	if r.stdout != want+"\n" {
		t.Errorf("stdout\n got: %q\nwant: %q", r.stdout, want+"\n")
	}
	if r.stderr != "" {
		t.Errorf("stderr is not empty: %q", r.stderr)
	}
	if r.code != 0 {
		t.Errorf("exit code %d", r.code)
	}
	if n := countCalls(f, "GET", "/features/DEMO-1"); n != 1 {
		t.Errorf("%d ApiGet calls on /features/DEMO-1, want 1", n)
	}
	expectNoWrite(t, f)

	f.Reset()
	r = run(t, sb.env(f, fixtureEnv...), "", "execute", "dry-run=true", "statement=UPDATE features SET status = 'done' WHERE id = 'DEMO-1'")
	if r.stdout != "" {
		t.Errorf("stdout is not empty: %q", r.stdout)
	}
	expectLine(t, "stderr", r.stderr, "error: workflow status 'done' not found for feature; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement")
	if r.code != 1 {
		t.Errorf("exit code %d, want 1", r.code)
	}
	expectNoWrite(t, f)
}

// EXAMPLE: manual_error_goes_to_standard_error
func TestManualErrorGoesToStandardError(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "", "query", "query=SELECT id, name FROM releases")
	if r.stdout != "" {
		t.Errorf("stdout is not empty: %q", r.stdout)
	}
	expectLine(t, "stderr", r.stderr, "error: releases are listed per product: add product = '...' to WHERE. Example: SELECT id, name, release_date FROM releases WHERE product = 'PROD'")
	if r.code != 1 {
		t.Errorf("exit code %d, want 1", r.code)
	}
}

// EXAMPLE: manual_statement_argument_is_missing
func TestManualStatementArgumentIsMissing(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "", "query")
	expectLine(t, "stderr", r.stderr, "error: query needs query=STATEMENT, or query=- to read the statement from standard input")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	r = run(t, sb.env(f, fixtureEnv...), "", "execute")
	expectLine(t, "stderr", r.stderr, "error: execute needs statement=STATEMENT, or statement=- to read the statement from standard input")
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	expectNoCalls(t, f)
}

// EXAMPLE: manual_execute_is_refused_in_read_only
func TestManualExecuteIsRefusedInReadOnly(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	line := "error: execute is not available, because read-only is true (option read-only or variable AHA_READ_ONLY)"
	r := run(t, sb.env(f, fixtureEnv...), "", "execute", "read-only=true", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	expectLine(t, "stderr", r.stderr, line)
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	r = run(t, sb.env(f, append(fixtureEnv, "AHA_READ_ONLY=true")...), "", "execute", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	expectLine(t, "stderr", r.stderr, line)
	if r.code != 2 {
		t.Errorf("exit code %d, want 2", r.code)
	}
	expectNoCalls(t, f)
}

// MILESTONE 0.5.0 acceptance lines for manual mode.
func TestManualModeAcceptance(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	env := []string{"HOME=" + sb.home, "AHA_SUBDOMAIN=example", "AHA_API_KEY=x"}
	r := run(t, env, "", "query", "query=DELETE FROM features WHERE id = 'DEMO-1'")
	expectLine(t, "stderr", r.stderr, "error: query only reads. Use the execute tool for INSERT, UPDATE and DELETE.")
	if r.code != 1 {
		t.Errorf("exit code %d, want 1", r.code)
	}
	r = run(t, env, "DELETE FROM features WHERE id = 'DEMO-1'\n", "query", "query=-")
	expectLine(t, "stderr", r.stderr, "error: query only reads. Use the execute tool for INSERT, UPDATE and DELETE.")
	r = run(t, env, "", "execute", "statement=SELECT id FROM features")
	expectLine(t, "stderr", r.stderr, "error: execute only changes data. Use the query tool for SELECT and DESCRIBE.")
	r = run(t, append(env, "HTTPS_PROXY=http://127.0.0.1:1"), "", "query", "query=SELECT id FROM features WHERE id = 'DEMO-1'")
	expectLine(t, "stderr", r.stderr, "error: cannot reach Aha!: the proxy did not open a connection to example.aha.io.")
	if r.stdout != "" {
		t.Errorf("stdout not empty: %q", r.stdout)
	}
}

// EXAMPLE: example_key_is_chosen_when_it_is_needed
func TestExampleKeyIsChosenWhenItIsNeeded(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "", "query", "query=SELECT id FROM features WHERE id = 'DEMO-3'")
	want := `{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`
	if r.stdout != want+"\n" {
		t.Errorf("stdout %q", r.stdout)
	}
	if n := countCalls(f, "GET", "/products"); n != 0 {
		t.Errorf("%d ApiList calls on /products, want 0", n)
	}
	f.Reset()
	r = run(t, sb.env(f, fixtureEnv...), "", "query", "query=SELECT id FROM bugs")
	expectLine(t, "stderr", r.stderr, "error: unknown entity 'bugs'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'")
	if r.code != 1 {
		t.Errorf("exit code %d, want 1", r.code)
	}
	expectCalls(t, f, "GET /products")
}

// EXAMPLE: example_key_is_the_default_product
func TestExampleKeyIsTheDefaultProduct(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, append(fixtureEnv, "AHA_DEFAULT_PRODUCT=demo")...))
	s.initialize()
	tools := s.listTools()
	for _, tl := range tools {
		if tl.Name == "query" {
			expectLine(t, "description of query", tl.Description, "id is the reference number of a record, for example 'DEMO-23'.")
		}
		if strings.Contains(tl.Description, "PROD") || strings.Contains(tl.Description, "OTHER") {
			t.Errorf("description of %s has PROD or OTHER:\n%s", tl.Name, tl.Description)
		}
	}
}

// EXAMPLE: example_key_without_a_default_product
func TestExampleKeyWithoutADefaultProduct(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, fixtureEnv...))
	s.initialize()
	for _, tl := range s.listTools() {
		if tl.Name == "query" {
			expectLine(t, "description of query", tl.Description, "Filter by parent: product = 'PROD', release = 'PROD-R-3', feature = 'PROD-23', initiative = 'PROD-S-1'.")
		}
		if strings.Contains(tl.Description, "DEMO") || strings.Contains(tl.Description, "OTHER") {
			t.Errorf("description of %s has DEMO or OTHER", tl.Name)
		}
	}
}

// EXAMPLE: example_key_is_not_a_product_of_the_account
func TestExampleKeyIsNotAProductOfTheAccount(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.AddProduct("PROD", "7000000000000000003", 0)
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, fixtureEnv...))
	s.initialize()
	found := false
	for _, tl := range s.listTools() {
		if tl.Name == "execute" {
			found = true
			expectLine(t, "description of execute", tl.Description, "INSERT INTO features (parent, name) VALUES ('PRODX-R-3', 'Export to CSV')")
		}
		if strings.Contains(tl.Description, "'PROD-") {
			t.Errorf("description of %s has 'PROD-", tl.Name)
		}
	}
	if !found {
		t.Errorf("no tool execute")
	}
}

// EXAMPLE: example_key_when_the_product_list_cannot_be_read (expected-long:
// FakeAha does not answer, and the server waits 60 seconds)
func TestExampleKeyWhenTheProductListCannotBeRead(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.NoAnswer("GET", "/products")
	sb := newSandbox(t)
	s := startStdio(t, sb.env(f, append(fixtureEnv, "AHA_DEFAULT_PRODUCT=demo")...))
	s.initialize()
	tools := s.listTools()
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.Name] = true
		if tl.Name == "query" {
			expectLine(t, "description of query", tl.Description, "id is the reference number of a record, for example 'PROD-23'.")
		}
	}
	if !names["query"] || !names["execute"] || len(tools) != 2 {
		t.Errorf("tool list %v", names)
	}
	errText := s.stderr.String()
	if !strings.Contains(strings.ToLower(errText), "warning") {
		t.Errorf("no warning on the diagnostic output:\n%s", errText)
	}
	if strings.Contains(errText, "fixture-key") {
		t.Errorf("the warning has the credential")
	}
}
