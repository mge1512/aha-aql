// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30
// tests by: claude-opus-5-5

package aqltests

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDescribeOneEntity(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`DESCRIBE features`)
	if r.isError {
		t.Fatal(r.text)
	}
	d := orderedObject(t, r.text)
	wantKeys := []string{"entity", "about", "fields", "custom_fields", "custom_fields_note", "filter_by_parent", "status_values", "change_with_execute", "example"}
	if !eqStrings(keysOf(d), wantKeys) {
		t.Fatalf("members: %v", keysOf(d))
	}
	if string(memberOf(t, d, "entity")) != `"features"` || string(memberOf(t, d, "about")) != `"Features. Each feature is in one release."` {
		t.Fatal(r.text)
	}
	var fields []json.RawMessage
	_ = json.Unmarshal(memberOf(t, d, "fields"), &fields)
	var names []string
	writable := map[string]bool{"name": true, "status": true, "release": true, "initiative": true, "assigned_to": true,
		"start_date": true, "due_date": true, "tags": true, "description": true}
	types := map[string]string{}
	for _, f := range fields {
		fm := orderedObject(t, string(f))
		var n, ty string
		_ = json.Unmarshal(memberOf(t, fm, "name"), &n)
		_ = json.Unmarshal(memberOf(t, fm, "type"), &ty)
		names = append(names, n)
		types[n] = ty
		if writable[n] {
			if string(memberOf(t, fm, "writable")) != "true" {
				t.Fatalf("%s not writable: %s", n, f)
			}
		} else if hasMember(fm, "writable") {
			t.Fatalf("%s has writable: %s", n, f)
		}
	}
	if !eqStrings(names, featureFields) {
		t.Fatalf("fields: %v", names)
	}
	for n, ty := range map[string]string{"release": "id of a release", "initiative": "id of an initiative",
		"assigned_to": "user (e-mail address)", "product": "product key", "start_date": "date (YYYY-MM-DD)", "created_at": "timestamp"} {
		if types[n] != ty {
			t.Fatalf("type of %s: %q", n, types[n])
		}
	}
	var cfs []json.RawMessage
	_ = json.Unmarshal(memberOf(t, d, "custom_fields"), &cfs)
	var cnames []string
	byName := map[string]string{}
	for _, c := range cfs {
		var m struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(c, &m)
		cnames = append(cnames, m.Name)
		byName[m.Name] = string(c)
	}
	if !eqStrings(cnames, []string{"custom.customer", "custom.effort_days", "custom.go_live", "custom.markets", "custom.platforms",
		"custom.priority", "custom.release_notes", "custom.reviewers", "custom.value_score"}) {
		t.Fatalf("custom fields: %v", cnames)
	}
	for n, want := range map[string]string{
		"custom.priority":    `{"name":"custom.priority","label":"Priority","type":"one option of a list","writable":true}`,
		"custom.reviewers":   `{"name":"custom.reviewers","label":"Reviewers","type":"users (e-mail addresses), separated by commas","writable":true}`,
		"custom.value_score": `{"name":"custom.value_score","label":"Value score","type":"number"}`,
	} {
		if byName[n] != want {
			t.Fatalf("%s\n got: %s\nwant: %s", n, byName[n], want)
		}
	}
	for k, want := range map[string]string{
		"custom_fields_note":  `"* shows custom fields only with id = '...', and only those that have a value. To select one by name: SELECT id, custom.customer FROM features WHERE id = 'PROD-23'"`,
		"filter_by_parent":    `["product = '...'","release = '...'","initiative = '...'","epic = '...'"]`,
		"status_values":       `"SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'"`,
		"change_with_execute": `["INSERT INTO features (parent, ...) VALUES (...): parent is the id of a release","UPDATE features SET field = 'value' WHERE id = '...'","DELETE FROM features WHERE id = '...'"]`,
		"example":             `"SELECT id, name, status FROM features WHERE release = 'PROD-R-3'"`,
	} {
		if got := string(memberOf(t, d, k)); got != want {
			t.Fatalf("%s\n got: %s\nwant: %s", k, got, want)
		}
	}
	e.wantCalls("GET /custom_field_definitions")
}

func TestDescribeWhenAhaCannotBeReached(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.fake.NoAnswer("GET", "/custom_field_definitions")
	s := e.stdio(nil)
	wantError(t, s.call("query", map[string]any{"query": `DESCRIBE features`}, 150e9), `cannot reach Aha!: no answer within 60 seconds.`)
	e.fake.Reset()
	r := s.query(`DESCRIBE comments`)
	d := orderedObject(t, r.text)
	if string(memberOf(t, d, "entity")) != `"comments"` || hasMember(d, "custom_fields") {
		t.Fatal(r.text)
	}
	e.wantNoCall()
}

func TestDescribeWithoutEntity(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`DESCRIBE`)
	d := orderedObject(t, r.text)
	var ents []json.RawMessage
	_ = json.Unmarshal(memberOf(t, d, "entities"), &ents)
	if len(ents) != 12 {
		t.Fatalf("entities: %d", len(ents))
	}
	if string(ents[0]) != `{"entity":"products","about":"Products (workspaces). The key of a product, for example PROD, starts the id of its records."}` {
		t.Fatalf("first: %s", ents[0])
	}
	order := []string{"products", "releases", "features", "epics", "requirements", "initiatives", "goals", "ideas", "comments", "users", "statuses", "me"}
	for i, en := range ents {
		m := orderedObject(t, string(en))
		if !eqStrings(keysOf(m), []string{"entity", "about"}) || string(m[0].val) != `"`+order[i]+`"` {
			t.Fatalf("entity %d: %s", i, en)
		}
	}
	if string(memberOf(t, d, "note")) != `"DESCRIBE entity lists the fields of one entity, for example: DESCRIBE features"` {
		t.Fatal(r.text)
	}
	e.wantNoCall()
}

func TestDescribeShowsAFieldThatIsSetInInsertOnly(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`DESCRIBE goals`)
	d := orderedObject(t, r.text)
	if string(memberOf(t, d, "about")) != `"Goals of a product. A new goal needs a success metric."` {
		t.Fatal(r.text)
	}
	var fields []json.RawMessage
	_ = json.Unmarshal(memberOf(t, d, "fields"), &fields)
	got := map[string]string{}
	for _, f := range fields {
		var m struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(f, &m)
		got[m.Name] = string(f)
	}
	for n, want := range map[string]string{
		"name":           `{"name":"name","type":"string","writable":true}`,
		"success_metric": `{"name":"success_metric","type":"string","writable":"INSERT only"}`,
		"product":        `{"name":"product","type":"product key"}`,
	} {
		if got[n] != want {
			t.Fatalf("%s\n got: %s\nwant: %s", n, got[n], want)
		}
	}
}

func TestCustomFieldsPerRecordType(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	r := s.query(`DESCRIBE ideas`)
	d := orderedObject(t, r.text)
	if got := string(memberOf(t, d, "custom_fields")); got != `[{"name":"custom.segment","label":"Segment","type":"one option of a list","writable":true}]` {
		t.Fatalf("ideas custom_fields: %s", got)
	}
	if got := string(memberOf(t, d, "custom_fields_note")); got != `"* shows custom fields only with id = '...', and only those that have a value. To select one by name: SELECT id, custom.segment FROM ideas WHERE id = 'PROD-I-1'"` {
		t.Fatalf("ideas note: %s", got)
	}
	r = s.query(`DESCRIBE products`)
	d = orderedObject(t, r.text)
	if got := string(memberOf(t, d, "custom_fields")); got != `[{"name":"custom.cost_center","label":"Cost center","type":"string"}]` {
		t.Fatalf("products custom_fields: %s", got)
	}
	if got := string(memberOf(t, d, "custom_fields_note")); got != `"* shows custom fields only with key = '...', and only those that have a value. To select one by name: SELECT key, custom.cost_center FROM products WHERE key = 'PROD'"` {
		t.Fatalf("products note: %s", got)
	}
	r = s.query(`DESCRIBE goals`)
	d = orderedObject(t, r.text)
	if hasMember(d, "custom_fields") || hasMember(d, "custom_fields_note") {
		t.Fatal(r.text)
	}
	if !strings.Contains(r.text, `"change_with_execute"`) {
		t.Fatal(r.text)
	}
}
