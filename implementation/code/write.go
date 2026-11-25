// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268

package main

import (
	"net/http"
	"strings"
)

// checked is one field of a write statement after check-write-values.
type checked struct {
	field  string
	val    any // string, float64, bool or []string (TextList)
	custom *CustomField
}

func display(v any) any {
	if l, ok := v.([]string); ok {
		return strings.Join(l, ", ")
	}
	return v
}

func parentText(e *EntityDef) string {
	switch e.ParentKind {
	case "release":
		return "a release"
	case "product":
		return "a product"
	case "feature":
		return "a feature"
	}
	return "the record to comment on"
}

// writeInfo is what the log line of execute needs.
type writeInfo struct {
	op, entity, id string
}

func (sc *stmtCtx) w24(value, kind string, p *Product) *AqlError {
	fe := entityOfKind(kind)
	list, err := sc.recordsOf(fe, p)
	if err != nil {
		return err
	}
	return aqlErr("wrong_kind", "%s is a product, not %s %s. Use the id of one of its %s: %s", quote(value), article(kind), kind, fe.Name, list)
}

func (sc *stmtCtx) w30(f *apiFailure) *AqlError {
	switch f.kind {
	case fNotAuthorized, fRateLimited, fUnreachable:
		return sc.failErr(f)
	}
	return aqlErr("api_error", "aha: API error (status %d) on %s %s: %s", f.status, f.method, f.route, answerText(f.text))
}

// productKeyByID gives the key of the product with a NumericId.
func (sc *stmtCtx) productKeyByID(id string) (string, *AqlError) {
	if _, err := sc.productList(false, false); err != nil {
		return "", err
	}
	if p := sc.productByID(id); p != nil {
		return p.Key, nil
	}
	return "", nil
}

func (sc *stmtCtx) runWrite(st *WriteStmt, dryRun bool) (omap, *writeInfo, *AqlError) {
	switch st.Kind {
	case "insert":
		return sc.runInsert(st, dryRun)
	}
	return sc.runChange(st, dryRun)
}

func (sc *stmtCtx) runInsert(st *WriteStmt, dryRun bool) (omap, *writeInfo, *AqlError) {
	e := st.E
	info := &writeInfo{op: "insert", entity: e.Name, id: "none"}
	if st.Parent == nil {
		return nil, info, aqlErr("bad_value", "parent is required: INSERT needs the column parent with the id of %s. Form: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)", parentText(e))
	}
	pv := st.Parent.written()
	up := strings.ToUpper(pv)
	var product *Product
	parentShown := up
	parentKind := e.ParentKind
	switch e.ParentKind {
	case "product":
		p, err := sc.resolveProduct(pv, e, false)
		if err != nil {
			return nil, info, err
		}
		product, parentShown = p, p.Key
	case "any":
		k := refKind(pv)
		if !commentParentKinds[k] {
			return nil, info, sc.errK08(pv)
		}
		parentKind = k
	default:
		if !(refKind(pv) == e.ParentKind || isNumericID(pv)) {
			p, err := sc.findProductQuiet(pv)
			if err != nil {
				return nil, info, err
			}
			if p != nil {
				return nil, info, sc.w24(pv, e.ParentKind, p)
			}
			return nil, info, sc.errW25(e.ParentKind, pv, "")
		}
	}
	has := func(f string) bool {
		for _, a := range st.Fields {
			if a.Field == f && !(a.Val.kind == vText && strings.TrimSpace(a.Val.s) == "") {
				return true
			}
		}
		return false
	}
	req := "name"
	if e.Name == "comments" {
		req = "body"
	}
	if !has(req) {
		return nil, info, aqlErr("bad_value", "%s is required to create %s %s", req, article(e.Singular), e.Singular)
	}
	if e.Name == "goals" && !has("success_metric") {
		return nil, info, aqlErr("bad_value", "success_metric is required to create a goal")
	}
	productID := ""
	if product != nil {
		productID = product.ID
	}
	parentProduct := func() (string, *AqlError) {
		if productID != "" {
			return productID, nil
		}
		pe := entityOfKind(parentKind)
		r, _ := pe.route("id")
		rec, f := sc.c.apiGet(sc.ctx, sc.acct, strings.ReplaceAll(r.Path, "{v}", esc(up)), pe.Singular)
		if f != nil {
			if f.kind == fNotFound {
				return "", sc.errW25(parentKind, up, "")
			}
			return "", sc.failErr(f)
		}
		productID = strOf(rec["product_id"])
		return productID, nil
	}
	vals, err := sc.checkWriteValues(e, st.Fields, parentProduct, "")
	if err != nil {
		return nil, info, err
	}
	if dryRun {
		out := omap{}
		for _, c := range vals {
			out = append(out, kv{c.field, display(c.val)})
		}
		out = append(out, kv{"message", "would create " + e.Singular + " under " + parentShown})
		return out, info, nil
	}
	body := writeBody(e, vals, true)
	var route string
	switch {
	case e.Name == "comments":
		route = "/" + entityOfKind(parentKind).Name + "/" + esc(up) + "/comments"
	case strings.Contains(e.CreateRoute, "{product}"):
		route = strings.ReplaceAll(e.CreateRoute, "{product}", esc(product.ID))
	default:
		route = strings.ReplaceAll(e.CreateRoute, "{parent}", esc(up))
	}
	rec, f := sc.c.apiWrite(sc.ctx, sc.acct, http.MethodPost, route, e.Singular, []byte(body))
	if f != nil {
		if f.kind == fNotFound {
			return nil, info, sc.errW25(parentKind, up, "")
		}
		return nil, info, sc.w30(f)
	}
	r, cerr := sc.convert(e, rec, "", parentShown, map[string]bool{}, nil)
	if cerr != nil {
		return nil, info, cerr
	}
	id := strOf(r.vals[e.idField()])
	info.id = strings.ToUpper(id)
	out := omap{{"id", id}}
	if e.Name == "comments" {
		out = append(out, kv{"parent", parentShown}, kv{"body", r.vals["body"]}, kv{"created_at", r.vals["created_at"]},
			kv{"message", "Comment added to " + parentShown})
		return out, info, nil
	}
	name := r.vals["name"]
	if noValue(name) {
		for _, c := range vals {
			if c.field == "name" {
				name = display(c.val)
			}
		}
	}
	out = append(out, kv{"name", name}, kv{"parent", parentShown}, kv{"message", capitalize(e.Singular) + " " + id + " created"})
	return out, info, nil
}

// checkID is step 1 of update-statement.
func (sc *stmtCtx) checkID(e *EntityDef, v string) *AqlError {
	if isNumericID(v) {
		return nil
	}
	k := refKind(v)
	if e.Name != "comments" && k == e.Kind {
		return nil
	}
	if k != "" {
		ke := entityOfKind(k)
		return aqlErr("wrong_kind", "%s is the id of %s %s, not of %s %s. To read it: SELECT * FROM %s WHERE id = '%s'",
			quote(v), article(k), k, article(e.Singular), e.Singular, ke.Name, strings.ToUpper(v))
	}
	return sc.errK04(v, e)
}

func guardCond(g Assign) Cond {
	if g.Val.kind == vNull || (g.Val.kind == vText && g.Val.s == "") {
		return &NullTest{Field: g.Field}
	}
	return &Cmp{Field: g.Field, Op: "=", Val: g.Val}
}

// runChange runs update-statement and delete-statement.
func (sc *stmtCtx) runChange(st *WriteStmt, dryRun bool) (omap, *writeInfo, *AqlError) {
	e := st.E
	info := &writeInfo{op: st.Kind, entity: e.Name, id: "none"}
	if isRecordID(st.ID) {
		info.id = strings.ToUpper(st.ID)
	}
	if err := sc.checkID(e, st.ID); err != nil {
		return nil, info, err
	}
	id := strings.ToUpper(st.ID)
	r0, _ := e.route("id")
	rec, f := sc.c.apiGet(sc.ctx, sc.acct, strings.ReplaceAll(r0.Path, "{v}", esc(id)), e.Singular)
	if f != nil {
		if f.kind == fNotFound {
			return nil, info, aqlErr("not_found", "there is no %s with the id %s", e.Singular, quote(id))
		}
		return nil, info, sc.failErr(f)
	}
	used := map[string]bool{}
	for _, a := range st.Fields {
		used[a.Field] = true
	}
	for _, g := range st.Guards {
		used[g.Field] = true
	}
	customs := sc.usedCustoms(used)
	r, err := sc.convert(e, rec, "", "", used, customs)
	if err != nil {
		return nil, info, err
	}
	recID := strOf(r.vals[e.idField()])
	if recID == "" {
		recID = id
	}
	var gc []Cond
	for _, g := range st.Guards {
		gc = append(gc, guardCond(g))
	}
	if err := sc.prepareLookups(e, &AllOf{M: gc}, map[Cond]bool{}); err != nil {
		return nil, info, err
	}
	for i, g := range st.Guards {
		if !sc.eval(e, r, gc[i]) {
			cur := r.vals[g.Field]
			if noValue(cur) {
				return nil, info, aqlErr("guard_failed", "nothing was changed: %s %s has no %s, not %s", e.Singular, recID, g.Field, quote(g.Val.written()))
			}
			return nil, info, aqlErr("guard_failed", "nothing was changed: %s %s has %s = %s, not %s", e.Singular, recID, g.Field, quote(textOf(cur)), quote(g.Val.written()))
		}
	}
	productID := strOf(rec["product_id"])
	if st.Kind == "delete" {
		if dryRun {
			return omap{{"id", recID}, {"message", "would delete " + e.Singular + " " + recID}}, info, nil
		}
		route := strings.ReplaceAll(strings.ReplaceAll(e.DeleteRoute, "{product}", esc(productID)), "{id}", esc(id))
		if f := sc.c.apiDelete(sc.ctx, sc.acct, route); f != nil {
			return nil, info, sc.w30(f)
		}
		return omap{{"id", recID}, {"message", capitalize(e.Singular) + " " + recID + " deleted"}}, info, nil
	}
	productKey := func() string {
		k, _ := sc.productKeyByID(productID)
		return k
	}
	vals, err := sc.checkWriteValues(e, st.Fields, func() (string, *AqlError) { return productID, nil }, "record")
	if err != nil {
		return nil, info, err
	}
	if dryRun {
		out := omap{{"id", recID}}
		for _, c := range vals {
			out = append(out, kv{c.field, display(c.val)})
		}
		out = append(out, kv{"message", "would update " + e.Singular + " " + recID})
		return out, info, nil
	}
	body := writeBody(e, vals, false)
	route := strings.ReplaceAll(e.UpdateRoute, "{id}", esc(id))
	ans, f := sc.c.apiWrite(sc.ctx, sc.acct, http.MethodPut, route, e.Singular, []byte(body))
	if f != nil {
		if f.kind == fNotFound {
			var refs []checked
			for _, c := range vals {
				if fd := e.field(c.field); fd != nil && fd.Type == tReference {
					refs = append(refs, c)
				}
			}
			if len(refs) == 1 {
				return nil, info, sc.errW25(e.field(refs[0].field).RefKind, textOf(refs[0].val), productKey())
			}
		}
		return nil, info, sc.w30(f)
	}
	after, err := sc.convert(e, ans, "", "", used, customs)
	if err != nil {
		return nil, info, err
	}
	out := omap{{"id", recID}}
	first := "name"
	if e.Name == "comments" {
		first = "parent"
	}
	out = append(out, kv{first, after.vals[first]})
	for _, c := range vals {
		if !out.has(c.field) {
			out = append(out, kv{c.field, after.vals[c.field]})
		}
	}
	msg := capitalize(e.Singular) + " " + recID + " updated"
	if len(vals) == 1 && vals[0].field == "status" {
		msg = capitalize(e.Singular) + " " + recID + " status changed to '" + textOf(after.vals["status"]) + "'"
	}
	out = append(out, kv{"message", msg})
	return out, info, nil
}

// checkWriteValues is check-write-values. productID gives the NumericId
// of the product for statuses; changed is "record" for an UPDATE.
func (sc *stmtCtx) checkWriteValues(e *EntityDef, fields []Assign, productID func() (string, *AqlError), changed string) ([]checked, *AqlError) {
	var out []checked
	recordProduct := func() string {
		if changed == "" {
			return ""
		}
		id, _ := productID()
		k, _ := sc.productKeyByID(id)
		return k
	}
	for _, a := range fields {
		v := a.Val
		if strings.HasPrefix(a.Field, "custom.") {
			cf := sc.custom[a.Field]
			c, err := sc.checkCustom(cf, a.Field, v)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
			continue
		}
		fd := e.field(a.Field)
		name := a.Field
		var val any
		switch fd.Type {
		case tString, tText:
			if v.kind != vText {
				return nil, aqlErr("bad_value", "%s must be text in single quotes", name)
			}
			val = v.s
			if e.Name == "features" && name == "tags" {
				val = splitTags(v.s)
			}
		case tNumber:
			n, ok := toNumber(v.raw())
			if !ok || v.kind == vBool {
				return nil, aqlErr("bad_value", "%s must be a number", name)
			}
			val = n
		case tBoolean:
			b, ok := v.raw().(bool)
			if v.kind == vText {
				switch strings.ToLower(v.s) {
				case "true":
					b, ok = true, true
				case "false":
					b, ok = false, true
				}
			}
			if !ok {
				return nil, aqlErr("bad_value", "%s must be true or false", name)
			}
			val = b
		case tDate:
			d, ok := checkDate(v)
			if !ok {
				return nil, aqlErr("bad_value", "%s: invalid date %s, use YYYY-MM-DD", name, quote(v.written()))
			}
			val = d
		case tUser:
			s := v.written()
			switch {
			case v.kind == vText && isMeWord(s):
				me, err := sc.currentUser()
				if err != nil {
					return nil, err
				}
				val = strOf(me["email"])
			case v.kind == vText && reEmail.MatchString(strings.TrimSpace(s)):
				val = strings.TrimSpace(s)
			case isNumericID(strings.TrimSpace(s)):
				val = strings.TrimSpace(s)
			default:
				return nil, aqlErr("bad_value", "%s: there is no user %s. Use an e-mail address, or 'me' for yourself", name, quote(s))
			}
		case tReference:
			s := v.written()
			if refKind(s) == fd.RefKind || isNumericID(s) {
				val = strings.ToUpper(s)
				break
			}
			p, err := sc.findProductQuiet(s)
			if err != nil {
				return nil, err
			}
			if p != nil {
				return nil, sc.w24(s, fd.RefKind, p)
			}
			return nil, sc.errW25(fd.RefKind, s, recordProduct())
		case tStatus:
			if v.kind != vText {
				return nil, aqlErr("bad_value", "%s must be text in single quotes", name)
			}
			pid, err := productID()
			if err != nil {
				return nil, err
			}
			names, err := sc.statusNames(e, pid)
			if err != nil {
				return nil, err
			}
			found := ""
			for _, n := range names {
				if textEq(n, v.s) {
					found = n
					break
				}
			}
			if found == "" {
				list := "(none)"
				if len(names) > 0 {
					list = strings.Join(names, ", ")
				}
				return nil, aqlErr("bad_value", "workflow status %s not found for %s; valid statuses: %s", quote(v.s), e.Singular, list)
			}
			val = found
		default:
			val = v.raw()
		}
		out = append(out, checked{field: a.Field, val: val})
	}
	return out, nil
}

func splitTags(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func checkDate(v Value) (string, bool) {
	if v.kind != vText {
		return "", false
	}
	s := strings.TrimSpace(v.s)
	if validDay(s) {
		return s, true
	}
	if len(s) > 10 && validDay(s[:10]) && isTimestamp(s) {
		return s[:10], true
	}
	return "", false
}

// statusNames gives the workflow statuses of the entity in the product.
func (sc *stmtCtx) statusNames(e *EntityDef, productID string) ([]string, *AqlError) {
	if productID == "" {
		return nil, nil
	}
	groups, err := sc.productStatuses(productID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.entity == e.Name {
			var out []string
			for _, s := range g.names {
				out = append(out, strOf(s["name"]))
			}
			return out, nil
		}
	}
	return nil, nil
}

// checkCustom checks a custom value by its kind (check-write-values, step 10).
func (sc *stmtCtx) checkCustom(cf *CustomField, name string, v Value) (checked, *AqlError) {
	c := checked{field: name, custom: cf}
	mustText := func() *AqlError {
		if v.kind != vText {
			return aqlErr("bad_value", "%s must be text in single quotes", name)
		}
		return nil
	}
	matchOption := func(s string) (string, bool, *AqlError) {
		opts, err := sc.optionsOf(cf)
		if err != nil {
			return "", false, err
		}
		for _, o := range opts {
			if textEq(o, s) {
				return o, true, nil
			}
		}
		return "", false, nil
	}
	optionsErr := func(s string) *AqlError {
		opts, _ := sc.optionsOf(cf)
		return aqlErr("bad_value", "%s: there is no option %s. Options: %s", name, quote(s), listMax(opts))
	}
	switch cf.kind() {
	case "text", "url", "note", "choice":
		if err := mustText(); err != nil {
			return c, err
		}
		c.val = v.s
	case "number":
		n, ok := toNumber(v.raw())
		if !ok || v.kind == vBool {
			return c, aqlErr("bad_value", "%s must be a number", name)
		}
		c.val = n
	case "date":
		d, ok := checkDate(v)
		if !ok {
			return c, aqlErr("bad_value", "%s: invalid date %s, use YYYY-MM-DD", name, quote(v.written()))
		}
		c.val = d
	case "fixed_choice":
		if err := mustText(); err != nil {
			return c, err
		}
		if v.s == "" {
			c.val = ""
			break
		}
		o, ok, err := matchOption(v.s)
		if err != nil {
			return c, err
		}
		if !ok {
			return c, optionsErr(v.s)
		}
		c.val = o
	case "tags":
		if err := mustText(); err != nil {
			return c, err
		}
		c.val = splitTags(v.s)
	case "fixed_tags":
		if err := mustText(); err != nil {
			return c, err
		}
		tags := splitTags(v.s)
		out := []string{}
		for _, t := range tags {
			o, ok, err := matchOption(t)
			if err != nil {
				return c, err
			}
			if !ok {
				return c, optionsErr(t)
			}
			out = append(out, o)
		}
		c.val = out
	case "users":
		if err := mustText(); err != nil {
			return c, err
		}
		out := []string{}
		for _, p := range splitParts(v.s) {
			switch {
			case isMeWord(p):
				me, err := sc.currentUser()
				if err != nil {
					return c, err
				}
				out = append(out, strOf(me["id"]))
			case reEmail.MatchString(p) || isNumericID(p):
				out = append(out, p)
			default:
				return c, aqlErr("bad_value", "%s: there is no user %s. Use an e-mail address, or 'me' for yourself", name, quote(p))
			}
		}
		c.val = out
	default:
		return c, aqlErr("bad_name", "the field %s cannot be set.", name)
	}
	return c, nil
}

// writeBody is write-request.
func writeBody(e *EntityDef, vals []checked, insert bool) string {
	inner := omap{}
	cfIndex := -1
	var cfObj omap
	smIndex := -1
	var smName any
	var goalStatus any
	hasGoalStatus := false
	for _, c := range vals {
		if c.custom != nil {
			if cfIndex < 0 {
				cfIndex = len(inner)
				inner = append(inner, kv{"custom_fields", nil})
			}
			cfObj = append(cfObj, kv{c.custom.Key, c.val})
			continue
		}
		if e.Name == "goals" && c.field == "success_metric" {
			if insert {
				smIndex = len(inner)
				smName = c.val
				inner = append(inner, kv{"success_metric", nil})
			}
			continue
		}
		if e.Name == "goals" && c.field == "status" && insert {
			goalStatus, hasGoalStatus = c.val, true
			continue
		}
		member := c.field
		switch {
		case c.field == "status":
			member = "workflow_status"
		case c.field == "assigned_to":
			member = "assigned_to_user"
		case e.Name == "releases" && c.field == "description":
			member = "theme"
		}
		val := c.val
		if l, ok := val.([]string); ok {
			val = strings.Join(l, ",")
		}
		inner = append(inner, kv{member, val})
	}
	if cfIndex >= 0 {
		inner[cfIndex].v = cfObj
	}
	if smIndex >= 0 {
		sm := omap{{"name", smName}}
		if hasGoalStatus {
			sm = append(sm, kv{"workflow_status", goalStatus})
		}
		inner[smIndex].v = sm
	}
	return encodeJSON(omap{{e.Singular, inner}})
}
