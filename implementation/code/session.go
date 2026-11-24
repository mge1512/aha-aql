// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// The Account of a call, the state of one statement (run), and the
// behaviours that several statements share: resolve-product,
// custom-fields, choose-example-key and the reading of workflows.

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Mode string

const (
	modeStdio  Mode = "stdio"
	modeHTTP   Mode = "http"
	modeManual Mode = "manual"
)

const (
	defaultLimit = 50
	maxLimit     = 200
	maxPages     = 15
	pageSize     = 200
	maxInValues  = 20
	maxListed    = 30
)

// keyMark stands for the ExampleKey in a text until the text is shown.
const keyMark = "\uE000KEY\uE000"

func withKey(s string) string { return strings.ReplaceAll(s, "{KEY}", keyMark) }

type Account struct {
	Subdomain  string
	Credential string
}

type Settings struct {
	Subdomain      string
	APIKeyFile     string
	DefaultProduct string
	ReadOnly       bool
	Listen         string
	DryRun         bool
}

type AqlError struct {
	Kind    string
	Message string
}

func (e *AqlError) Error() string { return e.Message }

func aqlErr(kind, format string, args ...any) *AqlError {
	return &AqlError{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

// session is the process in stdio and manual mode, and one HTTP request in
// http mode. It has the Account and the ExampleKey.
type session struct {
	mode     Mode
	settings *Settings
	acct     Account
	keyMu    sync.Mutex
	key      string
	chosen   bool
}

// run is the state of one statement or one tool list. Nothing of it is kept
// after the call.
type run struct {
	s            *session
	products     []Product
	productRecs  []map[string]any
	productsErr  *AqlError
	productsRead bool
	defs         *defsResult
	defsErr      *AqlError
	me           map[string]any
	meErr        *AqlError
	options      map[string][]string
	productRec   map[string]map[string]any
	workflows    map[string][]any
	custom       map[string]*CustomField
}

func newRun(s *session) *run {
	return &run{s: s, options: map[string][]string{}, productRec: map[string]map[string]any{},
		workflows: map[string][]any{}, custom: map[string]*CustomField{}}
}

func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// ---- record ids ----

var (
	reRefNum    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*-([A-Za-z]-)?[0-9]+(-[0-9]+)?$`)
	reNumericID = regexp.MustCompile(`^[0-9]{6,}$`)
	reProdKey   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	reSubdomain = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	reEmail     = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	reDate      = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

// credential pattern ^[!-~]{1,4096}$ (Go's regexp limits repeat counts to 1000)
type credPattern struct{}

func (credPattern) MatchString(s string) bool {
	if len(s) < 1 || len(s) > 4096 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '!' || s[i] > '~' {
			return false
		}
	}
	return true
}

var reCred credPattern

func isRefNum(s string) bool    { return reRefNum.MatchString(s) }
func isNumericID(s string) bool { return reNumericID.MatchString(s) }

// refKind gives the record kind of a ReferenceNumber, or "".
func refKind(s string) string {
	if !isRefNum(s) {
		return ""
	}
	parts := strings.Split(s, "-")
	switch len(parts) {
	case 2:
		return "feature"
	case 3:
		if _, err := strconv.Atoi(parts[1]); err == nil {
			return "requirement"
		}
		switch strings.ToUpper(parts[1]) {
		case "R":
			return "release"
		case "E":
			return "epic"
		case "S":
			return "initiative"
		case "G":
			return "goal"
		case "I":
			return "idea"
		}
	}
	return ""
}

// ---- comparison form of evaluate-condition ----

var typographic = strings.NewReplacer("\u2018", "'", "\u2019", "'", "\u201b", "'", "\u02bc", "'", "\u2032", "'",
	"\u201c", `"`, "\u201d", `"`, "\u201e", `"`, "\u201f", `"`, "\u00a0", " ")

func cform(s string) string {
	return strings.ToLower(strings.TrimSpace(typographic.Replace(s)))
}

func textEq(a, b string) bool { return cform(a) == cform(b) }

// ---- failures ----

func (r *run) reason(f *ApiFailure) string {
	host := r.s.acct.Subdomain + ".aha.io"
	switch f.Cause {
	case "no_answer":
		return "no answer within 60 seconds"
	case "certificate":
		return "the certificate of " + host + " could not be verified"
	case "proxy":
		return "the proxy did not open a connection to " + host
	}
	return "the connection to " + host + " failed"
}

func answerText(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return "(no text)"
	}
	rs := []rune(t)
	if len(rs) > 300 {
		rs = rs[:300]
	}
	return string(rs)
}

// failErr turns a failure into the error of plan-fetch step 10 and of the
// other behaviours: X04, X01, X02 and X03 (W30 for a write request).
func (r *run) failErr(f *ApiFailure, method, route string) *AqlError {
	switch f.Kind {
	case failNotAuthorized:
		return aqlErr("not_authorized", "Aha! refused the request (status %d): the API key is not valid for the account %s, or it has no permission for this record. Repeating the statement does not help.", f.Status, r.s.acct.Subdomain)
	case failRateLimited:
		return aqlErr("rate_limited", "aha: API error (status 429): too many requests")
	case failUnreachable:
		hint := ""
		if method != http.MethodGet && f.Cause != "certificate" && f.Cause != "proxy" {
			hint = " The change may have been made. Read the record before you send the statement again."
		}
		return aqlErr("network", "cannot reach Aha!: %s.%s", r.reason(f), hint)
	}
	status := f.Status
	return aqlErr("api_error", "aha: API error (status %d) on %s %s: %s", status, method, route, answerText(f.Text))
}

// ---- API operations with the Account of the run ----

func (r *run) list(route string, params url.Values) (*ListPage, *ApiFailure) {
	return apiList(r.s.acct, route, params)
}

func (r *run) get(route, member string) (map[string]any, *ApiFailure) {
	ans, f := apiGet(r.s.acct, route, member, false)
	if f != nil {
		return nil, f
	}
	return ans[member].(map[string]any), nil
}

func (r *run) getList(route, member string) ([]any, *ApiFailure) {
	ans, f := apiGet(r.s.acct, route, member, true)
	if f != nil {
		return nil, f
	}
	return ans[member].([]any), nil
}

// readAll reads a list to its last page.
func (r *run) readAll(route string, fields []string) ([]map[string]any, *ApiFailure) {
	var out []map[string]any
	for page := 1; ; page++ {
		p := url.Values{}
		p.Set("page", strconv.Itoa(page))
		p.Set("per_page", strconv.Itoa(pageSize))
		if len(fields) > 0 {
			p.Set("fields", strings.Join(fields, ","))
		}
		lp, f := r.list(route, p)
		if f != nil {
			return nil, f
		}
		out = append(out, lp.Records...)
		if page >= lp.TotalPages || len(lp.Records) == 0 {
			return out, nil
		}
	}
}

// ---- values in API records ----

// source takes the value of a.b.c from an API record.
func source(rec map[string]any, path string) any {
	var cur any = rec
	for _, p := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[p]
		if !ok {
			return nil
		}
	}
	return cur
}

func str(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	}
	return textOf(v)
}

// ---- resolve-product ----

type Product struct {
	Key  string
	ID   string
	Name string
}

func listFields(e *EntityDef, used map[string]bool, custom bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range e.Fields {
		if f.NoSource || (f.Heavy && !used[f.Name]) {
			continue
		}
		top := f.Source
		if i := strings.Index(top, "."); i >= 0 {
			top = top[:i]
		}
		if !seen[top] {
			seen[top] = true
			out = append(out, top)
		}
	}
	if custom {
		out = append(out, "custom_fields")
	}
	return out
}

func (r *run) productList() ([]Product, *AqlError) {
	if r.productsRead {
		return r.products, r.productsErr
	}
	r.productsRead = true
	recs, f := r.readAll("/products", listFields(entityByName("products"), map[string]bool{}, false))
	if f != nil {
		r.productsErr = r.failErr(f, "GET", "/products")
		return nil, r.productsErr
	}
	r.productRecs = recs
	for _, rec := range recs {
		r.products = append(r.products, Product{Key: str(rec["reference_prefix"]), ID: str(rec["id"]), Name: str(rec["name"])})
	}
	return r.products, nil
}

// findProduct compares the value with the key and the NumericId of every
// product, and then with the names.
func (r *run) findProduct(v any) (*Product, *AqlError) {
	ps, err := r.productList()
	if err != nil {
		return nil, err
	}
	s := textOf(v)
	for i := range ps {
		if textEq(ps[i].Key, s) || textEq(ps[i].ID, s) {
			return &ps[i], nil
		}
	}
	for i := range ps {
		if textEq(ps[i].Name, s) {
			return &ps[i], nil
		}
	}
	return nil, nil
}

func (r *run) productByID(id string) *Product {
	ps, _ := r.productList()
	for i := range ps {
		if ps[i].ID == id {
			return &ps[i]
		}
	}
	return nil
}

func (r *run) resolveProduct(v any, e *EntityDef, hints bool) (*Product, *AqlError) {
	p, err := r.findProduct(v)
	if err != nil {
		return nil, err
	}
	if p != nil {
		return p, nil
	}
	s := textOf(v)
	if hints {
		k := refKind(s)
		up := strings.ToUpper(s)
		switch {
		case k != "" && e.HasParentFilter(k):
			return nil, aqlErr("wrong_kind", "%s is a %s, not a product. Use: WHERE %s = '%s'", quoteVal(s), k, k, up)
		case k != "" && k == e.Kind:
			return nil, aqlErr("wrong_kind", "%s is the id of a %s, not a product. Use: WHERE id = '%s'", quoteVal(s), k, up)
		case k != "":
			if pp := r.productWithKey(strings.SplitN(s, "-", 2)[0]); pp != nil {
				return nil, aqlErr("wrong_kind", "%s is a %s, not a product. Its product is %s: WHERE product = '%s'", quoteVal(s), k, pp.Key, pp.Key)
			}
		}
	}
	return nil, r.errP01(s)
}

func (r *run) productWithKey(k string) *Product {
	ps, _ := r.productList()
	for i := range ps {
		if strings.EqualFold(ps[i].Key, k) {
			return &ps[i]
		}
	}
	return nil
}

func (r *run) errP01(s string) *AqlError {
	ps, _ := r.productList()
	var keys []string
	for _, p := range ps {
		keys = append(keys, p.Key)
	}
	sort.Strings(keys)
	return aqlErr("bad_product", "unknown product %s. Product keys: %s. To see their names: SELECT key, name FROM products", quoteVal(s), listMax(keys))
}

func listMax(items []string) string {
	if len(items) <= maxListed {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:maxListed], ", ") + fmt.Sprintf(", and %d more", len(items)-maxListed)
}

// ---- choose-example-key ----

func (s *session) exampleKey(r *run) string {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	if s.chosen {
		return s.key
	}
	s.chosen = true
	if r == nil || r.s != s {
		r = newRun(s)
	}
	ps, err := r.productList()
	if err != nil {
		warn("the product list could not be read; the examples use the key PROD")
		s.key = "PROD"
		return s.key
	}
	if dp := s.settings.DefaultProduct; dp != "" {
		if p, _ := r.findProduct(dp); p != nil {
			s.key = p.Key
			return s.key
		}
		warn("default-product does not name a product of the account; the examples use another key")
	}
	key := "PROD"
	for {
		taken := false
		for _, p := range ps {
			if strings.EqualFold(p.Key, key) {
				taken = true
			}
		}
		if !taken {
			break
		}
		key += "X"
	}
	s.key = key
	return s.key
}

// show replaces the mark of the ExampleKey; the key is chosen only when a
// text shows it.
func (r *run) show(text string) string {
	if !strings.Contains(text, keyMark) {
		return text
	}
	return strings.ReplaceAll(text, keyMark, r.s.exampleKey(r))
}

// ---- custom-fields ----

type CustomField struct {
	Name       string // custom.<key in lower case>
	Key        string
	Label      string
	Definition string
	Kind       string
	TypeName   string
}

type rawDef struct {
	ID, Key, Name, TypeName, RecordType string
}

type defsResult struct {
	defs       []rawDef
	unreadable bool
	status     int
}

var customKinds = map[string]string{
	"textfield": "text", "urlfield": "url", "notefield": "note", "numberfield": "number", "datefield": "date",
	"selecteditable": "choice", "selectconstant": "fixed_choice", "selectmultipleeditable": "tags",
	"selectmultipleconstant": "fixed_tags", "usersfield": "users", "scorecardfield": "scorecard",
}

var customKindType = map[string]FieldType{
	"text": ftString, "url": ftString, "note": ftText, "number": ftNumber, "date": ftDate, "choice": ftString,
	"fixed_choice": ftString, "tags": ftString, "fixed_tags": ftString, "users": ftUser, "scorecard": ftNumber,
}

var customKindText = map[string]string{
	"text": "string", "url": "string (web address)", "note": "text", "number": "number", "date": "date (YYYY-MM-DD)",
	"choice": "string", "fixed_choice": "one option of a list", "tags": "tags, separated by commas",
	"fixed_tags": "options of a list, separated by commas", "users": "users (e-mail addresses), separated by commas", "scorecard": "number",
}

func afterColons(s string) string {
	if i := strings.LastIndex(s, "::"); i >= 0 {
		return s[i+2:]
	}
	return s
}

func (r *run) definitions() (*defsResult, *AqlError) {
	if r.defs != nil || r.defsErr != nil {
		return r.defs, r.defsErr
	}
	var all []rawDef
	for page := 1; ; page++ {
		p := url.Values{}
		p.Set("page", strconv.Itoa(page))
		p.Set("per_page", strconv.Itoa(pageSize))
		lp, f := r.list("/custom_field_definitions", p)
		if f != nil {
			if f.Kind == failNotAuthorized || f.Kind == failNotFound {
				r.defs = &defsResult{unreadable: true, status: f.Status}
				return r.defs, nil
			}
			r.defsErr = r.failErr(f, "GET", "/custom_field_definitions")
			return nil, r.defsErr
		}
		for _, d := range lp.Records {
			all = append(all, rawDef{ID: str(d["id"]), Key: str(d["key"]), Name: str(d["name"]),
				TypeName: afterColons(str(d["type"])), RecordType: afterColons(str(d["custom_fieldable_type"]))})
		}
		if page >= lp.TotalPages || len(lp.Records) == 0 {
			break
		}
	}
	r.defs = &defsResult{defs: all}
	return r.defs, nil
}

// entityDefs gives the definitions of the entity, the first one per key.
func entityDefs(e *EntityDef, d *defsResult) []rawDef {
	var out []rawDef
	seen := map[string]bool{}
	for _, def := range d.defs {
		ok := false
		for _, rt := range e.CustomRecordTypes {
			if strings.EqualFold(rt, def.RecordType) {
				ok = true
			}
		}
		if !ok || seen[strings.ToLower(def.Key)] {
			continue
		}
		seen[strings.ToLower(def.Key)] = true
		out = append(out, def)
	}
	return out
}

func toCustom(def rawDef) *CustomField {
	kind, ok := customKinds[strings.ToLower(def.TypeName)]
	if !ok {
		return nil
	}
	return &CustomField{Name: "custom." + strings.ToLower(def.Key), Key: def.Key, Label: def.Name, Definition: def.ID, Kind: kind, TypeName: def.TypeName}
}

// customFields is custom-fields without a key.
func (r *run) customFields(e *EntityDef) ([]*CustomField, *AqlError) {
	if len(e.CustomRecordTypes) == 0 {
		return nil, nil
	}
	d, err := r.definitions()
	if err != nil {
		return nil, err
	}
	if d.unreadable {
		return nil, nil
	}
	var out []*CustomField
	for _, def := range entityDefs(e, d) {
		if cf := toCustom(def); cf != nil {
			out = append(out, cf)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (r *run) customNames(e *EntityDef, settable bool) []string {
	cfs, _ := r.customFields(e)
	var out []string
	for _, c := range cfs {
		if settable && c.Kind == "scorecard" {
			continue
		}
		out = append(out, c.Name)
	}
	return out
}

func (r *run) customHint(e *EntityDef) string {
	l := r.customNames(e, false)
	if len(l) == 0 {
		return ""
	}
	return fmt.Sprintf(" Custom fields of %s: %s.", e.Name, listMax(l))
}

// customField is custom-fields with a key. nil without an error means none.
func (r *run) customField(e *EntityDef, key, word string, explicit bool) (*CustomField, *AqlError) {
	if len(e.CustomRecordTypes) == 0 {
		if explicit {
			return nil, r.errN06(e)
		}
		return nil, nil
	}
	d, err := r.definitions()
	if err != nil {
		return nil, err
	}
	if d.unreadable {
		if explicit {
			return nil, aqlErr("bad_name", "the custom fields of the account cannot be read with this API key (status %d). Fields of %s: %s. Example: %s",
				d.status, e.Name, fieldList(e), withKey(e.Example))
		}
		return nil, nil
	}
	defs := entityDefs(e, d)
	for _, def := range defs {
		if strings.EqualFold(def.Key, key) {
			cf := toCustom(def)
			if cf == nil {
				return nil, aqlErr("bad_name", "custom.%s has the type %s, and the server cannot read or set custom fields of that type.%s",
					strings.ToLower(def.Key), def.TypeName, r.customHint(e))
			}
			r.custom[cf.Name] = cf
			return cf, nil
		}
	}
	if !explicit {
		return nil, nil
	}
	cfs, _ := r.customFields(e)
	if len(cfs) > 0 {
		return nil, aqlErr("bad_name", "unknown custom field '%s' on %s. Custom fields of %s: %s. To see their types: DESCRIBE %s",
			word, e.Name, e.Name, listMax(r.customNames(e, false)), e.Name)
	}
	return nil, r.errN06(e)
}

func (r *run) errN06(e *EntityDef) *AqlError {
	return aqlErr("bad_name", "%s have no custom fields that can be used here. Fields of %s: %s. Example: %s", e.Name, e.Name, fieldList(e), withKey(e.Example))
}

func fieldList(e *EntityDef) string {
	var n []string
	for _, f := range e.Fields {
		n = append(n, f.Name)
	}
	return strings.Join(n, ", ")
}

func (r *run) optionsOf(cf *CustomField) ([]string, *AqlError) {
	if o, ok := r.options[cf.Definition]; ok {
		return o, nil
	}
	route := routeWith("/custom_field_definitions/{v}/options", "v", cf.Definition)
	l, f := r.getList(route, "options")
	if f != nil {
		return nil, r.failErr(f, "GET", route)
	}
	var out []string
	for _, x := range l {
		if m, ok := x.(map[string]any); ok {
			out = append(out, str(m["text"]))
		}
	}
	r.options[cf.Definition] = out
	return out, nil
}

// ---- lookups ----

func (r *run) currentUser() (map[string]any, *AqlError) {
	if r.me != nil || r.meErr != nil {
		return r.me, r.meErr
	}
	m, f := r.get("/me", "user")
	if f != nil {
		r.meErr = r.failErr(f, "GET", "/me")
		return nil, r.meErr
	}
	r.me = m
	return m, nil
}

// productRecord and productWorkflows read the two answers of the entity
// statuses for one product.
func (r *run) productRecord(id string) (map[string]any, *ApiFailure, string) {
	route := routeWith("/products/{v}", "v", id)
	if m, ok := r.productRec[id]; ok {
		return m, nil, route
	}
	m, f := r.get(route, "product")
	if f != nil {
		return nil, f, route
	}
	r.productRec[id] = m
	return m, nil, route
}

func (r *run) productWorkflows(id string) ([]any, *ApiFailure, string) {
	route := routeWith("/products/{v}/workflows", "v", id)
	if w, ok := r.workflows[id]; ok {
		return w, nil, route
	}
	w, f := r.getList(route, "workflows")
	if f != nil {
		return nil, f, route
	}
	r.workflows[id] = w
	return w, nil, route
}

type statusRec struct {
	Name     string
	Entity   string
	Prefix   string
	Position any
	Complete any
}

// statusRecords gives the records of the entity statuses for one product.
func (r *run) statusRecords(id string) ([]statusRec, *AqlError) {
	prod, f, route := r.productRecord(id)
	if f != nil {
		return nil, r.failErr(f, "GET", route)
	}
	wfs, f, route := r.productWorkflows(id)
	if f != nil {
		return nil, r.failErr(f, "GET", route)
	}
	type member struct{ name, entity, prefix string }
	var members []member
	known := map[string]bool{}
	for _, m := range statusWorkflowMembers {
		known[m.Member] = true
		members = append(members, member{m.Member, m.Entity, strings.TrimSuffix(m.Member, "_workflow")})
	}
	var others []string
	for k := range prod {
		if strings.HasSuffix(k, "_workflow") && !known[k] {
			others = append(others, k)
		}
	}
	sort.Strings(others)
	for _, k := range others {
		p := strings.TrimSuffix(k, "_workflow")
		members = append(members, member{k, p, p})
	}
	var out []statusRec
	for _, m := range members {
		obj, ok := prod[m.name].(map[string]any)
		if !ok {
			continue
		}
		wid := str(obj["id"])
		for _, w := range wfs {
			wm, ok := w.(map[string]any)
			if !ok || str(wm["id"]) != wid || wid == "" {
				continue
			}
			sts, _ := wm["workflow_statuses"].([]any)
			for _, st := range sts {
				sm, ok := st.(map[string]any)
				if !ok {
					continue
				}
				out = append(out, statusRec{Name: str(sm["name"]), Entity: m.entity, Prefix: m.prefix, Position: sm["position"], Complete: sm["complete"]})
			}
			break
		}
	}
	return out, nil
}
