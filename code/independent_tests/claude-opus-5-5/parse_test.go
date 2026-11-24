// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// tokenize, parse-statement and error-messages.

import (
	"strings"
	"testing"
)

const writeForms = "INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...) | UPDATE entity SET field = 'value', ... WHERE id = 'id' | DELETE FROM entity WHERE id = 'id'"
const readForm = "SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]"
const featureExample = "SELECT id, name, status FROM features WHERE release = 'PROD-R-3'"

// EXAMPLE: doubled_apostrophe_in_text
func TestDoubledApostropheInText(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND name = 'Rider''s handbook'"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-5"}]}`)
	expectResult(t, s.query(`SELECT id FROM features WHERE product = "DEMO" AND name = 'Rider\'s handbook'`),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-5"}]}`)
}

// EXAMPLE: date_without_quotes
func TestDateWithoutQuotes(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'DEMO' AND release_date > 2026-06-01"),
		`{"entity":"releases","scope":"product DEMO","count":2,"records":[{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`)
}

// EXAMPLE: text_that_is_not_closed
func TestTextThatIsNotClosed(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT id FROM features WHERE name = 'Offline maps"),
		"the text that starts with the apostrophe at position 38 is not closed. Text goes in single quotes: 'like this'.")
	expectError(t, s.query(`SELECT id FROM features WHERE name = "Offline maps`),
		"the text that starts with the double quote at position 38 is not closed. Use single quotes for text: 'like this'.")
	expectNoCalls(t, f)
}

// EXAMPLE: unexpected_character
func TestUnexpectedCharacter(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT id FROM features WHERE name ~ 'x'"), "unexpected character ~ at position 36.")
	expectNoCalls(t, f)
}

// EXAMPLE: apostrophe_inside_a_word_is_read_as_text
func TestApostropheInsideAWordIsReadAsText(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'DEMO' AND name = 'Rider's cup release'"),
		`{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-4"}]}`)
}

// EXAMPLE: strict_reading_is_kept
func TestStrictReadingIsKept(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM features WHERE release='DEMO-R-3'AND status='Shipped'"),
		`{"entity":"features","scope":"release DEMO-R-3","count":1,"records":[{"id":"DEMO-5"}]}`)
}

// EXAMPLE: statements_without_select
func TestStatementsWithoutSelect(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("FROM releases WHERE product = 'DEMO'"),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","name":"What's next","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","name":"Spring release","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","name":"Summer release","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","name":"Rider's cup release","release_date":"2026-12-15","parking_lot":false}],"note":"* shows id, name, release_date, parking_lot. Other fields of releases: start_date, status, released, product, progress, description, created_at, url."}`)
	expectResult(t, s.query("COUNT(*) FROM features WHERE product = 'DEMO'"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`)
	expectResult(t, s.query("SELECT COUNT(1) FROM features WHERE product = 'DEMO';"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`)
}

// EXAMPLE: text_after_the_statement
func TestTextAfterTheStatement(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectErrorPrefix(t, s.query("SELECT id FROM features WHERE product = 'DEMO' LIMIT 5 please"),
		"cannot read the statement at position 56 (please): expected the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order.")
	expectNoCalls(t, f)
	expectError(t, s.query("SELECT id FROM features WHERE product = 'DEMO' LIMIT 5 please"),
		"cannot read the statement at position 56 (please): expected the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order. Form: "+readForm+". Example: "+featureExample)
}

// EXAMPLE: join_is_refused
func TestJoinIsRefused(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT f.name FROM features f JOIN releases r ON f.release = r.id"),
		"JOIN is not supported. Read one entity per statement and filter by its parent, for example: "+featureExample)
	expectError(t, s.query("SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status HAVING COUNT(*) > 1"),
		"HAVING is not supported. Use WHERE, GROUP BY, ORDER BY and LIMIT. Example: SELECT status, COUNT(*) FROM features WHERE product = 'PROD' GROUP BY status")
	expectError(t, s.query("SELECT id FROM features WHERE product = 'DEMO' LIMIT 5 OFFSET 10"),
		"OFFSET is not supported. Use WHERE to narrow the result and LIMIT to shorten it.")
	expectError(t, s.query("SELECT id FROM features WHERE product = 'DEMO' LIMIT 5, 10"),
		"OFFSET is not supported. Use WHERE to narrow the result and LIMIT to shorten it.")
	expectError(t, s.query("SELECT id FROM features WHERE release IN (SELECT id FROM releases)"),
		"subqueries are not supported. Send one statement per call. Form: "+readForm)
	expectError(t, s.query("SELECT LOWER(name) FROM features WHERE product = 'DEMO'"),
		"unknown function LOWER. Functions: COUNT, SUM, AVG, MIN, MAX. Example: SELECT COUNT(*) FROM features WHERE product = 'PROD'")
	expectError(t, s.query("SELECT id FROM features WHERE status = Shipped"),
		"text goes in single quotes: write 'Shipped' instead of Shipped")
	expectError(t, s.query("hello world"),
		"a statement starts with SELECT or DESCRIBE. Form: "+readForm+". Example: "+featureExample)
	expectNoCalls(t, f)
}

// EXAMPLE: id_or_entity_alone
func TestIDOrEntityAlone(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("DEMO-3"), "DEMO-3 is an id, not a statement. To read that record: SELECT * FROM features WHERE id = 'DEMO-3'")
	expectError(t, s.query("features"), "features is an entity, not a statement. To list its records: "+featureExample+". To see its fields: DESCRIBE features")
}

// EXAMPLE: describe_takes_nothing_else
func TestDescribeTakesNothingElse(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("DESCRIBE features WHERE product = 'DEMO'"),
		"DESCRIBE features lists the fields of features and takes nothing else. To read records: "+featureExample)
}

// EXAMPLE: apostrophe_hint_in_an_error
func TestApostropheHintInAnError(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	res := s.query("SELECT id FROM features WHERE name = 'it's")
	if !res.IsError || !strings.Contains(res.Text, "The text contains an apostrophe (it's). Inside text, write an apostrophe twice: it''s") {
		t.Errorf("message %q", res.Text)
	}
}

// S03 messages: what was expected.
func TestReadSyntaxMessages(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	tail := " Form: " + readForm + ". Example: " + featureExample
	for _, c := range []struct{ q, msg string }{
		{"SELECT id", "cannot read the statement at the end of the statement: expected FROM and an entity."},
		{"SELECT id FROM", "cannot read the statement at the end of the statement: expected an entity (products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me)."},
		{"SELECT FROM features", "cannot read the statement at position 8 (FROM features): expected a field, * or COUNT(*) after SELECT."},
		{"SELECT id FROM features LIMIT x", "cannot read the statement at position 31 (x): expected a number after LIMIT."},
		{"SELECT id FROM features GROUP status", "cannot read the statement at position 31 (status): expected BY after GROUP."},
		{"SELECT id FROM features WHERE product IS 'DEMO'", "cannot read the statement at position 42 ('DEMO'): expected NULL after IS."},
		{"SELECT id FROM features WHERE product = 'DEMO' ORDER id", "cannot read the statement at position 54 (id): expected BY after ORDER."},
	} {
		expectError(t, s.query(c.q), c.msg+tail)
	}
	expectNoCalls(t, f)
}

// EXAMPLE: write_without_an_id_is_refused
func TestWriteWithoutAnIDIsRefused(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	msg := "one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record."
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE status = 'Shipped'"), msg)
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE id = 'DEMO-1' OR id = 'DEMO-2'"), msg)
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE id = 'DEMO-1' AND status != 'Shipped'"),
		"after WHERE id = '...', further conditions must have the form AND field = 'value'")
	expectNoCalls(t, f)
}

// EXAMPLE: write_without_where_is_refused
func TestWriteWithoutWhereIsRefused(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET name = 'X'"),
		"UPDATE needs WHERE id = '...' with the id of the one record to change. Forms: "+writeForms)
	expectErrorPrefix(t, s.execute("DELETE FROM features"), "DELETE needs WHERE id = '...' with the id of the one record to change.")
	expectNoCalls(t, f)
}

// W02, W03 and W08 to W10 of the write forms.
func TestWriteSyntaxMessages(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("hello"), "a statement starts with INSERT, UPDATE or DELETE. Forms: "+writeForms)
	expectError(t, s.execute("INSERT features (name) VALUES ('x')"), "expected INTO after INSERT. Forms: "+writeForms)
	expectError(t, s.execute("DELETE features WHERE id = 'DEMO-1'"), "expected FROM after DELETE. Forms: "+writeForms)
	expectError(t, s.execute("UPDATE features name = 'x' WHERE id = 'DEMO-1'"), "expected SET after the entity. Forms: "+writeForms)
	expectError(t, s.execute("UPDATE features SET name 'x' WHERE id = 'DEMO-1'"), "expected = after the field name. Forms: "+writeForms)
	expectError(t, s.execute("DELETE FROM features WHERE id = 'DEMO-1' 'x'"), "unexpected text at position 42 ('x'). One statement, one record. Forms: "+writeForms)
	expectError(t, s.execute("INSERT INTO features (parent, name) VALUES ('DEMO-R-3')"), "2 fields but 1 values")
	expectError(t, s.execute("INSERT INTO features (parent, release, name) VALUES ('DEMO-R-3', 'DEMO-R-2', 'A')"), "the parent is given twice with different values")
	expectError(t, s.execute("UPDATE features SET name = 'X' 'Y' WHERE id = 'DEMO-1'"), "a field can only be set to a value, not to an expression")
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE id = TRUE"), "WHERE id = needs the id in single quotes. Forms: "+writeForms)
	expectNoCalls(t, f)
}

// EXAMPLE: insert_with_two_rows_is_refused
func TestInsertWithTwoRowsIsRefused(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'A'), ('DEMO-R-3', 'B')"),
		"one INSERT creates one record. Send one statement per record.")
	expectNoWrite(t, f)
}

const featureWritable = "Fields of features that can be set: name, status, release, initiative, assigned_to, start_date, due_date, tags, description, and parent in INSERT. Custom fields, with UPDATE only: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers"

// EXAMPLE: insert_with_an_unknown_column_is_refused
func TestInsertWithAnUnknownColumnIsRefused(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("INSERT INTO features (parent, name, bogus) VALUES ('DEMO-R-3', 'A', 'x')"),
		"unknown field 'bogus' on features. "+featureWritable)
	expectError(t, s.execute("INSERT INTO features (parent, name, score) VALUES ('DEMO-R-3', 'A', 5)"),
		"the field score of features cannot be set. "+featureWritable)
	expectError(t, s.execute("INSERT INTO features (parent, id, name) VALUES ('DEMO-R-3', 'DEMO-9', 'A')"),
		"the id of a record cannot be set. Fields of features that can be set: name, status, release, initiative, assigned_to, start_date, due_date, tags, description")
	expectNoWrite(t, f)
}

// EXAMPLE: update_of_a_field_that_is_set_in_insert_only
func TestUpdateOfAFieldThatIsSetInInsertOnly(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE goals SET success_metric = 'Weekly viewers' WHERE id = 'DEMO-G-1'"),
		"the field success_metric of goals can be set in INSERT only. Fields of goals that can be set: name, success_metric (INSERT only), status, progress, description, and parent in INSERT")
	if countCalls(f, "GET", "/goals/DEMO-G-1") != 0 {
		t.Errorf("ApiGet on /goals/DEMO-G-1")
	}
	expectNoWrite(t, f)
}

// EXAMPLE: assignment_of_a_word_or_an_expression
func TestAssignmentOfAWordOrAnExpression(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET name = name WHERE id = 'DEMO-1'"), "text goes in single quotes: write 'name' instead of name")
	expectError(t, s.execute("UPDATE features SET name = name + ' (old)' WHERE id = 'DEMO-1'"), "unexpected character + at position 33.")
	expectNoCalls(t, f)
}

// EXAMPLE: null_cannot_be_written
func TestNullCannotBeWritten(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET due_date = NULL WHERE id = 'DEMO-1'"), "NULL cannot be written. Leave the field out.")
	expectNoCalls(t, f)
}

// EXAMPLE: write_to_a_read_only_entity
func TestWriteToAReadOnlyEntity(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE users SET name = 'X' WHERE id = '7000000000000000901'"),
		"users cannot be changed. Entities that can be changed: releases, features, epics, requirements, initiatives, goals, ideas, comments")
	expectError(t, s.execute("DELETE FROM tickets WHERE id = '1'"),
		"unknown entity 'tickets'. Entities that can be changed: releases, features, epics, requirements, initiatives, goals, ideas, comments")
	expectNoCalls(t, f)
}

// EXAMPLE: products_cannot_be_changed
func TestProductsCannotBeChanged(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("INSERT INTO products (name, key) VALUES ('New product', 'NEW')"),
		"products cannot be changed. Entities that can be changed: releases, features, epics, requirements, initiatives, goals, ideas, comments")
	expectNoCalls(t, f)
}

// EXAMPLE: label_of_a_custom_field
func TestLabelOfACustomField(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT f.custom.priority, f.custom.customer AS customer FROM features f WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"custom.priority":"High","customer":"Acme"}]}`)
}

// EXAMPLE: insert_with_a_custom_field_is_refused
func TestInsertWithACustomFieldIsRefused(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	msg := "custom fields cannot be set in INSERT. Leave out custom.priority, create the record, and then send: UPDATE features SET custom.priority = '...' WHERE id = '...'"
	expectError(t, s.execute("INSERT INTO features (parent, name, custom.priority) VALUES ('DEMO-R-3', 'A', 'High')"), msg)
	expectError(t, s.execute("INSERT INTO features (parent, name, priority) VALUES ('DEMO-R-3', 'A', 'High')"), msg)
	expectNoWrite(t, f)
}

// EXAMPLE: custom_field_that_cannot_be_set
func TestCustomFieldThatCannotBeSet(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET custom.value_score = 40 WHERE id = 'DEMO-1'"),
		"the field custom.value_score of features cannot be set. "+featureWritable)
	if countCalls(f, "GET", "/features/DEMO-1") != 0 {
		t.Errorf("ApiGet on /features/DEMO-1")
	}
	expectNoWrite(t, f)
}

// W14 and W17 of the write rules.
func TestWriteNameMessages(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE id = 'DEMO-1' AND bogus = 'x'"),
		"unknown field 'bogus' on features. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score")
	expectNoWrite(t, f)
}

// EXAMPLE: statements_in_messages_can_be_sent
func TestStatementsInMessagesCanBeSent(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	// message openings of parse-statement and resolve-names: none of the
	// statements may get one of them
	parseErrors := []string{"the text that starts", "unexpected character", "cannot read the statement", "a statement starts with",
		"is an id, not a statement", "is an entity, not a statement", "query only reads", "JOIN is not", "HAVING is not",
		"OFFSET is not", "subqueries are not", "unknown function", "text goes in single quotes", "lists the fields of",
		"execute only changes", "expected ", "needs WHERE id", "one statement changes one record", "after WHERE id", "one INSERT creates",
		"fields but", "the parent is given twice", "a field can only be set", "NULL cannot be written", "unknown entity", "cannot be changed",
		"cannot be created", "cannot be deleted", "the id of a record cannot be set", "unknown field", "the field ", "custom fields cannot be set",
		"unknown custom field", "has the type", "have no custom fields", "the custom fields of the account", "WHERE id = needs"}
	check := func(stmt string, res toolResult) {
		t.Helper()
		if !res.IsError {
			return
		}
		for _, p := range parseErrors {
			if strings.HasPrefix(res.Text, p) || strings.Contains(res.Text, "Forms: ") {
				t.Errorf("parse-statement does not accept %q: %s", stmt, res.Text)
				return
			}
		}
	}
	reads := []string{
		"SELECT key, name FROM products",
		"SELECT id, name, release_date FROM releases WHERE product = 'PROD'",
		"SELECT id, name, status FROM features WHERE release = 'PROD-R-3'",
		"SELECT id, name, status FROM epics WHERE product = 'PROD'",
		"SELECT id, name, status FROM requirements WHERE feature = 'PROD-23'",
		"SELECT id, name, status FROM initiatives WHERE product = 'PROD'",
		"SELECT id, name FROM goals WHERE product = 'PROD'",
		"SELECT id, name, votes FROM ideas WHERE product = 'PROD' ORDER BY votes DESC LIMIT 5",
		"SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'",
		"SELECT name, email FROM users",
		"SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'",
		"SELECT name, email FROM me",
		"SELECT status, COUNT(*) FROM features WHERE product = 'PROD' GROUP BY status",
		"SELECT id, name, release_date FROM releases WHERE product = 'PROD' AND parking_lot = false ORDER BY release_date",
		"SELECT email FROM me",
		"SELECT COUNT(*) FROM features WHERE product = 'PROD'",
		"SELECT id, name FROM releases WHERE product = 'PROD'",
		"SELECT * FROM features WHERE name CONTAINS 'x'",
		"SELECT id, body FROM comments WHERE parent = 'PROD-23'",
		"DESCRIBE features",
		"SELECT id, custom.customer FROM features WHERE id = 'PROD-23'",
	}
	for _, q := range reads {
		check(q, s.query(q))
	}
	sb := newSandbox(t)
	writes := []string{
		"INSERT INTO features (parent, name) VALUES ('PROD-R-3', 'Export to CSV')",
		"UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = 'PROD-23'",
		"INSERT INTO comments (parent, body) VALUES ('PROD-23', 'It''s ready.')",
		"UPDATE features SET custom.priority = '...' WHERE id = '...'",
	}
	for _, w := range writes {
		r := run(t, sb.env(f, fixtureEnv...), "", "execute", "dry-run=true", "statement="+w)
		msg := strings.TrimPrefix(strings.TrimSpace(r.stderr), "error: ")
		check(w, toolResult{Text: msg, IsError: r.code != 0})
	}
	expectNoWrite(t, f)
}
