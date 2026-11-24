// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// insert-statement, update-statement, delete-statement, check-write-values,
// write-request.

import (
	"encoding/json"
	"regexp"
	"testing"
)

func expectBody(t *testing.T, f *FakeAha, method, route, body string) {
	t.Helper()
	cs := f.CallsTo(method, "/api/v1"+route)
	if len(cs) != 1 {
		t.Errorf("%d %s calls on %s, want 1; calls: %s", len(cs), method, route, describeCalls(f.Calls()))
		return
	}
	if cs[0].Body != body {
		t.Errorf("body of %s %s\n got: %s\nwant: %s", method, route, cs[0].Body, body)
	}
	if cs[0].Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type %q", cs[0].Header.Get("Content-Type"))
	}
}

// EXAMPLE: insert_with_other_spellings
func TestInsertWithOtherSpellings(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.execute("INSERT INTO features (release_id, title, assignee) VALUES ('DEMO-R-2', 'Dark mode', 'me')"),
		`{"id":"DEMO-7","name":"Dark mode","parent":"DEMO-R-2","message":"Feature DEMO-7 created"}`)
	expectBody(t, f, "POST", "/releases/DEMO-R-2/features", `{"feature":{"name":"Dark mode","assigned_to_user":"test.user@example.com"}}`)
}

// EXAMPLE: insert_comment_with_apostrophe
func TestInsertCommentWithApostrophe(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("INSERT INTO comments (parent, body) VALUES ('DEMO-3', 'It''s ready.')")
	expectBody(t, f, "POST", "/features/DEMO-3/comments", `{"comment":{"body":"It's ready."}}`)
	if res.IsError {
		t.Fatalf("error %s", res.Text)
	}
	if got := memberNames(t, res.Text); len(got) != 5 || got[0] != "id" || got[1] != "parent" || got[2] != "body" || got[3] != "created_at" || got[4] != "message" {
		t.Errorf("members %v", got)
	}
	var m map[string]any
	_ = json.Unmarshal([]byte(res.Text), &m)
	if !regexp.MustCompile(`^[0-9]{6,}$`).MatchString(m["id"].(string)) || m["parent"] != "DEMO-3" || m["body"] != "It's ready." ||
		m["message"] != "Comment added to DEMO-3" || m["created_at"] == nil {
		t.Errorf("result %s", res.Text)
	}
}

// EXAMPLE: insert_without_parent
func TestInsertWithoutParent(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("INSERT INTO features (name) VALUES ('Dark mode')"),
		"parent is required: INSERT needs the column parent with the id of a release. Form: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)")
	expectError(t, s.execute("INSERT INTO features (parent) VALUES ('DEMO-R-3')"), "name is required to create a feature")
	expectError(t, s.execute("INSERT INTO comments (parent, body) VALUES ('DEMO', 'x')"),
		"parent = needs the id of the commented record, for example 'PROD-23'. Got 'DEMO'.")
	expectNoWrite(t, f)
}

// EXAMPLE: insert_with_a_product_as_release
func TestInsertWithAProductAsRelease(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("INSERT INTO features (parent, name) VALUES ('DEMO', 'Dark mode')"),
		"'DEMO' is a product, not a release. Use the id of one of its releases: DEMO-R-1 (What's next), DEMO-R-2 (Spring release), DEMO-R-3 (Summer release), DEMO-R-4 (Rider's cup release)")
	expectNoWrite(t, f)
}

// EXAMPLE: insert_under_an_unknown_release
func TestInsertUnderAnUnknownRelease(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("INSERT INTO features (parent, name) VALUES ('DEMO-R-99', 'Dark mode')"),
		"there is no release with the id 'DEMO-R-99'. To find the id, use the query tool: SELECT id, name FROM releases WHERE product = 'DEMO'")
	// the account has no new feature
	expectResult(t, s.query("SELECT COUNT(*) FROM features WHERE product = 'DEMO'"), `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`)
}

// EXAMPLE: insert_goal_without_a_success_metric
func TestInsertGoalWithoutASuccessMetric(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("INSERT INTO goals (parent, name) VALUES ('DEMO', 'Win the cup')"), "success_metric is required to create a goal")
	expectError(t, s.execute("INSERT INTO goals (parent, name, success_metric) VALUES ('DEMO', 'Win the cup', '')"), "success_metric is required to create a goal")
	expectError(t, s.execute("INSERT INTO goals (parent) VALUES ('DEMO')"), "name is required to create a goal")
	expectNoWrite(t, f)
}

// EXAMPLE: update_status_in_any_case
func TestUpdateStatusInAnyCase(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.execute("UPDATE features SET status = 'in design' WHERE id = 'demo-2'"),
		`{"id":"DEMO-2","name":"Offline maps","status":"In design","message":"Feature DEMO-2 status changed to 'In design'"}`)
	if countCalls(f, "GET", "/features/DEMO-2") != 1 {
		t.Errorf("ApiGet calls on /features/DEMO-2: %d", countCalls(f, "GET", "/features/DEMO-2"))
	}
	expectBody(t, f, "PUT", "/features/DEMO-2", `{"feature":{"workflow_status":"In design"}}`)
	for _, c := range f.Calls() {
		if c.Method == "GET" && (c.Path == "/api/v1/features" || regexp.MustCompile(`/features$`).MatchString(c.Path)) {
			t.Errorf("ApiList on a feature list: %s", c.Path)
		}
	}
}

// EXAMPLE: update_assigned_to_me
func TestUpdateAssignedToMe(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.execute("UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-2'"),
		`{"id":"DEMO-2","name":"Offline maps","assigned_to":"test.user@example.com","message":"Feature DEMO-2 updated"}`)
}

// EXAMPLE: update_with_a_true_guard
func TestUpdateWithATrueGuard(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.execute("UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-3' AND status = 'in development'"),
		`{"id":"DEMO-3","name":"Live dashboard","status":"Shipped","message":"Feature DEMO-3 status changed to 'Shipped'"}`)
}

// EXAMPLE: update_with_a_false_guard
func TestUpdateWithAFalseGuard(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-2' AND name = 'CSV export'"),
		"nothing was changed: feature DEMO-2 has name = 'Offline maps', not 'CSV export'")
	expectCalls(t, f, "GET /features/DEMO-2")
	expectNoWrite(t, f)
	expectError(t, s.execute("UPDATE features SET name = 'x' WHERE id = 'DEMO-2' AND assigned_to = 'me'"),
		"nothing was changed: feature DEMO-2 has no assigned_to, not 'me'")
	expectNoWrite(t, f)
}

// EXAMPLE: update_of_an_unknown_record
func TestUpdateOfAnUnknownRecord(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE id = 'DEMO-99'"), "there is no feature with the id 'DEMO-99'")
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE id = 'DEMO-R-3'"),
		"'DEMO-R-3' is the id of a release, not of a feature. To read it: SELECT * FROM releases WHERE id = 'DEMO-R-3'")
	expectError(t, s.execute("UPDATE features SET name = 'X' WHERE id = 'CSV export'"),
		"'CSV export' is not an id. An id looks like 'PROD-23'. To search by name: SELECT * FROM features WHERE name CONTAINS 'CSV export'")
	expectNoWrite(t, f)
}

// EXAMPLE: update_sends_empty_text
func TestUpdateSendsEmptyText(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("UPDATE features SET description = '' WHERE id = 'DEMO-1'")
	expectBody(t, f, "PUT", "/features/DEMO-1", `{"feature":{"description":""}}`)
	expectResult(t, res, `{"id":"DEMO-1","name":"CSV export","description":null,"message":"Feature DEMO-1 updated"}`)
}

// EXAMPLE: update_with_a_date_without_quotes
func TestUpdateWithADateWithoutQuotes(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.execute("UPDATE features SET due_date = 2026-12-01 WHERE id = 'DEMO-1'"),
		`{"id":"DEMO-1","name":"CSV export","due_date":"2026-12-01","message":"Feature DEMO-1 updated"}`)
}

// EXAMPLE: update_sets_a_custom_field
func TestUpdateSetsACustomField(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.execute("UPDATE features SET custom.priority = 'low' WHERE id = 'demo-2'"),
		`{"id":"DEMO-2","name":"Offline maps","custom.priority":"Low","message":"Feature DEMO-2 updated"}`)
	expectCalls(t, f, "GET /custom_field_definitions", "GET /features/DEMO-2", "GET /custom_field_definitions/7000000000000000801/options", "PUT /features/DEMO-2")
	expectBody(t, f, "PUT", "/features/DEMO-2", `{"feature":{"custom_fields":{"priority":"Low"}}}`)
}

// EXAMPLE: update_with_a_custom_guard
func TestUpdateWithACustomGuard(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-1' AND custom.priority = 'Low'"),
		"nothing was changed: feature DEMO-1 has custom.priority = 'High', not 'Low'")
	expectNoWrite(t, f)
	expectResult(t, s.execute("UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-1' AND custom.priority = 'high'"),
		`{"id":"DEMO-1","name":"CSV export","status":"Shipped","message":"Feature DEMO-1 status changed to 'Shipped'"}`)
}

// EXAMPLE: dry_run_shows_a_list_as_text
func TestDryRunShowsAListAsText(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	sb := newSandbox(t)
	r := run(t, sb.env(f, fixtureEnv...), "", "execute", "dry-run=true", "statement=UPDATE features SET custom.markets = 'EMEA, LATAM' WHERE id = 'DEMO-1'")
	if r.stdout != `{"id":"DEMO-1","custom.markets":"EMEA, LATAM","message":"would update feature DEMO-1"}`+"\n" || r.code != 0 {
		t.Errorf("stdout %q code %d stderr %q", r.stdout, r.code, r.stderr)
	}
	expectNoWrite(t, f)
	f.Reset()
	r = run(t, sb.env(f, fixtureEnv...), "", "execute", "dry-run=true", "statement=INSERT INTO features (parent, name) VALUES ('demo-r-3', 'A')")
	if r.stdout != `{"name":"A","message":"would create feature under DEMO-R-3"}`+"\n" || r.code != 0 {
		t.Errorf("stdout %q code %d stderr %q", r.stdout, r.code, r.stderr)
	}
	r = run(t, sb.env(f, fixtureEnv...), "", "execute", "dry-run=true", "statement=DELETE FROM features WHERE id = 'DEMO-6'")
	if r.stdout != `{"id":"DEMO-6","message":"would delete feature DEMO-6"}`+"\n" || r.code != 0 {
		t.Errorf("stdout %q code %d stderr %q", r.stdout, r.code, r.stderr)
	}
	expectNoWrite(t, f)
}

// EXAMPLE: delete_one_feature
func TestDeleteOneFeature(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.execute("DELETE FROM features WHERE id = 'DEMO-6'"), `{"id":"DEMO-6","message":"Feature DEMO-6 deleted"}`)
	expectCalls(t, f, "GET /features/DEMO-6", "DELETE /features/DEMO-6")
}

// EXAMPLE: delete_a_release_through_its_product
func TestDeleteAReleaseThroughItsProduct(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.execute("DELETE FROM releases WHERE id = 'DEMO-R-4'"), `{"id":"DEMO-R-4","message":"Release DEMO-R-4 deleted"}`)
	if countCalls(f, "DELETE", "/products/7000000000000000001/releases/DEMO-R-4") != 1 {
		t.Errorf("calls %s", describeCalls(f.Calls()))
	}
}

// EXAMPLE: delete_with_a_false_guard
func TestDeleteWithAFalseGuard(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("DELETE FROM features WHERE id = 'DEMO-6' AND status = 'Shipped'"),
		"nothing was changed: feature DEMO-6 has status = 'Under consideration', not 'Shipped'")
	expectNoWrite(t, f)
}

// EXAMPLE: delete_of_an_unknown_record
func TestDeleteOfAnUnknownRecord(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("DELETE FROM features WHERE id = 'DEMO-99'"), "there is no feature with the id 'DEMO-99'")
	expectNoWrite(t, f)
}

// EXAMPLE: delete_with_a_custom_guard
func TestDeleteWithACustomGuard(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("DELETE FROM features WHERE id = 'DEMO-2' AND custom.priority = 'High'"),
		"nothing was changed: feature DEMO-2 has custom.priority = 'Medium', not 'High'")
	expectNoWrite(t, f)
	expectResult(t, s.execute("DELETE FROM features WHERE id = 'DEMO-2' AND custom.priority = 'medium'"), `{"id":"DEMO-2","message":"Feature DEMO-2 deleted"}`)
}

// EXAMPLE: tags_are_split_at_commas
func TestTagsAreSplitAtCommas(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("UPDATE features SET tags = 'export, csv ,' WHERE id = 'DEMO-1'")
	expectBody(t, f, "PUT", "/features/DEMO-1", `{"feature":{"tags":"export,csv"}}`)
	expectResult(t, res, `{"id":"DEMO-1","name":"CSV export","tags":"export, csv","message":"Feature DEMO-1 updated"}`)
}

// EXAMPLE: unknown_status_lists_the_statuses
func TestUnknownStatusListsTheStatuses(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET status = 'Done' WHERE id = 'DEMO-2'"),
		"workflow status 'Done' not found for feature; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement")
	expectNoWrite(t, f)
}

// EXAMPLE: status_of_an_epic
func TestStatusOfAnEpic(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("UPDATE epics SET status = 'in development' WHERE id = 'DEMO-E-1'")
	expectBody(t, f, "PUT", "/epics/DEMO-E-1", `{"epic":{"workflow_status":"In development"}}`)
	expectResult(t, res, `{"id":"DEMO-E-1","name":"Race day","status":"In development","message":"Epic DEMO-E-1 status changed to 'In development'"}`)
	f.Reset()
	expectError(t, s.execute("UPDATE epics SET status = 'On track' WHERE id = 'DEMO-E-1'"),
		"workflow status 'On track' not found for epic; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement")
	expectNoWrite(t, f)
}

// EXAMPLE: date_that_is_not_a_date
func TestDateThatIsNotADate(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET due_date = 'next friday' WHERE id = 'DEMO-1'"), "due_date: invalid date 'next friday', use YYYY-MM-DD")
	expectError(t, s.execute("UPDATE features SET due_date = '2026-02-30' WHERE id = 'DEMO-1'"), "due_date: invalid date '2026-02-30', use YYYY-MM-DD")
	expectError(t, s.execute("UPDATE initiatives SET value = 'much' WHERE id = 'DEMO-S-1'"), "value must be a number")
	expectError(t, s.execute("UPDATE initiatives SET presented = 'maybe' WHERE id = 'DEMO-S-1'"), "presented must be true or false")
	expectError(t, s.execute("UPDATE features SET name = 5 WHERE id = 'DEMO-1'"), "name must be text in single quotes")
	expectNoWrite(t, f)
}

// EXAMPLE: user_by_name
func TestUserByName(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET assigned_to = 'Second User' WHERE id = 'DEMO-2'"),
		"assigned_to: there is no user 'Second User'. Use an e-mail address, or 'me' for yourself")
	expectNoWrite(t, f)
}

// EXAMPLE: custom_values_are_checked_by_kind
func TestCustomValuesAreCheckedByKind(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("UPDATE features SET custom.effort_days = '4.5', custom.go_live = 2026-09-01, custom.customer = 'Initech', custom.platforms = 'ios, web' WHERE id = 'DEMO-2'")
	expectBody(t, f, "PUT", "/features/DEMO-2", `{"feature":{"custom_fields":{"effort_days":4.5,"go_live":"2026-09-01","customer":"Initech","platforms":["iOS","Web"]}}}`)
	expectResult(t, res, `{"id":"DEMO-2","name":"Offline maps","custom.effort_days":4.5,"custom.go_live":"2026-09-01","custom.customer":"Initech","custom.platforms":"iOS, Web","message":"Feature DEMO-2 updated"}`)
}

// EXAMPLE: option_that_is_not_in_the_list
func TestOptionThatIsNotInTheList(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.execute("UPDATE features SET custom.priority = 'Urgent' WHERE id = 'DEMO-2'"), "custom.priority: there is no option 'Urgent'. Options: High, Medium, Low")
	expectError(t, s.execute("UPDATE features SET custom.platforms = 'web, Linux' WHERE id = 'DEMO-2'"), "custom.platforms: there is no option 'Linux'. Options: Web, iOS, Android")
	expectError(t, s.execute("UPDATE features SET custom.effort_days = 'many' WHERE id = 'DEMO-2'"), "custom.effort_days must be a number")
	expectError(t, s.execute("UPDATE features SET custom.reviewers = 'Second User' WHERE id = 'DEMO-2'"), "custom.reviewers: there is no user 'Second User'. Use an e-mail address, or 'me' for yourself")
	expectNoWrite(t, f)
}

// EXAMPLE: body_of_an_update_request
func TestBodyOfAnUpdateRequest(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("UPDATE features SET status = 'in design', assigned_to = 'me', due_date = '2026-07-01' WHERE id = 'DEMO-2'")
	if res.IsError {
		t.Errorf("error %s", res.Text)
	}
	expectBody(t, f, "PUT", "/features/DEMO-2", `{"feature":{"workflow_status":"In design","assigned_to_user":"test.user@example.com","due_date":"2026-07-01"}}`)
}

// write-request: the examples of the behaviour text.
func TestBodiesOfTheWriteRequestText(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	s.execute("UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = 'DEMO-1'")
	expectBody(t, f, "PUT", "/features/DEMO-1", `{"feature":{"workflow_status":"In design","assigned_to_user":"test.user@example.com"}}`)
	f.Reset()
	s.execute("UPDATE features SET name = 'CSV export 2', custom.priority = 'low' WHERE id = 'DEMO-1'")
	expectBody(t, f, "PUT", "/features/DEMO-1", `{"feature":{"name":"CSV export 2","custom_fields":{"priority":"Low"}}}`)
}

// EXAMPLE: body_of_a_create_request
func TestBodyOfACreateRequest(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("INSERT INTO releases (parent, name, release_date, description) VALUES ('demo', 'Winter release', '2027-01-31', 'Maintenance only')")
	expectBody(t, f, "POST", "/products/7000000000000000001/releases", `{"release":{"name":"Winter release","release_date":"2027-01-31","theme":"Maintenance only"}}`)
	expectResult(t, res, `{"id":"DEMO-R-5","name":"Winter release","parent":"DEMO","message":"Release DEMO-R-5 created"}`)
}

// EXAMPLE: body_with_custom_fields
func TestBodyWithCustomFields(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("UPDATE features SET due_date = '2026-05-01', custom.markets = 'EMEA, LATAM ,', custom.reviewers = 'me, second.user@example.com', start_date = '2026-04-01', custom.release_notes = 'Ready.' WHERE id = 'DEMO-1'")
	expectBody(t, f, "PUT", "/features/DEMO-1", `{"feature":{"due_date":"2026-05-01","custom_fields":{"markets":["EMEA","LATAM"],"reviewers":["7000000000000000900","second.user@example.com"],"release_notes":"Ready."},"start_date":"2026-04-01"}}`)
	expectResult(t, res, `{"id":"DEMO-1","name":"CSV export","due_date":"2026-05-01","custom.markets":"EMEA, LATAM","custom.reviewers":"test.user@example.com, second.user@example.com","start_date":"2026-04-01","custom.release_notes":"Ready.","message":"Feature DEMO-1 updated"}`)
}

// EXAMPLE: bodies_for_a_goal
func TestBodiesForAGoal(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.execute("INSERT INTO goals (parent, name, success_metric, status) VALUES ('DEMO', 'Win the cup', 'Podium places', 'at risk')")
	expectBody(t, f, "POST", "/products/7000000000000000001/goals", `{"goal":{"name":"Win the cup","success_metric":{"name":"Podium places","workflow_status":"At risk"}}}`)
	var m map[string]any
	if json.Unmarshal([]byte(res.Text), &m) != nil || m["message"] != "Goal DEMO-G-2 created" {
		t.Errorf("result %s", res.Text)
	}
	f.Reset()
	res = s.execute("UPDATE goals SET status = 'achieved' WHERE id = 'DEMO-G-1'")
	expectBody(t, f, "PUT", "/goals/DEMO-G-1", `{"goal":{"workflow_status":"Achieved"}}`)
	expectResult(t, res, `{"id":"DEMO-G-1","name":"Grow the audience","status":"Achieved","message":"Goal DEMO-G-1 status changed to 'Achieved'"}`)
}
