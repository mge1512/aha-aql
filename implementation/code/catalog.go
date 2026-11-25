// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30

package main

import "strings"

// FieldType is the type of a catalog field (TYPES: FieldType).
type FieldType int

const (
	tString FieldType = iota
	tText
	tNumber
	tBoolean
	tDate
	tDatetime
	tStatus
	tUser
	tKey
	tReference
)

// Writable states of a catalog field.
const (
	wNo = iota
	wYes
	wInsertOnly
)

// FieldDef is one field of the catalog (TYPES: FieldDef).
type FieldDef struct {
	Name         string
	Type         FieldType
	RefKind      string // for reference fields
	Source       string // member of the API record; "a.b" is member b of member a
	Heavy        bool
	Writable     int
	FalseDefault bool // false when the API record has no value
	AsName       bool // a user field that is read as a name
}

// Route is one read route of an entity.
type Route struct {
	Field string
	Path  string // "{v}" is replaced by the value, "{product}" by a product id
}

// EntityDef is one entity of the catalog (TYPES: EntityDef).
type EntityDef struct {
	Name            string
	Singular        string
	OtherNames      []string
	Kind            string
	About           string
	Fields          []*FieldDef
	DefaultFields   []string
	OtherFieldNames map[string]string
	ReadRoutes      []Route
	FixedSource     string // products, users, statuses, me
	AccountList     string
	NameSearch      bool
	ParentFilters   []string
	ParentKind      string // product | release | feature | any | ""
	ParentField     string // the field that INSERT takes as the parent
	CanCreate       bool
	CanUpdate       bool
	CanDelete       bool
	CreateRoute     string
	UpdateRoute     string
	DeleteRoute     string
	CustomTypes     []string
	Example         string
}

func f(name string, t FieldType, src string, w int) *FieldDef {
	return &FieldDef{Name: name, Type: t, Source: src, Writable: w}
}

func ref(name, kind, src string, w int) *FieldDef {
	return &FieldDef{Name: name, Type: tReference, RefKind: kind, Source: src, Writable: w}
}

func heavy(d *FieldDef) *FieldDef    { d.Heavy = true; return d }
func falseDef(d *FieldDef) *FieldDef { d.FalseDefault = true; return d }
func asName(d *FieldDef) *FieldDef   { d.AsName = true; return d }
func names(s string) []string        { return strings.Split(s, ", ") }
func means(pairs ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		for _, w := range strings.Split(pairs[i], ", ") {
			m[w] = pairs[i+1]
		}
	}
	return m
}

// commonOtherNames is the table "Common other names for fields".
var commonOtherNames = means(
	"reference_num, ref_num, ref, reference, reference_number, ref_number, refnum", "id",
	"product_id, product_key, key, project, project_id, workspace, product_name", "product",
	"workflow_status, state", "status",
	"title", "name",
	"created", "created_at",
	"updated", "updated_at",
	"assignee, assigned_to_user, assigned_user, owner, assigned", "assigned_to",
)

// catalog is the single table of entities (entity-catalog).
var catalog = []*EntityDef{
	{
		Name: "products", Singular: "product", OtherNames: names("project, projects, workspace, workspaces"), Kind: "product",
		About:         "Products (workspaces). The key of a product, for example {KEY}, starts the id of its records.",
		DefaultFields: names("key, name"),
		OtherFieldNames: means("id, reference_prefix, prefix, reference_num, ref_num, ref, product, product_key, product_id", "key",
			"parent_id", "parent", "title", "name"),
		FixedSource: "products",
		CustomTypes: []string{"Project"},
		Example:     "SELECT key, name FROM products",
		Fields: []*FieldDef{
			f("key", tKey, "reference_prefix", wNo),
			f("name", tString, "name", wNo),
			falseDef(f("product_line", tBoolean, "product_line", wNo)),
			f("parent", tKey, "parent_id", wNo),
			heavy(f("description", tText, "description", wNo)),
			f("created_at", tDatetime, "created_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "releases", Singular: "release", OtherNames: names("versions"), Kind: "release",
		About:           "Releases of a product. The parking lot is a release with parking_lot = true and no date.",
		DefaultFields:   names("id, name, release_date, parking_lot"),
		OtherFieldNames: means("parent, parent_id", "product", "theme", "description", "date, ship_date", "release_date"),
		ReadRoutes:      []Route{{"id", "/releases/{v}"}, {"product", "/products/{v}/releases"}},
		ParentFilters:   names("product"), ParentKind: "product", ParentField: "product",
		CustomTypes: []string{"Release"},
		CanCreate:   true, CreateRoute: "/products/{product}/releases",
		CanUpdate: true, UpdateRoute: "/releases/{id}",
		CanDelete: true, DeleteRoute: "/products/{product}/releases/{id}",
		Example: "SELECT id, name, release_date FROM releases WHERE product = '{KEY}'",
		Fields: []*FieldDef{
			f("id", tString, "reference_num", wNo),
			f("name", tString, "name", wYes),
			f("release_date", tDate, "release_date", wYes),
			f("start_date", tDate, "start_date", wYes),
			f("status", tStatus, "workflow_status.name", wNo),
			falseDef(f("parking_lot", tBoolean, "parking_lot", wYes)),
			falseDef(f("released", tBoolean, "released", wNo)),
			f("product", tKey, "product_id", wNo),
			f("progress", tNumber, "progress", wNo),
			heavy(f("description", tText, "theme", wYes)),
			f("created_at", tDatetime, "created_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "features", Singular: "feature", Kind: "feature",
		About:         "Features. Each feature is in one release.",
		DefaultFields: names("id, name, status, release"),
		OtherFieldNames: means("release_id, release_ref, release_reference_num, parent, parent_id", "release",
			"initiative_id", "initiative", "epic_id", "epic"),
		ReadRoutes: []Route{{"id", "/features/{v}"}, {"release", "/releases/{v}/features"}, {"epic", "/epics/{v}/features"},
			{"initiative", "/initiatives/{v}/features"}, {"product", "/products/{v}/features"}},
		AccountList: "/features", NameSearch: true,
		ParentFilters: names("product, release, initiative, epic"), ParentKind: "release", ParentField: "release",
		CustomTypes: []string{"Feature"},
		CanCreate:   true, CreateRoute: "/releases/{parent}/features",
		CanUpdate: true, UpdateRoute: "/features/{id}",
		CanDelete: true, DeleteRoute: "/features/{id}",
		Example: "SELECT id, name, status FROM features WHERE release = '{KEY}-R-3'",
		Fields: []*FieldDef{
			f("id", tString, "reference_num", wNo),
			f("name", tString, "name", wYes),
			f("status", tStatus, "workflow_status.name", wYes),
			ref("release", "release", "release_reference_num", wYes),
			ref("initiative", "initiative", "initiative_reference_num", wYes),
			ref("epic", "epic", "epic_reference_num", wNo),
			f("assigned_to", tUser, "assigned_to_user.email", wYes),
			f("product", tKey, "product_id", wNo),
			f("start_date", tDate, "start_date", wYes),
			f("due_date", tDate, "due_date", wYes),
			f("tags", tString, "tags", wYes),
			f("score", tNumber, "score", wNo),
			f("progress", tNumber, "progress", wNo),
			heavy(f("description", tText, "description", wYes)),
			f("created_at", tDatetime, "created_at", wNo),
			f("updated_at", tDatetime, "updated_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "epics", Singular: "epic", OtherNames: names("master_features"), Kind: "epic",
		About:           "Epics. An epic is in one release and groups features.",
		DefaultFields:   names("id, name, status, release"),
		OtherFieldNames: means("release_id, parent, parent_id", "release", "initiative_id", "initiative"),
		ReadRoutes: []Route{{"id", "/epics/{v}"}, {"release", "/releases/{v}/epics"}, {"initiative", "/initiatives/{v}/epics"},
			{"product", "/products/{v}/epics"}},
		AccountList: "/epics", NameSearch: true,
		ParentFilters: names("product, release, initiative"), ParentKind: "release", ParentField: "release",
		CustomTypes: []string{"Epic", "MasterFeature"},
		CanCreate:   true, CreateRoute: "/releases/{parent}/epics",
		CanUpdate: true, UpdateRoute: "/epics/{id}",
		CanDelete: true, DeleteRoute: "/epics/{id}",
		Example: "SELECT id, name, status FROM epics WHERE product = '{KEY}'",
		Fields: []*FieldDef{
			f("id", tString, "reference_num", wNo),
			f("name", tString, "name", wYes),
			f("status", tStatus, "workflow_status.name", wYes),
			ref("release", "release", "release.reference_num", wNo),
			ref("initiative", "initiative", "initiative_reference_num", wYes),
			f("assigned_to", tUser, "assigned_to_user.email", wNo),
			f("product", tKey, "product_id", wNo),
			f("start_date", tDate, "start_date", wYes),
			f("due_date", tDate, "due_date", wYes),
			f("progress", tNumber, "progress", wYes),
			heavy(f("description", tText, "description", wYes)),
			f("created_at", tDatetime, "created_at", wNo),
			f("updated_at", tDatetime, "updated_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "requirements", Singular: "requirement", Kind: "requirement",
		About:           "Requirements of a feature.",
		DefaultFields:   names("id, name, status, feature"),
		OtherFieldNames: means("feature_id, parent, parent_id", "feature"),
		ReadRoutes:      []Route{{"id", "/requirements/{v}"}, {"feature", "/features/{v}/requirements"}},
		ParentFilters:   names("feature"), ParentKind: "feature", ParentField: "feature",
		CustomTypes: []string{"Requirement"},
		CanCreate:   true, CreateRoute: "/features/{parent}/requirements",
		CanUpdate: true, UpdateRoute: "/requirements/{id}",
		CanDelete: true, DeleteRoute: "/requirements/{id}",
		Example: "SELECT id, name, status FROM requirements WHERE feature = '{KEY}-23'",
		Fields: []*FieldDef{
			f("id", tString, "reference_num", wNo),
			f("name", tString, "name", wYes),
			f("status", tStatus, "workflow_status.name", wYes),
			ref("feature", "feature", "feature.reference_num", wNo),
			f("assigned_to", tUser, "assigned_to_user.email", wYes),
			f("original_estimate", tNumber, "original_estimate", wYes),
			f("work_done", tNumber, "work_done", wYes),
			heavy(f("description", tText, "description", wYes)),
			f("created_at", tDatetime, "created_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "initiatives", Singular: "initiative", Kind: "initiative",
		About:           "Initiatives of a product. Features can be linked to an initiative.",
		DefaultFields:   names("id, name, status"),
		OtherFieldNames: means("parent, parent_id", "product"),
		ReadRoutes:      []Route{{"id", "/initiatives/{v}"}, {"product", "/products/{v}/initiatives"}},
		AccountList:     "/initiatives", NameSearch: true,
		ParentFilters: names("product"), ParentKind: "product", ParentField: "product",
		CustomTypes: []string{"Initiative"},
		CanCreate:   true, CreateRoute: "/products/{product}/initiatives",
		CanUpdate: true, UpdateRoute: "/initiatives/{id}",
		CanDelete: true, DeleteRoute: "/products/{product}/initiatives/{id}",
		Example: "SELECT id, name, status FROM initiatives WHERE product = '{KEY}'",
		Fields: []*FieldDef{
			f("id", tString, "reference_num", wNo),
			f("name", tString, "name", wYes),
			f("status", tStatus, "workflow_status.name", wYes),
			f("product", tKey, "product_id", wNo),
			f("start_date", tDate, "start_date", wYes),
			f("end_date", tDate, "end_date", wYes),
			f("progress", tNumber, "progress", wNo),
			f("value", tNumber, "value", wYes),
			f("effort", tNumber, "effort", wYes),
			f("color", tString, "color", wYes),
			f("presented", tBoolean, "presented", wYes),
			heavy(f("description", tText, "description", wYes)),
			f("created_at", tDatetime, "created_at", wNo),
			f("updated_at", tDatetime, "updated_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "goals", Singular: "goal", OtherNames: names("strategic_imperatives"), Kind: "goal",
		About:           "Goals of a product. A new goal needs a success metric.",
		DefaultFields:   names("id, name"),
		OtherFieldNames: means("parent, parent_id", "product"),
		ReadRoutes:      []Route{{"id", "/goals/{v}"}, {"product", "/products/{v}/goals"}},
		AccountList:     "/goals", NameSearch: true,
		ParentFilters: names("product"), ParentKind: "product", ParentField: "product",
		CustomTypes: []string{"StrategicImperative", "Goal"},
		CanCreate:   true, CreateRoute: "/products/{product}/goals",
		CanUpdate: true, UpdateRoute: "/goals/{id}",
		CanDelete: true, DeleteRoute: "/products/{product}/goals/{id}",
		Example: "SELECT id, name FROM goals WHERE product = '{KEY}'",
		Fields: []*FieldDef{
			f("id", tString, "reference_num", wNo),
			f("name", tString, "name", wYes),
			f("success_metric", tString, "success_metric.name", wInsertOnly),
			f("status", tStatus, "success_metric.workflow_status.name", wYes),
			f("product", tKey, "product_id", wNo),
			f("progress", tNumber, "progress", wYes),
			f("value", tNumber, "value", wNo),
			f("effort", tNumber, "effort", wNo),
			heavy(f("description", tText, "description", wYes)),
			f("created_at", tDatetime, "created_at", wNo),
			f("updated_at", tDatetime, "updated_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "ideas", Singular: "idea", Kind: "idea",
		About:           "Ideas submitted for a product. feature is the feature an idea was promoted to.",
		DefaultFields:   names("id, name, status, votes"),
		OtherFieldNames: means("parent, parent_id", "product", "votes_count", "votes", "feature_id", "feature"),
		ReadRoutes:      []Route{{"id", "/ideas/{v}"}, {"product", "/products/{v}/ideas"}},
		AccountList:     "/ideas", NameSearch: true,
		ParentFilters: names("product"), ParentKind: "product", ParentField: "product",
		CustomTypes: []string{"Idea"},
		CanCreate:   true, CreateRoute: "/products/{product}/ideas",
		CanUpdate: true, UpdateRoute: "/ideas/{id}",
		CanDelete: true, DeleteRoute: "/ideas/{id}",
		Example: "SELECT id, name, votes FROM ideas WHERE product = '{KEY}' ORDER BY votes DESC LIMIT 5",
		Fields: []*FieldDef{
			f("id", tString, "reference_num", wNo),
			f("name", tString, "name", wYes),
			f("status", tStatus, "workflow_status.name", wYes),
			f("votes", tNumber, "votes", wNo),
			f("product", tKey, "product_id", wNo),
			ref("feature", "feature", "feature.reference_num", wNo),
			f("visibility", tString, "visibility", wYes),
			heavy(f("description", tText, "description", wYes)),
			f("created_at", tDatetime, "created_at", wNo),
			f("updated_at", tDatetime, "updated_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "comments", Singular: "comment", Kind: "comment",
		About:         "Comments on a record. parent is the id of the commented record.",
		DefaultFields: names("id, body, user, parent, created_at"),
		OtherFieldNames: means("content, text, comment, message, description", "body",
			"author, created_by, author_name, user_name", "user",
			"feature, feature_id, epic, epic_id, idea, idea_id, release, release_id, initiative, initiative_id, goal, goal_id, requirement, requirement_id, record, record_id, parent_id, commentable, product, product_id, product_key", "parent"),
		ReadRoutes:    []Route{{"id", "/comments/{v}"}, {"parent", ""}},
		ParentFilters: names("parent"), ParentKind: "any", ParentField: "parent",
		CanCreate: true, CreateRoute: "",
		CanUpdate: true, UpdateRoute: "/comments/{id}",
		CanDelete: true, DeleteRoute: "/comments/{id}",
		Example: "SELECT body, user, created_at FROM comments WHERE parent = '{KEY}-23'",
		Fields: []*FieldDef{
			f("id", tString, "id", wNo),
			f("body", tText, "body", wYes),
			asName(f("user", tUser, "user.name", wNo)),
			f("parent", tString, "commentable.url", wNo),
			f("created_at", tDatetime, "created_at", wNo),
			f("url", tString, "url", wNo),
		},
	},
	{
		Name: "users", Singular: "user", Kind: "user",
		About:           "Users of the account. For the user you are working as, use: SELECT name, email FROM me",
		DefaultFields:   names("id, name, email"),
		OtherFieldNames: means("mail, e_mail", "email", "user_id", "id"),
		FixedSource:     "users",
		Example:         "SELECT name, email FROM users",
		Fields: []*FieldDef{
			f("id", tString, "id", wNo),
			f("name", tString, "name", wNo),
			f("email", tString, "email", wNo),
			f("created_at", tDatetime, "created_at", wNo),
		},
	},
	{
		Name: "statuses", Singular: "status", OtherNames: names("workflow_statuses, workflows, workflow"),
		About:           "The workflow statuses a record can have, per product and entity.",
		DefaultFields:   names("name, entity"),
		OtherFieldNames: means("status, id, title", "name", "type, entity_type, kind", "entity"),
		FixedSource:     "statuses",
		ParentFilters:   names("product, entity"),
		Example:         "SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features'",
		Fields: []*FieldDef{
			f("name", tString, "name", wNo),
			f("entity", tString, "", wNo),
			f("position", tNumber, "position", wNo),
			falseDef(f("complete", tBoolean, "complete", wNo)),
			f("product", tKey, "", wNo),
		},
	},
	{
		Name: "me", Singular: "me", OtherNames: names("current_user, myself, whoami"),
		About:           "The user you are working as. One record.",
		DefaultFields:   names("name, email"),
		OtherFieldNames: means("mail", "email", "user", "name", "user_id", "id"),
		FixedSource:     "me",
		Example:         "SELECT name, email FROM me",
		Fields: []*FieldDef{
			f("id", tString, "id", wNo),
			f("name", tString, "name", wNo),
			f("email", tString, "email", wNo),
		},
	},
}

// workflowMembers is the table of members of a product that name a
// workflow, in the order of the entity statuses.
var workflowMembers = []struct{ member, entity string }{
	{"release_workflow", "releases"},
	{"feature_workflow", "features"},
	{"epic_workflow", "epics"},
	{"requirement_workflow", "requirements"},
	{"initiative_workflow", "initiatives"},
	{"strategic_imperative_workflow", "goals"},
	{"idea_workflow", "ideas"},
	{"key_result_workflow", "key_results"},
	{"page_workflow", "pages"},
}

func entityByName(name string) *EntityDef {
	for _, e := range catalog {
		if e.Name == name {
			return e
		}
	}
	return nil
}

// entityOfKind gives the entity whose records have the kind.
func entityOfKind(kind string) *EntityDef {
	for _, e := range catalog {
		if e.Kind == kind {
			return e
		}
	}
	return nil
}

func (e *EntityDef) field(name string) *FieldDef {
	for _, fd := range e.Fields {
		if fd.Name == name {
			return fd
		}
	}
	return nil
}

// idField is "key" for products, "id" for every other entity that has a
// field id, and "" for statuses.
func (e *EntityDef) idField() string {
	if e.Name == "products" {
		return "key"
	}
	if e.field("id") != nil {
		return "id"
	}
	return ""
}

func (e *EntityDef) hasOps() bool { return e.CanCreate || e.CanUpdate || e.CanDelete }

func (e *EntityDef) route(field string) (Route, bool) {
	for _, r := range e.ReadRoutes {
		if r.Field == field {
			return r, true
		}
	}
	return Route{}, false
}

func (e *EntityDef) hasParentFilter(name string) bool {
	for _, p := range e.ParentFilters {
		if p == name {
			return true
		}
	}
	return false
}

func entityNames() []string {
	var out []string
	for _, e := range catalog {
		out = append(out, e.Name)
	}
	return out
}

func fieldNames(e *EntityDef) []string {
	var out []string
	for _, fd := range e.Fields {
		out = append(out, fd.Name)
	}
	return out
}

// writableList is <writable list> of error-messages.
func writableList(e *EntityDef) string {
	var out []string
	for _, fd := range e.Fields {
		switch fd.Writable {
		case wYes:
			out = append(out, fd.Name)
		case wInsertOnly:
			out = append(out, fd.Name+" (INSERT only)")
		}
	}
	return strings.Join(out, ", ")
}

func changeableList() string {
	var out []string
	for _, e := range catalog {
		if e.hasOps() {
			out = append(out, e.Name)
		}
	}
	return strings.Join(out, ", ")
}

// routeFields gives the fields of the read routes other than id, joined.
func routeFields(e *EntityDef) []string {
	var out []string
	for _, r := range e.ReadRoutes {
		if r.Field != "id" {
			out = append(out, r.Field)
		}
	}
	return out
}

// sourceMembers gives the top-level members of the API sources of the
// fields that a list request names (MECHANISM of plan-fetch, step 8).
func sourceMembers(e *EntityDef, used map[string]bool, custom bool) string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, fd := range e.Fields {
		if fd.Heavy && !used[fd.Name] {
			continue
		}
		src := fd.Source
		if i := strings.Index(src, "."); i >= 0 {
			src = src[:i]
		}
		add(src)
	}
	if custom {
		add("custom_fields")
	}
	return strings.Join(out, ",")
}
