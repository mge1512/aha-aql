// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// entity-catalog: the single table of entities, fields, types, API sources,
// routes, parents and operations. Every other part of the server takes these
// facts from here.

import "strings"

type FieldType int

const (
	ftString FieldType = iota
	ftText
	ftNumber
	ftBoolean
	ftDate
	ftDatetime
	ftStatus
	ftUser
	ftKey
	ftReference
)

type Writable int

const (
	wNo Writable = iota
	wYes
	wInsertOnly
)

type FieldDef struct {
	Name            string
	Type            FieldType
	RefKind         string // record kind, for type reference
	Source          string // member of the API record; a.b is member b of member a
	Heavy           bool
	Writable        Writable
	RequestMember   string
	FalseWhenAbsent bool
	UserAsName      bool
	NoSource        bool // statuses: product of the statement
}

type Route struct {
	Field string
	Path  string // with one parameter {v}
}

type EntityDef struct {
	Name              string
	Singular          string
	OtherNames        []string
	Kind              string
	About             string
	Fields            []FieldDef
	DefaultFields     []string
	OtherFieldNames   map[string]string
	ReadRoutes        []Route
	AccountList       string
	NameSearch        bool
	ParentFilters     []string
	ParentKind        string // "product", a record kind, "any", or ""
	ParentField       string
	CanCreate         bool
	CanUpdate         bool
	CanDelete         bool
	CreateRoute       string // {v} is the parent
	UpdateRoute       string // {id}
	DeleteRoute       string // {id} and {product}
	CustomRecordTypes []string
	Example           string
	Fixed             string // products, users, statuses, me: read from a fixed source
}

func (e *EntityDef) Field(name string) *FieldDef {
	for i := range e.Fields {
		if e.Fields[i].Name == name {
			return &e.Fields[i]
		}
	}
	return nil
}

func (e *EntityDef) HasField(name string) bool { return e.Field(name) != nil }

// IDField is key for products, id for every other entity that has a field
// id, and none for statuses.
func (e *EntityDef) IDField() string {
	if e.Name == "products" {
		return "key"
	}
	if e.HasField("id") {
		return "id"
	}
	return ""
}

func (e *EntityDef) HasOperation() bool { return e.CanCreate || e.CanUpdate || e.CanDelete }

func (e *EntityDef) RouteFor(field string) string {
	for _, r := range e.ReadRoutes {
		if r.Field == field {
			return r.Path
		}
	}
	return ""
}

func (e *EntityDef) HasParentFilter(f string) bool {
	for _, p := range e.ParentFilters {
		if p == f {
			return true
		}
	}
	return false
}

func fd(name string, t FieldType, source string, w Writable) FieldDef {
	return FieldDef{Name: name, Type: t, Source: source, Writable: w}
}

func ref(name, kind, source string, w Writable) FieldDef {
	return FieldDef{Name: name, Type: ftReference, RefKind: kind, Source: source, Writable: w}
}

func heavy(f FieldDef) FieldDef { f.Heavy = true; return f }

func falseAbsent(f FieldDef) FieldDef { f.FalseWhenAbsent = true; return f }

func member(f FieldDef, m string) FieldDef { f.RequestMember = m; return f }

func names(pairs string) map[string]string {
	// "a b c=x; d=y": every word before = means the name after it
	out := map[string]string{}
	for _, part := range strings.Split(pairs, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eq := strings.LastIndex(part, "=")
		target := strings.TrimSpace(part[eq+1:])
		for _, w := range strings.Fields(part[:eq]) {
			out[w] = target
		}
	}
	return out
}

// commonOtherNames apply to every entity except products, when the entity
// has the field in the right column.
var commonOtherNames = names(`reference_num ref_num ref reference reference_number ref_number refnum=id;
	product_id product_key key project project_id workspace product_name=product;
	workflow_status state=status; title=name; created=created_at; updated=updated_at;
	assignee assigned_to_user assigned_user owner assigned=assigned_to`)

// productWords are the words that mean product on other entities (N03).
var productWords = []string{"product", "product_id", "product_key", "key", "project", "project_id", "workspace", "product_name"}

var catalog = []*EntityDef{
	{
		Name: "products", Singular: "product", OtherNames: []string{"project", "projects", "workspace", "workspaces"},
		Kind:          "product",
		About:         "Products (workspaces). The key of a product, for example {KEY}, starts the id of its records.",
		DefaultFields: []string{"key", "name"},
		OtherFieldNames: names(`id reference_prefix prefix reference_num ref_num ref product product_key product_id=key;
			parent_id=parent; title=name`),
		Fields: []FieldDef{
			fd("key", ftKey, "reference_prefix", wNo),
			fd("name", ftString, "name", wNo),
			falseAbsent(fd("product_line", ftBoolean, "product_line", wNo)),
			fd("parent", ftKey, "parent_id", wNo),
			fd("description", ftText, "description", wNo),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("url", ftString, "url", wNo),
		},
		CustomRecordTypes: []string{"Project"},
		Example:           "SELECT key, name FROM products",
		Fixed:             "products",
	},
	{
		Name: "releases", Singular: "release", OtherNames: []string{"versions"}, Kind: "release",
		About:           "Releases of a product. The parking lot is a release with parking_lot = true and no date.",
		DefaultFields:   []string{"id", "name", "release_date", "parking_lot"},
		OtherFieldNames: names(`parent parent_id=product; theme=description; date ship_date=release_date`),
		ReadRoutes:      []Route{{"id", "/releases/{v}"}, {"product", "/products/{v}/releases"}},
		ParentFilters:   []string{"product"},
		ParentKind:      "product", ParentField: "product",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		CreateRoute: "/products/{v}/releases", UpdateRoute: "/releases/{id}", DeleteRoute: "/products/{product}/releases/{id}",
		CustomRecordTypes: []string{"Release"},
		Example:           "SELECT id, name, release_date FROM releases WHERE product = '{KEY}'",
		Fields: []FieldDef{
			fd("id", ftString, "reference_num", wNo),
			fd("name", ftString, "name", wYes),
			fd("release_date", ftDate, "release_date", wYes),
			fd("start_date", ftDate, "start_date", wYes),
			fd("status", ftStatus, "workflow_status.name", wNo),
			falseAbsent(fd("parking_lot", ftBoolean, "parking_lot", wYes)),
			falseAbsent(fd("released", ftBoolean, "released", wNo)),
			fd("product", ftKey, "product_id", wNo),
			fd("progress", ftNumber, "progress", wNo),
			member(heavy(fd("description", ftText, "theme", wYes)), "theme"),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "features", Singular: "feature", Kind: "feature",
		About:         "Features. Each feature is in one release.",
		DefaultFields: []string{"id", "name", "status", "release"},
		OtherFieldNames: names(`release_id release_ref release_reference_num parent parent_id=release;
			initiative_id=initiative; epic_id=epic`),
		ReadRoutes: []Route{{"id", "/features/{v}"}, {"release", "/releases/{v}/features"}, {"epic", "/epics/{v}/features"},
			{"initiative", "/initiatives/{v}/features"}, {"product", "/products/{v}/features"}},
		AccountList: "/features", NameSearch: true,
		ParentFilters: []string{"product", "release", "initiative", "epic"},
		ParentKind:    "release", ParentField: "release",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		CreateRoute: "/releases/{v}/features", UpdateRoute: "/features/{id}", DeleteRoute: "/features/{id}",
		CustomRecordTypes: []string{"Feature"},
		Example:           "SELECT id, name, status FROM features WHERE release = '{KEY}-R-3'",
		Fields: []FieldDef{
			fd("id", ftString, "reference_num", wNo),
			fd("name", ftString, "name", wYes),
			member(fd("status", ftStatus, "workflow_status.name", wYes), "workflow_status"),
			ref("release", "release", "release_reference_num", wYes),
			ref("initiative", "initiative", "initiative_reference_num", wYes),
			ref("epic", "epic", "epic_reference_num", wNo),
			member(fd("assigned_to", ftUser, "assigned_to_user.email", wYes), "assigned_to_user"),
			fd("product", ftKey, "product_id", wNo),
			fd("start_date", ftDate, "start_date", wYes),
			fd("due_date", ftDate, "due_date", wYes),
			fd("tags", ftString, "tags", wYes),
			fd("score", ftNumber, "score", wNo),
			fd("progress", ftNumber, "progress", wNo),
			heavy(fd("description", ftText, "description", wYes)),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("updated_at", ftDatetime, "updated_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "epics", Singular: "epic", OtherNames: []string{"master_features"}, Kind: "epic",
		About:           "Epics. An epic is in one release and groups features.",
		DefaultFields:   []string{"id", "name", "status", "release"},
		OtherFieldNames: names(`release_id parent parent_id=release; initiative_id=initiative`),
		ReadRoutes: []Route{{"id", "/epics/{v}"}, {"release", "/releases/{v}/epics"}, {"initiative", "/initiatives/{v}/epics"},
			{"product", "/products/{v}/epics"}},
		AccountList: "/epics", NameSearch: true,
		ParentFilters: []string{"product", "release", "initiative"},
		ParentKind:    "release", ParentField: "release",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		CreateRoute: "/releases/{v}/epics", UpdateRoute: "/epics/{id}", DeleteRoute: "/epics/{id}",
		CustomRecordTypes: []string{"Epic", "MasterFeature"},
		Example:           "SELECT id, name, status FROM epics WHERE product = '{KEY}'",
		Fields: []FieldDef{
			fd("id", ftString, "reference_num", wNo),
			fd("name", ftString, "name", wYes),
			member(fd("status", ftStatus, "workflow_status.name", wYes), "workflow_status"),
			ref("release", "release", "release.reference_num", wNo),
			ref("initiative", "initiative", "initiative_reference_num", wYes),
			member(fd("assigned_to", ftUser, "assigned_to_user.email", wNo), "assigned_to_user"),
			fd("product", ftKey, "product_id", wNo),
			fd("start_date", ftDate, "start_date", wYes),
			fd("due_date", ftDate, "due_date", wYes),
			fd("progress", ftNumber, "progress", wYes),
			heavy(fd("description", ftText, "description", wYes)),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("updated_at", ftDatetime, "updated_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "requirements", Singular: "requirement", Kind: "requirement",
		About:           "Requirements of a feature.",
		DefaultFields:   []string{"id", "name", "status", "feature"},
		OtherFieldNames: names(`feature_id parent parent_id=feature`),
		ReadRoutes:      []Route{{"id", "/requirements/{v}"}, {"feature", "/features/{v}/requirements"}},
		ParentFilters:   []string{"feature"},
		ParentKind:      "feature", ParentField: "feature",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		CreateRoute: "/features/{v}/requirements", UpdateRoute: "/requirements/{id}", DeleteRoute: "/requirements/{id}",
		CustomRecordTypes: []string{"Requirement"},
		Example:           "SELECT id, name, status FROM requirements WHERE feature = '{KEY}-23'",
		Fields: []FieldDef{
			fd("id", ftString, "reference_num", wNo),
			fd("name", ftString, "name", wYes),
			member(fd("status", ftStatus, "workflow_status.name", wYes), "workflow_status"),
			ref("feature", "feature", "feature.reference_num", wNo),
			member(fd("assigned_to", ftUser, "assigned_to_user.email", wYes), "assigned_to_user"),
			fd("original_estimate", ftNumber, "original_estimate", wYes),
			fd("work_done", ftNumber, "work_done", wYes),
			heavy(fd("description", ftText, "description", wYes)),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "initiatives", Singular: "initiative", Kind: "initiative",
		About:           "Initiatives of a product. Features can be linked to an initiative.",
		DefaultFields:   []string{"id", "name", "status"},
		OtherFieldNames: names(`parent parent_id=product`),
		ReadRoutes:      []Route{{"id", "/initiatives/{v}"}, {"product", "/products/{v}/initiatives"}},
		AccountList:     "/initiatives", NameSearch: true,
		ParentFilters: []string{"product"},
		ParentKind:    "product", ParentField: "product",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		CreateRoute: "/products/{v}/initiatives", UpdateRoute: "/initiatives/{id}", DeleteRoute: "/products/{product}/initiatives/{id}",
		CustomRecordTypes: []string{"Initiative"},
		Example:           "SELECT id, name, status FROM initiatives WHERE product = '{KEY}'",
		Fields: []FieldDef{
			fd("id", ftString, "reference_num", wNo),
			fd("name", ftString, "name", wYes),
			member(fd("status", ftStatus, "workflow_status.name", wYes), "workflow_status"),
			fd("product", ftKey, "product_id", wNo),
			fd("start_date", ftDate, "start_date", wYes),
			fd("end_date", ftDate, "end_date", wYes),
			fd("progress", ftNumber, "progress", wNo),
			fd("value", ftNumber, "value", wYes),
			fd("effort", ftNumber, "effort", wYes),
			fd("color", ftString, "color", wYes),
			fd("presented", ftBoolean, "presented", wYes),
			heavy(fd("description", ftText, "description", wYes)),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("updated_at", ftDatetime, "updated_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "goals", Singular: "goal", OtherNames: []string{"strategic_imperatives"}, Kind: "goal",
		About:           "Goals of a product. A new goal needs a success metric.",
		DefaultFields:   []string{"id", "name"},
		OtherFieldNames: names(`parent parent_id=product`),
		ReadRoutes:      []Route{{"id", "/goals/{v}"}, {"product", "/products/{v}/goals"}},
		AccountList:     "/goals", NameSearch: true,
		ParentFilters: []string{"product"},
		ParentKind:    "product", ParentField: "product",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		CreateRoute: "/products/{v}/goals", UpdateRoute: "/goals/{id}", DeleteRoute: "/products/{product}/goals/{id}",
		CustomRecordTypes: []string{"StrategicImperative", "Goal"},
		Example:           "SELECT id, name FROM goals WHERE product = '{KEY}'",
		Fields: []FieldDef{
			fd("id", ftString, "reference_num", wNo),
			fd("name", ftString, "name", wYes),
			member(fd("success_metric", ftString, "success_metric.name", wInsertOnly), "success_metric"),
			member(fd("status", ftStatus, "success_metric.workflow_status.name", wYes), "workflow_status"),
			fd("product", ftKey, "product_id", wNo),
			fd("progress", ftNumber, "progress", wYes),
			fd("value", ftNumber, "value", wNo),
			fd("effort", ftNumber, "effort", wNo),
			heavy(fd("description", ftText, "description", wYes)),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("updated_at", ftDatetime, "updated_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "ideas", Singular: "idea", Kind: "idea",
		About:           "Ideas submitted for a product. feature is the feature an idea was promoted to.",
		DefaultFields:   []string{"id", "name", "status", "votes"},
		OtherFieldNames: names(`parent parent_id=product; votes_count=votes; feature_id=feature`),
		ReadRoutes:      []Route{{"id", "/ideas/{v}"}, {"product", "/products/{v}/ideas"}},
		AccountList:     "/ideas", NameSearch: true,
		ParentFilters: []string{"product"},
		ParentKind:    "product", ParentField: "product",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		CreateRoute: "/products/{v}/ideas", UpdateRoute: "/ideas/{id}", DeleteRoute: "/ideas/{id}",
		CustomRecordTypes: []string{"Idea"},
		Example:           "SELECT id, name, votes FROM ideas WHERE product = '{KEY}' ORDER BY votes DESC LIMIT 5",
		Fields: []FieldDef{
			fd("id", ftString, "reference_num", wNo),
			fd("name", ftString, "name", wYes),
			member(fd("status", ftStatus, "workflow_status.name", wYes), "workflow_status"),
			fd("votes", ftNumber, "votes", wNo),
			fd("product", ftKey, "product_id", wNo),
			ref("feature", "feature", "feature.reference_num", wNo),
			fd("visibility", ftString, "visibility", wYes),
			heavy(fd("description", ftText, "description", wYes)),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("updated_at", ftDatetime, "updated_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "comments", Singular: "comment", Kind: "comment",
		About:         "Comments on a record. parent is the id of the commented record.",
		DefaultFields: []string{"id", "body", "user", "parent", "created_at"},
		OtherFieldNames: names(`content text comment message description=body; author created_by author_name user_name=user;
			feature feature_id epic epic_id idea idea_id release release_id initiative initiative_id goal goal_id
			requirement requirement_id record record_id parent_id commentable product product_id product_key=parent`),
		ReadRoutes:    []Route{{"id", "/comments/{v}"}, {"parent", ""}},
		ParentFilters: []string{"parent"},
		ParentKind:    "any", ParentField: "parent",
		CanCreate: true, CanUpdate: true, CanDelete: true,
		UpdateRoute: "/comments/{id}", DeleteRoute: "/comments/{id}",
		Example: "SELECT body, user, created_at FROM comments WHERE parent = '{KEY}-23'",
		Fields: []FieldDef{
			fd("id", ftString, "id", wNo),
			fd("body", ftText, "body", wYes),
			{Name: "user", Type: ftUser, Source: "user.name", UserAsName: true},
			fd("parent", ftString, "commentable.url", wNo),
			fd("created_at", ftDatetime, "created_at", wNo),
			fd("url", ftString, "url", wNo),
		},
	},
	{
		Name: "users", Singular: "user", Kind: "user",
		About:           "Users of the account. For the user you are working as, use: SELECT name, email FROM me",
		DefaultFields:   []string{"id", "name", "email"},
		OtherFieldNames: names(`mail e_mail=email; user_id=id`),
		Example:         "SELECT name, email FROM users",
		Fixed:           "users",
		Fields: []FieldDef{
			fd("id", ftString, "id", wNo),
			fd("name", ftString, "name", wNo),
			fd("email", ftString, "email", wNo),
			fd("created_at", ftDatetime, "created_at", wNo),
		},
	},
	{
		Name: "statuses", Singular: "status", OtherNames: []string{"workflow_statuses", "workflows", "workflow"},
		About:           "The workflow statuses a record can have, per product and entity.",
		DefaultFields:   []string{"name", "entity"},
		OtherFieldNames: names(`status id title=name; type entity_type kind=entity`),
		ParentFilters:   []string{"product", "entity"},
		Example:         "SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features'",
		Fixed:           "statuses",
		Fields: []FieldDef{
			fd("name", ftString, "name", wNo),
			fd("entity", ftString, "entity", wNo),
			fd("position", ftNumber, "position", wNo),
			falseAbsent(fd("complete", ftBoolean, "complete", wNo)),
			{Name: "product", Type: ftKey, NoSource: true},
		},
	},
	{
		Name: "me", Singular: "me", OtherNames: []string{"current_user", "myself", "whoami"},
		About:           "The user you are working as. One record.",
		DefaultFields:   []string{"name", "email"},
		OtherFieldNames: names(`mail=email; user=name; user_id=id`),
		Example:         "SELECT name, email FROM me",
		Fixed:           "me",
		Fields: []FieldDef{
			fd("id", ftString, "id", wNo),
			fd("name", ftString, "name", wNo),
			fd("email", ftString, "email", wNo),
		},
	},
}

// statusWorkflowMembers: the member of a product record that names the
// workflow of a record kind, and the value of the field entity.
var statusWorkflowMembers = []struct{ Member, Entity string }{
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

func entityOfKind(kind string) *EntityDef {
	for _, e := range catalog {
		if e.Kind == kind {
			return e
		}
	}
	return nil
}

func entityNames() []string {
	var out []string
	for _, e := range catalog {
		out = append(out, e.Name)
	}
	return out
}

// commentRoute is the comment list of a record of the given kind.
func commentRoute(kind string) string {
	switch kind {
	case "feature", "release", "epic", "idea", "initiative", "goal", "requirement":
		return "/" + entityOfKind(kind).Name + "/{v}/comments"
	case "product":
		return "/products/{v}/comments"
	}
	return ""
}

var commentableKinds = map[string]bool{"feature": true, "epic": true, "idea": true, "release": true, "initiative": true, "goal": true, "requirement": true}
