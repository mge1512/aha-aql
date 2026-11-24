// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// insert-statement, update-statement, delete-statement,
// check-write-values and write-request.

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type checked struct {
	Field  string
	Value  any // string, float64, bool or []string (a TextList)
	Custom *CustomField
}

// writeOutcome is what the log line of execute needs.
type writeOutcome struct {
	op, entity, id string
}

func display(v any) any {
	if l, ok := v.([]string); ok {
		return strings.Join(l, ", ")
	}
	return v
}

func splitList(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isDay(s string) bool {
	if !reDate.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// productKeyOf gives the key of a product by its NumericId, for messages.
func (r *run) productKeyOf(id string) string {
	if p := r.productByID(id); p != nil {
		return p.Key
	}
	return ""
}

func (r *run) w25Product(recordProduct, value string) string {
	if recordProduct != "" {
		if k := r.productKeyOf(recordProduct); k != "" {
			return k
		}
	}
	if i := strings.Index(value, "-"); i > 0 {
		if p := r.productWithKey(value[:i]); p != nil {
			return p.Key
		}
	}
	return keyMark
}

func (r *run) errW25(kind, value, recordProduct string) *AqlError {
	ke := entityOfKind(kind)
	name := kind + "s"
	if ke != nil {
		name = ke.Name
	}
	return aqlErr("not_found", "there is no %s with the id %s. To find the id, use the query tool: SELECT id, name FROM %s WHERE product = '%s'",
		kind, quoteVal(value), name, r.w25Product(recordProduct, value))
}

// checkValues is check-write-values.
func (r *run) checkValues(e *EntityDef, fields []Assign, productID string, parentKind, parent string, formOnly bool) ([]checked, *AqlError) {
	var out []checked
	for _, a := range fields {
		if cf := r.custom[a.Field]; cf != nil && strings.HasPrefix(a.Field, "custom.") {
			v, err := r.checkCustom(cf, a.Value)
			if err != nil {
				return nil, err
			}
			out = append(out, checked{Field: a.Field, Value: v, Custom: cf})
			continue
		}
		f := e.Field(a.Field)
		var v any
		var err *AqlError
		switch {
		case f.Name == "tags" && e.Name == "features":
			s, ok := a.Value.(string)
			if !ok {
				return nil, aqlErr("bad_value", "%s must be text in single quotes", f.Name)
			}
			v = splitList(s)
		case f.Type == ftStatus:
			if s, ok := a.Value.(string); !ok {
				return nil, aqlErr("bad_value", "%s must be text in single quotes", f.Name)
			} else if formOnly {
				v = s
			} else {
				v, err = r.checkStatus(e, s, productID, parentKind, parent)
			}
		default:
			v, err = r.checkByType(f.Name, f.Type, f.RefKind, a.Value, productID)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, checked{Field: f.Name, Value: v})
	}
	return out, nil
}

func (r *run) checkByType(name string, t FieldType, refKind_ string, val any, productID string) (any, *AqlError) {
	switch t {
	case ftString, ftText:
		s, ok := val.(string)
		if !ok {
			return nil, aqlErr("bad_value", "%s must be text in single quotes", name)
		}
		return s, nil
	case ftNumber:
		f, ok := numberOf(val)
		if _, isBool := val.(bool); !ok || isBool {
			return nil, aqlErr("bad_value", "%s must be a number", name)
		}
		return f, nil
	case ftBoolean:
		switch x := val.(type) {
		case bool:
			return x, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(x)) {
			case "true":
				return true, nil
			case "false":
				return false, nil
			}
		}
		return nil, aqlErr("bad_value", "%s must be true or false", name)
	case ftDate:
		s, _ := val.(string)
		if len(s) > 10 && isDay(s[:10]) && (s[10] == 'T' || s[10] == ' ') {
			return s[:10], nil
		}
		if isDay(s) {
			return s, nil
		}
		return nil, aqlErr("bad_value", "%s: invalid date %s, use YYYY-MM-DD", name, quoteVal(val))
	case ftUser:
		s, ok := val.(string)
		if ok && isMe(s) {
			me, err := r.currentUser()
			if err != nil {
				return nil, err
			}
			return str(me["email"]), nil
		}
		if ok && (reEmail.MatchString(strings.TrimSpace(s)) || isNumericID(strings.TrimSpace(s))) {
			return strings.TrimSpace(s), nil
		}
		return nil, aqlErr("bad_value", "%s: there is no user %s. Use an e-mail address, or 'me' for yourself", name, quoteVal(val))
	case ftReference:
		s := textOf(val)
		if refKind(s) == refKind_ || isNumericID(s) {
			return strings.ToUpper(s), nil
		}
		p, err := r.findProduct(val)
		if err != nil {
			return nil, err
		}
		if p != nil {
			recs, err := r.recordsOfKind(refKind_, p)
			if err != nil {
				return nil, err
			}
			return nil, aqlErr("wrong_kind", "%s is a product, not a %s. Use the id of one of its %s: %s", quoteVal(s), refKind_, entityOfKind(refKind_).Name, recs)
		}
		return nil, r.errW25(refKind_, s, productID)
	}
	return val, nil
}

func (r *run) statusNames(e *EntityDef, productID string) ([]string, *AqlError) {
	srs, err := r.statusRecords(productID)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, sr := range srs {
		if sr.Entity == e.Name {
			names = append(names, sr.Name)
		}
	}
	return names, nil
}

func (r *run) checkStatus(e *EntityDef, value, productID, parentKind, parent string) (any, *AqlError) {
	if productID == "" && parentKind != "" && parentKind != "product" {
		pe := entityOfKind(parentKind)
		route := routeWith(pe.RouteFor("id"), "v", parent)
		rec, f := r.get(route, pe.Singular)
		if f != nil {
			if f.Kind == failNotFound {
				return nil, r.errW25(parentKind, parent, "")
			}
			return nil, r.failErr(f, "GET", route)
		}
		productID = str(rec["product_id"])
	}
	names, err := r.statusNames(e, productID)
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		if textEq(n, value) {
			return n, nil
		}
	}
	l := "(none)"
	if len(names) > 0 {
		l = strings.Join(names, ", ")
	}
	return nil, aqlErr("bad_value", "workflow status %s not found for %s; valid statuses: %s", quoteVal(value), e.Singular, l)
}

func (r *run) matchOption(cf *CustomField, v string) (string, bool, *AqlError) {
	opts, err := r.optionsOf(cf)
	if err != nil {
		return "", false, err
	}
	for _, o := range opts {
		if textEq(o, v) {
			return o, true, nil
		}
	}
	return "", false, nil
}

func (r *run) errW32(cf *CustomField, v string) *AqlError {
	opts, _ := r.optionsOf(cf)
	return aqlErr("bad_value", "%s: there is no option %s. Options: %s", cf.Name, quoteVal(v), listMax(opts))
}

func (r *run) checkCustom(cf *CustomField, val any) (any, *AqlError) {
	text := func() (string, *AqlError) {
		s, ok := val.(string)
		if !ok {
			return "", aqlErr("bad_value", "%s must be text in single quotes", cf.Name)
		}
		return s, nil
	}
	switch cf.Kind {
	case "text", "url", "note", "choice":
		return text()
	case "number":
		return r.checkByType(cf.Name, ftNumber, "", val, "")
	case "date":
		return r.checkByType(cf.Name, ftDate, "", val, "")
	case "fixed_choice":
		s, err := text()
		if err != nil || s == "" {
			return s, err
		}
		o, ok, err := r.matchOption(cf, s)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, r.errW32(cf, s)
		}
		return o, nil
	case "tags", "fixed_tags":
		s, err := text()
		if err != nil {
			return nil, err
		}
		tags := splitList(s)
		if cf.Kind == "fixed_tags" {
			for i, t := range tags {
				o, ok, err := r.matchOption(cf, t)
				if err != nil {
					return nil, err
				}
				if !ok {
					return nil, r.errW32(cf, t)
				}
				tags[i] = o
			}
		}
		return tags, nil
	case "users":
		s, err := text()
		if err != nil {
			return nil, err
		}
		parts := splitList(s)
		for i, p := range parts {
			switch {
			case isMe(p):
				me, err := r.currentUser()
				if err != nil {
					return nil, err
				}
				parts[i] = str(me["id"])
			case reEmail.MatchString(p) || isNumericID(p):
			default:
				return nil, aqlErr("bad_value", "%s: there is no user %s. Use an e-mail address, or 'me' for yourself", cf.Name, quoteVal(p))
			}
		}
		return parts, nil
	}
	return nil, aqlErr("bad_name", "the field %s cannot be set", cf.Name)
}

// writeBody is write-request.
func writeBody(e *EntityDef, fields []checked, insert bool) []byte {
	var inner Obj
	var custom Obj
	customAt := -1
	var goalStatus any
	for _, c := range fields {
		if c.Custom != nil {
			if customAt < 0 {
				customAt = len(inner)
				inner = append(inner, KV{"custom_fields", nil})
			}
			v := c.Value
			if l, ok := v.([]string); ok {
				v = l
			}
			custom = append(custom, KV{c.Custom.Key, v})
			continue
		}
		f := e.Field(c.Field)
		if e.Name == "goals" && insert {
			if f.Name == "success_metric" {
				inner = append(inner, KV{"success_metric", Obj{{"name", c.Value}}})
				continue
			}
			if f.Name == "status" {
				goalStatus = c.Value
				continue
			}
		}
		name := f.Name
		if f.RequestMember != "" {
			name = f.RequestMember
		}
		v := c.Value
		if l, ok := v.([]string); ok {
			v = strings.Join(l, ",")
		}
		inner = append(inner, KV{name, v})
	}
	if customAt >= 0 {
		inner[customAt].V = custom
	}
	if goalStatus != nil {
		for i := range inner {
			if inner[i].K == "success_metric" {
				o := inner[i].V.(Obj)
				inner[i].V = append(o, KV{"workflow_status", goalStatus})
			}
		}
	}
	if inner == nil {
		inner = Obj{}
	}
	return []byte(toJSON(Obj{{e.Singular, inner}}))
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (r *run) writeFail(f *ApiFailure, method, route string) *AqlError {
	switch f.Kind {
	case failNotAuthorized, failRateLimited, failUnreachable:
		return r.failErr(f, method, route)
	}
	return aqlErr("api_error", "aha: API error (status %d) on %s %s: %s", f.Status, method, route, answerText(f.Text))
}

// ---- insert-statement ----

func (r *run) insert(st *InsertStmt, dryRun bool, out *writeOutcome) (Obj, *AqlError) {
	e := st.Entity
	if st.Parent == nil {
		pt := map[string]string{"product": "a product", "release": "a release", "feature": "a feature", "any": "the record to comment on"}[e.ParentKind]
		return nil, aqlErr("bad_value", "parent is required: INSERT needs the column parent with the id of %s. Form: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)", pt)
	}
	s := textOf(st.Parent)
	up := strings.ToUpper(s)
	var route, parentShown, parentKind, productID string
	switch e.ParentKind {
	case "product":
		p, err := r.resolveProduct(st.Parent, e, false)
		if err != nil {
			return nil, err
		}
		route = routeWith(e.CreateRoute, "v", p.ID)
		parentShown, parentKind, productID = p.Key, "product", p.ID
	case "any":
		k := refKind(s)
		if !commentableKinds[k] {
			return nil, aqlErr("wrong_kind", "parent = needs the id of the commented record, for example '%s-23'. Got %s.", keyMark, quoteVal(s))
		}
		route = routeWith(commentRoute(k), "v", up)
		parentShown, parentKind = up, k
	default:
		if !(refKind(s) == e.ParentKind || isNumericID(s)) {
			p, err := r.findProduct(st.Parent)
			if err != nil {
				return nil, err
			}
			if p != nil {
				recs, err := r.recordsOfKind(e.ParentKind, p)
				if err != nil {
					return nil, err
				}
				return nil, aqlErr("wrong_kind", "%s is a product, not a %s. Use the id of one of its %s: %s", quoteVal(s), e.ParentKind, entityOfKind(e.ParentKind).Name, recs)
			}
			return nil, r.errW25(e.ParentKind, s, "")
		}
		route = routeWith(e.CreateRoute, "v", up)
		parentShown, parentKind = up, e.ParentKind
	}
	required := "name"
	if e.Name == "comments" {
		required = "body"
	}
	missing := func(f string) bool {
		for _, a := range st.Fields {
			if a.Field == f {
				s, ok := a.Value.(string)
				return ok && strings.TrimSpace(s) == ""
			}
		}
		return true
	}
	if missing(required) {
		return nil, aqlErr("bad_value", "%s is required to create a %s", required, e.Singular)
	}
	if e.Name == "goals" && missing("success_metric") {
		return nil, aqlErr("bad_value", "success_metric is required to create a goal")
	}
	fields, err := r.checkValues(e, st.Fields, productID, parentKind, up, false)
	if err != nil {
		return nil, err
	}
	if dryRun {
		res := Obj{}
		for _, c := range fields {
			res = append(res, KV{c.Field, display(c.Value)})
		}
		res = append(res, KV{"message", fmt.Sprintf("would create %s under %s", e.Singular, parentShown)})
		return res, nil
	}
	body := writeBody(e, fields, true)
	ans, f := apiCall(r.s.acct, http.MethodPost, route, nil, body, e.Singular, false)
	if f != nil {
		if f.Kind == failNotFound {
			return nil, r.errW25(parentKind, s, productID)
		}
		return nil, r.writeFail(f, "POST", route)
	}
	rec := r.convert(e, fetched{api: ans[e.Singular].(map[string]any), knownParent: parentShown}, nil, map[string]bool{})
	id := textOf(rec.vals[e.IDField()])
	out.id = strings.ToUpper(id)
	res := Obj{{"id", id}}
	if e.Name == "comments" {
		res = append(res, KV{"parent", parentShown}, KV{"body", rec.vals["body"]}, KV{"created_at", rec.vals["created_at"]})
		res = append(res, KV{"message", "Comment added to " + parentShown})
		return res, nil
	}
	name := rec.vals["name"]
	if name == nil {
		for _, c := range fields {
			if c.Field == "name" {
				name = c.Value
			}
		}
	}
	res = append(res, KV{"name", name}, KV{"parent", parentShown}, KV{"message", fmt.Sprintf("%s %s created", capital(e.Singular), id)})
	return res, nil
}

// ---- update-statement and delete-statement ----

func (r *run) checkID(e *EntityDef, id any) (string, *AqlError) {
	s := textOf(id)
	k := refKind(s)
	if e.Name == "comments" {
		if isNumericID(s) {
			return s, nil
		}
	} else if isNumericID(s) || (k != "" && k == e.Kind) {
		return strings.ToUpper(s), nil
	}
	if k != "" {
		return "", aqlErr("wrong_kind", "%s is the id of a %s, not of a %s. To read it: SELECT * FROM %s WHERE id = '%s'", quoteVal(s), k, e.Kind, entityOfKind(k).Name, strings.ToUpper(s))
	}
	if e.Name == "comments" {
		return "", aqlErr("wrong_kind", "%s is not the id of a comment. To find the id: SELECT id, body FROM comments WHERE parent = '%s-23'", quoteVal(s), keyMark)
	}
	return "", aqlErr("wrong_kind", "%s is not an id. An id looks like '%s'. To search by name: SELECT * FROM %s WHERE name CONTAINS %s", quoteVal(s), exampleID(e.Kind), e.Name, quoteVal(s))
}

// readForWrite reads the record by its id and checks the guards.
func (r *run) readForWrite(e *EntityDef, id string, assigned, guards []Assign) (*Record, *AqlError) {
	route := routeWith(e.UpdateRoute, "id", id)
	if e.RouteFor("id") != "" {
		route = routeWith(e.RouteFor("id"), "v", id)
	}
	api, f := r.get(route, e.Singular)
	if f != nil {
		if f.Kind == failNotFound {
			return nil, aqlErr("not_found", "there is no %s with the id %s", e.Singular, quoteVal(id))
		}
		return nil, r.failErr(f, "GET", route)
	}
	used := map[string]bool{}
	var custom []*CustomField
	for _, a := range append(append([]Assign{}, assigned...), guards...) {
		used[a.Field] = true
		if cf := r.custom[a.Field]; cf != nil && strings.HasPrefix(a.Field, "custom.") {
			custom = append(custom, cf)
		}
	}
	gc := &AllOf{}
	for _, g := range guards {
		gc.Members = append(gc.Members, &Cmp{Field: g.Field, Op: "=", Value: g.Value})
	}
	if err := r.condLookups(e, gc, map[Cond]bool{}); err != nil {
		return nil, err
	}
	rec := r.convert(e, fetched{api: api}, custom, used)
	recID := textOf(rec.vals[e.IDField()])
	if recID == "" {
		recID = id
	}
	for i, g := range guards {
		if !r.eval(rec, gc.Members[i], e, map[Cond]bool{}) {
			cur := rec.vals[g.Field]
			if noValue(cur) {
				return nil, aqlErr("guard_failed", "nothing was changed: %s %s has no %s, not %s", e.Singular, recID, g.Field, quoteVal(g.Value))
			}
			return nil, aqlErr("guard_failed", "nothing was changed: %s %s has %s = %s, not %s", e.Singular, recID, g.Field, quoteVal(cur), quoteVal(g.Value))
		}
	}
	return rec, nil
}

func (r *run) update(st *UpdateStmt, dryRun bool, out *writeOutcome) (Obj, *AqlError) {
	e := st.Entity
	id, err := r.checkID(e, st.ID)
	if err != nil {
		return nil, err
	}
	rec, err := r.readForWrite(e, id, st.Fields, st.Guards)
	if err != nil {
		return nil, err
	}
	if v := textOf(rec.vals[e.IDField()]); v != "" {
		id = v
	}
	productID := str(rec.api["product_id"])
	fields, err := r.checkValues(e, st.Fields, productID, "", "", false)
	if err != nil {
		return nil, err
	}
	if dryRun {
		res := Obj{{"id", id}}
		for _, c := range fields {
			res = append(res, KV{c.Field, display(c.Value)})
		}
		res = append(res, KV{"message", fmt.Sprintf("would update %s %s", e.Singular, id)})
		return res, nil
	}
	route := routeWith(e.UpdateRoute, "id", id)
	ans, f := apiCall(r.s.acct, http.MethodPut, route, nil, writeBody(e, fields, false), e.Singular, false)
	if f != nil {
		if f.Kind == failNotFound {
			var refs []checked
			for _, c := range fields {
				if c.Custom == nil && e.Field(c.Field).Type == ftReference {
					refs = append(refs, c)
				}
			}
			if len(refs) == 1 {
				return nil, r.errW25(e.Field(refs[0].Field).RefKind, textOf(refs[0].Value), productID)
			}
		}
		return nil, r.writeFail(f, "PUT", route)
	}
	var custom []*CustomField
	used := map[string]bool{}
	for _, c := range fields {
		used[c.Field] = true
		if c.Custom != nil {
			custom = append(custom, c.Custom)
		}
	}
	after := r.convert(e, fetched{api: ans[e.Singular].(map[string]any)}, custom, used)
	res := Obj{{"id", id}}
	if e.Name == "comments" {
		res = append(res, KV{"parent", after.vals["parent"]})
	} else {
		res = append(res, KV{"name", after.vals["name"]})
	}
	for _, c := range fields {
		res = append(res, KV{c.Field, after.vals[c.Field]})
	}
	msg := fmt.Sprintf("%s %s updated", capital(e.Singular), id)
	if len(fields) == 1 && fields[0].Field == "status" {
		msg = fmt.Sprintf("%s %s status changed to %s", capital(e.Singular), id, quoteVal(after.vals["status"]))
	}
	res = append(res, KV{"message", msg})
	return res, nil
}

func (r *run) delete(st *DeleteStmt, dryRun bool, out *writeOutcome) (Obj, *AqlError) {
	e := st.Entity
	id, err := r.checkID(e, st.ID)
	if err != nil {
		return nil, err
	}
	rec, err := r.readForWrite(e, id, nil, st.Guards)
	if err != nil {
		return nil, err
	}
	if v := textOf(rec.vals[e.IDField()]); v != "" {
		id = v
	}
	if dryRun {
		return Obj{{"id", id}, {"message", fmt.Sprintf("would delete %s %s", e.Singular, id)}}, nil
	}
	route := routeWith(e.DeleteRoute, "id", id, "product", str(rec.api["product_id"]))
	if _, f := apiCall(r.s.acct, http.MethodDelete, route, nil, nil, "", false); f != nil {
		return nil, r.writeFail(f, "DELETE", route)
	}
	return Obj{{"id", id}, {"message", fmt.Sprintf("%s %s deleted", capital(e.Singular), id)}}, nil
}
