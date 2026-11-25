// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30

package main

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
	"unicode"
	"unicode/utf8"
)

// record is the output of convert-record.
type record struct {
	vals  map[string]any        // every field: nil, string, float64 or bool
	alts  map[string][]string   // other spellings, accepted by conditions
	multi map[string][][]string // the single values of a field with several values
	api   apiRec
}

// ---- convert-record ----

var (
	reBlockEnd = regexp.MustCompile(`(?i)</\s*(p|li|h[1-6]|div|tr|blockquote|pre|ul|ol|table|section|article|header|footer|dd|dt)\s*>|<\s*(br|hr)\s*/?\s*>`)
	reTag      = regexp.MustCompile(`(?s)<[^>]*>`)
	reSpaces   = regexp.MustCompile(`[ \t\f\v\x{00a0}]+`)
)

// htmlText turns HTML into plain text (convert-record, step 2 a).
func htmlText(s string) any {
	s = reBlockEnd.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		l = strings.TrimSpace(reSpaces.ReplaceAllString(l, " "))
		if l != "" {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return strings.Join(out, "\n")
}

func numberOf(v any) any {
	if f, ok := toNumber(v); ok {
		return f
	}
	return nil
}

func plain(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		return x
	case bool:
		return x
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return f
		}
		return x.String()
	case float64:
		return x
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func listOf(v any) ([]any, bool) {
	l, ok := v.([]any)
	return l, ok
}

func (sc *stmtCtx) convert(e *EntityDef, api apiRec, knownProduct, knownParent string, used map[string]bool, customs []*CustomField) (*record, *AqlError) {
	r := &record{vals: map[string]any{}, alts: map[string][]string{}, multi: map[string][][]string{}, api: api}
	for _, fd := range e.Fields {
		raw := lookup(api, fd.Source)
		var v any
		switch fd.Type {
		case tText:
			if m, ok := raw.(map[string]any); ok {
				raw = m["body"]
			}
			if raw != nil {
				v = htmlText(strOf(raw))
			}
		case tDate:
			s := strOf(raw)
			if len(s) >= 10 {
				s = s[:10]
			}
			if s != "" {
				v = s
			}
		case tBoolean:
			if b, ok := truth(raw); ok && raw != nil {
				v = b
			} else if fd.FalseDefault {
				v = false
			}
		case tNumber:
			v = numberOf(raw)
		case tKey:
			switch {
			case e.Name == "products" && fd.Name == "key":
				if raw != nil {
					v = strOf(raw)
				}
			case fd.Name == "product" && knownProduct != "":
				v = knownProduct
			case used[fd.Name] && raw != nil:
				if _, err := sc.productList(false, false); err != nil {
					return nil, err
				}
				if p := sc.productByID(strOf(raw)); p != nil {
					v = p.Key
				}
			}
		default:
			if l, ok := listOf(raw); ok && fd.Name == "tags" {
				var tags []string
				var items [][]string
				for _, t := range l {
					tags = append(tags, strOf(t))
					items = append(items, []string{strOf(t)})
				}
				if len(tags) > 0 {
					v = strings.Join(tags, ", ")
				}
				r.multi[fd.Name] = items
			} else if raw != nil {
				v = plain(raw)
				if _, isStr := v.(string); !isStr && fd.Type != tString {
					v = strOf(raw)
				}
			}
		}
		if e.Name == "comments" && fd.Name == "parent" {
			if s, ok := v.(string); ok && s != "" {
				v = s[strings.LastIndex(s, "/")+1:]
			} else if knownParent != "" {
				v = knownParent
			}
		}
		if e.Name == "requirements" && fd.Name == "feature" && noValue(v) && knownParent != "" {
			v = knownParent
		}
		r.vals[fd.Name] = v
	}
	// other spellings (step 5)
	addAlt := func(f string, vs ...any) {
		for _, x := range vs {
			if s := strOf(x); s != "" {
				r.alts[f] = append(r.alts[f], s)
			}
		}
	}
	if id := e.idField(); id != "" {
		addAlt(id, api["id"])
	}
	if e.field("product") != nil {
		addAlt("product", api["product_id"])
	}
	for _, fd := range e.Fields {
		switch {
		case fd.Name == "assigned_to":
			addAlt(fd.Name, lookup(api, "assigned_to_user.name"), lookup(api, "assigned_to_user.id"))
		case fd.Type == tReference:
			if i := strings.Index(fd.Source, "."); i >= 0 {
				obj := fd.Source[:i]
				addAlt(fd.Name, lookup(api, obj+".id"), lookup(api, obj+".name"))
			} else {
				obj := strings.TrimSuffix(fd.Source, "_reference_num")
				addAlt(fd.Name, lookup(api, obj+".id"), lookup(api, obj+".name"), api[obj+"_id"])
			}
		case e.Name == "comments" && fd.Name == "user":
			addAlt(fd.Name, lookup(api, "user.email"), lookup(api, "user.id"))
		case e.Name == "comments" && fd.Name == "parent":
			addAlt(fd.Name, lookup(api, "commentable.id"))
		}
	}
	for _, cf := range customs {
		sc.convertCustom(r, cf, api)
	}
	return r, nil
}

// customEntry is the entry of a custom field in an API record.
func customEntry(api apiRec, key string) map[string]any {
	list, ok := api["custom_fields"].([]any)
	if !ok {
		return nil
	}
	for _, e := range list {
		m, ok := e.(map[string]any)
		if ok && strings.EqualFold(strOf(m["key"]), key) {
			return m
		}
	}
	return nil
}

func emptyValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok && s == "" {
		return true
	}
	if l, ok := v.([]any); ok && len(l) == 0 {
		return true
	}
	return false
}

// convertCustom takes the value of a custom field (convert-record, step 4).
func (sc *stmtCtx) convertCustom(r *record, cf *CustomField, api apiRec) {
	r.vals[cf.Name] = nil
	entry := customEntry(api, cf.Key)
	if entry == nil || emptyValue(entry["value"]) {
		return
	}
	val := entry["value"]
	texts := func(v any) ([]string, bool) {
		l, ok := v.([]any)
		if !ok {
			return nil, false
		}
		var out []string
		for _, x := range l {
			out = append(out, strOf(x))
		}
		return out, true
	}
	switch cf.kind() {
	case "text", "url", "choice", "fixed_choice":
		r.vals[cf.Name] = textOrJSON(val)
	case "note":
		if s, ok := val.(string); ok {
			r.vals[cf.Name] = htmlText(s)
		} else {
			r.vals[cf.Name] = textOrJSON(val)
		}
	case "number", "scorecard":
		if f, ok := toNumber(val); ok {
			r.vals[cf.Name] = f
		} else {
			r.vals[cf.Name] = textOrJSON(val)
		}
	case "date":
		if s, ok := val.(string); ok {
			if len(s) >= 10 {
				s = s[:10]
			}
			r.vals[cf.Name] = s
		} else {
			r.vals[cf.Name] = textOrJSON(val)
		}
	case "tags", "fixed_tags":
		if l, ok := texts(val); ok {
			r.vals[cf.Name] = strings.Join(l, ", ")
			for _, t := range l {
				r.multi[cf.Name] = append(r.multi[cf.Name], []string{t})
			}
		} else {
			r.vals[cf.Name] = textOrJSON(val)
		}
	case "users":
		ids, _ := texts(val)
		emails, hasEmails := texts(entry["email_value"])
		var full []map[string]any
		if l, ok := entry["full_value"].([]any); ok {
			for _, x := range l {
				if m, ok := x.(map[string]any); ok {
					full = append(full, m)
				}
			}
		}
		if !hasEmails {
			if full != nil {
				emails = nil
				for _, m := range full {
					emails = append(emails, strOf(m["email"]))
				}
			} else if ids != nil {
				emails = ids
			} else {
				r.vals[cf.Name] = textOrJSON(val)
				return
			}
		}
		if len(emails) == 0 {
			return
		}
		r.vals[cf.Name] = strings.Join(emails, ", ")
		var namesL, idsL []string
		for i, em := range emails {
			item := []string{em}
			var m map[string]any
			if i < len(full) {
				m = full[i]
			}
			for _, fm := range full {
				if strings.EqualFold(strOf(fm["email"]), em) {
					m = fm
				}
			}
			if m != nil {
				item = append(item, strOf(m["name"]), strOf(m["id"]))
				namesL = append(namesL, strOf(m["name"]))
				idsL = append(idsL, strOf(m["id"]))
			} else if i < len(ids) {
				item = append(item, ids[i])
				idsL = append(idsL, ids[i])
			}
			r.multi[cf.Name] = append(r.multi[cf.Name], item)
		}
		if len(namesL) > 0 {
			r.alts[cf.Name] = append(r.alts[cf.Name], strings.Join(namesL, ", "))
		}
		if len(idsL) > 0 {
			r.alts[cf.Name] = append(r.alts[cf.Name], strings.Join(idsL, ", "))
		}
	}
}

func textOrJSON(v any) any {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// ---- evaluate-condition ----

func isMeWord(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "me", "myself", "current_user":
		return true
	}
	return false
}

// splitParts splits a text at commas; empty parts are left out.
func splitParts(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func conjuncts(c Cond) []Cond {
	if c == nil {
		return nil
	}
	if a, ok := c.(*AllOf); ok {
		return a.M
	}
	return []Cond{c}
}

// prepareLookups reads the current user and the product list where the
// condition needs them (plan-fetch, step 9).
func (sc *stmtCtx) prepareLookups(e *EntityDef, c Cond, applied map[Cond]bool) *AqlError {
	var err *AqlError
	var walk func(c Cond, top bool)
	check := func(field string, v Value, isApplied bool) {
		if err != nil || v.kind != vText {
			return
		}
		ft := sc.fieldType(e, field)
		if ft == tUser {
			need := isMeWord(v.s)
			if sc.multiValued(e, field) {
				for _, p := range splitParts(v.s) {
					if isMeWord(p) {
						need = true
					}
				}
			}
			if need {
				_, err = sc.currentUser()
			}
		}
		if ft == tKey && !isApplied {
			_, err = sc.productList(false, false)
		}
	}
	walk = func(c Cond, top bool) {
		switch x := c.(type) {
		case *AllOf:
			for _, m := range x.M {
				walk(m, top)
			}
		case *AnyOf:
			for _, m := range x.M {
				walk(m, false)
			}
		case *Negation:
			walk(x.M, false)
		case *Cmp:
			check(x.Field, x.Val, applied[c])
		case *InList:
			for _, v := range x.Vals {
				check(x.Field, v, applied[c])
			}
		}
	}
	walk(c, true)
	return err
}

func (sc *stmtCtx) meValue(e *EntityDef, field string) string {
	if sc.me == nil {
		return ""
	}
	if e.Name == "comments" && field == "user" {
		return strOf(sc.me["name"])
	}
	if fd := e.field(field); fd != nil && fd.AsName {
		return strOf(sc.me["name"])
	}
	return strOf(sc.me["email"])
}

// replaceValue applies step 4 of evaluate-condition.
func (sc *stmtCtx) replaceValue(e *EntityDef, field string, v any) any {
	s, isText := v.(string)
	if !isText {
		if f, ok := v.(float64); ok {
			s, isText = fmtNum(f), sc.fieldType(e, field) == tKey
			if !isText {
				return v
			}
		} else {
			return v
		}
	}
	ft := sc.fieldType(e, field)
	switch {
	case ft == tUser && isMeWord(s) && sc.me != nil:
		return sc.meValue(e, field)
	case ft == tKey:
		if p := matchProduct(sc.products, s); p != nil {
			return p.Key
		}
	case e.Name == "statuses" && field == "entity":
		if en := resolveEntity(s); en != nil {
			return en.Name
		}
	}
	return v
}

func utcDay(v any) string {
	s := textOf(v)
	if t, ok := parseTimestamp(s); ok {
		return t.UTC().Format("2006-01-02")
	}
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func day(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// eq is the operator = (evaluate-condition, steps 6 and 11).
func (sc *stmtCtx) eq(e *EntityDef, r *record, field string, v Value) bool {
	val := r.vals[field]
	if noValue(val) {
		return false
	}
	ft := sc.fieldType(e, field)
	if sc.multiValued(e, field) && v.kind == vText {
		items := r.multi[field]
		isUsers := ft == tUser
		match := func(part string) bool {
			if isUsers && isMeWord(part) && sc.me != nil {
				part = strOf(sc.me["email"])
			}
			for _, it := range items {
				for _, sp := range it {
					if textEq(sp, part) {
						return true
					}
				}
			}
			return false
		}
		if match(v.s) {
			return true
		}
		if strings.Contains(v.s, ",") {
			parts := splitParts(v.s)
			if len(parts) == 0 {
				return false
			}
			for _, p := range parts {
				if !match(p) {
					return false
				}
			}
			return true
		}
		return false
	}
	sv := sc.replaceValue(e, field, v.raw())
	if ft == tBoolean || v.kind == vBool {
		a, ok1 := truth(val)
		b, ok2 := truth(sv)
		return ok1 && ok2 && a == b
	}
	switch ft {
	case tNumber:
		a, ok1 := toNumber(val)
		b, ok2 := toNumber(sv)
		if ok1 && ok2 {
			return a == b
		}
	case tDate:
		return day(textOf(val)) == day(textOf(sv))
	case tDatetime:
		s := textOf(sv)
		if validDay(s) {
			return utcDay(val) == s
		}
		if t2, ok := parseTimestamp(s); ok {
			if t1, ok := parseTimestamp(textOf(val)); ok {
				return t1.Equal(t2)
			}
		}
	}
	if textEq(textOf(val), textOf(sv)) {
		return true
	}
	for _, a := range r.alts[field] {
		if textEq(a, textOf(sv)) {
			return true
		}
	}
	return false
}

func likeMatch(pattern, text string) bool {
	p, t := []rune(pattern), []rune(text)
	var m func(i, j int) bool
	memo := map[[2]int]bool{}
	seen := map[[2]int]bool{}
	m = func(i, j int) bool {
		k := [2]int{i, j}
		if seen[k] {
			return memo[k]
		}
		seen[k] = true
		var res bool
		switch {
		case i == len(p):
			res = j == len(t)
		case p[i] == '%':
			res = m(i+1, j) || (j < len(t) && m(i, j+1))
		case j < len(t) && (p[i] == '_' || p[i] == t[j]):
			res = m(i+1, j+1)
		}
		memo[k] = res
		return res
	}
	return m(0, 0)
}

func (sc *stmtCtx) evalCmp(e *EntityDef, r *record, c *Cmp) bool {
	val := r.vals[c.Field]
	switch c.Op {
	case "=":
		return sc.eq(e, r, c.Field, c.Val)
	case "!=":
		return !sc.eq(e, r, c.Field, c.Val)
	case "contains", "like":
		if noValue(val) || c.Val.kind == vNull {
			return false
		}
		sv := textOf(sc.replaceValue(e, c.Field, c.Val.raw()))
		texts := append([]string{textOf(val)}, r.alts[c.Field]...)
		if c.Op == "contains" {
			for _, t := range texts {
				if strings.Contains(compForm(t), compForm(sv)) {
					return true
				}
			}
			return false
		}
		return likeMatch(compForm(sv), compForm(textOf(val)))
	}
	if noValue(val) || c.Val.kind == vNull {
		return false
	}
	sv := sc.replaceValue(e, c.Field, c.Val.raw())
	cmp := 0
	ft := sc.fieldType(e, c.Field)
	done := false
	switch ft {
	case tNumber:
		a, ok1 := toNumber(val)
		b, ok2 := toNumber(sv)
		if ok1 && ok2 {
			cmp, done = cmpFloat(a, b), true
		}
	case tDate:
		cmp, done = strings.Compare(day(textOf(val)), day(textOf(sv))), true
	case tDatetime:
		s := textOf(sv)
		if validDay(s) {
			if _, ok := parseTimestamp(textOf(val)); ok {
				cmp, done = strings.Compare(utcDay(val), s), true
			}
		} else if t2, ok := parseTimestamp(s); ok {
			if t1, ok := parseTimestamp(textOf(val)); ok {
				cmp, done = t1.Compare(t2), true
			}
		}
		if !done {
			cmp, done = strings.Compare(textOf(val), s), true
		}
	}
	if !done {
		cmp = orderTexts(textOf(val), textOf(sv))
	}
	switch c.Op {
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
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

func (sc *stmtCtx) eval(e *EntityDef, r *record, c Cond) bool {
	switch x := c.(type) {
	case nil:
		return true
	case *AllOf:
		for _, m := range x.M {
			if !sc.eval(e, r, m) {
				return false
			}
		}
		return true
	case *AnyOf:
		for _, m := range x.M {
			if sc.eval(e, r, m) {
				return true
			}
		}
		return false
	case *Negation:
		return !sc.eval(e, r, x.M)
	case *NullTest:
		return noValue(r.vals[x.Field]) != x.Negated
	case *InList:
		for _, v := range x.Vals {
			if v.kind == vNull || (v.kind == vText && v.s == "") {
				if noValue(r.vals[x.Field]) {
					return true
				}
				continue
			}
			if sc.eq(e, r, x.Field, v) {
				return true
			}
		}
		return false
	case *Cmp:
		return sc.evalCmp(e, r, x)
	}
	return false
}

// keep evaluates the WHERE condition; an applied conjunct counts as true.
func (sc *stmtCtx) keep(e *EntityDef, r *record, where Cond, applied map[Cond]bool) bool {
	if where == nil {
		return true
	}
	if applied[where] {
		return true
	}
	if a, ok := where.(*AllOf); ok {
		for _, m := range a.M {
			if applied[m] {
				continue
			}
			if !sc.eval(e, r, m) {
				return false
			}
		}
		return true
	}
	return sc.eval(e, r, where)
}

// ---- order (build-rows, step 4) ----

// orderTexts compares two texts by their comparison form, with runs of
// digits compared as numbers.
func orderTexts(a, b string) int {
	x, y := []rune(compForm(a)), []rune(compForm(b))
	i, j := 0, 0
	for i < len(x) && j < len(y) {
		if unicode.IsDigit(x[i]) && unicode.IsDigit(y[j]) {
			si := i
			for i < len(x) && unicode.IsDigit(x[i]) {
				i++
			}
			sj := j
			for j < len(y) && unicode.IsDigit(y[j]) {
				j++
			}
			na := strings.TrimLeft(string(x[si:i]), "0")
			nb := strings.TrimLeft(string(y[sj:j]), "0")
			if len(na) != len(nb) {
				if len(na) < len(nb) {
					return -1
				}
				return 1
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
			continue
		}
		if x[i] != y[j] {
			if x[i] < y[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case i < len(x):
		return 1
	case j < len(y):
		return -1
	}
	return strings.Compare(a, b)
}

func isNum(v any) bool { _, ok := v.(float64); return ok }

// orderValues: numbers before text, numbers by value, text by orderTexts.
func orderValues(a, b any) int {
	an, bn := isNum(a), isNum(b)
	switch {
	case an && bn:
		return cmpFloat(a.(float64), b.(float64))
	case an:
		return -1
	case bn:
		return 1
	}
	return orderTexts(textOf(a), textOf(b))
}

// ---- plan-fetch ----

type fetched struct {
	api    apiRec
	prod   string
	parent string
	pre    *record
}

type fetchResult struct {
	recs      []fetched
	scope     string
	total     int
	hasTotal  bool
	applied   map[Cond]bool
	countOnly bool
	idConj    Cond   // conjunct whose id route was read
	idFound   string // id of the record that the id route read
	groups    []statusGroup
	product   *Product
}

type target struct {
	field  string
	route  string
	scope  string
	prod   string
	parent string
	single bool // an ApiGet by id
	kind   string
	value  string
	singul string
}

func (sc *stmtCtx) eqConj(conj []Cond, field string) *Cmp {
	for _, c := range conj {
		if x, ok := c.(*Cmp); ok && x.Field == field && x.Op == "=" && (x.Val.kind == vText || x.Val.kind == vNumber) {
			return x
		}
	}
	return nil
}

var commentParentKinds = map[string]bool{"feature": true, "release": true, "epic": true, "idea": true, "initiative": true, "goal": true, "requirement": true}

// resolveTarget checks one value for one route (plan-fetch, step 3).
func (sc *stmtCtx) resolveTarget(e *EntityDef, field string, v string, conj []Cond) (*target, *AqlError) {
	up := strings.ToUpper(v)
	r, _ := e.route(field)
	switch {
	case field == "id":
		ok := isNumericID(v) || (e.Name != "comments" && refKind(v) == e.Kind)
		if !ok {
			return nil, sc.wrongKind(e, field, v, conj)
		}
		return &target{field: field, route: strings.ReplaceAll(r.Path, "{v}", esc(up)), single: true, value: up}, nil
	case field == "product":
		p, err := sc.resolveProduct(v, e, true)
		if err != nil {
			return nil, err
		}
		return &target{field: field, route: strings.ReplaceAll(r.Path, "{v}", esc(p.ID)), scope: "product " + p.Key, prod: p.Key, value: p.Key, kind: "product"}, nil
	case e.Name == "comments" && field == "parent":
		if k := refKind(v); commentParentKinds[k] {
			ke := entityOfKind(k)
			return &target{field: field, route: "/" + ke.Name + "/" + esc(up) + "/comments", scope: k + " " + up, parent: up, value: up, kind: k}, nil
		}
		p, err := sc.findProductQuiet(v)
		if err != nil {
			return nil, err
		}
		if p != nil {
			return &target{field: field, route: "/products/" + esc(p.ID) + "/comments", scope: "product " + p.Key, parent: p.Key, prod: p.Key, value: p.Key, kind: "product"}, nil
		}
		return nil, sc.errK08(v)
	}
	fd := e.field(field)
	kind := ""
	if fd != nil {
		kind = fd.RefKind
	}
	if !(isNumericID(v) || (kind != "" && refKind(v) == kind)) {
		return nil, sc.wrongKind(e, field, v, conj)
	}
	return &target{field: field, route: strings.ReplaceAll(r.Path, "{v}", esc(up)), scope: field + " " + up, parent: up, value: up, kind: kind}, nil
}

// wrongKind gives the message of plan-fetch, step 4.
func (sc *stmtCtx) wrongKind(e *EntityDef, field, v string, conj []Cond) *AqlError {
	kind := refKind(v)
	up := strings.ToUpper(v)
	expected := e.Kind
	if field != "id" {
		if fd := e.field(field); fd != nil {
			expected = fd.RefKind
		}
	}
	if expected == "" {
		expected = singularOf(e)
	}
	if kind != "" && field == "id" {
		for _, rt := range e.ReadRoutes {
			if fd := e.field(rt.Field); fd != nil && fd.RefKind == kind {
				return aqlErr("wrong_kind", "%s is the id of %s %s, not of %s %s. For the %s of that %s: SELECT * FROM %s WHERE %s = '%s'",
					quote(v), article(kind), kind, article(singularOf(e)), singularOf(e), e.Name, kind, e.Name, rt.Field, up)
			}
		}
	}
	if kind != "" {
		ke := entityOfKind(kind)
		return aqlErr("wrong_kind", "%s is the id of %s %s, not of %s %s. To read it: SELECT * FROM %s WHERE id = '%s'",
			quote(v), article(kind), kind, article(expected), expected, ke.Name, up)
	}
	p, err := sc.findProductQuiet(v)
	if err != nil {
		return err
	}
	if p != nil && e.hasParentFilter("product") {
		return aqlErr("wrong_kind", "%s is a product, not %s %s. Use: WHERE product = '%s'", quote(v), article(expected), expected, p.Key)
	}
	if field == "id" {
		return sc.errK04(v, e)
	}
	fe := entityOfKind(expected)
	if field == "release" || field == "initiative" || field == "epic" {
		if pc := sc.eqConj(conj, "product"); pc != nil {
			if pp, err := sc.findProductQuiet(pc.Val.written()); err != nil {
				return err
			} else if pp != nil {
				list, err := sc.recordsOf(fe, pp)
				if err != nil {
					return err
				}
				return aqlErr("wrong_kind", "%s = needs the id of the %s, not its name. The %s of %s: %s", field, field, fe.Name, pp.Key, list)
			}
		}
	}
	name := expected + "s"
	if fe != nil {
		name = fe.Name
	}
	return aqlErr("wrong_kind", "%s = needs the id of the %s, for example '%s', not its name. To find the id: SELECT id, name FROM %s WHERE product = '%s'",
		field, field, sc.exampleID(expected), name, sc.key())
}

// recordsOf reads the records of an entity in a product with one list
// request and gives <records>.
func (sc *stmtCtx) recordsOf(fe *EntityDef, p *Product) (string, *AqlError) {
	params := url.Values{}
	params.Set("page", "1")
	params.Set("per_page", strconv.Itoa(pageSize))
	params.Set("fields", sourceMembers(fe, map[string]bool{}, false))
	lp, f := sc.c.apiList(sc.ctx, sc.acct, "/products/"+esc(p.ID)+"/"+fe.Name, params)
	if f != nil {
		return "", sc.failErr(f)
	}
	var items []string
	for _, r := range lp.records {
		items = append(items, fmt.Sprintf("%s (%s)", strOf(lookup(r, fe.field("id").Source)), strOf(r["name"])))
	}
	if len(items) == 0 {
		return "none", nil
	}
	return listMax(items), nil
}

type listMode struct {
	countOnly bool
	early     bool
	enough    int
	account   bool
}

// readList reads a list page by page (plan-fetch, step 8).
func (sc *stmtCtx) readList(e *EntityDef, route string, used map[string]bool, custom bool, q string, mode listMode, st *ReadStmt, conj []Cond) ([]apiRec, int, bool, bool, *AqlError) {
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(pageSize))
	params.Set("fields", sourceMembers(e, used, custom))
	if q != "" {
		params.Set("q", q)
	}
	var out []apiRec
	for page := 1; ; page++ {
		params.Set("page", strconv.Itoa(page))
		lp, f := sc.c.apiList(sc.ctx, sc.acct, route, params)
		if f != nil {
			return nil, 0, false, false, listFail(sc, f)
		}
		if page == 1 && custom && len(lp.records) > 0 && !anyHasMember(lp.records, "custom_fields") {
			return nil, 0, false, false, sc.errX05(e)
		}
		if page == 1 && mode.countOnly {
			return nil, lp.totalRecords, true, true, nil
		}
		out = append(out, lp.records...)
		if mode.early && len(out) >= mode.enough {
			if page < lp.totalPages {
				return out, lp.totalRecords, true, false, nil
			}
			return out, 0, false, false, nil
		}
		if page == 1 && lp.totalPages > maxPages {
			return nil, 0, false, false, aqlErr("scope", "this would read %d records. %s", lp.totalRecords, sc.narrowHint(e, mode.account, st, conj))
		}
		if page >= lp.totalPages || page >= maxPages {
			return out, 0, false, false, nil
		}
	}
}

// listFail maps a failed list request: NotFound is K07 for the caller.
func listFail(sc *stmtCtx, f *apiFailure) *AqlError {
	if f.kind == fNotFound {
		return &AqlError{Kind: "_notfound"}
	}
	return sc.failErr(f)
}

func condUses(c Cond, field string) bool {
	switch x := c.(type) {
	case *AllOf:
		for _, m := range x.M {
			if condUses(m, field) {
				return true
			}
		}
	case *AnyOf:
		for _, m := range x.M {
			if condUses(m, field) {
				return true
			}
		}
	case *Negation:
		return condUses(x.M, field)
	case *Cmp:
		return x.Field == field
	case *NullTest:
		return x.Field == field
	case *InList:
		return x.Field == field
	}
	return false
}

func (sc *stmtCtx) narrowHint(e *EntityDef, account bool, st *ReadStmt, conj []Cond) string {
	if account {
		return "Narrow it down, for example with product = '" + sc.key() + "'"
	}
	for _, pf := range e.ParentFilters {
		if pf == "product" || condUses(st.Where, pf) {
			continue
		}
		kind := "feature"
		if fd := e.field(pf); fd != nil && fd.RefKind != "" {
			kind = fd.RefKind
		}
		return "Narrow it down, for example with " + pf + " = '" + sc.exampleID(kind) + "'"
	}
	return "Use LIMIT without ORDER BY, GROUP BY and DISTINCT to read the first records, or COUNT(*) alone to count them."
}

func (st *ReadStmt) hasAggregate() bool {
	for _, it := range st.Items {
		if it.aggregate() {
			return true
		}
	}
	return false
}

func (st *ReadStmt) limit() int {
	l := defaultLimit
	if st.HasLimit {
		l = st.Limit
	}
	if l > maxLimit {
		l = maxLimit
	}
	return l
}

func (sc *stmtCtx) readTarget(e *EntityDef, t *target, used map[string]bool, custom bool, mode listMode, st *ReadStmt, conj []Cond) ([]fetched, int, bool, bool, string, *AqlError) {
	if t.single {
		rec, f := sc.c.apiGet(sc.ctx, sc.acct, t.route, e.Singular)
		if f != nil {
			if f.kind == fNotFound {
				return nil, 0, false, false, singularOf(e) + " " + t.value, nil
			}
			return nil, 0, false, false, "", sc.failErr(f)
		}
		id := strOf(lookup(rec, e.field("id").Source))
		if id == "" {
			id = t.value
		}
		return []fetched{{api: rec}}, 0, false, false, singularOf(e) + " " + id, nil
	}
	recs, total, hasTotal, countOnly, err := sc.readList(e, t.route, used, custom, "", mode, st, conj)
	if err != nil {
		if err.Kind == "_notfound" {
			kind := t.kind
			if kind == "" {
				kind = t.field
			}
			return nil, 0, false, false, "", sc.errK07(kind, t.value)
		}
		return nil, 0, false, false, "", err
	}
	var out []fetched
	for _, r := range recs {
		out = append(out, fetched{api: r, prod: t.prod, parent: t.parent})
	}
	return out, total, hasTotal, countOnly, t.scope, nil
}

func (sc *stmtCtx) planFetch(st *ReadStmt, used map[string]bool, custom bool) (*fetchResult, *AqlError) {
	e := st.E
	conj := conjuncts(st.Where)
	fr := &fetchResult{applied: map[Cond]bool{}}
	countOnlyItems := len(st.Items) == 1 && st.Items[0].CountStar && len(st.GroupBy) == 0
	allApplied := func() bool {
		for _, c := range conj {
			if !fr.applied[c] {
				return false
			}
		}
		return true
	}
	mode := func() listMode {
		m := listMode{enough: st.limit()}
		if allApplied() {
			m.countOnly = countOnlyItems
			m.early = len(st.OrderBy) == 0 && len(st.GroupBy) == 0 && !st.Distinct && !st.hasAggregate()
		}
		return m
	}
	switch e.FixedSource {
	case "me":
		rec, f := sc.c.apiGet(sc.ctx, sc.acct, "/me", "user")
		if f != nil {
			return nil, sc.failErr(f)
		}
		fr.recs = []fetched{{api: rec}}
		fr.scope = "current user"
		return fr, nil
	case "products":
		prods, err := sc.productList(used["description"], custom)
		if err != nil {
			return nil, err
		}
		for _, p := range prods {
			fr.recs = append(fr.recs, fetched{api: p.rec})
		}
		fr.scope = "all products"
		return fr, nil
	case "users":
		recs, total, hasTotal, countOnly, err := sc.readList(e, "/users", used, false, "", mode(), st, conj)
		if err != nil {
			if err.Kind == "_notfound" {
				return nil, aqlErr("api_error", "aha: API error (status 404) on GET /users: (no text)")
			}
			return nil, err
		}
		for _, r := range recs {
			fr.recs = append(fr.recs, fetched{api: r})
		}
		fr.total, fr.hasTotal, fr.countOnly = total, hasTotal, countOnly
		fr.scope = "all users"
		return fr, nil
	case "statuses":
		pc := sc.eqConj(conj, "product")
		if pc == nil {
			return nil, sc.errC01(e)
		}
		p, err := sc.resolveProduct(pc.Val.written(), e, true)
		if err != nil {
			return nil, err
		}
		groups, err := sc.productStatuses(p.ID)
		if err != nil {
			return nil, err
		}
		fr.groups, fr.product = groups, p
		fr.applied[pc] = true
		for _, g := range groups {
			for _, s := range g.names {
				r := &record{vals: map[string]any{}, alts: map[string][]string{}, multi: map[string][][]string{}}
				r.vals["name"] = plain(s["name"])
				r.vals["entity"] = g.entity
				r.vals["position"] = numberOf(s["position"])
				if b, ok := s["complete"].(bool); ok {
					r.vals["complete"] = b
				} else {
					r.vals["complete"] = false
				}
				r.vals["product"] = p.Key
				if en := entityByName(g.entity); en != nil {
					r.alts["entity"] = append(r.alts["entity"], en.Singular)
				}
				r.alts["entity"] = append(r.alts["entity"], g.prefix)
				fr.recs = append(fr.recs, fetched{pre: r})
			}
		}
		fr.scope = "product " + p.Key
		return fr, nil
	}
	// step 2: route by an id or a parent
	for _, rt := range e.ReadRoutes {
		if rt.Field == "product" {
			continue
		}
		c := sc.eqConj(conj, rt.Field)
		if c == nil {
			continue
		}
		t, err := sc.resolveTarget(e, rt.Field, c.Val.written(), conj)
		if err != nil {
			return nil, err
		}
		fr.applied[c] = true
		recs, total, hasTotal, countOnly, scope, err := sc.readTarget(e, t, used, custom, mode(), st, conj)
		if err != nil {
			return nil, err
		}
		if rt.Field == "id" {
			fr.idConj = c
			if len(recs) > 0 {
				fr.idFound = strOf(lookup(recs[0].api, e.field("id").Source))
			}
		}
		fr.recs, fr.total, fr.hasTotal, fr.countOnly, fr.scope = recs, total, hasTotal, countOnly, scope
		return fr, nil
	}
	// step 5: route by a list of values
	for _, c := range conj {
		in, ok := c.(*InList)
		if !ok || len(in.Vals) > maxInValues {
			continue
		}
		if _, has := e.route(in.Field); !has {
			continue
		}
		usable := true
		for _, v := range in.Vals {
			if v.kind == vNull || v.kind == vBool || (v.kind == vText && v.s == "") {
				usable = false
			}
		}
		if !usable {
			continue
		}
		var targets []*target
		for _, v := range in.Vals {
			t, err := sc.resolveTarget(e, in.Field, v.written(), conj)
			if err != nil {
				return nil, err
			}
			dup := false
			for _, o := range targets {
				if strings.EqualFold(o.value, t.value) {
					dup = true
				}
			}
			if !dup {
				targets = append(targets, t)
			}
		}
		fr.applied[c] = true
		seen := map[string]bool{}
		var scopes []string
		for _, t := range targets {
			recs, _, _, _, scope, err := sc.readTarget(e, t, used, custom, listMode{}, st, conj)
			if err != nil {
				return nil, err
			}
			for _, r := range recs {
				id := strOf(r.api["id"])
				if id != "" && seen[id] {
					continue
				}
				seen[id] = true
				fr.recs = append(fr.recs, r)
			}
			dupScope := false
			for _, s := range scopes {
				if s == scope {
					dupScope = true
				}
			}
			if !dupScope {
				scopes = append(scopes, scope)
			}
		}
		fr.scope = strings.Join(scopes, ", ")
		return fr, nil
	}
	// step 6: route by product
	if _, has := e.route("product"); has {
		if c := sc.eqConj(conj, "product"); c != nil {
			t, err := sc.resolveTarget(e, "product", c.Val.written(), conj)
			if err != nil {
				return nil, err
			}
			fr.applied[c] = true
			recs, total, hasTotal, countOnly, scope, err := sc.readTarget(e, t, used, custom, mode(), st, conj)
			if err != nil {
				return nil, err
			}
			fr.recs, fr.total, fr.hasTotal, fr.countOnly, fr.scope = recs, total, hasTotal, countOnly, scope
			return fr, nil
		}
	}
	// step 7: account-wide list
	if e.AccountList == "" {
		return nil, sc.errC01(e)
	}
	q := ""
	if e.NameSearch {
		for _, c := range conj {
			if x, ok := c.(*Cmp); ok && x.Field == "name" && (x.Op == "=" || x.Op == "contains") && x.Val.kind == vText && x.Val.s != "" {
				q = x.Val.s
				break
			}
		}
	}
	m := mode()
	m.account = true
	recs, total, hasTotal, countOnly, err := sc.readList(e, e.AccountList, used, custom, q, m, st, conj)
	if err != nil {
		if err.Kind == "_notfound" {
			return nil, aqlErr("api_error", "aha: API error (status 404) on GET %s: (no text)", e.AccountList)
		}
		return nil, err
	}
	for _, r := range recs {
		fr.recs = append(fr.recs, fetched{api: r})
	}
	fr.total, fr.hasTotal, fr.countOnly = total, hasTotal, countOnly
	fr.scope = "all products"
	return fr, nil
}

// ---- read-statement ----

func (sc *stmtCtx) collectUsed(st *ReadStmt) map[string]bool {
	used := map[string]bool{}
	for _, it := range st.Items {
		if it.Field != "" {
			used[it.Field] = true
		}
	}
	var walk func(c Cond)
	walk = func(c Cond) {
		switch x := c.(type) {
		case *AllOf:
			for _, m := range x.M {
				walk(m)
			}
		case *AnyOf:
			for _, m := range x.M {
				walk(m)
			}
		case *Negation:
			walk(x.M)
		case *Cmp:
			used[x.Field] = true
		case *NullTest:
			used[x.Field] = true
		case *InList:
			used[x.Field] = true
		}
	}
	walk(st.Where)
	for _, g := range st.GroupBy {
		used[g] = true
	}
	for _, k := range st.OrderBy {
		if k.Field != "" {
			used[k.Field] = true
		}
		if k.Item != nil && k.Item.Field != "" {
			used[k.Item.Field] = true
		}
	}
	return used
}

func (sc *stmtCtx) usedCustoms(used map[string]bool) []*CustomField {
	var names []string
	for n := range used {
		if strings.HasPrefix(n, "custom.") && sc.custom[n] != nil {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	var out []*CustomField
	for _, n := range names {
		out = append(out, sc.custom[n])
	}
	return out
}

// runRead is read-statement; it gives the result object of the tool query.
func (sc *stmtCtx) runRead(st *ReadStmt) (omap, *AqlError) {
	e := st.E
	conj := conjuncts(st.Where)
	idf := e.idField()
	idConj := idf != "" && sc.eqConj(conj, idf) != nil
	var notes []string
	// step 1: replace a Star item by fields
	hasStar := false
	for _, it := range st.Items {
		if it.Star {
			hasStar = true
		}
	}
	if hasStar {
		var fields []string
		if idConj {
			fields = fieldNames(e)
		} else {
			fields = e.DefaultFields
			var others []string
			def := map[string]bool{}
			for _, d := range e.DefaultFields {
				def[d] = true
			}
			for _, fd := range e.Fields {
				if !def[fd.Name] {
					others = append(others, fd.Name)
				}
			}
			notes = append(notes, fmt.Sprintf("* shows %s. Other fields of %s: %s.", strings.Join(e.DefaultFields, ", "), e.Name, strings.Join(others, ", ")))
		}
		var items []*Item
		for _, it := range st.Items {
			if !it.Star {
				items = append(items, it)
				continue
			}
			for _, f := range fields {
				items = append(items, &Item{Field: f, Label: f})
			}
		}
		var kept []*Item
		for _, it := range items {
			dup := false
			for _, k := range kept {
				if k.Func == "" && it.Func == "" && k.Label == it.Label && k.Field == it.Field {
					dup = true
				}
			}
			if !dup {
				kept = append(kept, it)
			}
		}
		st.Items = kept
	}
	used := sc.collectUsed(st)
	customs := sc.usedCustoms(used)
	fr, err := sc.planFetch(st, used, len(customs) > 0)
	if err != nil {
		return nil, err
	}
	if err := sc.prepareLookups(e, st.Where, fr.applied); err != nil {
		return nil, err
	}
	var kept []*record
	var keptFetched []fetched
	for _, f := range fr.recs {
		r := f.pre
		if r == nil {
			var cerr *AqlError
			r, cerr = sc.convert(e, f.api, f.prod, f.parent, used, customs)
			if cerr != nil {
				return nil, cerr
			}
		}
		if sc.keep(e, r, st.Where, fr.applied) {
			kept = append(kept, r)
			keptFetched = append(keptFetched, f)
		}
	}
	grouped := len(st.GroupBy) > 0 || st.hasAggregate()
	// step 6 a: star custom fields
	var starCustom []*CustomField
	if hasStar && idConj && !grouped && len(kept) == 1 {
		all, err := sc.customFieldsOf(e)
		if err != nil {
			return nil, err
		}
		labels := map[string]bool{}
		selectedNames := map[string]bool{}
		for _, it := range st.Items {
			labels[it.Label] = true
			if it.Func == "" {
				selectedNames[it.Field] = true
			}
		}
		for _, c := range all {
			if selectedNames[c.Name] || labels[c.Name] {
				continue
			}
			starCustom = append(starCustom, c)
		}
		if len(starCustom) > 0 {
			api := kept[0].api
			if e.Name == "products" {
				rec, f := sc.productRecord(strOf(api["id"]))
				switch {
				case f == nil:
					api = rec
				case f.kind == fNotFound:
					api = apiRec{}
				default:
					return nil, sc.failErr(f)
				}
			}
			for _, c := range starCustom {
				sc.convertCustom(kept[0], c, api)
			}
		}
	}
	rows, count, total, hasTotal, truncated := sc.buildRows(st, kept, starCustom, fr)
	if truncated {
		notes = append(notes, fmt.Sprintf("Showing %d of %d records. Narrow the result with WHERE, raise LIMIT (up to %d), or count with COUNT(*).", count, total, maxLimit))
	}
	if len(kept) == 0 && !grouped {
		notes = append(notes, sc.emptyNotes(st, fr)...)
	}
	_ = keptFetched
	out := omap{{"entity", e.Name}, {"scope", fr.scope}, {"count", count}}
	if hasTotal {
		out = append(out, kv{"total", total})
	}
	if truncated {
		out = append(out, kv{"truncated", true})
	}
	recs := []any{}
	for _, r := range rows {
		recs = append(recs, r)
	}
	out = append(out, kv{"records", recs})
	if len(notes) > 0 {
		out = append(out, kv{"note", strings.Join(notes, " ")})
	}
	return out, nil
}

// emptyNotes explains an empty result (read-statement, step 8).
func (sc *stmtCtx) emptyNotes(st *ReadStmt, fr *fetchResult) []string {
	e := st.E
	var notes []string
	idf := e.idField()
	for _, c := range conjuncts(st.Where) {
		if len(notes) >= 2 {
			break
		}
		x, ok := c.(*Cmp)
		if !ok || x.Val.kind != vText {
			continue
		}
		v := strings.TrimSuffix(strings.TrimPrefix(x.Val.s, "%"), "%")
		switch {
		case x.Field == idf && x.Op == "=" && idf != "":
			if e.Name == "products" || e.Name == "users" {
				found := ""
				for _, f := range fr.recs {
					if textEq(strOf(lookup(f.api, e.field(idf).Source)), v) || textEq(strOf(f.api["id"]), v) {
						found = strOf(lookup(f.api, e.field(idf).Source))
					}
				}
				if found == "" {
					notes = append(notes, fmt.Sprintf("There is no %s with the %s %s.", singularOf(e), idf, quote(v)))
				} else {
					notes = append(notes, fmt.Sprintf("The %s %s exists, and another condition of the statement is not true for it. To see the record: SELECT * FROM %s WHERE %s = '%s'", singularOf(e), found, e.Name, idf, found))
				}
				continue
			}
			if fr.idConj != c {
				continue
			}
			if fr.idFound == "" {
				notes = append(notes, fmt.Sprintf("There is no %s with the %s %s.", singularOf(e), idf, quote(v)))
			} else {
				notes = append(notes, fmt.Sprintf("The %s %s exists, and another condition of the statement is not true for it. To see the record: SELECT * FROM %s WHERE %s = '%s'", singularOf(e), fr.idFound, e.Name, idf, fr.idFound))
			}
		case e.Name == "products" && x.Field == "name" && matchProduct(sc.products, v) != nil:
			p := matchProduct(sc.products, v)
			notes = append(notes, fmt.Sprintf("%s is the key of a product, not its name. Use: SELECT key, name FROM products WHERE key = '%s'", quote(v), p.Key))
		case x.Field == "name" && refKind(v) != "" && refKind(v) == e.Kind:
			notes = append(notes, fmt.Sprintf("%s looks like an id, not a name. Use: WHERE id = '%s'", quote(v), strings.ToUpper(v)))
		case x.Field == "name" && refKind(v) != "" && e.hasParentFilter(refKind(v)):
			k := refKind(v)
			notes = append(notes, fmt.Sprintf("%s looks like the id of %s %s, not a name. Use: WHERE %s = '%s'", quote(v), article(k), k, k, strings.ToUpper(v)))
		case x.Field == "name" && x.Op == "=":
			notes = append(notes, fmt.Sprintf("No %s has exactly the name %s. name = compares the whole name; to search inside names use: name CONTAINS %s", singularOf(e), quote(v), quote(v)))
		case e.Name == "statuses" && x.Field == "entity" && x.Op == "=" && fr.product != nil:
			want := v
			if en := resolveEntity(v); en != nil {
				want = en.Name
			}
			has := false
			var ents []string
			seen := map[string]bool{}
			for _, g := range fr.groups {
				if g.entity == want || strings.EqualFold(g.prefix, v) {
					has = true
				}
				if !seen[g.entity] {
					seen[g.entity] = true
					ents = append(ents, g.entity)
				}
			}
			if !has {
				list := "(none)"
				if len(ents) > 0 {
					list = strings.Join(ents, ", ")
				}
				notes = append(notes, fmt.Sprintf("%s has statuses for: %s.", fr.product.Key, list))
			}
		}
	}
	return notes
}

// ---- build-rows ----

func roundAgg(f float64) any {
	if f == math.Trunc(f) {
		return f
	}
	return math.Round(f*100) / 100
}

func (sc *stmtCtx) aggregate(it *Item, group []*record) any {
	if it.CountStar {
		return float64(len(group))
	}
	var vals []any
	seen := map[string]bool{}
	for _, r := range group {
		v := r.vals[it.Field]
		if noValue(v) {
			continue
		}
		if it.Distinct {
			k := compForm(textOf(v))
			if seen[k] {
				continue
			}
			seen[k] = true
		}
		vals = append(vals, v)
	}
	switch it.Func {
	case "COUNT":
		return float64(len(vals))
	case "SUM", "AVG":
		sum, n := 0.0, 0
		for _, v := range vals {
			if _, isBool := v.(bool); isBool {
				continue
			}
			if f, ok := toNumber(v); ok {
				sum += f
				n++
			}
		}
		if n == 0 {
			return nil
		}
		if it.Func == "AVG" {
			return roundAgg(sum / float64(n))
		}
		return roundAgg(sum)
	case "MIN", "MAX":
		var best any
		for _, v := range vals {
			if best == nil {
				best = v
				continue
			}
			c := orderValues(v, best)
			if (it.Func == "MIN" && c < 0) || (it.Func == "MAX" && c > 0) {
				best = v
			}
		}
		return best
	}
	return nil
}

type rowWithKeys struct {
	row  omap
	keys []any
}

func sortRows(rows []rowWithKeys, order []OrderKey) {
	sort.SliceStable(rows, func(i, j int) bool {
		for k, key := range order {
			a, b := rows[i].keys[k], rows[j].keys[k]
			an, bn := noValue(a), noValue(b)
			if an && bn {
				continue
			}
			if an {
				return false
			}
			if bn {
				return true
			}
			c := orderValues(a, b)
			if c == 0 {
				continue
			}
			if key.Desc {
				return c > 0
			}
			return c < 0
		}
		return false
	})
}

func (sc *stmtCtx) buildRows(st *ReadStmt, recs []*record, starCustom []*CustomField, fr *fetchResult) ([]omap, int, int, bool, bool) {
	if fr.countOnly {
		label := "COUNT(*)"
		if len(st.Items) > 0 {
			label = st.Items[0].Label
		}
		return []omap{{{label, float64(fr.total)}}}, 1, 0, false, false
	}
	var rows []rowWithKeys
	if len(st.GroupBy) > 0 || st.hasAggregate() {
		groupFields := append([]string{}, st.GroupBy...)
		selected := map[string]bool{}
		for _, it := range st.Items {
			if it.Func == "" && !it.Star {
				selected[it.Field] = true
				in := false
				for _, g := range groupFields {
					if g == it.Field {
						in = true
					}
				}
				if !in {
					groupFields = append(groupFields, it.Field)
				}
			}
		}
		var keys []string
		groups := map[string][]*record{}
		for _, r := range recs {
			var parts []string
			for _, g := range groupFields {
				v := r.vals[g]
				if noValue(v) {
					parts = append(parts, "\x00null")
				} else {
					parts = append(parts, compForm(textOf(v)))
				}
			}
			k := strings.Join(parts, "\x01")
			if _, ok := groups[k]; !ok {
				keys = append(keys, k)
			}
			groups[k] = append(groups[k], r)
		}
		if len(groupFields) == 0 && len(keys) == 0 {
			keys = []string{""}
			groups[""] = nil
		}
		for _, k := range keys {
			g := groups[k]
			row := omap{}
			for _, gf := range st.GroupBy {
				if !selected[gf] {
					var v any
					if len(g) > 0 {
						v = g[0].vals[gf]
					}
					row = append(row, kv{gf, v})
				}
			}
			for _, it := range st.Items {
				if it.Func != "" {
					row = append(row, kv{it.Label, sc.aggregate(it, g)})
				} else {
					var v any
					if len(g) > 0 {
						v = g[0].vals[it.Field]
					}
					row = append(row, kv{it.Label, v})
				}
			}
			var okeys []any
			for _, ok := range st.OrderBy {
				switch {
				case ok.Item != nil && ok.Item.Func != "":
					okeys = append(okeys, sc.aggregate(ok.Item, g))
				case ok.Item != nil:
					v, _ := row.get(ok.Item.Label)
					okeys = append(okeys, v)
				default:
					var v any
					if len(g) > 0 {
						v = g[0].vals[ok.Field]
					}
					okeys = append(okeys, v)
				}
			}
			rows = append(rows, rowWithKeys{row, okeys})
		}
		sortRows(rows, st.OrderBy)
	} else {
		for _, r := range recs {
			row := omap{}
			for _, it := range st.Items {
				row = append(row, kv{it.Label, r.vals[it.Field]})
			}
			for _, c := range starCustom {
				if v := r.vals[c.Name]; !noValue(v) {
					row = append(row, kv{c.Name, v})
				}
			}
			var okeys []any
			for _, ok := range st.OrderBy {
				if ok.Item != nil {
					okeys = append(okeys, r.vals[ok.Item.Field])
				} else {
					okeys = append(okeys, r.vals[ok.Field])
				}
			}
			rows = append(rows, rowWithKeys{row, okeys})
		}
		sortRows(rows, st.OrderBy)
		if st.Distinct {
			var kept []rowWithKeys
			seen := map[string]bool{}
			for _, r := range rows {
				k := encodeJSON(r.row)
				if seen[k] {
					continue
				}
				seen[k] = true
				kept = append(kept, r)
			}
			rows = kept
		}
	}
	limit := st.limit()
	total, hasTotal := 0, false
	if len(rows) > limit {
		total, hasTotal = len(rows), true
		rows = rows[:limit]
	}
	if fr.hasTotal {
		total, hasTotal = fr.total, true
	}
	truncated := hasTotal && (!st.HasLimit || st.Limit > maxLimit)
	var out []omap
	for _, r := range rows {
		out = append(out, r.row)
	}
	return out, len(out), total, hasTotal, truncated
}

var _ = utf8.RuneLen
