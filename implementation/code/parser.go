// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e

package main

import (
	"strconv"
	"strings"
)

type valKind int

const (
	vText valKind = iota
	vNumber
	vBool
	vNull
)

// Value is a value of a statement (TYPES: Value).
type Value struct {
	kind valKind
	s    string // text, or the number as written
	n    float64
	b    bool
}

func (v Value) raw() any {
	switch v.kind {
	case vText:
		return v.s
	case vNumber:
		return v.n
	case vBool:
		return v.b
	}
	return nil
}

// written gives the value as the statement writes it, without quotes.
func (v Value) written() string {
	switch v.kind {
	case vBool:
		if v.b {
			return "TRUE"
		}
		return "FALSE"
	case vNull:
		return "NULL"
	}
	return v.s
}

// Conditions (TYPES: Condition).
type Cond interface{}

type Cmp struct {
	Field string
	Op    string // = != < <= > >= contains like
	Val   Value
}
type NullTest struct {
	Field   string
	Negated bool
}
type InList struct {
	Field string
	Vals  []Value
}
type AllOf struct{ M []Cond }
type AnyOf struct{ M []Cond }
type Negation struct{ M Cond }

// Item is a select item (TYPES: SelectItem).
type Item struct {
	Star      bool
	Field     string
	Label     string
	Func      string // COUNT, SUM, AVG, MIN, MAX; "" for a field item
	Distinct  bool
	CountStar bool
	word      string
	hasAS     bool
	pos       int
}

func (it *Item) aggregate() bool { return it.Func != "" }

func (it *Item) content() string {
	switch {
	case it.Star:
		return "*"
	case it.CountStar:
		return "COUNT(*)"
	case it.Func != "":
		d := ""
		if it.Distinct {
			d = "DISTINCT "
		}
		return it.Func + "(" + d + it.Field + ")"
	}
	return "field:" + it.Field
}

// OrderKey is a key of ORDER BY.
type OrderKey struct {
	Item  *Item // a label or an aggregate
	Field string
	Desc  bool
}

type ReadStmt struct {
	E        *EntityDef
	Items    []*Item
	Distinct bool
	Where    Cond
	GroupBy  []string
	OrderBy  []OrderKey
	Limit    int
	HasLimit bool
}

type DescribeStmt struct{ E *EntityDef }

type Assign struct {
	Field string
	Val   Value
	Word  string
}

type WriteStmt struct {
	Kind   string // insert | update | delete
	E      *EntityDef
	Parent *Value
	Fields []Assign
	ID     string
	Guards []Assign
}

var keywords = map[string]bool{}

func init() {
	for _, k := range strings.Fields("SELECT FROM WHERE AND OR NOT IN IS NULL LIKE CONTAINS BETWEEN GROUP BY ORDER ASC DESC LIMIT OFFSET AS DISTINCT TRUE FALSE JOIN INNER LEFT RIGHT FULL CROSS OUTER NATURAL ON HAVING INSERT INTO VALUES UPDATE SET DELETE DESCRIBE") {
		keywords[k] = true
	}
}

func isKeyword(w string) bool { return keywords[strings.ToUpper(w)] }

func isFunc(w string) bool {
	switch strings.ToUpper(w) {
	case "COUNT", "SUM", "AVG", "MIN", "MAX":
		return true
	}
	return false
}

type parser struct {
	sc   *stmtCtx
	src  []rune
	toks []token
	i    int
}

func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) at(k int) token {
	if p.i+k < len(p.toks) {
		return p.toks[p.i+k]
	}
	return token{kind: tkEOF}
}
func (p *parser) next() token {
	t := p.toks[p.i]
	if t.kind != tkEOF {
		p.i++
	}
	return t
}
func (p *parser) s03(t token, what string) *AqlError { return p.sc.errS03(p.src, t.pos, what) }

// parseStatement is parse-statement.
func (sc *stmtCtx) parseStatement(text string, reads, writes bool) (any, *AqlError) {
	src := []rune(text)
	if reads {
		t := strings.Trim(text, " \t\r\n'\";")
		if k := refKind(t); k != "" {
			id := strings.ToUpper(t)
			return nil, aqlErr("syntax", "%s is an id, not a statement. To read that record: SELECT * FROM %s WHERE id = '%s'", id, entityOfKind(k).Name, id)
		}
	}
	st, err := sc.parseOnce(src, false, reads, writes)
	if err == nil {
		return st, nil
	}
	if err.Kind == "syntax" && hasInnerApostrophe(src) {
		if st2, err2 := sc.parseOnce(src, true, reads, writes); err2 == nil {
			return st2, nil
		}
	}
	return nil, err
}

func (sc *stmtCtx) parseOnce(src []rune, lenient, reads, writes bool) (any, *AqlError) {
	toks, err := tokenize(src, lenient)
	if err != nil {
		return nil, err
	}
	p := &parser{sc: sc, src: src, toks: toks}
	t := p.peek()
	kind := ""
	switch {
	case t.is("SELECT") || t.is("FROM"):
		kind = "select"
	case t.punct("*") || (t.kind == tkWord && isFunc(t.val) && p.at(1).punct("(")):
		kind = "select"
	case t.is("DESCRIBE"):
		kind = "describe"
	case t.is("INSERT"):
		kind = "insert"
	case t.is("UPDATE"):
		kind = "update"
	case t.is("DELETE"):
		kind = "delete"
	}
	write := kind == "insert" || kind == "update" || kind == "delete"
	if kind != "" && write && !writes {
		return nil, aqlErr("syntax", "query only reads. Use the execute tool for INSERT, UPDATE and DELETE.")
	}
	if kind != "" && !write && !reads {
		return nil, errW01()
	}
	if kind == "" {
		if reads {
			if t.kind == tkWord && (p.at(1).kind == tkEOF || (p.at(1).punct(";") && p.at(2).kind == tkEOF)) {
				if e := resolveEntity(t.val); e != nil {
					return nil, aqlErr("syntax", "%s is an entity, not a statement. To list its records: %s. To see its fields: DESCRIBE %s", t.val, sc.exampleOf(e), e.Name)
				}
			}
			return nil, sc.errS04(src)
		}
		return nil, aqlErr("syntax", "a statement starts with INSERT, UPDATE or DELETE.%s Forms: %s", apostropheHint(src), writeFormText)
	}
	switch kind {
	case "select":
		return p.parseRead()
	case "describe":
		return p.parseDescribe()
	}
	return p.parseWrite(kind)
}

// ---- resolve-names ----

func resolveEntity(word string) *EntityDef {
	for _, e := range catalog {
		if strings.EqualFold(e.Name, word) || strings.EqualFold(e.Singular, word) {
			return e
		}
		for _, o := range e.OtherNames {
			if strings.EqualFold(o, word) {
				return e
			}
		}
	}
	return nil
}

// splitWord gives the name of a field word, whether it is a custom field
// word, and its label (the word without a qualifier).
func splitWord(word string) (name string, custom bool, label string) {
	parts := strings.Split(word, ".")
	name = parts[len(parts)-1]
	label = name
	if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-2], "custom") {
		custom = true
		label = parts[len(parts)-2] + "." + name
	}
	return
}

// catalogField runs steps 4 a to 4 d of resolve-names.
func catalogField(e *EntityDef, name string) string {
	lw := strings.ToLower(name)
	if fd := e.field(lw); fd != nil {
		return fd.Name
	}
	for k, v := range e.OtherFieldNames {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	if e.Name != "products" {
		if v, ok := commonOtherNames[lw]; ok && e.field(v) != nil {
			return v
		}
	}
	sing := strings.ToLower(e.Singular)
	if id := e.idField(); id != "" && (lw == sing || lw == sing+"_id") {
		return id
	}
	if lw == sing+"_name" && e.field("name") != nil {
		return "name"
	}
	return ""
}

// resolveField is resolve-names for a field word. With writing true an
// unknown word gives "" and no error.
func (sc *stmtCtx) resolveField(e *EntityDef, word string, writing bool) (string, *AqlError) {
	name, custom, _ := splitWord(word)
	if custom {
		cf, err := sc.customLookup(e, name, true)
		if err != nil {
			return "", err
		}
		return cf.Name, nil
	}
	if n := catalogField(e, name); n != "" {
		return n, nil
	}
	if name != "" {
		cf, err := sc.customLookup(e, name, false)
		if err != nil {
			return "", err
		}
		if cf != nil {
			return cf.Name, nil
		}
	}
	if writing {
		return "", nil
	}
	if e.Name == "users" {
		lw := strings.ToLower(name)
		if lw == "product" || commonOtherNames[lw] == "product" {
			return "", sc.errN03()
		}
	}
	return "", sc.errN02(name, e)
}

// ---- read statements ----

func specialWord(t token) bool {
	return t.is("JOIN") || t.is("INNER") || t.is("LEFT") || t.is("RIGHT") || t.is("FULL") || t.is("CROSS") || t.is("OUTER") || t.is("NATURAL") || t.is("HAVING") || t.is("OFFSET")
}

func (p *parser) special(t token) *AqlError {
	switch {
	case t.is("HAVING"):
		return p.sc.errS09()
	case t.is("OFFSET"):
		return errS10()
	}
	return p.sc.errS08()
}

func (p *parser) parseRead() (any, *AqlError) {
	st := &ReadStmt{}
	var items []*Item
	var err *AqlError
	t := p.peek()
	switch {
	case t.is("SELECT"):
		p.next()
		if p.peek().is("DISTINCT") {
			st.Distinct = true
			p.next()
		}
		items, err = p.parseItems(true)
	case t.is("FROM"):
		items = []*Item{{Star: true, pos: t.pos}}
	default:
		items, err = p.parseItems(false)
	}
	if err != nil {
		return nil, err
	}
	if !p.peek().is("FROM") {
		if specialWord(p.peek()) {
			return nil, p.special(p.peek())
		}
		return nil, p.s03(p.peek(), "FROM and an entity")
	}
	p.next()
	et := p.peek()
	if et.punct("(") && p.at(1).is("SELECT") {
		return nil, errS11()
	}
	if et.kind != tkWord || isKeyword(et.val) {
		return nil, p.s03(et, "an entity ("+strings.Join(entityNames(), ", ")+")")
	}
	e := resolveEntity(et.val)
	if e == nil {
		return nil, p.sc.errN01(et.val)
	}
	st.E = e
	p.next()
	if p.peek().is("AS") {
		p.next()
		if a := p.peek(); a.kind != tkWord || isKeyword(a.val) {
			return nil, p.s03(a, "a name after AS")
		}
		p.next()
	} else if a := p.peek(); a.kind == tkWord && !isKeyword(a.val) {
		p.next()
	}
	for _, it := range items {
		if it.Star || it.CountStar {
			continue
		}
		f, err := p.sc.resolveField(e, it.word, false)
		if err != nil {
			return nil, err
		}
		it.Field = f
	}
	if p.peek().is("WHERE") {
		p.next()
		c, err := p.parseOr(e)
		if err != nil {
			return nil, err
		}
		st.Where = c
	}
	type gkey struct {
		field string
		pos   int
	}
	if p.peek().is("GROUP") {
		p.next()
		if !p.peek().is("BY") {
			return nil, p.s03(p.peek(), "BY after GROUP")
		}
		p.next()
		for {
			g := p.peek()
			if g.kind != tkWord || isKeyword(g.val) {
				return nil, p.s03(g, "a field name ("+strings.Join(fieldNames(e), ", ")+")")
			}
			f, err := p.sc.resolveField(e, g.val, false)
			if err != nil {
				return nil, err
			}
			p.next()
			dup := false
			for _, x := range st.GroupBy {
				if x == f {
					dup = true
				}
			}
			if !dup {
				st.GroupBy = append(st.GroupBy, f)
			}
			if !p.peek().punct(",") {
				break
			}
			p.next()
		}
	}
	if p.peek().is("HAVING") {
		return nil, p.sc.errS09()
	}
	// items: leave out repeated items; S15 and S16.
	var kept []*Item
	for _, it := range items {
		drop := false
		for _, k := range kept {
			if k.Label == it.Label && !it.Star && !k.Star {
				if k.content() == it.content() {
					drop = true
					break
				}
				return nil, errS15(it.Label)
			}
		}
		if !drop {
			kept = append(kept, it)
		}
	}
	items = kept
	hasStar := false
	for _, it := range items {
		if it.Star {
			hasStar = true
		}
	}
	selected := map[string]bool{}
	for _, it := range items {
		if !it.Star && it.Func == "" {
			selected[it.Field] = true
		}
	}
	for _, it := range items {
		if !it.hasAS {
			continue
		}
		for _, g := range st.GroupBy {
			if g == it.Label && !selected[g] {
				return nil, errS16(it.Label, e)
			}
		}
		if hasStar && e.field(it.Label) != nil && !(it.Func == "" && it.Field == it.Label) {
			return nil, errS16(it.Label, e)
		}
	}
	st.Items = items
	grouped := len(st.GroupBy) > 0
	for _, it := range items {
		if it.aggregate() {
			grouped = true
		}
	}
	if p.peek().is("ORDER") {
		p.next()
		if !p.peek().is("BY") {
			return nil, p.s03(p.peek(), "BY after ORDER")
		}
		p.next()
		for {
			k := p.peek()
			var key OrderKey
			switch {
			case k.kind == tkWord && isFunc(k.val) && p.at(1).punct("("):
				it, err := p.parseFunc()
				if err != nil {
					return nil, err
				}
				if !it.CountStar {
					f, err := p.sc.resolveField(e, it.word, false)
					if err != nil {
						return nil, err
					}
					it.Field = f
				}
				if !grouped {
					return nil, p.s03(k, "a field name")
				}
				key.Item = it
			case k.kind == tkWord && !isKeyword(k.val):
				var byLabel *Item
				for _, it := range items {
					if !it.Star && it.Label == k.val {
						byLabel = it
						break
					}
				}
				p.next()
				if byLabel != nil {
					key.Item = byLabel
				} else {
					f, err := p.sc.resolveField(e, k.val, false)
					if err != nil {
						return nil, err
					}
					if grouped {
						ok := selected[f]
						for _, g := range st.GroupBy {
							if g == f {
								ok = true
							}
						}
						if !ok {
							return nil, p.s03(k, "a field of GROUP BY, a selected field or an aggregate after ORDER BY")
						}
					}
					key.Field = f
				}
			default:
				return nil, p.s03(k, "a field name")
			}
			if p.peek().is("DESC") {
				key.Desc = true
				p.next()
			} else if p.peek().is("ASC") {
				p.next()
			}
			st.OrderBy = append(st.OrderBy, key)
			if !p.peek().punct(",") {
				break
			}
			p.next()
		}
	}
	if p.peek().is("LIMIT") {
		p.next()
		n := p.peek()
		if n.kind != tkNumber || strings.ContainsAny(n.val, ".-") {
			return nil, p.s03(n, "a number after LIMIT")
		}
		v, convErr := strconv.Atoi(n.val)
		if convErr != nil {
			v = 1 << 30
		}
		st.Limit, st.HasLimit = v, true
		p.next()
		if p.peek().punct(",") || p.peek().is("OFFSET") {
			return nil, errS10()
		}
	}
	if p.peek().is("OFFSET") {
		return nil, errS10()
	}
	if p.peek().punct(";") {
		p.next()
	}
	if t := p.peek(); t.kind != tkEOF {
		if specialWord(t) {
			return nil, p.special(t)
		}
		return nil, p.s03(t, "the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order")
	}
	return st, nil
}

func fieldLabel(word string) string { _, _, l := splitWord(word); return l }

func (p *parser) parseItems(afterSelect bool) ([]*Item, *AqlError) {
	var items []*Item
	first := true
	for {
		t := p.peek()
		var it *Item
		switch {
		case t.punct("*"):
			p.next()
			it = &Item{Star: true, pos: t.pos}
		case t.kind == tkWord && strings.HasSuffix(t.val, ".") && p.at(1).punct("*"):
			p.next()
			p.next()
			it = &Item{Star: true, pos: t.pos}
		case t.kind == tkWord && p.at(1).punct("("):
			if !isFunc(t.val) {
				return nil, p.sc.errS12(t.val)
			}
			var err *AqlError
			it, err = p.parseFunc()
			if err != nil {
				return nil, err
			}
		case t.kind == tkWord && !isKeyword(t.val):
			p.next()
			it = &Item{word: t.val, Label: fieldLabel(t.val), pos: t.pos}
		case t.punct("(") && p.at(1).is("SELECT"):
			return nil, errS11()
		default:
			what := "a field, * or COUNT(*)"
			if first && afterSelect {
				what += " after SELECT"
			}
			return nil, p.s03(t, what)
		}
		first = false
		if p.peek().is("AS") {
			p.next()
			l := p.peek()
			switch {
			case l.kind == tkWord && !isKeyword(l.val):
				it.Label = l.val
			case l.kind == tkText && l.val != "":
				it.Label = l.val
			default:
				return nil, p.s03(l, "a name after AS")
			}
			it.hasAS = true
			p.next()
		}
		items = append(items, it)
		if !p.peek().punct(",") {
			return items, nil
		}
		p.next()
	}
}

// parseFunc reads function ( * | [DISTINCT] field ).
func (p *parser) parseFunc() (*Item, *AqlError) {
	ft := p.next()
	name := strings.ToUpper(ft.val)
	p.next() // (
	it := &Item{Func: name, pos: ft.pos}
	a := p.peek()
	switch {
	case a.punct("*"):
		if name != "COUNT" {
			return nil, p.s03(a, "a field inside "+name+"( )")
		}
		it.CountStar = true
		p.next()
	case a.kind == tkNumber && a.val == "1" && name == "COUNT":
		it.CountStar = true
		p.next()
	case a.is("DISTINCT"):
		p.next()
		b := p.peek()
		if b.punct("*") {
			return nil, p.s03(b, "a field after DISTINCT")
		}
		if b.kind != tkWord || isKeyword(b.val) {
			return nil, p.s03(b, "a field after DISTINCT")
		}
		it.Distinct = true
		it.word = b.val
		p.next()
	case a.kind == tkWord && !isKeyword(a.val):
		if name == "COUNT" && entityByName(strings.ToLower(a.val)) != nil {
			it.CountStar = true
		} else {
			it.word = a.val
		}
		p.next()
	default:
		if name == "COUNT" {
			return nil, p.s03(a, "a field or * inside COUNT( )")
		}
		return nil, p.s03(a, "a field inside "+name+"( )")
	}
	if !p.peek().punct(")") {
		return nil, p.s03(p.peek(), "')'")
	}
	p.next()
	if it.CountStar {
		it.Func = "COUNT"
		it.Label = "COUNT(*)"
	} else {
		d := ""
		if it.Distinct {
			d = "DISTINCT "
		}
		it.Label = name + "(" + d + it.word + ")"
	}
	return it, nil
}

func (p *parser) parseValue() (Value, *AqlError) {
	t := p.peek()
	switch {
	case t.kind == tkText:
		p.next()
		return Value{kind: vText, s: t.val}, nil
	case t.kind == tkNumber:
		p.next()
		f, _ := strconv.ParseFloat(t.val, 64)
		return Value{kind: vNumber, s: t.val, n: f}, nil
	case t.is("TRUE"):
		p.next()
		return Value{kind: vBool, b: true}, nil
	case t.is("FALSE"):
		p.next()
		return Value{kind: vBool, b: false}, nil
	case t.is("NULL"):
		p.next()
		return Value{kind: vNull}, nil
	case t.kind == tkWord && !isKeyword(t.val):
		return Value{}, errS13(t.val)
	}
	return Value{}, p.s03(t, "a value in single quotes")
}

func (p *parser) parseOr(e *EntityDef) (Cond, *AqlError) {
	left, err := p.parseAnd(e)
	if err != nil {
		return nil, err
	}
	if !p.peek().is("OR") {
		return left, nil
	}
	any := &AnyOf{M: []Cond{left}}
	for p.peek().is("OR") {
		p.next()
		r, err := p.parseAnd(e)
		if err != nil {
			return nil, err
		}
		any.M = append(any.M, r)
	}
	return any, nil
}

func (p *parser) parseAnd(e *EntityDef) (Cond, *AqlError) {
	left, err := p.parseNot(e)
	if err != nil {
		return nil, err
	}
	if !p.peek().is("AND") {
		return left, nil
	}
	all := &AllOf{}
	add := func(c Cond) {
		if a, ok := c.(*AllOf); ok {
			all.M = append(all.M, a.M...)
		} else {
			all.M = append(all.M, c)
		}
	}
	add(left)
	for p.peek().is("AND") {
		p.next()
		r, err := p.parseNot(e)
		if err != nil {
			return nil, err
		}
		add(r)
	}
	return all, nil
}

func (p *parser) parseNot(e *EntityDef) (Cond, *AqlError) {
	if p.peek().is("NOT") {
		p.next()
		m, err := p.parseNot(e)
		if err != nil {
			return nil, err
		}
		return &Negation{M: m}, nil
	}
	if p.peek().punct("(") {
		if p.at(1).is("SELECT") {
			return nil, errS11()
		}
		p.next()
		c, err := p.parseOr(e)
		if err != nil {
			return nil, err
		}
		if !p.peek().punct(")") {
			return nil, p.s03(p.peek(), "')'")
		}
		p.next()
		return c, nil
	}
	return p.parseComparison(e)
}

func (p *parser) parseComparison(e *EntityDef) (Cond, *AqlError) {
	ft := p.peek()
	if ft.kind != tkWord || isKeyword(ft.val) {
		return nil, p.s03(ft, "a field name ("+strings.Join(fieldNames(e), ", ")+")")
	}
	field, err := p.sc.resolveField(e, ft.val, false)
	if err != nil {
		return nil, err
	}
	p.next()
	ftype := p.sc.fieldType(e, field)
	op := p.peek()
	negate := false
	if op.is("NOT") && (p.at(1).is("IN") || p.at(1).is("CONTAINS") || p.at(1).is("LIKE") || p.at(1).is("BETWEEN")) {
		negate = true
		p.next()
		op = p.peek()
	}
	wrap := func(c Cond) Cond {
		if negate {
			return &Negation{M: c}
		}
		return c
	}
	switch {
	case op.kind == tkOp && !negate:
		p.next()
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if v.kind == vNull || (v.kind == vText && v.s == "") {
			if op.val == "=" {
				return &NullTest{Field: field}, nil
			}
			if op.val == "!=" {
				return &NullTest{Field: field, Negated: true}, nil
			}
		}
		if err := checkCondValue(field, ftype, op.val, v); err != nil {
			return nil, err
		}
		return &Cmp{Field: field, Op: op.val, Val: v}, nil
	case op.is("IS") && !negate:
		p.next()
		neg := false
		if p.peek().is("NOT") {
			neg = true
			p.next()
		}
		if !p.peek().is("NULL") {
			return nil, p.s03(p.peek(), "NULL after IS")
		}
		p.next()
		return &NullTest{Field: field, Negated: neg}, nil
	case op.is("IN"):
		p.next()
		if !p.peek().punct("(") {
			return nil, p.s03(p.peek(), "'(' after IN")
		}
		p.next()
		if p.peek().is("SELECT") {
			return nil, errS11()
		}
		in := &InList{Field: field}
		for {
			v, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			if err := checkCondValue(field, ftype, "=", v); err != nil {
				return nil, err
			}
			in.Vals = append(in.Vals, v)
			if !p.peek().punct(",") {
				break
			}
			p.next()
		}
		if !p.peek().punct(")") {
			return nil, p.s03(p.peek(), "')'")
		}
		p.next()
		return wrap(in), nil
	case op.is("CONTAINS") || op.is("LIKE"):
		p.next()
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		return wrap(&Cmp{Field: field, Op: strings.ToLower(op.val), Val: v}), nil
	case op.is("BETWEEN"):
		p.next()
		a, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if !p.peek().is("AND") {
			return nil, p.s03(p.peek(), "AND in BETWEEN")
		}
		p.next()
		b, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if err := checkCondValue(field, ftype, ">=", a); err != nil {
			return nil, err
		}
		if err := checkCondValue(field, ftype, "<=", b); err != nil {
			return nil, err
		}
		return wrap(&AllOf{M: []Cond{&Cmp{Field: field, Op: ">=", Val: a}, &Cmp{Field: field, Op: "<=", Val: b}}}), nil
	}
	return nil, p.s03(op, "a comparison after the field "+ft.val+" (=, !=, <, >, CONTAINS, LIKE, IN, IS NULL)")
}

// checkCondValue checks a value against the type of its field (table of
// values in a condition).
func checkCondValue(field string, t FieldType, op string, v Value) *AqlError {
	if t == tBoolean && (op == "<" || op == "<=" || op == ">" || op == ">=") {
		return errV02(field)
	}
	if v.kind == vNull {
		return nil
	}
	switch t {
	case tNumber:
		if _, ok := toNumber(v.raw()); !ok || v.kind == vBool {
			return errV01(field, t, v.written())
		}
	case tBoolean:
		if _, ok := truth(v.raw()); !ok {
			return errV01(field, t, v.written())
		}
	case tDate, tDatetime:
		if v.kind != vText || !(validDay(v.s) || isTimestamp(v.s)) {
			return errV01(field, t, v.written())
		}
	}
	return nil
}

// ---- describe ----

func (p *parser) parseDescribe() (any, *AqlError) {
	p.next()
	t := p.peek()
	if t.kind == tkEOF || (t.punct(";") && p.at(1).kind == tkEOF) {
		return &DescribeStmt{}, nil
	}
	if t.kind != tkWord {
		return nil, p.s03(t, "an entity ("+strings.Join(entityNames(), ", ")+")")
	}
	e := resolveEntity(t.val)
	if e == nil {
		return nil, p.sc.errN01(t.val)
	}
	p.next()
	if p.peek().punct(";") {
		p.next()
	}
	if p.peek().kind != tkEOF {
		return nil, p.sc.errS14(e)
	}
	return &DescribeStmt{E: e}, nil
}

// ---- write statements ----

func (p *parser) writeValue() (Value, *AqlError) {
	t := p.peek()
	switch {
	case t.is("NULL"):
		return Value{}, aqlErr("syntax", "NULL cannot be written. Leave the field out.")
	case t.kind == tkText || t.kind == tkNumber || t.is("TRUE") || t.is("FALSE"):
		return p.parseValue()
	case t.kind == tkWord && !isKeyword(t.val):
		return Value{}, errS13(t.val)
	}
	return Value{}, errW03("expected a value in single quotes at "+whereText(p.src, t.pos)+".", p.src)
}

func (p *parser) writeEntity(kind string) (*EntityDef, *AqlError) {
	t := p.peek()
	if t.kind != tkWord || isKeyword(t.val) {
		return nil, errW03("expected a field name at "+whereText(p.src, t.pos)+".", p.src)
	}
	e := resolveEntity(t.val)
	if e == nil {
		return nil, aqlErr("bad_name", "unknown entity '%s'. Entities that can be changed: %s", t.val, changeableList())
	}
	if !e.hasOps() {
		return nil, errW13(e)
	}
	if (kind == "insert" && !e.CanCreate) || (kind == "update" && !e.CanUpdate) || (kind == "delete" && !e.CanDelete) {
		return nil, errW14(e, kind)
	}
	p.next()
	return e, nil
}

func (p *parser) parseWrite(kind string) (any, *AqlError) {
	sc := p.sc
	p.next()
	st := &WriteStmt{Kind: kind}
	type col struct {
		word string
		pos  int
	}
	switch kind {
	case "insert":
		if !p.peek().is("INTO") {
			return nil, errW03("expected INTO after INSERT.", p.src)
		}
		p.next()
		e, err := p.writeEntity(kind)
		if err != nil {
			return nil, err
		}
		st.E = e
		if !p.peek().punct("(") {
			return nil, errW03("expected ( and the field names after the entity.", p.src)
		}
		p.next()
		var cols []col
		for {
			t := p.peek()
			if t.kind != tkWord || isKeyword(t.val) {
				return nil, errW03("expected a field name at "+whereText(p.src, t.pos)+".", p.src)
			}
			cols = append(cols, col{t.val, t.pos})
			p.next()
			if !p.peek().punct(",") {
				break
			}
			p.next()
		}
		if !p.peek().punct(")") {
			return nil, errW03("expected ) after the field names at "+whereText(p.src, p.peek().pos)+".", p.src)
		}
		p.next()
		if !p.peek().is("VALUES") {
			return nil, errW03("expected VALUES at "+whereText(p.src, p.peek().pos)+".", p.src)
		}
		p.next()
		if !p.peek().punct("(") {
			return nil, errW03("expected ( after VALUES.", p.src)
		}
		p.next()
		var vals []Value
		for {
			v, err := p.writeValue()
			if err != nil {
				return nil, err
			}
			vals = append(vals, v)
			if !p.peek().punct(",") {
				break
			}
			p.next()
		}
		if !p.peek().punct(")") {
			return nil, errW03("expected ) after the values at "+whereText(p.src, p.peek().pos)+".", p.src)
		}
		p.next()
		if p.peek().punct(",") {
			return nil, aqlErr("syntax", "one INSERT creates one record. Send one statement per record.")
		}
		if err := p.endOfWrite(); err != nil {
			return nil, err
		}
		if len(cols) != len(vals) {
			return nil, aqlErr("syntax", "%d fields but %d values", len(cols), len(vals))
		}
		seen := map[string]bool{}
		for i, c := range cols {
			v := vals[i]
			name, isCustom, _ := splitWord(c.word)
			isParent := false
			if !isCustom {
				if strings.EqualFold(name, "parent") || strings.EqualFold(name, e.ParentKind) || catalogField(e, name) == e.ParentField {
					isParent = true
				}
			}
			if isParent {
				if st.Parent != nil && !textEq(st.Parent.written(), v.written()) {
					return nil, aqlErr("syntax", "the parent is given twice with different values")
				}
				vv := v
				st.Parent = &vv
				continue
			}
			f, err := sc.resolveField(e, c.word, true)
			if err != nil {
				return nil, err
			}
			if err := sc.checkAssignable(e, f, c.word, true); err != nil {
				return nil, err
			}
			if seen[f] {
				return nil, aqlErr("syntax", "the field %s is given twice. A statement sets each field once.", f)
			}
			seen[f] = true
			st.Fields = append(st.Fields, Assign{Field: f, Val: v, Word: c.word})
		}
		return st, nil
	case "update":
		e, err := p.writeEntity(kind)
		if err != nil {
			return nil, err
		}
		st.E = e
		if !p.peek().is("SET") {
			return nil, errW03("expected SET after the entity.", p.src)
		}
		p.next()
		var assigns []Assign
		for {
			t := p.peek()
			if t.kind != tkWord || isKeyword(t.val) {
				return nil, errW03("expected a field name at "+whereText(p.src, t.pos)+".", p.src)
			}
			p.next()
			if !(p.peek().kind == tkOp && p.peek().val == "=") {
				return nil, errW03("expected = after the field "+t.val+".", p.src)
			}
			p.next()
			v, err := p.writeValue()
			if err != nil {
				return nil, err
			}
			assigns = append(assigns, Assign{Word: t.val, Val: v})
			if p.peek().punct(",") {
				p.next()
				continue
			}
			if p.peek().is("WHERE") || p.peek().kind == tkEOF || p.peek().punct(";") {
				break
			}
			return nil, aqlErr("syntax", "a field can only be set to a value, not to an expression")
		}
		if err := p.parseWhereID(st); err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, a := range assigns {
			f, err := sc.resolveField(e, a.Word, true)
			if err != nil {
				return nil, err
			}
			if err := sc.checkAssignable(e, f, a.Word, false); err != nil {
				return nil, err
			}
			if seen[f] {
				return nil, aqlErr("syntax", "the field %s is given twice. A statement sets each field once.", f)
			}
			seen[f] = true
			a.Field = f
			st.Fields = append(st.Fields, a)
		}
		if err := sc.resolveGuards(st); err != nil {
			return nil, err
		}
		return st, nil
	}
	// delete
	if !p.peek().is("FROM") {
		return nil, errW03("expected FROM after DELETE.", p.src)
	}
	p.next()
	e, err := p.writeEntity(kind)
	if err != nil {
		return nil, err
	}
	st.E = e
	if err := p.parseWhereID(st); err != nil {
		return nil, err
	}
	if err := sc.resolveGuards(st); err != nil {
		return nil, err
	}
	return st, nil
}

func (p *parser) endOfWrite() *AqlError {
	if p.peek().punct(";") {
		p.next()
	}
	if t := p.peek(); t.kind != tkEOF {
		return errW03("unexpected text at "+whereText(p.src, t.pos)+". One statement, one record.", p.src)
	}
	return nil
}

// parseWhereID reads WHERE id = value {AND field = value}.
func (p *parser) parseWhereID(st *WriteStmt) *AqlError {
	if !p.peek().is("WHERE") {
		if t := p.peek(); t.kind != tkEOF && !t.punct(";") {
			return errW03("unexpected text at "+whereText(p.src, t.pos)+". One statement, one record.", p.src)
		}
		return errW04(strings.ToUpper(st.Kind), p.src)
	}
	p.next()
	t := p.peek()
	if t.kind != tkWord || isKeyword(t.val) || catalogField(st.E, fieldLabelName(t.val)) != st.E.idField() || st.E.idField() == "" {
		return errW05()
	}
	p.next()
	if !(p.peek().kind == tkOp && p.peek().val == "=") {
		return errW05()
	}
	p.next()
	v := p.peek()
	switch {
	case v.kind == tkText || v.kind == tkNumber:
		st.ID = v.val
		p.next()
	case v.is("NULL"):
		return aqlErr("syntax", "NULL cannot be written. Leave the field out.")
	case v.kind == tkWord && !isKeyword(v.val):
		return errS13(v.val)
	default:
		return errW03("WHERE id = needs the id in single quotes.", p.src)
	}
	for {
		n := p.peek()
		if n.is("OR") || n.is("NOT") || n.punct("(") || n.punct(")") {
			return errW05()
		}
		if !n.is("AND") {
			break
		}
		p.next()
		g := p.peek()
		if g.is("NOT") || g.punct("(") {
			return errW05()
		}
		if g.kind != tkWord || isKeyword(g.val) {
			return errW03("expected a field name at "+whereText(p.src, g.pos)+".", p.src)
		}
		p.next()
		op := p.peek()
		if !(op.kind == tkOp && op.val == "=") {
			return errW06()
		}
		p.next()
		val, err := p.parseValue()
		if err != nil {
			return err
		}
		st.Guards = append(st.Guards, Assign{Word: g.val, Val: val})
	}
	return p.endOfWrite()
}

func fieldLabelName(word string) string { n, _, _ := splitWord(word); return n }

// resolveGuards resolves the fields of the guards and checks their values.
func (sc *stmtCtx) resolveGuards(st *WriteStmt) *AqlError {
	for i, g := range st.Guards {
		f, err := sc.resolveField(st.E, g.Word, true)
		if err != nil {
			return err
		}
		if f == "" {
			return sc.errW17(fieldLabelName(g.Word), st.E)
		}
		st.Guards[i].Field = f
		if g.Val.kind == vNull {
			continue
		}
		if g.Val.kind == vText && g.Val.s == "" {
			continue
		}
		if err := checkCondValue(f, sc.fieldType(st.E, f), "=", g.Val); err != nil {
			return err
		}
	}
	return nil
}

// checkAssignable applies the write rules for a column or an assigned field.
func (sc *stmtCtx) checkAssignable(e *EntityDef, f, word string, insert bool) *AqlError {
	if f == "" {
		return sc.errW16("unknown field '"+fieldLabelName(word)+"' on "+e.Name+".", e)
	}
	if f == e.idField() {
		return aqlErr("bad_name", "the id of a record cannot be set. Fields of %s that can be set: %s", e.Name, writableList(e))
	}
	if strings.HasPrefix(f, "custom.") {
		cf := sc.custom[f]
		if cf == nil || !kindTable[cf.TypeKey].settable {
			return sc.errW16("the field "+f+" of "+e.Name+" cannot be set.", e)
		}
		if insert {
			return aqlErr("bad_name", "custom fields cannot be set in INSERT. Leave out %s, create the record, and then send: UPDATE %s SET %s = '...' WHERE id = '...'", f, e.Name, f)
		}
		return nil
	}
	fd := e.field(f)
	switch {
	case fd == nil || fd.Writable == wNo:
		return sc.errW16("the field "+f+" of "+e.Name+" cannot be set.", e)
	case fd.Writable == wInsertOnly && !insert:
		return sc.errW16("the field "+f+" of "+e.Name+" can be set in INSERT only.", e)
	}
	return nil
}
