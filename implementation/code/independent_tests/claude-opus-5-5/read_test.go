// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// read-statement, describe-statement, entity-catalog, custom-fields,
// resolve-names, resolve-product, plan-fetch, convert-record,
// evaluate-condition, build-rows.

import (
	"encoding/json"
	"strings"
	"testing"
)

var featureFields = []string{"id", "name", "status", "release", "initiative", "epic", "assigned_to", "product", "start_date",
	"due_date", "tags", "score", "progress", "description", "created_at", "updated_at", "url"}

func expectMembers(t *testing.T, raw json.RawMessage, want []string) {
	t.Helper()
	got := memberNames(t, string(raw))
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("row labels\n got: %v\nwant: %v", got, want)
	}
}

func member(t *testing.T, raw json.RawMessage, name string) any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("not an object: %s", raw)
	}
	return m[name]
}

// EXAMPLE: count_per_status
func TestCountPerStatus(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status"),
		`{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"In design","COUNT(*)":1},{"status":"Under consideration","COUNT(*)":3},{"status":"In development","COUNT(*)":1},{"status":"Shipped","COUNT(*)":1}]}`)
}

// EXAMPLE: star_shows_default_fields
func TestStarShowsDefaultFields(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT * FROM features WHERE product = 'DEMO' AND status = 'under consideration'"),
		`{"entity":"features","scope":"product DEMO","count":3,"records":[{"id":"DEMO-2","name":"Offline maps","status":"Under consideration","release":"DEMO-R-2"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration","release":"DEMO-R-3"},{"id":"DEMO-6","name":"Route planner","status":"Under consideration","release":"DEMO-R-1"}],"note":"* shows id, name, status, release. Other fields of features: initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url."}`)
}

// EXAMPLE: record_by_id_has_all_fields
func TestRecordByIDHasAllFields(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.query("SELECT * FROM features WHERE id = 'demo-3'")
	if res.IsError {
		t.Fatalf("error: %s", res.Text)
	}
	var top map[string]any
	_ = json.Unmarshal([]byte(res.Text), &top)
	if top["scope"] != "feature DEMO-3" || top["count"] != float64(1) {
		t.Errorf("scope %v count %v", top["scope"], top["count"])
	}
	if _, ok := top["note"]; ok {
		t.Errorf("the result has a note")
	}
	recs := records(t, res.Text)
	if len(recs) != 1 {
		t.Fatalf("records %d", len(recs))
	}
	want := append(append([]string{}, featureFields...), "custom.customer", "custom.effort_days", "custom.go_live", "custom.markets", "custom.priority", "custom.reviewers")
	expectMembers(t, recs[0], want)
	for k, v := range map[string]any{"id": "DEMO-3", "name": "Live dashboard", "status": "In development", "release": "DEMO-R-3",
		"initiative": "DEMO-S-1", "product": "DEMO", "epic": nil, "assigned_to": nil} {
		if got := member(t, recs[0], k); got != v {
			t.Errorf("%s = %v, want %v", k, got, v)
		}
	}
	if countCalls(f, "GET", "/features/DEMO-3") != 1 || countCalls(f, "GET", "/products") != 1 || countCalls(f, "GET", "/custom_field_definitions") != 1 || len(f.Calls()) != 3 {
		t.Errorf("calls: %s", describeCalls(f.Calls()))
	}
}

// EXAMPLE: field_without_value_is_null
func TestFieldWithoutValueIsNull(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id, assigned_to FROM features WHERE release = 'DEMO-R-2'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","assigned_to":"test.user@example.com"},{"id":"DEMO-2","assigned_to":null}]}`)
}

// EXAMPLE: label_is_the_word_of_the_statement
func TestLabelIsTheWordOfTheStatement(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT ref_num, title FROM features WHERE release_id = 'DEMO-R-2'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"ref_num":"DEMO-1","title":"CSV export"},{"ref_num":"DEMO-2","title":"Offline maps"}]}`)
}

// EXAMPLE: name_equality_and_search
func TestNameEqualityAndSearch(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND name = 'live'"),
		`{"entity":"features","scope":"product DEMO","count":0,"records":[],"note":"No feature has exactly the name 'live'. name = compares the whole name; to search inside names use: name CONTAINS 'live'"}`)
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND name CONTAINS 'live'"),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`)
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND name = 'offline MAPS'"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-2"}]}`)
}

// EXAMPLE: unknown_id_gives_an_empty_result
func TestUnknownIDGivesAnEmptyResult(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT * FROM features WHERE id = 'DEMO-99'"),
		`{"entity":"features","scope":"feature DEMO-99","count":0,"records":[],"note":"There is no feature with the id 'DEMO-99'."}`)
	if countCalls(f, "GET", "/custom_field_definitions") != 0 {
		t.Errorf("a call on /custom_field_definitions")
	}
}

// EXAMPLE: default_limit_is_reported
func TestDefaultLimitIsReported(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.AddProduct("BULK", "7000000000000000010", 60)
	s := fixtureServer(t, f)
	res := s.query("SELECT id, name FROM features WHERE product = 'BULK'")
	var top struct {
		Count     int               `json:"count"`
		Total     int               `json:"total"`
		Truncated bool              `json:"truncated"`
		Note      string            `json:"note"`
		Records   []json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal([]byte(res.Text), &top); err != nil || res.IsError {
		t.Fatalf("result: %s", res.Text)
	}
	if top.Count != 50 || top.Total != 60 || !top.Truncated || len(top.Records) != 50 {
		t.Errorf("count %d total %d truncated %v records %d", top.Count, top.Total, top.Truncated, len(top.Records))
	}
	if len(top.Records) == 50 && (member(t, top.Records[0], "id") != "BULK-1" || member(t, top.Records[49], "id") != "BULK-50") {
		t.Errorf("records are not BULK-1 to BULK-50")
	}
	if top.Note != "Showing 50 of 60 records. Narrow the result with WHERE, raise LIMIT (up to 200), or count with COUNT(*)." {
		t.Errorf("note %q", top.Note)
	}
	if want := []string{"entity", "scope", "count", "total", "truncated", "records", "note"}; strings.Join(memberNames(t, res.Text), ",") != strings.Join(want, ",") {
		t.Errorf("member order %v", memberNames(t, res.Text))
	}
}

// EXAMPLE: releases_need_a_product
func TestReleasesNeedAProduct(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT id, name FROM releases"),
		"releases are listed per product: add product = '...' to WHERE. Example: SELECT id, name, release_date FROM releases WHERE product = 'PROD'")
}

// EXAMPLE: star_with_an_id_shows_custom_values
func TestStarWithAnIDShowsCustomValues(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.query("SELECT * FROM features WHERE id = 'DEMO-1'")
	recs := records(t, res.Text)
	if len(recs) != 1 {
		t.Fatalf("result %s", res.Text)
	}
	expectMembers(t, recs[0], append(append([]string{}, featureFields...), "custom.customer", "custom.effort_days", "custom.go_live",
		"custom.markets", "custom.priority", "custom.release_notes", "custom.reviewers", "custom.value_score"))
	for k, v := range map[string]any{"custom.customer": "Acme", "custom.effort_days": 12.5, "custom.go_live": "2026-04-15",
		"custom.markets": "EMEA, APAC", "custom.priority": "High", "custom.release_notes": "Adds the export.\nSee the handbook.",
		"custom.reviewers": "second.user@example.com", "custom.value_score": float64(31)} {
		if got := member(t, recs[0], k); got != v {
			t.Errorf("%s = %#v, want %#v", k, got, v)
		}
	}
	if !strings.Contains(string(recs[0]), `"custom.release_notes":"Adds the export.\nSee the handbook."`) {
		t.Errorf("release_notes as JSON string: %s", recs[0])
	}
	if countCalls(f, "GET", "/custom_field_definitions") != 1 {
		t.Errorf("calls on /custom_field_definitions: %d", countCalls(f, "GET", "/custom_field_definitions"))
	}
	recs = records(t, s.query("SELECT * FROM features WHERE id = 'DEMO-2'").Text)
	if len(recs) == 1 {
		expectMembers(t, recs[0], append(append([]string{}, featureFields...), "custom.effort_days", "custom.priority"))
		if member(t, recs[0], "custom.effort_days") != float64(8) || !strings.Contains(string(recs[0]), `"custom.effort_days":8,`) {
			t.Errorf("effort_days: %s", recs[0])
		}
	}
	recs = records(t, s.query("SELECT *, custom.platforms FROM features WHERE id = 'DEMO-2'").Text)
	if len(recs) == 1 {
		expectMembers(t, recs[0], append(append([]string{}, featureFields...), "custom.platforms", "custom.effort_days", "custom.priority"))
		if member(t, recs[0], "custom.platforms") != nil {
			t.Errorf("custom.platforms must be null")
		}
	}
	expectResult(t, s.query("SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`)
}

// EXAMPLE: star_without_an_id_has_catalog_fields_only
func TestStarWithoutAnIDHasCatalogFieldsOnly(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.query("SELECT * FROM features WHERE release = 'DEMO-R-2'")
	recs := records(t, res.Text)
	if len(recs) != 2 {
		t.Fatalf("result %s", res.Text)
	}
	for _, r := range recs {
		expectMembers(t, r, []string{"id", "name", "status", "release"})
	}
	var top map[string]any
	_ = json.Unmarshal([]byte(res.Text), &top)
	if n, _ := top["note"].(string); !strings.HasPrefix(n, "* shows id, name, status, release.") {
		t.Errorf("note %q", top["note"])
	}
	if countCalls(f, "GET", "/custom_field_definitions") != 0 {
		t.Errorf("call on /custom_field_definitions")
	}
	for _, c := range f.CallsTo("GET", "/api/v1/releases/DEMO-R-2/features") {
		if strings.Contains(c.Query.Get("fields"), "custom_fields") {
			t.Errorf("fields names custom_fields: %s", c.Query.Get("fields"))
		}
	}
	f.Reset()
	res = s.query("SELECT * FROM products")
	recs = records(t, res.Text)
	if len(recs) != 2 {
		t.Fatalf("result %s", res.Text)
	}
	for _, r := range recs {
		expectMembers(t, r, []string{"key", "name"})
	}
	if countCalls(f, "GET", "/custom_field_definitions") != 0 || countCalls(f, "GET", "/products/"+demoID) != 0 || countCalls(f, "GET", "/products/"+otherID) != 0 {
		t.Errorf("calls: %s", describeCalls(f.Calls()))
	}
}

var productFields = []string{"key", "name", "product_line", "parent", "description", "created_at", "url"}

// EXAMPLE: star_shows_the_custom_values_of_a_product
func TestStarShowsTheCustomValuesOfAProduct(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	recs := records(t, s.query("SELECT * FROM products WHERE key = 'DEMO'").Text)
	if len(recs) != 1 {
		t.Fatal("no record")
	}
	expectMembers(t, recs[0], append(append([]string{}, productFields...), "custom.cost_center"))
	if member(t, recs[0], "custom.cost_center") != "CC-100" {
		t.Errorf("cost_center %v", member(t, recs[0], "custom.cost_center"))
	}
	if countCalls(f, "GET", "/custom_field_definitions") != 1 || countCalls(f, "GET", "/products/"+demoID) != 1 {
		t.Errorf("calls: %s", describeCalls(f.Calls()))
	}
	recs = records(t, s.query("SELECT * FROM products WHERE key = 'OTHER'").Text)
	if len(recs) == 1 {
		expectMembers(t, recs[0], productFields)
	}
	f.Answer("GET", "/custom_field_definitions", 403, nil, `{"error":"Forbidden"}`, 0)
	f.Reset()
	recs = records(t, s.query("SELECT * FROM products WHERE key = 'DEMO'").Text)
	if len(recs) == 1 {
		expectMembers(t, recs[0], productFields)
	}
	if countCalls(f, "GET", "/products/"+demoID) != 0 {
		t.Errorf("ApiGet on a product: %s", describeCalls(f.Calls()))
	}
}

// EXAMPLE: custom_fields_in_group_by_and_order_by
func TestCustomFieldsInGroupByAndOrderBy(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT custom.priority, COUNT(*) FROM features WHERE product = 'DEMO' AND custom.priority IS NOT NULL GROUP BY custom.priority"),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"custom.priority":"High","COUNT(*)":2},{"custom.priority":"Medium","COUNT(*)":1}]}`)
	expectResult(t, s.query("SELECT SUM(custom.effort_days) FROM features WHERE product = 'DEMO'"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"SUM(custom.effort_days)":23.5}]}`)
	expectResult(t, s.query("SELECT id, custom.effort_days FROM features WHERE product = 'DEMO' ORDER BY custom.effort_days DESC LIMIT 2"),
		`{"entity":"features","scope":"product DEMO","count":2,"total":6,"records":[{"id":"DEMO-1","custom.effort_days":12.5},{"id":"DEMO-2","custom.effort_days":8}]}`)
}

// EXAMPLE: describe_one_entity
func TestDescribeOneEntity(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.query("DESCRIBE features")
	if res.IsError {
		t.Fatalf("error %s", res.Text)
	}
	expectCalls(t, f, "GET /custom_field_definitions")
	var d struct {
		Entity string `json:"entity"`
		About  string `json:"about"`
		Fields []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Writable any    `json:"writable"`
		} `json:"fields"`
		CustomFields     []json.RawMessage `json:"custom_fields"`
		CustomFieldsNote string            `json:"custom_fields_note"`
		FilterByParent   []string          `json:"filter_by_parent"`
		StatusValues     string            `json:"status_values"`
		Change           []string          `json:"change_with_execute"`
		Example          string            `json:"example"`
	}
	if err := json.Unmarshal([]byte(res.Text), &d); err != nil {
		t.Fatalf("%v: %s", err, res.Text)
	}
	if want := []string{"entity", "about", "fields", "custom_fields", "custom_fields_note", "filter_by_parent", "status_values", "change_with_execute", "example"}; strings.Join(memberNames(t, res.Text), ",") != strings.Join(want, ",") {
		t.Errorf("members %v", memberNames(t, res.Text))
	}
	if d.Entity != "features" || d.About != "Features. Each feature is in one release." {
		t.Errorf("entity %q about %q", d.Entity, d.About)
	}
	var names []string
	writable := map[string]bool{"name": true, "status": true, "release": true, "initiative": true, "assigned_to": true, "start_date": true, "due_date": true, "tags": true, "description": true}
	types := map[string]string{}
	for _, fd := range d.Fields {
		names = append(names, fd.Name)
		types[fd.Name] = fd.Type
		if writable[fd.Name] && fd.Writable != true {
			t.Errorf("%s must have writable true", fd.Name)
		}
		if !writable[fd.Name] && fd.Writable != nil {
			t.Errorf("%s must have no member writable", fd.Name)
		}
	}
	if strings.Join(names, ",") != strings.Join(featureFields, ",") {
		t.Errorf("fields %v", names)
	}
	for k, v := range map[string]string{"release": "id of a release", "initiative": "id of an initiative", "assigned_to": "user (e-mail address)",
		"product": "product key", "start_date": "date (YYYY-MM-DD)", "created_at": "timestamp", "status": "status name", "score": "number"} {
		if types[k] != v {
			t.Errorf("type of %s %q, want %q", k, types[k], v)
		}
	}
	var cnames []string
	for _, c := range d.CustomFields {
		cnames = append(cnames, member(t, c, "name").(string))
		switch member(t, c, "name") {
		case "custom.priority":
			if string(c) != `{"name":"custom.priority","label":"Priority","type":"one option of a list","writable":true}` {
				t.Errorf("priority %s", c)
			}
		case "custom.reviewers":
			if string(c) != `{"name":"custom.reviewers","label":"Reviewers","type":"users (e-mail addresses), separated by commas","writable":true}` {
				t.Errorf("reviewers %s", c)
			}
		case "custom.value_score":
			if string(c) != `{"name":"custom.value_score","label":"Value score","type":"number"}` {
				t.Errorf("value_score %s", c)
			}
		}
	}
	if strings.Join(cnames, ",") != "custom.customer,custom.effort_days,custom.go_live,custom.markets,custom.platforms,custom.priority,custom.release_notes,custom.reviewers,custom.value_score" {
		t.Errorf("custom fields %v", cnames)
	}
	if d.CustomFieldsNote != "* shows custom fields only with id = '...', and only those that have a value. To select one by name: SELECT id, custom.customer FROM features WHERE id = 'PROD-23'" {
		t.Errorf("custom_fields_note %q", d.CustomFieldsNote)
	}
	if strings.Join(d.FilterByParent, "|") != "product = '...'|release = '...'|initiative = '...'|epic = '...'" {
		t.Errorf("filter_by_parent %v", d.FilterByParent)
	}
	if d.StatusValues != "SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'" {
		t.Errorf("status_values %q", d.StatusValues)
	}
	if strings.Join(d.Change, "|") != "INSERT INTO features (parent, ...) VALUES (...): parent is the id of a release|UPDATE features SET field = 'value' WHERE id = '...'|DELETE FROM features WHERE id = '...'" {
		t.Errorf("change_with_execute %v", d.Change)
	}
	if d.Example != "SELECT id, name, status FROM features WHERE release = 'PROD-R-3'" {
		t.Errorf("example %q", d.Example)
	}
}

// EXAMPLE: describe_when_aha_cannot_be_reached (expected-long: 60 seconds)
func TestDescribeWhenAhaCannotBeReached(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.NoAnswer("GET", "/custom_field_definitions")
	s := fixtureServer(t, f)
	expectError(t, s.query("DESCRIBE features"), "cannot reach Aha!: no answer within 60 seconds.")
	f.Reset()
	res := s.query("DESCRIBE comments")
	if res.IsError || member(t, json.RawMessage(res.Text), "entity") != "comments" || strings.Contains(res.Text, `"custom_fields"`) {
		t.Errorf("DESCRIBE comments: %s", res.Text)
	}
	expectNoCalls(t, f)
}

// EXAMPLE: describe_without_entity
func TestDescribeWithoutEntity(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.query("DESCRIBE")
	var d struct {
		Entities []json.RawMessage `json:"entities"`
		Note     string            `json:"note"`
	}
	if err := json.Unmarshal([]byte(res.Text), &d); err != nil || res.IsError {
		t.Fatalf("result %s", res.Text)
	}
	if len(d.Entities) != 12 {
		t.Errorf("%d entities", len(d.Entities))
	}
	if len(d.Entities) > 0 && string(d.Entities[0]) != `{"entity":"products","about":"Products (workspaces). The key of a product, for example PROD, starts the id of its records."}` {
		t.Errorf("first %s", d.Entities[0])
	}
	order := []string{"products", "releases", "features", "epics", "requirements", "initiatives", "goals", "ideas", "comments", "users", "statuses", "me"}
	for i, e := range d.Entities {
		expectMembers(t, e, []string{"entity", "about"})
		if i < len(order) && member(t, e, "entity") != order[i] {
			t.Errorf("entity %d is %v", i, member(t, e, "entity"))
		}
	}
	if d.Note != "DESCRIBE entity lists the fields of one entity, for example: DESCRIBE features" {
		t.Errorf("note %q", d.Note)
	}
	expectNoCalls(t, f)
}

// EXAMPLE: describe_shows_a_field_that_is_set_in_insert_only
func TestDescribeShowsAFieldThatIsSetInInsertOnly(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	res := s.query("DESCRIBE goals")
	var d struct {
		About  string            `json:"about"`
		Fields []json.RawMessage `json:"fields"`
	}
	_ = json.Unmarshal([]byte(res.Text), &d)
	if d.About != "Goals of a product. A new goal needs a success metric." {
		t.Errorf("about %q", d.About)
	}
	got := map[string]string{}
	for _, fd := range d.Fields {
		got[member(t, fd, "name").(string)] = string(fd)
	}
	for k, v := range map[string]string{"name": `{"name":"name","type":"string","writable":true}`,
		"success_metric": `{"name":"success_metric","type":"string","writable":"INSERT only"}`,
		"product":        `{"name":"product","type":"product key"}`} {
		if got[k] != v {
			t.Errorf("%s: %s, want %s", k, got[k], v)
		}
	}
}

// EXAMPLE: describe_and_read_have_the_same_fields
func TestDescribeAndReadHaveTheSameFields(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	res := s.query("DESCRIBE releases")
	var d struct {
		Fields []struct {
			Name string `json:"name"`
		} `json:"fields"`
	}
	_ = json.Unmarshal([]byte(res.Text), &d)
	var names []string
	for _, f := range d.Fields {
		names = append(names, f.Name)
	}
	want := []string{"id", "name", "release_date", "start_date", "status", "parking_lot", "released", "product", "progress", "description", "created_at", "url"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("fields %v", names)
	}
	// DEMO-R-2 has no custom value; DEMO-R-3 has budget_owner
	recs := records(t, s.query("SELECT * FROM releases WHERE id = 'DEMO-R-2'").Text)
	if len(recs) == 1 {
		expectMembers(t, recs[0], want)
	}
	recs = records(t, s.query("SELECT * FROM releases WHERE id = 'DEMO-R-3'").Text)
	if len(recs) == 1 {
		expectMembers(t, recs[0], append(want, "custom.budget_owner"))
	}
}

// EXAMPLE: goal_status_from_the_success_metric
func TestGoalStatusFromTheSuccessMetric(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id, success_metric, status FROM goals WHERE product = 'DEMO'"),
		`{"entity":"goals","scope":"product DEMO","count":1,"records":[{"id":"DEMO-G-1","success_metric":"Monthly viewers","status":"On track"}]}`)
	for _, c := range f.CallsTo("GET", "/api/v1/products/"+demoID+"/goals") {
		names := strings.Split(c.Query.Get("fields"), ",")
		n := 0
		for _, x := range names {
			if x == "success_metric" {
				n++
			}
			if x == "workflow_status" {
				t.Errorf("fields names workflow_status")
			}
		}
		if n != 1 {
			t.Errorf("fields names success_metric %d times: %s", n, c.Query.Get("fields"))
		}
	}
	if len(f.CallsTo("GET", "/api/v1/products/"+demoID+"/goals")) != 1 {
		t.Errorf("calls %s", describeCalls(f.Calls()))
	}
}

// EXAMPLE: release_of_an_epic
func TestReleaseOfAnEpic(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id, release FROM epics WHERE product = 'DEMO'"),
		`{"entity":"epics","scope":"product DEMO","count":1,"records":[{"id":"DEMO-E-1","release":"DEMO-R-3"}]}`)
	cs := f.CallsTo("GET", "/api/v1/products/"+demoID+"/epics")
	if len(cs) != 1 {
		t.Fatalf("calls %s", describeCalls(f.Calls()))
	}
	names := "," + cs[0].Query.Get("fields") + ","
	if !strings.Contains(names, ",release,") || strings.Contains(names, "release_reference_num") || strings.Contains(names, ",color,") {
		t.Errorf("fields %s", cs[0].Query.Get("fields"))
	}
	f.Reset()
	expectResult(t, s.query("SELECT name FROM epics WHERE release = 'demo-r-3'"),
		`{"entity":"epics","scope":"release DEMO-R-3","count":1,"records":[{"name":"Race day"}]}`)
	if countCalls(f, "GET", "/releases/DEMO-R-3/epics") != 1 {
		t.Errorf("calls %s", describeCalls(f.Calls()))
	}
}

// EXAMPLE: statuses_of_an_entity
func TestStatusesOfAnEntity(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	six := `{"entity":"statuses","scope":"product DEMO","count":6,"records":[{"name":"Under consideration"},{"name":"In design"},{"name":"Ready to develop"},{"name":"In development"},{"name":"Shipped"},{"name":"Will not implement"}]}`
	expectResult(t, s.query("SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'features'"), six)
	if countCalls(f, "GET", "/products/"+demoID) != 1 || countCalls(f, "GET", "/products/"+demoID+"/workflows") != 1 {
		t.Errorf("calls %s", describeCalls(f.Calls()))
	}
	expectResult(t, s.query("SELECT name FROM statuses WHERE product = 'demo' AND entity = 'feature'"), six)
	expectResult(t, s.query("SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'epics'"), six)
	expectResult(t, s.query("SELECT entity FROM statuses WHERE product = 'DEMO' AND name = 'Shipped'"),
		`{"entity":"statuses","scope":"product DEMO","count":5,"records":[{"entity":"releases"},{"entity":"features"},{"entity":"epics"},{"entity":"requirements"},{"entity":"ideas"}]}`)
	expectResult(t, s.query("SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'strategic_imperative'"),
		`{"entity":"statuses","scope":"product DEMO","count":4,"records":[{"name":"Not started"},{"name":"On track"},{"name":"At risk"},{"name":"Achieved"}]}`)
	expectError(t, s.query("SELECT name FROM statuses"),
		"statuses are listed per product: add product = '...' to WHERE. Example: SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'")
}

// EXAMPLE: current_user
func TestCurrentUser(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT email FROM me"), `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`)
}

// EXAMPLE: comments_of_a_record
func TestCommentsOfARecord(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	want := `{"entity":"comments","scope":"feature DEMO-3","count":1,"records":[{"body":"Verified with the provider.","user":"Test User"}]}`
	expectResult(t, s.query("SELECT body, user FROM comments WHERE parent = 'DEMO-3'"), want)
	expectResult(t, s.query("SELECT body, user FROM comments WHERE feature = 'demo-3'"), want)
}

// EXAMPLE: custom_fields_per_record_type
func TestCustomFieldsPerRecordType(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	get := func(stmt, name string) string {
		res := s.query(stmt)
		var m map[string]json.RawMessage
		_ = json.Unmarshal([]byte(res.Text), &m)
		return string(m[name])
	}
	if v := get("DESCRIBE ideas", "custom_fields"); v != `[{"name":"custom.segment","label":"Segment","type":"one option of a list","writable":true}]` {
		t.Errorf("ideas custom_fields %s", v)
	}
	if v := get("DESCRIBE ideas", "custom_fields_note"); v != `"* shows custom fields only with id = '...', and only those that have a value. To select one by name: SELECT id, custom.segment FROM ideas WHERE id = 'PROD-I-1'"` {
		t.Errorf("ideas note %s", v)
	}
	if v := get("DESCRIBE products", "custom_fields"); v != `[{"name":"custom.cost_center","label":"Cost center","type":"string"}]` {
		t.Errorf("products custom_fields %s", v)
	}
	if v := get("DESCRIBE products", "custom_fields_note"); v != `"* shows custom fields only with key = '...', and only those that have a value. To select one by name: SELECT key, custom.cost_center FROM products WHERE key = 'PROD'"` {
		t.Errorf("products note %s", v)
	}
	if get("DESCRIBE goals", "custom_fields") != "" || get("DESCRIBE goals", "custom_fields_note") != "" {
		t.Errorf("goals have custom fields")
	}
}

// EXAMPLE: custom_fields_are_read_by_name
func TestCustomFieldsAreReadByName(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id, custom.priority, custom.go_live FROM features WHERE release = 'DEMO-R-2'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","custom.priority":"High","custom.go_live":"2026-04-15"},{"id":"DEMO-2","custom.priority":"Medium","custom.go_live":null}]}`)
	expectCalls(t, f, "GET /custom_field_definitions", "GET /releases/DEMO-R-2/features")
}

// EXAMPLE: unknown_custom_field_lists_the_custom_fields
func TestUnknownCustomFieldListsTheCustomFields(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT id, custom.severity FROM features WHERE product = 'DEMO'"),
		"unknown custom field 'custom.severity' on features. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score. To see their types: DESCRIBE features")
	expectCalls(t, f, "GET /custom_field_definitions")
}

// EXAMPLE: custom_field_of_a_type_that_is_not_supported
func TestCustomFieldOfATypeThatIsNotSupported(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT id, custom.business_case FROM features WHERE id = 'DEMO-1'"),
		"custom.business_case has the type EquationField, and the server cannot read or set custom fields of that type. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score.")
	if countCalls(f, "GET", "/features/DEMO-1") != 0 {
		t.Errorf("ApiGet on /features/DEMO-1")
	}
}

// EXAMPLE: entity_without_custom_fields
// Spec ambiguity (see TRANSLATION_REPORT): the EXAMPLE lists the fields of
// goals without success_metric, while <field list> is "the names of all
// fields of the entity" and the catalog has success_metric. The test
// follows the rule and the catalog.
func TestEntityWithoutCustomFields(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT body, custom.priority FROM comments WHERE parent = 'DEMO-3'"),
		"comments have no custom fields that can be used here. Fields of comments: id, body, user, parent, created_at, url. Example: SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'")
	expectNoCalls(t, f)
	expectError(t, s.query("SELECT id, custom.priority FROM goals WHERE product = 'DEMO'"),
		"goals have no custom fields that can be used here. Fields of goals: id, name, success_metric, status, product, progress, value, effort, description, created_at, updated_at, url. Example: SELECT id, name FROM goals WHERE product = 'PROD'")
}

// EXAMPLE: custom_fields_that_the_key_cannot_read
func TestCustomFieldsThatTheKeyCannotRead(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.Answer("GET", "/custom_field_definitions", 403, nil, `{"error":"Forbidden"}`, 0)
	s := fixtureServer(t, f)
	fieldList := "id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url"
	expectError(t, s.query("SELECT id, custom.priority FROM features WHERE product = 'DEMO'"),
		"the custom fields of the account cannot be read with this API key (status 403). Fields of features: "+fieldList+". Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'")
	expectError(t, s.query("SELECT id, severity FROM features WHERE product = 'DEMO'"),
		"unknown field 'severity' on features. Fields of features: "+fieldList+". Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'")
	res := s.query("DESCRIBE features")
	if res.IsError || strings.Contains(res.Text, `"custom_fields"`) {
		t.Errorf("DESCRIBE features: %s", res.Text)
	}
	res = s.query("SELECT * FROM features WHERE id = 'DEMO-1'")
	if res.IsError {
		t.Errorf("error %s", res.Text)
	} else if recs := records(t, res.Text); len(recs) == 1 {
		expectMembers(t, recs[0], featureFields)
	}
	expectResult(t, s.query("SELECT id, name FROM features WHERE release = 'DEMO-R-2'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","name":"CSV export"},{"id":"DEMO-2","name":"Offline maps"}]}`)
}

// EXAMPLE: custom_field_that_was_added_later
func TestCustomFieldThatWasAddedLater(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`)
	f.AddDefinition("7000000000000000820", "risk", "Risk", "CustomFieldDefinitions::TextField", "Feature")
	expectResult(t, s.query("SELECT id, custom.risk FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.risk":null}]}`)
}

// EXAMPLE: own_name_of_the_entity
func TestOwnNameOfTheEntity(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT feature_id, feature_name FROM features WHERE feature = 'DEMO-5'"),
		`{"entity":"features","scope":"feature DEMO-5","count":1,"records":[{"feature_id":"DEMO-5","feature_name":"Rider's handbook"}]}`)
	expectCalls(t, f, "GET /features/DEMO-5")
}

// EXAMPLE: unknown_field_lists_the_fields
func TestUnknownFieldListsTheFields(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT id, severity FROM features WHERE product = 'DEMO'"),
		"unknown field 'severity' on features. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'")
	expectCalls(t, f, "GET /custom_field_definitions")
	expectError(t, s.query("SELECT id FROM features WHERE goal = 'DEMO-G-1'"),
		"unknown field 'goal' on features. Features cannot be filtered by goal. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'")
}

// EXAMPLE: unknown_entity_lists_the_entities
func TestUnknownEntityListsTheEntities(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT * FROM tickets"),
		"unknown entity 'tickets'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'")
}

// EXAMPLE: users_cannot_be_filtered_by_product
func TestUsersCannotBeFilteredByProduct(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT name FROM users WHERE product = 'DEMO'"),
		"users cannot be filtered by product. Fields of users: id, name, email, created_at. Example: SELECT name, email FROM users")
	expectResult(t, s.query("SELECT name, email FROM users"),
		`{"entity":"users","scope":"all users","count":2,"records":[{"name":"Test User","email":"test.user@example.com"},{"name":"Second User","email":"second.user@example.com"}]}`)
}

// EXAMPLE: bare_key_of_a_custom_field
func TestBareKeyOfACustomField(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id, priority FROM features WHERE id = 'demo-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","priority":"High"}]}`)
	expectResult(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-2' AND Custom.Priority = 'medium'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":1,"records":[{"id":"DEMO-2"}]}`)
}

// EXAMPLE: catalog_name_wins_over_a_custom_key
func TestCatalogNameWinsOverACustomKey(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.AddDefinition("7000000000000000821", "status", "Review status", "CustomFieldDefinitions::TextField", "Feature")
	f.SetCustomValue("features", "DEMO-1", textEntry("status", "Review status", "Checked"))
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT status, custom.status FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"status":"In design","custom.status":"Checked"}]}`)
}

// EXAMPLE: product_in_three_spellings
func TestProductInThreeSpellings(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	want := `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1"},{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'demo'"), want)
	expectResult(t, s.query("SELECT id FROM releases WHERE product = '7000000000000000001'"), want)
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'Demo Product'"), want)
}

// EXAMPLE: products_are_not_mixed
func TestProductsAreNotMixed(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id, name FROM goals WHERE product = 'other'"),
		`{"entity":"goals","scope":"product OTHER","count":1,"records":[{"id":"OTHER-G-1","name":"Other goal"}]}`)
}

// EXAMPLE: unknown_product_lists_the_keys
func TestUnknownProductListsTheKeys(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT id FROM releases WHERE product = 'DEMOX'"),
		"unknown product 'DEMOX'. Product keys: DEMO, OTHER. To see their names: SELECT key, name FROM products")
}

// EXAMPLE: release_id_where_a_product_is_needed
func TestReleaseIDWhereAProductIsNeeded(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT id FROM features WHERE product = 'DEMO-R-3'"),
		"'DEMO-R-3' is a release, not a product. Use: WHERE release = 'DEMO-R-3'")
}

// EXAMPLE: in_list_reads_each_parent
func TestInListReadsEachParent(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id, release FROM features WHERE release IN ('DEMO-R-1', 'DEMO-R-2')"),
		`{"entity":"features","scope":"release DEMO-R-1, release DEMO-R-2","count":3,"records":[{"id":"DEMO-6","release":"DEMO-R-1"},{"id":"DEMO-1","release":"DEMO-R-2"},{"id":"DEMO-2","release":"DEMO-R-2"}]}`)
	expectCalls(t, f, "GET /releases/DEMO-R-1/features", "GET /releases/DEMO-R-2/features")
}

// EXAMPLE: count_needs_one_list_request
func TestCountNeedsOneListRequest(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.AddProduct("BULK", "7000000000000000010", 60)
	s := fixtureServer(t, f)
	res := s.query("SELECT COUNT(*) FROM features WHERE product = 'BULK'")
	expectResult(t, res, `{"entity":"features","scope":"product BULK","count":1,"records":[{"COUNT(*)":60}]}`)
	if n := countCalls(f, "GET", "/products/7000000000000000010/features"); n != 1 {
		t.Errorf("%d list calls, want 1", n)
	}
}

// EXAMPLE: every_page_is_read_once
func TestEveryPageIsReadOnce(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.AddProduct("BIG", "7000000000000000011", 450)
	s := fixtureServer(t, f)
	res := s.query("SELECT id FROM features WHERE product = 'BIG' ORDER BY id")
	cs := f.CallsTo("GET", "/api/v1/products/7000000000000000011/features")
	var pages []string
	for _, c := range cs {
		pages = append(pages, c.Query.Get("page"))
		if c.Query.Get("per_page") != "200" {
			t.Errorf("per_page %q", c.Query.Get("per_page"))
		}
	}
	if strings.Join(pages, ",") != "1,2,3" {
		t.Errorf("pages %v", pages)
	}
	var top map[string]any
	_ = json.Unmarshal([]byte(res.Text), &top)
	if top["count"] != float64(50) || top["total"] != float64(450) || top["truncated"] != true {
		t.Errorf("count %v total %v truncated %v", top["count"], top["total"], top["truncated"])
	}
}

// EXAMPLE: too_many_pages
func TestTooManyPages(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.AddProduct("HUGE", "7000000000000000012", 3100)
	s := fixtureServer(t, f)
	route := "/products/7000000000000000012/features"
	expectError(t, s.query("SELECT id, name FROM features WHERE product = 'HUGE' ORDER BY name"),
		"this would read 3100 records. Narrow it down, for example with release = 'PROD-R-3'")
	if n := countCalls(f, "GET", route); n != 1 {
		t.Errorf("%d list calls, want 1", n)
	}
	res := s.query("SELECT id, name FROM features WHERE product = 'HUGE'")
	var top map[string]any
	_ = json.Unmarshal([]byte(res.Text), &top)
	if top["count"] != float64(50) || top["total"] != float64(3100) || top["truncated"] != true {
		t.Errorf("count %v total %v truncated %v", top["count"], top["total"], top["truncated"])
	}
	if n := countCalls(f, "GET", route); n != 2 {
		t.Errorf("%d list calls in all, want 2", n)
	}
}

// EXAMPLE: release_by_name_lists_the_releases
func TestReleaseByNameListsTheReleases(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND release = 'Summer release'"),
		"release = needs the id of the release, not its name. The releases of DEMO: DEMO-R-1 (What's next), DEMO-R-2 (Spring release), DEMO-R-3 (Summer release), DEMO-R-4 (Rider's cup release)")
}

// EXAMPLE: parent_that_does_not_exist
func TestParentThatDoesNotExist(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-99'"),
		"there is no release with the id 'DEMO-R-99'. To find it: SELECT id, name, release_date FROM releases WHERE product = 'PROD'")
}

// EXAMPLE: release_id_where_a_feature_id_is_needed
func TestReleaseIDWhereAFeatureIDIsNeeded(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT * FROM features WHERE id = 'DEMO-R-3'"),
		"'DEMO-R-3' is the id of a release, not of a feature. For the features of that release: SELECT * FROM features WHERE release = 'DEMO-R-3'")
	for _, c := range f.Calls() {
		if strings.HasPrefix(c.Path, "/api/v1/features/") || strings.HasPrefix(c.Path, "/api/v1/releases/") {
			t.Errorf("ApiGet call %s", c.Path)
		}
	}
	expectError(t, s.query("SELECT * FROM features WHERE id = 'CSV export'"),
		"'CSV export' is not an id. An id looks like 'PROD-23'. To search by name: SELECT * FROM features WHERE name CONTAINS 'CSV export'")
}

// EXAMPLE: comments_need_a_parent
func TestCommentsNeedAParent(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectError(t, s.query("SELECT body FROM comments"),
		"comments are listed per record: add parent = '...' to WHERE. Example: SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'")
}

// EXAMPLE: answers_in_the_json_form_of_aha
func TestAnswersInTheJSONFormOfAha(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id, status FROM features WHERE release = 'DEMO-R-2'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","status":"In design"},{"id":"DEMO-2","status":"Under consideration"}]}`)
	cs := f.CallsTo("GET", "/api/v1/releases/DEMO-R-2/features")
	if len(cs) != 1 {
		t.Fatalf("calls %s", describeCalls(f.Calls()))
	}
	names := "," + cs[0].Query.Get("fields") + ","
	if !strings.Contains(names, ",reference_num,") || !strings.Contains(names, ",workflow_status,") || strings.Contains(names, ".") {
		t.Errorf("fields %s", cs[0].Query.Get("fields"))
	}
	if strings.Contains(names, ",description,") {
		t.Errorf("fields names the heavy field description: %s", names)
	}
	expectResult(t, s.query("SELECT id, status FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","status":"In design"}]}`)
}

// EXAMPLE: answer_with_an_unknown_form
func TestAnswerWithAnUnknownForm(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.Answer("GET", "/releases/DEMO-R-2/features", 200, map[string]string{"Content-Type": "application/json"}, `{"items":[]}`, 0)
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-2'"),
		"aha: API error (status 200) on GET /releases/DEMO-R-2/features: the answer has an unknown form")
	f.Answer("GET", "/releases/DEMO-R-3/features", 200, nil, `not json`, 0)
	expectError(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-3'"),
		"aha: API error (status 200) on GET /releases/DEMO-R-3/features: the answer is not JSON")
	f.Answer("GET", "/releases/DEMO-R-1/features", 500, nil, `Internal Server Error`, 0)
	expectError(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-1'"),
		"aha: API error (status 500) on GET /releases/DEMO-R-1/features: Internal Server Error")
}

// EXAMPLE: custom_fields_are_requested_only_when_used
func TestCustomFieldsAreRequestedOnlyWhenUsed(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	res := s.query("SELECT id, name FROM features WHERE release = 'DEMO-R-2'")
	if res.IsError {
		t.Errorf("error %s", res.Text)
	}
	expectCalls(t, f, "GET /releases/DEMO-R-2/features")
	for _, c := range f.Calls() {
		if strings.Contains(c.Query.Get("fields"), "custom_fields") {
			t.Errorf("fields names custom_fields")
		}
	}
	f.Reset()
	expectResult(t, s.query("SELECT id, custom.effort_days FROM features WHERE release = 'DEMO-R-2' ORDER BY custom.effort_days"),
		`{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-2","custom.effort_days":8},{"id":"DEMO-1","custom.effort_days":12.5}]}`)
	cs := f.CallsTo("GET", "/api/v1/releases/DEMO-R-2/features")
	if len(cs) != 1 || !strings.Contains(","+cs[0].Query.Get("fields")+",", ",custom_fields,") {
		t.Errorf("list call does not name custom_fields: %s", describeCalls(f.Calls()))
	}
	for _, c := range f.Calls() {
		if strings.HasPrefix(c.Path, "/api/v1/features/") {
			t.Errorf("ApiGet on a feature: %s", c.Path)
		}
	}
}

// EXAMPLE: list_without_custom_fields_is_an_error
func TestListWithoutCustomFieldsIsAnError(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	f.LeaveOutCustomFieldsOfLists()
	s := fixtureServer(t, f)
	expectError(t, s.query("SELECT id, custom.priority FROM features WHERE release = 'DEMO-R-2'"),
		"Aha! returned this list of features without custom fields. A record that is read by its id has them: add id = '...' to WHERE.")
	for _, c := range f.Calls() {
		if strings.HasPrefix(c.Path, "/api/v1/features/") {
			t.Errorf("ApiGet on a feature: %s", c.Path)
		}
	}
	expectResult(t, s.query("SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`)
}

// EXAMPLE: custom_field_of_a_product
func TestCustomFieldOfAProduct(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT key, custom.cost_center FROM products"),
		`{"entity":"products","scope":"all products","count":2,"records":[{"key":"DEMO","custom.cost_center":"CC-100"},{"key":"OTHER","custom.cost_center":null}]}`)
	ok := false
	for _, c := range f.CallsTo("GET", "/api/v1/products") {
		if strings.Contains(","+c.Query.Get("fields")+",", ",custom_fields,") {
			ok = true
		}
	}
	if !ok {
		t.Errorf("no ApiList on /products with custom_fields: %s", describeCalls(f.Calls()))
	}
}

// EXAMPLE: parking_lot_and_missing_date
func TestParkingLotAndMissingDate(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id, release_date, parking_lot FROM releases WHERE product = 'DEMO'"),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","release_date":"2026-12-15","parking_lot":false}]}`)
}

// EXAMPLE: description_is_plain_text
func TestDescriptionIsPlainText(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT description FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"description":"Export all columns.\nKeep the order."}]}`)
}

// EXAMPLE: custom_values_by_kind
func TestCustomValuesByKind(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT custom.effort_days, custom.markets, custom.reviewers, custom.release_notes, custom.value_score FROM features WHERE id = 'DEMO-1'"),
		`{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"custom.effort_days":12.5,"custom.markets":"EMEA, APAC","custom.reviewers":"second.user@example.com","custom.release_notes":"Adds the export.\nSee the handbook.","custom.value_score":31}]}`)
	expectResult(t, s.query("SELECT custom.effort_days, custom.markets, custom.go_live FROM features WHERE id = 'DEMO-2'"),
		`{"entity":"features","scope":"feature DEMO-2","count":1,"records":[{"custom.effort_days":8,"custom.markets":null,"custom.go_live":null}]}`)
}

// EXAMPLE: or_and_not_are_evaluated
func TestOrAndNotAreEvaluated(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND (status = 'Shipped' OR status = 'In design')"),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-5"}]}`)
	expectResult(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-3' AND NOT status = 'Shipped'"),
		`{"entity":"features","scope":"release DEMO-R-3","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`)
	// further forms of parse-statement: IN, NOT IN, LIKE, BETWEEN, != NULL
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND status NOT IN ('Under consideration', 'Shipped')"),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-3"}]}`)
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND name LIKE 'live%'"),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`)
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'DEMO' AND release_date BETWEEN '2026-03-01' AND '2026-07-01'"),
		`{"entity":"releases","scope":"product DEMO","count":2,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"}]}`)
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND initiative != NULL"),
		`{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`)
}

// EXAMPLE: me_as_a_value
func TestMeAsAValue(t *testing.T) {
	t.Parallel()
	f := newFakeAha(t)
	s := fixtureServer(t, f)
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND assigned_to = 'me'"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`)
	if countCalls(f, "GET", "/me") != 1 {
		t.Errorf("calls on /me: %d", countCalls(f, "GET", "/me"))
	}
}

// EXAMPLE: fields_without_a_value_in_conditions
func TestFieldsWithoutAValueInConditions(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'DEMO' AND release_date IS NULL"),
		`{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-1"}]}`)
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'DEMO' AND parking_lot = false"),
		`{"entity":"releases","scope":"product DEMO","count":3,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`)
	expectResult(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-2' AND assigned_to != 'test.user@example.com'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":1,"records":[{"id":"DEMO-2"}]}`)
}

// EXAMPLE: conditions_on_custom_fields
func TestConditionsOnCustomFields(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	for _, c := range []struct{ q, want string }{
		{"SELECT id FROM features WHERE product = 'DEMO' AND custom.priority = 'high'", `[{"id":"DEMO-1"},{"id":"DEMO-3"}]`},
		{"SELECT id FROM features WHERE product = 'DEMO' AND custom.effort_days > 5", `[{"id":"DEMO-1"},{"id":"DEMO-2"}]`},
		{"SELECT id FROM features WHERE product = 'DEMO' AND custom.go_live < '2026-06-01'", `[{"id":"DEMO-1"}]`},
		{"SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers CONTAINS 'me'", `[{"id":"DEMO-3"}]`},
		{"SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers CONTAINS 'second user'", `[{"id":"DEMO-1"},{"id":"DEMO-3"}]`},
		{"SELECT id FROM features WHERE product = 'DEMO' AND custom.priority IS NULL", `[{"id":"DEMO-4"},{"id":"DEMO-5"},{"id":"DEMO-6"}]`},
	} {
		n := strings.Count(c.want, "{")
		expectResult(t, s.query(c.q), `{"entity":"features","scope":"product DEMO","count":`+string(rune('0'+n))+`,"records":`+c.want+`}`)
	}
}

// EXAMPLE: grouping_without_group_by
func TestGroupingWithoutGroupBy(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT release, COUNT(*) FROM features WHERE product = 'DEMO'"),
		`{"entity":"features","scope":"product DEMO","count":3,"records":[{"release":"DEMO-R-2","COUNT(*)":2},{"release":"DEMO-R-3","COUNT(*)":3},{"release":"DEMO-R-1","COUNT(*)":1}]}`)
}

// EXAMPLE: rows_without_a_value_sort_last
func TestRowsWithoutAValueSortLast(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date"),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"},{"id":"DEMO-R-1"}]}`)
	expectResult(t, s.query("SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date DESC"),
		`{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-4"},{"id":"DEMO-R-3"},{"id":"DEMO-R-2"},{"id":"DEMO-R-1"}]}`)
}

// EXAMPLE: count_of_nothing_is_zero
func TestCountOfNothingIsZero(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT COUNT(*) FROM features WHERE product = 'DEMO' AND status = 'Will not implement'"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":0}]}`)
}

// EXAMPLE: limit_of_the_statement_keeps_the_total
func TestLimitOfTheStatementKeepsTheTotal(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' LIMIT 2"),
		`{"entity":"features","scope":"product DEMO","count":2,"total":6,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`)
}

// Aggregates and DISTINCT of build-rows.
func TestAggregatesAndDistinct(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT COUNT(assigned_to), COUNT(DISTINCT status), AVG(custom.effort_days), MIN(name), MAX(custom.effort_days) FROM features WHERE product = 'DEMO'"),
		`{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(assigned_to)":1,"COUNT(DISTINCT status)":4,"AVG(custom.effort_days)":7.83,"MIN(name)":"CSV export","MAX(custom.effort_days)":12.5}]}`)
	expectResult(t, s.query("SELECT DISTINCT release FROM features WHERE product = 'DEMO'"),
		`{"entity":"features","scope":"product DEMO","count":3,"records":[{"release":"DEMO-R-2"},{"release":"DEMO-R-3"},{"release":"DEMO-R-1"}]}`)
	expectResult(t, s.query("SELECT status, COUNT(*) AS n FROM features WHERE product = 'DEMO' GROUP BY status ORDER BY n DESC LIMIT 1"),
		`{"entity":"features","scope":"product DEMO","count":1,"total":4,"records":[{"status":"Under consideration","n":3}]}`)
}

// Empty results get notes H04 to H06 (read-statement step 8).
func TestNotesForEmptyResults(t *testing.T) {
	t.Parallel()
	s := fixtureServer(t, newFakeAha(t))
	expectResult(t, s.query("SELECT key FROM products WHERE name = 'DEMO'"),
		`{"entity":"products","scope":"all products","count":0,"records":[],"note":"'DEMO' is the key of a product, not its name. Use: SELECT key, name FROM products WHERE key = 'DEMO'"}`)
	expectResult(t, s.query("SELECT id FROM features WHERE release = 'DEMO-R-2' AND name = 'DEMO-3'"),
		`{"entity":"features","scope":"release DEMO-R-2","count":0,"records":[],"note":"'DEMO-3' looks like an id, not a name. Use: WHERE id = 'DEMO-3'"}`)
	expectResult(t, s.query("SELECT id FROM features WHERE product = 'DEMO' AND name = 'demo-r-3'"),
		`{"entity":"features","scope":"product DEMO","count":0,"records":[],"note":"'demo-r-3' looks like the id of a release, not a name. Use: WHERE release = 'DEMO-R-3'"}`)
}
