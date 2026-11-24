// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// read-statement, plan-fetch, convert-record, evaluate-condition,
// build-rows and describe-statement.

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ---- helpers on conditions ----

func conjuncts(c Cond) []Cond {
	switch x := c.(type) {
	case nil:
		return nil
	case *AllOf:
		return x.Members
	}
	return []Cond{c}
}

func eqConj(c Cond, field string) (*Cmp, bool) {
	if cmp, ok := c.(*Cmp); ok && cmp.Field == field && cmp.Op == "=" {
		return cmp, true
	}
	return nil, false
}

func walkCond(c Cond, fn func(Cond)) {
	if c == nil {
		return
	}
	fn(c)
	switch x := c.(type) {
	case *AllOf:
		for _, m := range x.Members {
			walkCond(m, fn)
		}
	case *AnyOf:
		for _, m := range x.Members {
			walkCond(m, fn)
		}
	case *Negation:
		walkCond(x.Member, fn)
	}
}

func condFields(c Cond, out map[string]bool) {
	walkCond(c, func(x Cond) {
		switch y := x.(type) {
		case *Cmp:
			out[y.Field] = true
		case *NullTest:
			out[y.Field] = true
		case *InList:
			out[y.Field] = true
		}
	})
}

var exampleIDs = map[string]string{"feature": "{KEY}-23", "release": "{KEY}-R-3", "initiative": "{KEY}-S-1", "epic": "{KEY}-E-1",
	"goal": "{KEY}-G-1", "idea": "{KEY}-I-1", "requirement": "{KEY}-23-1", "product": "{KEY}", "comment": "{KEY}-23"}

func exampleID(kind string) string { return withKey(exampleIDs[kind]) }

func article(word string) string {
	if word != "" && strings.ContainsRune("aeiouAEIOU", rune(word[0])) {
		return "an " + word
	}
	return "a " + word
}

// ---- fetched records ----

type fetched struct {
	api          map[string]any
	knownProduct *Product
	knownParent  string
}

type fetchResult struct {
	recs      []fetched
	scope     string
	total     *int
	applied   map[Cond]bool
	countOnly bool
}

type listOpts struct {
	short     bool // count only and early end may apply
	allApply  bool
	custom    bool
	used      map[string]bool
	q         string
	accountWd bool
}

func (r *run) readList(st *ReadStmt, route string, o listOpts) ([]map[string]any, *int, bool, *AqlError) {
	e := st.Entity
	countOnly := o.short && o.allApply && len(st.Items) == 1 && st.Items[0].Fn == "COUNT" && st.Items[0].Field == "" && !st.Items[0].Distinct && len(st.GroupBy) == 0
	early := o.short && o.allApply && len(st.OrderBy) == 0 && len(st.GroupBy) == 0 && !st.Distinct && !hasAggregate(st)
	limit := effectiveLimit(st)
	fields := listFields(e, o.used, o.custom)
	var acc []map[string]any
	for page := 1; ; page++ {
		p := url.Values{}
		p.Set("page", strconv.Itoa(page))
		p.Set("per_page", strconv.Itoa(pageSize))
		p.Set("fields", strings.Join(fields, ","))
		if o.q != "" {
			p.Set("q", o.q)
		}
		lp, f := r.list(route, p)
		if f != nil {
			if f.Kind == failNotFound {
				return nil, nil, false, errNotFoundList
			}
			return nil, nil, false, r.failErr(f, "GET", route)
		}
		if page == 1 {
			if countOnly {
				t := lp.TotalRecords
				return nil, &t, true, nil
			}
			if o.custom && len(lp.Records) > 0 {
				has := false
				for _, rec := range lp.Records {
					if _, ok := rec["custom_fields"]; ok {
						has = true
					}
				}
				if !has {
					return nil, nil, false, aqlErr("api_error", "Aha! returned this list of %s without custom fields. A record that is read by its id has them: add %s = '...' to WHERE.", e.Name, e.IDField())
				}
			}
			if lp.TotalPages > maxPages && !(early && len(lp.Records) >= limit) {
				return nil, nil, false, r.errC02(st, lp.TotalRecords, o.accountWd)
			}
		}
		acc = append(acc, lp.Records...)
		if early && len(acc) >= limit {
			if page < lp.TotalPages {
				t := lp.TotalRecords
				return acc, &t, false, nil
			}
			return acc, nil, false, nil
		}
		if page >= lp.TotalPages || len(lp.Records) == 0 {
			return acc, nil, false, nil
		}
	}
}

// errNotFoundList marks a list route that answered NotFound (K07).
var errNotFoundList = &AqlError{Kind: "not_found", Message: "\x00list"}

func (r *run) errC02(st *ReadStmt, total int, accountWide bool) *AqlError {
	var hint string
	if accountWide {
		hint = "Narrow it down, for example with product = '" + keyMark + "'"
	} else {
		usedF := map[string]bool{}
		condFields(st.Where, usedF)
		for _, pf := range st.Entity.ParentFilters {
			if pf == "product" || usedF[pf] {
				continue
			}
			kind := pf
			if f := st.Entity.Field(pf); f != nil && f.RefKind != "" {
				kind = f.RefKind
			}
			if pf == "parent" {
				kind = "feature"
			}
			hint = fmt.Sprintf("Narrow it down, for example with %s = '%s'", pf, exampleID(kind))
			break
		}
		if hint == "" {
			hint = "Use LIMIT without ORDER BY, GROUP BY and DISTINCT to read the first records, or COUNT(*) alone to count them."
		}
	}
	return aqlErr("scope", "this would read %d records. %s", total, hint)
}

func hasAggregate(st *ReadStmt) bool {
	for _, it := range st.Items {
		if it.Fn != "" {
			return true
		}
	}
	return false
}

func effectiveLimit(st *ReadStmt) int {
	l := defaultLimit
	if st.Limit != nil {
		l = *st.Limit
	}
	if l > maxLimit {
		l = maxLimit
	}
	return l
}

func errC01(e *EntityDef) *AqlError {
	switch e.Name {
	case "comments":
		return aqlErr("scope", "comments are listed per record: add parent = '...' to WHERE. Example: %s", exampleOf(e))
	case "statuses":
		return aqlErr("scope", "statuses are listed per product: add product = '...' to WHERE. Example: %s", exampleOf(e))
	}
	var parents, conds []string
	for _, rt := range e.ReadRoutes {
		if rt.Field == "id" {
			continue
		}
		parents = append(parents, rt.Field)
		conds = append(conds, rt.Field+" = '...'")
	}
	return aqlErr("scope", "%s are listed per %s: add %s to WHERE. Example: %s", e.Name, strings.Join(parents, " or "), strings.Join(conds, " or "), exampleOf(e))
}

// wrongKind gives the message of plan-fetch step 4.
func (r *run) wrongKind(st *ReadStmt, field string, value any) *AqlError {
	e := st.Entity
	s := textOf(value)
	k := refKind(s)
	up := strings.ToUpper(s)
	expected := e.Kind
	if field != "id" {
		if f := e.Field(field); f != nil && f.RefKind != "" {
			expected = f.RefKind
		}
	}
	if k != "" && field == "id" {
		for _, rt := range e.ReadRoutes {
			if rt.Field == k || (rt.Field == "parent" && commentableKinds[k]) {
				return aqlErr("wrong_kind", "%s is the id of a %s, not of a %s. For the %s of that %s: SELECT * FROM %s WHERE %s = '%s'", quoteVal(s), k, e.Singular, e.Name, k, e.Name, k, up)
			}
		}
	}
	if k != "" {
		return aqlErr("wrong_kind", "%s is the id of a %s, not of a %s. To read it: SELECT * FROM %s WHERE id = '%s'", quoteVal(s), k, expected, entityOfKind(k).Name, up)
	}
	if e.HasParentFilter("product") {
		p, err := r.findProduct(value)
		if err != nil {
			return err
		}
		if p != nil {
			return aqlErr("wrong_kind", "%s is a product, not a %s. Use: WHERE product = '%s'", quoteVal(s), expected, p.Key)
		}
	}
	if field == "id" {
		if e.Name == "comments" {
			return aqlErr("wrong_kind", "%s is not the id of a comment. To find the id: SELECT id, body FROM comments WHERE parent = '%s-23'", quoteVal(s), keyMark)
		}
		return aqlErr("wrong_kind", "%s is not an id. An id looks like '%s'. To search by name: SELECT * FROM %s WHERE name CONTAINS %s", quoteVal(s), exampleID(e.Kind), e.Name, quoteVal(s))
	}
	fe := entityOfKind(expected)
	if field == "release" || field == "initiative" || field == "epic" {
		for _, c := range conjuncts(st.Where) {
			if cmp, ok := eqConj(c, "product"); ok {
				p, err := r.findProduct(cmp.Value)
				if err != nil {
					return err
				}
				if p != nil {
					recs, err := r.recordsOfKind(expected, p)
					if err != nil {
						return err
					}
					return aqlErr("wrong_kind", "%s = needs the id of the %s, not its name. The %s of %s: %s", field, field, fe.Name, p.Key, recs)
				}
			}
		}
	}
	return aqlErr("wrong_kind", "%s = needs the id of the %s, for example '%s', not its name. To find the id: SELECT id, name FROM %s WHERE product = '%s'", field, field, exampleID(expected), fe.Name, keyMark)
}

// recordsOfKind lists the records of a kind in a product with one list request.
func (r *run) recordsOfKind(kind string, p *Product) (string, *AqlError) {
	e := entityOfKind(kind)
	route := routeWith(e.RouteFor("product"), "v", p.ID)
	params := url.Values{}
	params.Set("page", "1")
	params.Set("per_page", strconv.Itoa(pageSize))
	params.Set("fields", "reference_num,name")
	lp, f := r.list(route, params)
	if f != nil {
		return "", r.failErr(f, "GET", route)
	}
	var items []string
	for _, rec := range lp.Records {
		items = append(items, fmt.Sprintf("%s (%s)", str(rec["reference_num"]), str(rec["name"])))
	}
	if len(items) == 0 {
		return "none", nil
	}
	if lp.TotalRecords > len(items) && len(items) >= maxListed {
		return strings.Join(items[:maxListed], ", ") + fmt.Sprintf(", and %d more", lp.TotalRecords-maxListed), nil
	}
	return listMax(items), nil
}

// readOne is plan-fetch step 3: reading one route for one value.
func (r *run) readOne(st *ReadStmt, field string, value any, o listOpts) ([]fetched, string, *int, bool, *AqlError) {
	e := st.Entity
	s := textOf(value)
	up := strings.ToUpper(s)
	k := refKind(s)
	switch {
	case field == "id":
		ok := isNumericID(s) || (e.Name != "comments" && k != "" && k == e.Kind)
		if !ok {
			return nil, "", nil, false, r.wrongKind(st, field, value)
		}
		route := routeWith(e.RouteFor("id"), "v", up)
		rec, f := r.get(route, e.Singular)
		if f != nil {
			if f.Kind == failNotFound {
				return nil, e.Singular + " " + up, nil, false, nil
			}
			return nil, "", nil, false, r.failErr(f, "GET", route)
		}
		idv := str(source(rec, e.Field("id").Source))
		if idv == "" {
			idv = up
		}
		return []fetched{{api: rec}}, e.Singular + " " + idv, nil, false, nil
	case field == "product":
		p, err := r.resolveProduct(value, e, true)
		if err != nil {
			return nil, "", nil, false, err
		}
		route := routeWith(e.RouteFor("product"), "v", p.ID)
		recs, total, co, err := r.readList(st, route, o)
		if err != nil {
			return nil, "", nil, false, r.k07(err, "product", s)
		}
		return wrap(recs, p, ""), "product " + p.Key, total, co, nil
	case field == "parent" && e.Name == "comments":
		var route, scope string
		var kp *Product
		if commentableKinds[k] {
			route = routeWith(commentRoute(k), "v", up)
			scope = k + " " + up
		} else {
			p, err := r.findProduct(value)
			if err != nil {
				return nil, "", nil, false, err
			}
			if p == nil {
				return nil, "", nil, false, aqlErr("wrong_kind", "parent = needs the id of the commented record, for example '%s-23'. Got %s.", keyMark, quoteVal(s))
			}
			route = routeWith(commentRoute("product"), "v", p.ID)
			scope = "product " + p.Key
			kp = p
		}
		recs, total, co, err := r.readList(st, route, o)
		if err != nil {
			return nil, "", nil, false, r.k07(err, k, s)
		}
		parent := up
		if kp != nil {
			parent = kp.Key
		}
		return wrap(recs, kp, parent), scope, total, co, nil
	}
	fdef := e.Field(field)
	kind := field
	if fdef != nil && fdef.RefKind != "" {
		kind = fdef.RefKind
	}
	if !(isNumericID(s) || k == kind) {
		return nil, "", nil, false, r.wrongKind(st, field, value)
	}
	route := routeWith(e.RouteFor(field), "v", up)
	recs, total, co, err := r.readList(st, route, o)
	if err != nil {
		return nil, "", nil, false, r.k07(err, kind, s)
	}
	return wrap(recs, nil, up), field + " " + up, total, co, nil
}

func (r *run) k07(err *AqlError, kind, value string) *AqlError {
	if err != errNotFoundList {
		return err
	}
	ex := ""
	if ke := entityOfKind(kind); ke != nil {
		ex = exampleOf(ke)
	}
	return aqlErr("not_found", "there is no %s with the id %s. To find it: %s", kind, quoteVal(value), ex)
}

func wrap(recs []map[string]any, p *Product, parent string) []fetched {
	out := make([]fetched, 0, len(recs))
	for _, rec := range recs {
		out = append(out, fetched{api: rec, knownProduct: p, knownParent: parent})
	}
	return out
}

func (r *run) planFetch(st *ReadStmt, used map[string]bool, hasCustom bool) (*fetchResult, *AqlError) {
	e := st.Entity
	res := &fetchResult{applied: map[Cond]bool{}}
	conj := conjuncts(st.Where)
	switch e.Fixed {
	case "me":
		m, err := r.currentUser()
		if err != nil {
			return nil, err
		}
		res.recs, res.scope = []fetched{{api: m}}, "current user"
		return res, r.lookups(st, res)
	case "products":
		res.scope = "all products"
		if hasCustom {
			recs, _, _, err := r.readList(st, "/products", listOpts{custom: true, used: used})
			if err != nil {
				return nil, err
			}
			res.recs = wrap(recs, nil, "")
		} else {
			if _, err := r.productList(); err != nil {
				return nil, err
			}
			res.recs = wrap(r.productRecs, nil, "")
		}
		return res, r.lookups(st, res)
	case "users":
		res.scope = "all users"
		recs, total, co, err := r.readList(st, "/users", listOpts{short: true, allApply: len(conj) == 0, used: used, accountWd: true})
		if err != nil {
			return nil, err
		}
		res.recs, res.total, res.countOnly = wrap(recs, nil, ""), total, co
		return res, r.lookups(st, res)
	case "statuses":
		var pc *Cmp
		for _, c := range conj {
			if cmp, ok := eqConj(c, "product"); ok {
				pc = cmp
				break
			}
		}
		if pc == nil {
			return nil, errC01(e)
		}
		p, err := r.resolveProduct(pc.Value, e, true)
		if err != nil {
			return nil, err
		}
		srs, err := r.statusRecords(p.ID)
		if err != nil {
			return nil, err
		}
		for _, sr := range srs {
			res.recs = append(res.recs, fetched{api: map[string]any{"name": sr.Name, "entity": sr.Entity, "_prefix": sr.Prefix,
				"position": sr.Position, "complete": sr.Complete}, knownProduct: p})
		}
		res.scope = "product " + p.Key
		res.applied[pc] = true
		return res, r.lookups(st, res)
	}
	allApplied := func(extra Cond) bool {
		for _, c := range conj {
			if c != extra && !res.applied[c] {
				return false
			}
		}
		return true
	}
	// step 2: a route by an id or a parent
	for _, rt := range e.ReadRoutes {
		if rt.Field == "product" {
			continue
		}
		for _, c := range conj {
			cmp, ok := eqConj(c, rt.Field)
			if !ok {
				continue
			}
			if _, isBool := cmp.Value.(bool); isBool {
				continue
			}
			recs, scope, total, co, err := r.readOne(st, rt.Field, cmp.Value, listOpts{short: true, allApply: allApplied(c), custom: hasCustom, used: used})
			if err != nil {
				return nil, err
			}
			res.recs, res.scope, res.total, res.countOnly = recs, scope, total, co
			res.applied[c] = true
			return res, r.lookups(st, res)
		}
	}
	// step 5: a list of values
	for _, c := range conj {
		in, ok := c.(*InList)
		if !ok || len(in.Values) > maxInValues || e.RouteFor(in.Field) == "" && !(e.Name == "comments" && in.Field == "parent") {
			continue
		}
		var scopes []string
		seen := map[string]bool{}
		for _, v := range in.Values {
			if v == nil {
				continue
			}
			recs, scope, _, _, err := r.readOne(st, in.Field, v, listOpts{custom: hasCustom, used: used})
			if err != nil {
				return nil, err
			}
			scopes = append(scopes, scope)
			for _, fr := range recs {
				key := str(fr.api["id"]) + "|" + str(fr.api["reference_num"])
				if seen[key] {
					continue
				}
				seen[key] = true
				res.recs = append(res.recs, fr)
			}
		}
		res.scope = strings.Join(scopes, ", ")
		res.applied[c] = true
		return res, r.lookups(st, res)
	}
	// step 6: route by product
	if e.RouteFor("product") != "" {
		for _, c := range conj {
			cmp, ok := eqConj(c, "product")
			if !ok {
				continue
			}
			recs, scope, total, co, err := r.readOne(st, "product", cmp.Value, listOpts{short: true, allApply: allApplied(c), custom: hasCustom, used: used})
			if err != nil {
				return nil, err
			}
			res.recs, res.scope, res.total, res.countOnly = recs, scope, total, co
			res.applied[c] = true
			return res, r.lookups(st, res)
		}
	}
	// step 7: the account-wide list
	if e.AccountList == "" {
		return nil, errC01(e)
	}
	q := ""
	if e.NameSearch {
		for _, c := range conj {
			if cmp, ok := c.(*Cmp); ok && cmp.Field == "name" && (cmp.Op == "=" || cmp.Op == "contains") {
				if s, ok := cmp.Value.(string); ok && s != "" {
					q = s
					break
				}
			}
		}
	}
	recs, total, co, err := r.readList(st, e.AccountList, listOpts{short: true, allApply: allApplied(nil), custom: hasCustom, used: used, q: q, accountWd: true})
	if err != nil {
		return nil, err
	}
	res.recs, res.scope, res.total, res.countOnly = wrap(recs, nil, ""), "all products", total, co
	return res, r.lookups(st, res)
}

// lookups is plan-fetch step 9.
func (r *run) lookups(st *ReadStmt, res *fetchResult) *AqlError {
	return r.condLookups(st.Entity, st.Where, res.applied)
}

func isMe(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "me" || s == "myself" || s == "current_user"
}

func (r *run) fieldType(e *EntityDef, field string) (FieldType, bool) {
	if cf := r.custom[field]; cf != nil && strings.HasPrefix(field, "custom.") {
		return customKindType[cf.Kind], false
	}
	if f := e.Field(field); f != nil {
		return f.Type, f.UserAsName
	}
	return ftString, false
}

func (r *run) condLookups(e *EntityDef, where Cond, applied map[Cond]bool) *AqlError {
	var needMe, needProducts bool
	walkCond(where, func(c Cond) {
		if applied[c] {
			return
		}
		var field string
		var vals []any
		switch x := c.(type) {
		case *Cmp:
			field, vals = x.Field, []any{x.Value}
		case *InList:
			field, vals = x.Field, x.Values
		default:
			return
		}
		t, _ := r.fieldType(e, field)
		if t == ftUser {
			for _, v := range vals {
				if isMe(v) {
					needMe = true
				}
			}
		}
		if t == ftKey {
			needProducts = true
		}
	})
	if needMe {
		if _, err := r.currentUser(); err != nil {
			return err
		}
	}
	if needProducts {
		if _, err := r.productList(); err != nil {
			return err
		}
	}
	return nil
}

// ---- convert-record ----

type Record struct {
	vals  map[string]any
	spell map[string][]any
	api   map[string]any
}

var (
	reBlockEnd = regexp.MustCompile(`(?i)</\s*(p|li|h[1-6]|div|blockquote|pre|tr|section|article|header|footer|ul|ol|table|dd|dt)\s*>|<\s*br\s*/?\s*>`)
	reTag      = regexp.MustCompile(`<[^>]*>`)
	reSpaces   = regexp.MustCompile(`[ \t\f\v\r]+`)
)

func htmlToText(s string) any {
	s = strings.NewReplacer("\r\n", " ", "\n", " ").Replace(s)
	s = reBlockEnd.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(reSpaces.ReplaceAllString(l, " "))
		if l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return strings.Join(lines, "\n")
}

func plainValue(v any) any {
	switch x := v.(type) {
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return x.String()
		}
		return f
	case map[string]any, []any:
		return toJSON(x)
	}
	return v
}

func (r *run) convert(e *EntityDef, fr fetched, custom []*CustomField, used map[string]bool) *Record {
	api := fr.api
	rec := &Record{vals: map[string]any{}, spell: map[string][]any{}, api: api}
	for _, f := range e.Fields {
		var v any
		if f.NoSource {
			if fr.knownProduct != nil {
				v = fr.knownProduct.Key
			}
			rec.vals[f.Name] = v
			continue
		}
		v = source(api, f.Source)
		switch f.Type {
		case ftText:
			if m, ok := v.(map[string]any); ok {
				v = m["body"]
			}
			if s, ok := v.(string); ok {
				v = htmlToText(s)
			} else if v != nil {
				v = plainValue(v)
			}
		case ftDate:
			if s, ok := v.(string); ok {
				if len(s) >= 10 {
					v = s[:10]
				} else if s == "" {
					v = nil
				}
			} else {
				v = plainValue(v)
			}
		case ftBoolean:
			switch x := v.(type) {
			case bool:
			case nil:
				if f.FalseWhenAbsent {
					v = false
				}
			case string:
				v = strings.EqualFold(x, "true")
			default:
				v = plainValue(v)
			}
		case ftKey:
			switch {
			case e.Name == "products" && f.Name == "key":
				v = plainValue(v)
			case f.Name == "product" && fr.knownProduct != nil:
				v = fr.knownProduct.Key
			case v == nil:
			default:
				id := str(v)
				v = nil
				if used[f.Name] {
					if p := r.productByID(id); p != nil {
						v = p.Key
					}
				}
			}
		default:
			v = plainValue(v)
		}
		rec.vals[f.Name] = v
	}
	// notes of the catalog entries
	switch e.Name {
	case "features":
		if l, ok := source(api, "tags").([]any); ok {
			var tags []string
			for _, t := range l {
				if s := str(t); s != "" {
					tags = append(tags, s)
				}
			}
			if len(tags) == 0 {
				rec.vals["tags"] = nil
			} else {
				rec.vals["tags"] = strings.Join(tags, ", ")
			}
		} else if s, ok := rec.vals["tags"].(string); ok && s == "" {
			rec.vals["tags"] = nil
		}
	case "requirements":
		if rec.vals["feature"] == nil && fr.knownParent != "" {
			rec.vals["feature"] = fr.knownParent
		}
	case "comments":
		if u, ok := rec.vals["parent"].(string); ok && u != "" {
			rec.vals["parent"] = u[strings.LastIndex(u, "/")+1:]
		} else if fr.knownParent != "" {
			rec.vals["parent"] = fr.knownParent
		}
	}
	for _, cf := range custom {
		rec.vals[cf.Name] = nil
		entry := customEntry(api, cf.Key)
		if entry != nil {
			v, sp := customValue(cf, entry)
			rec.vals[cf.Name] = v
			if len(sp) > 0 {
				rec.spell[cf.Name] = sp
			}
		}
	}
	// other spellings
	add := func(f string, v any) {
		if s := str(v); s != "" {
			rec.spell[f] = append(rec.spell[f], s)
		}
	}
	switch idf := e.IDField(); {
	case e.Name == "products":
		add("key", api["id"])
	case idf == "id" && e.Field("id").Source != "id":
		add("id", api["id"])
	}
	if e.HasField("product") && e.Field("product").Source == "product_id" {
		add("product", api["product_id"])
	}
	if e.HasField("assigned_to") {
		add("assigned_to", source(api, "assigned_to_user.name"))
		add("assigned_to", source(api, "assigned_to_user.id"))
	}
	for _, f := range e.Fields {
		if f.Type == ftReference && strings.Contains(f.Source, ".") {
			obj := f.Source[:strings.Index(f.Source, ".")]
			add(f.Name, source(api, obj+".id"))
			add(f.Name, source(api, obj+".name"))
		}
	}
	if e.Name == "comments" {
		add("user", source(api, "user.email"))
		add("user", source(api, "user.id"))
		add("parent", source(api, "commentable.id"))
	}
	if e.Name == "statuses" {
		if ent := entityByName(str(api["entity"])); ent != nil {
			add("entity", ent.Singular)
		}
		add("entity", api["_prefix"])
	}
	return rec
}

func customEntry(api map[string]any, key string) map[string]any {
	l, _ := api["custom_fields"].([]any)
	for _, x := range l {
		if m, ok := x.(map[string]any); ok && str(m["key"]) == key {
			return m
		}
	}
	return nil
}

func stringsOf(v any) []string {
	l, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, x := range l {
		if s := str(x); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// customValue converts the value of an entry by the kind of the field.
func customValue(cf *CustomField, entry map[string]any) (any, []any) {
	v := entry["value"]
	if v == nil {
		return nil, nil
	}
	if s, ok := v.(string); ok && s == "" {
		return nil, nil
	}
	if l, ok := v.([]any); ok && len(l) == 0 {
		return nil, nil
	}
	switch cf.Kind {
	case "text", "url", "choice", "fixed_choice":
		if s, ok := v.(string); ok {
			return s, nil
		}
	case "note":
		if s, ok := v.(string); ok {
			return htmlToText(s), nil
		}
	case "number", "scorecard":
		if f, ok := numberOf(v); ok {
			return f, nil
		}
	case "date":
		if s, ok := v.(string); ok {
			if len(s) >= 10 {
				return s[:10], nil
			}
			return s, nil
		}
	case "tags", "fixed_tags":
		if l := stringsOf(v); l != nil {
			return strings.Join(l, ", "), nil
		}
	case "users":
		var emails, names, ids []string
		if l := stringsOf(entry["email_value"]); l != nil {
			emails = l
		}
		if full, ok := entry["full_value"].([]any); ok {
			var fe []string
			for _, x := range full {
				if m, ok := x.(map[string]any); ok {
					if s := str(m["email"]); s != "" {
						fe = append(fe, s)
					}
					if s := str(m["name"]); s != "" {
						names = append(names, s)
					}
					if s := str(m["id"]); s != "" {
						ids = append(ids, s)
					}
				}
			}
			if emails == nil {
				emails = fe
			}
		}
		if emails == nil {
			emails = stringsOf(v)
		}
		if ids == nil {
			ids = stringsOf(v)
		}
		var sp []any
		if len(names) > 0 {
			sp = append(sp, strings.Join(names, ", "))
		}
		if len(ids) > 0 {
			sp = append(sp, strings.Join(ids, ", "))
		}
		if len(emails) == 0 {
			return nil, sp
		}
		return strings.Join(emails, ", "), sp
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return toJSON(v), nil
}

// ---- evaluate-condition ----

func noValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok && s == "" {
		return true
	}
	return false
}

func truth(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	default:
		if f, ok := numberOf(v); ok {
			if f == 1 {
				return true, true
			}
			if f == 0 {
				return false, true
			}
		}
	}
	return false, false
}

func valueEq(fv, v any, t FieldType) bool {
	if noValue(fv) {
		return false
	}
	_, fb := fv.(bool)
	_, vb := v.(bool)
	if fb || vb {
		a, ok1 := truth(fv)
		b, ok2 := truth(v)
		return ok1 && ok2 && a == b
	}
	if t == ftNumber {
		a, ok1 := numberOf(fv)
		b, ok2 := numberOf(v)
		if ok1 && ok2 {
			return a == b
		}
		return false
	}
	vs := textOf(v)
	if (t == ftDate || t == ftDatetime) && len([]rune(strings.TrimSpace(vs))) == 10 {
		fs := textOf(fv)
		if len(fs) >= 10 {
			return cform(fs[:10]) == cform(vs)
		}
	}
	return cform(textOf(fv)) == cform(vs)
}

func likeMatch(pattern, text string) bool {
	var b strings.Builder
	b.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile("(?s)" + b.String())
	return err == nil && re.MatchString(text)
}

func (r *run) substitute(e *EntityDef, field string, v any) any {
	t, asName := r.fieldType(e, field)
	if t == ftUser && isMe(v) && r.me != nil {
		if asName {
			return str(r.me["name"])
		}
		return str(r.me["email"])
	}
	if t == ftKey && r.productsRead {
		if p, _ := r.findProduct(v); p != nil {
			return p.Key
		}
	}
	if e.Name == "statuses" && field == "entity" {
		if s, ok := v.(string); ok {
			if ent := entityWord(strings.TrimSpace(s)); ent != nil {
				return ent.Name
			}
		}
	}
	return v
}

func (r *run) eval(rec *Record, c Cond, e *EntityDef, applied map[Cond]bool) bool {
	if applied[c] {
		return true
	}
	switch x := c.(type) {
	case nil:
		return true
	case *AllOf:
		for _, m := range x.Members {
			if !r.eval(rec, m, e, applied) {
				return false
			}
		}
		return true
	case *AnyOf:
		for _, m := range x.Members {
			if r.eval(rec, m, e, applied) {
				return true
			}
		}
		return false
	case *Negation:
		return !r.eval(rec, x.Member, e, applied)
	case *NullTest:
		nv := noValue(rec.vals[x.Field])
		if x.Negated {
			return !nv
		}
		return nv
	case *InList:
		for _, v := range x.Values {
			if v != nil && r.compare(rec, e, x.Field, "=", v) {
				return true
			}
		}
		return false
	case *Cmp:
		return r.compare(rec, e, x.Field, x.Op, x.Value)
	}
	return false
}

func (r *run) compare(rec *Record, e *EntityDef, field, op string, value any) bool {
	v := r.substitute(e, field, value)
	t, _ := r.fieldType(e, field)
	fv := rec.vals[field]
	spells := rec.spell[field]
	switch op {
	case "=":
		return r.equals(fv, spells, v, t)
	case "!=":
		return !r.equals(fv, spells, v, t)
	case "contains":
		if noValue(fv) {
			return false
		}
		needle := cform(textOf(v))
		if strings.Contains(cform(textOf(fv)), needle) {
			return true
		}
		for _, s := range spells {
			if strings.Contains(cform(textOf(s)), needle) {
				return true
			}
		}
		return false
	case "like":
		if noValue(fv) {
			return false
		}
		pat := cform(textOf(v))
		if likeMatch(pat, cform(textOf(fv))) {
			return true
		}
		for _, s := range spells {
			if likeMatch(pat, cform(textOf(s))) {
				return true
			}
		}
		return false
	}
	if noValue(fv) {
		return false
	}
	var cmpv int
	switch {
	case t == ftNumber:
		a, ok1 := numberOf(fv)
		b, ok2 := numberOf(v)
		if !ok1 || !ok2 {
			return false
		}
		cmpv = cmpFloat(a, b)
	case t == ftDate || t == ftDatetime:
		fs, vs := textOf(fv), strings.TrimSpace(textOf(v))
		if len(vs) == 10 && len(fs) >= 10 {
			fs = fs[:10]
		}
		cmpv = strings.Compare(fs, vs)
	default:
		cmpv = strings.Compare(cform(textOf(fv)), cform(textOf(v)))
	}
	switch op {
	case "<":
		return cmpv < 0
	case "<=":
		return cmpv <= 0
	case ">":
		return cmpv > 0
	case ">=":
		return cmpv >= 0
	}
	return false
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func (r *run) equals(fv any, spells []any, v any, t FieldType) bool {
	if valueEq(fv, v, t) {
		return true
	}
	if noValue(fv) {
		return false
	}
	for _, s := range spells {
		if valueEq(s, v, ftString) {
			return true
		}
	}
	return false
}

// ---- build-rows ----

func groupKey(v any) string {
	switch x := v.(type) {
	case nil:
		return "\x00"
	case string:
		return "s" + cform(x)
	case float64:
		return "n" + fmtFloat(x)
	case bool:
		return "b" + strconv.FormatBool(x)
	}
	return "s" + cform(textOf(v))
}

// orderCmp: numbers before text, numbers by value, text by its comparison
// form. Values without a value are handled by the caller.
func orderCmp(a, b any) int {
	fa, na := a.(float64)
	fb, nb := b.(float64)
	switch {
	case na && nb:
		return cmpFloat(fa, fb)
	case na:
		return -1
	case nb:
		return 1
	}
	return strings.Compare(cform(textOf(a)), cform(textOf(b)))
}

func round2(f float64) float64 {
	if f == math.Trunc(f) {
		return f
	}
	return math.Round(f*100) / 100
}

func aggregate(it SelectItem, recs []*Record) any {
	if it.Fn == "COUNT" && it.Field == "" {
		return float64(len(recs))
	}
	var vals []any
	seen := map[string]bool{}
	for _, rc := range recs {
		v := rc.vals[it.Field]
		if noValue(v) {
			continue
		}
		if it.Distinct {
			k := groupKey(v)
			if seen[k] {
				continue
			}
			seen[k] = true
		}
		vals = append(vals, v)
	}
	switch it.Fn {
	case "COUNT":
		return float64(len(vals))
	case "SUM", "AVG":
		sum, n := 0.0, 0
		for _, v := range vals {
			if f, ok := v.(float64); ok {
				sum += f
				n++
			}
		}
		if n == 0 {
			return nil
		}
		if it.Fn == "AVG" {
			return round2(sum / float64(n))
		}
		return round2(sum)
	case "MIN", "MAX":
		if len(vals) == 0 {
			return nil
		}
		best := vals[0]
		for _, v := range vals[1:] {
			c := orderCmp(v, best)
			if (it.Fn == "MIN" && c < 0) || (it.Fn == "MAX" && c > 0) {
				best = v
			}
		}
		if f, ok := best.(float64); ok {
			return round2(f)
		}
		return best
	}
	return nil
}

type rowsResult struct {
	rows      []Obj
	count     int
	total     *int
	truncated bool
}

func sortRows[T any](items []T, keys []OrderKey, value func(T, OrderKey) any) {
	sort.SliceStable(items, func(i, j int) bool {
		for _, k := range keys {
			a, b := value(items[i], k), value(items[j], k)
			an, bn := noValue(a), noValue(b)
			switch {
			case an && bn:
				continue
			case an:
				return false
			case bn:
				return true
			}
			c := orderCmp(a, b)
			if c == 0 {
				continue
			}
			if k.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
}

func buildRows(st *ReadStmt, recs []*Record, starCustom []*CustomField, listTotal *int, countOnly bool) rowsResult {
	if countOnly {
		row := Obj{{st.Items[0].Label, float64(*listTotal)}}
		return rowsResult{rows: []Obj{row}, count: 1}
	}
	var rows []Obj
	if hasAggregate(st) || len(st.GroupBy) > 0 {
		groupFields := append([]string{}, st.GroupBy...)
		inGroup := map[string]bool{}
		for _, g := range groupFields {
			inGroup[g] = true
		}
		selected := map[string]bool{}
		for _, it := range st.Items {
			if it.Fn == "" {
				selected[it.Field] = true
				if !inGroup[it.Field] {
					groupFields = append(groupFields, it.Field)
					inGroup[it.Field] = true
				}
			}
		}
		type group struct{ recs []*Record }
		var groups []*group
		index := map[string]*group{}
		for _, rc := range recs {
			var kb strings.Builder
			for _, g := range groupFields {
				kb.WriteString(groupKey(rc.vals[g]))
				kb.WriteString("\x01")
			}
			g := index[kb.String()]
			if g == nil {
				g = &group{}
				index[kb.String()] = g
				groups = append(groups, g)
			}
			g.recs = append(g.recs, rc)
		}
		if len(groupFields) == 0 && len(groups) == 0 {
			groups = []*group{{}}
		}
		first := func(g *group, f string) any {
			if len(g.recs) == 0 {
				return nil
			}
			return g.recs[0].vals[f]
		}
		sortRows(groups, st.OrderBy, func(g *group, k OrderKey) any {
			if k.Agg != nil {
				return aggregate(*k.Agg, g.recs)
			}
			return first(g, k.Field)
		})
		for _, g := range groups {
			var row Obj
			for _, gf := range st.GroupBy {
				if !selected[gf] {
					row = append(row, KV{gf, first(g, gf)})
				}
			}
			for _, it := range st.Items {
				if it.Fn != "" {
					row = append(row, KV{it.Label, aggregate(it, g.recs)})
				} else {
					row = append(row, KV{it.Label, first(g, it.Field)})
				}
			}
			rows = append(rows, row)
		}
	} else {
		sorted := append([]*Record{}, recs...)
		sortRows(sorted, st.OrderBy, func(rc *Record, k OrderKey) any { return rc.vals[k.Field] })
		seen := map[string]bool{}
		for _, rc := range sorted {
			var row Obj
			for _, it := range st.Items {
				row = append(row, KV{it.Label, rc.vals[it.Field]})
			}
			if st.Distinct {
				k := toJSON(row)
				if seen[k] {
					continue
				}
				seen[k] = true
			}
			for _, cf := range starCustom {
				if v := rc.vals[cf.Name]; !noValue(v) {
					row = append(row, KV{cf.Name, v})
				}
			}
			rows = append(rows, row)
		}
	}
	res := rowsResult{}
	limit := effectiveLimit(st)
	if len(rows) > limit {
		n := len(rows)
		res.total = &n
		rows = rows[:limit]
	}
	if listTotal != nil {
		t := *listTotal
		res.total = &t
	}
	res.rows = rows
	res.count = len(rows)
	res.truncated = res.total != nil && (st.Limit == nil || *st.Limit > maxLimit)
	return res
}

// ---- read-statement ----

func (r *run) readStatement(st *ReadStmt) (Obj, *AqlError) {
	e := st.Entity
	idf := e.IDField()
	idConj := false
	for _, c := range conjuncts(st.Where) {
		if _, ok := eqConj(c, idf); ok && idf != "" {
			idConj = true
		}
	}
	var notes []string
	// step 1: replace a Star item by fields
	var items []SelectItem
	labels := map[string]bool{}
	hasStar := false
	for _, it := range st.Items {
		if it.Star {
			hasStar = true
		}
	}
	for _, it := range st.Items {
		if !it.Star {
			continue
		}
		var names []string
		if idConj {
			for _, f := range e.Fields {
				names = append(names, f.Name)
			}
		} else {
			names = e.DefaultFields
			var others []string
			def := map[string]bool{}
			for _, d := range e.DefaultFields {
				def[d] = true
			}
			for _, f := range e.Fields {
				if !def[f.Name] {
					others = append(others, f.Name)
				}
			}
			notes = append(notes, fmt.Sprintf("* shows %s. Other fields of %s: %s.", strings.Join(e.DefaultFields, ", "), e.Name, strings.Join(others, ", ")))
		}
		for _, n := range names {
			if !labels[n] {
				labels[n] = true
				items = append(items, SelectItem{Field: n, Label: n})
			}
		}
		break
	}
	for _, it := range st.Items {
		if it.Star {
			continue
		}
		if hasStar && labels[it.Label] {
			continue
		}
		labels[it.Label] = true
		items = append(items, it)
	}
	st.Items = items
	// step 2: the used fields
	used := map[string]bool{}
	for _, it := range st.Items {
		if it.Field != "" {
			used[it.Field] = true
		}
	}
	condFields(st.Where, used)
	for _, g := range st.GroupBy {
		used[g] = true
	}
	for _, o := range st.OrderBy {
		if o.Field != "" {
			used[o.Field] = true
		}
		if o.Agg != nil && o.Agg.Field != "" {
			used[o.Agg.Field] = true
		}
	}
	var custom []*CustomField
	for f := range used {
		if cf := r.custom[f]; cf != nil && strings.HasPrefix(f, "custom.") {
			custom = append(custom, cf)
		}
	}
	sort.Slice(custom, func(i, j int) bool { return custom[i].Name < custom[j].Name })
	// step 3
	fr, err := r.planFetch(st, used, len(custom) > 0)
	if err != nil {
		return nil, err
	}
	// steps 4 and 5
	var kept []*Record
	for _, f := range fr.recs {
		rec := r.convert(e, f, custom, used)
		if r.eval(rec, st.Where, e, fr.applied) {
			kept = append(kept, rec)
		}
	}
	// step 6 a and b: star custom fields
	var starCustom []*CustomField
	if hasStar && idConj && !hasAggregate(st) && len(st.GroupBy) == 0 && len(kept) == 1 && !fr.countOnly {
		cfs, err := r.customFields(e)
		if err != nil {
			return nil, err
		}
		for _, cf := range cfs {
			if !used[cf.Name] {
				starCustom = append(starCustom, cf)
			}
		}
		if len(starCustom) > 0 {
			api := kept[0].api
			if e.Name == "products" {
				id := str(api["id"])
				route := routeWith("/products/{v}", "v", id)
				m, f := r.get(route, "product")
				switch {
				case f == nil:
					api = m
				case f.Kind == failNotFound:
					api = map[string]any{}
				default:
					return nil, r.failErr(f, "GET", route)
				}
			}
			for _, cf := range starCustom {
				kept[0].vals[cf.Name] = nil
				if entry := customEntry(api, cf.Key); entry != nil {
					v, _ := customValue(cf, entry)
					kept[0].vals[cf.Name] = v
				}
			}
		}
	}
	rr := buildRows(st, kept, starCustom, fr.total, fr.countOnly)
	if rr.truncated {
		notes = append(notes, fmt.Sprintf("Showing %d of %d records. Narrow the result with WHERE, raise LIMIT (up to %d), or count with COUNT(*).", rr.count, *rr.total, maxLimit))
	}
	if len(rr.rows) == 0 && !hasAggregate(st) && len(st.GroupBy) == 0 {
		notes = append(notes, r.emptyNotes(st)...)
	}
	out := Obj{{"entity", e.Name}, {"scope", fr.scope}, {"count", rr.count}}
	if rr.total != nil {
		out = append(out, KV{"total", *rr.total})
	}
	if rr.truncated {
		out = append(out, KV{"truncated", true})
	}
	rows := rr.rows
	if rows == nil {
		rows = []Obj{}
	}
	out = append(out, KV{"records", rows})
	if len(notes) > 0 {
		out = append(out, KV{"note", strings.Join(notes, " ")})
	}
	return out, nil
}

func (r *run) emptyNotes(st *ReadStmt) []string {
	e := st.Entity
	var notes []string
	for _, c := range conjuncts(st.Where) {
		if len(notes) >= 2 {
			break
		}
		cmp, ok := c.(*Cmp)
		if !ok {
			continue
		}
		s, ok := cmp.Value.(string)
		if !ok {
			continue
		}
		t := strings.TrimSuffix(strings.TrimPrefix(s, "%"), "%")
		k := refKind(t)
		switch {
		case cmp.Field == e.IDField() && cmp.Op == "=":
			notes = append(notes, fmt.Sprintf("There is no %s with the id %s.", e.Singular, quoteVal(s)))
		case cmp.Field != "name":
		case e.Name == "products" && r.productFound(t) != nil:
			p := r.productFound(t)
			notes = append(notes, fmt.Sprintf("%s is the key of a product, not its name. Use: SELECT key, name FROM products WHERE key = '%s'", quoteVal(s), p.Key))
		case k != "" && k == e.Kind:
			notes = append(notes, fmt.Sprintf("%s looks like an id, not a name. Use: WHERE id = '%s'", quoteVal(s), strings.ToUpper(t)))
		case k != "" && e.HasParentFilter(k):
			notes = append(notes, fmt.Sprintf("%s looks like the id of a %s, not a name. Use: WHERE %s = '%s'", quoteVal(s), k, k, strings.ToUpper(t)))
		case cmp.Op == "=":
			notes = append(notes, fmt.Sprintf("No %s has exactly the name %s. name = compares the whole name; to search inside names use: name CONTAINS %s", e.Singular, quoteVal(s), quoteVal(s)))
		}
	}
	return notes
}

func (r *run) productFound(v string) *Product {
	p, _ := r.findProduct(v)
	return p
}

// ---- describe-statement ----

func typeText(f *FieldDef) string {
	switch f.Type {
	case ftString:
		return "string"
	case ftText:
		return "text"
	case ftNumber:
		return "number"
	case ftBoolean:
		return "boolean"
	case ftKey:
		return "product key"
	case ftUser:
		if f.UserAsName {
			return "user (name)"
		}
		return "user (e-mail address)"
	case ftStatus:
		return "status name"
	case ftDate:
		return "date (YYYY-MM-DD)"
	case ftDatetime:
		return "timestamp"
	case ftReference:
		return "id of " + article(f.RefKind)
	}
	return "string"
}

func (r *run) describe(st *DescribeStmt) (Obj, *AqlError) {
	if st.Entity == nil {
		var ents []Obj
		for _, e := range catalog {
			ents = append(ents, Obj{{"entity", e.Name}, {"about", withKey(e.About)}})
		}
		return Obj{{"entities", ents}, {"note", "DESCRIBE entity lists the fields of one entity, for example: DESCRIBE features"}}, nil
	}
	e := st.Entity
	out := Obj{{"entity", e.Name}, {"about", withKey(e.About)}}
	var fields []Obj
	for i := range e.Fields {
		f := &e.Fields[i]
		o := Obj{{"name", f.Name}, {"type", typeText(f)}}
		switch f.Writable {
		case wYes:
			o = append(o, KV{"writable", true})
		case wInsertOnly:
			o = append(o, KV{"writable", "INSERT only"})
		}
		fields = append(fields, o)
	}
	out = append(out, KV{"fields", fields})
	cfs, err := r.customFields(e)
	if err != nil {
		return nil, err
	}
	if len(cfs) > 0 {
		var l []Obj
		for _, cf := range cfs {
			o := Obj{{"name", cf.Name}, {"label", cf.Label}, {"type", customKindText[cf.Kind]}}
			if cf.Kind != "scorecard" && e.CanUpdate {
				o = append(o, KV{"writable", true})
			}
			l = append(l, o)
		}
		out = append(out, KV{"custom_fields", l})
		idf := e.IDField()
		kind := e.Kind
		out = append(out, KV{"custom_fields_note", fmt.Sprintf("* shows custom fields only with %s = '...', and only those that have a value. To select one by name: SELECT %s, %s FROM %s WHERE %s = '%s'",
			idf, idf, cfs[0].Name, e.Name, idf, exampleID(kind))})
	}
	if len(e.ParentFilters) > 0 {
		var l []string
		for _, p := range e.ParentFilters {
			l = append(l, p+" = '...'")
		}
		out = append(out, KV{"filter_by_parent", l})
	}
	if e.HasField("status") {
		out = append(out, KV{"status_values", fmt.Sprintf("SELECT name FROM statuses WHERE product = '%s' AND entity = '%s'", keyMark, e.Name)})
	}
	if !e.HasOperation() {
		out = append(out, KV{"change_with_execute", "read-only"})
	} else {
		var l []string
		if e.CanCreate {
			what := "the id of the commented record"
			switch {
			case e.ParentKind == "product":
				what = "a product key"
			case e.ParentKind != "any":
				what = "the id of " + article(e.ParentKind)
			}
			l = append(l, fmt.Sprintf("INSERT INTO %s (parent, ...) VALUES (...): parent is %s", e.Name, what))
		}
		if e.CanUpdate {
			l = append(l, fmt.Sprintf("UPDATE %s SET field = 'value' WHERE id = '...'", e.Name))
		}
		if e.CanDelete {
			l = append(l, fmt.Sprintf("DELETE FROM %s WHERE id = '...'", e.Name))
		}
		out = append(out, KV{"change_with_execute", l})
	}
	out = append(out, KV{"example", exampleOf(e)})
	return out, nil
}
