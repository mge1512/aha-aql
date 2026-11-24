// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// parse-statement and resolve-names.

import (
	"fmt"
	"strconv"
	"strings"
)

// ---- statements ----

type Cond interface{}

type Cmp struct {
	Field string
	Op    string // = != < <= > >= contains like
	Value any    // string, Num, bool
}
type NullTest struct {
	Field   string
	Negated bool
}
type InList struct {
	Field  string
	Values []any
}
type AllOf struct{ Members []Cond }
type AnyOf struct{ Members []Cond }
type Negation struct{ Member Cond }

type SelectItem struct {
	Star     bool
	Field    string // "" for COUNT(*)
	Label    string
	Fn       string // COUNT SUM AVG MIN MAX, or ""
	Distinct bool
}

type OrderKey struct {
	Field string
	Label string
	Agg   *SelectItem
	Desc  bool
}

type ReadStmt struct {
	Entity   *EntityDef
	Items    []SelectItem
	Distinct bool
	Where    Cond
	GroupBy  []string
	OrderBy  []OrderKey
	Limit    *int
}

type DescribeStmt struct{ Entity *EntityDef }

type Assign struct {
	Field string
	Value any
	Word  string
}

type InsertStmt struct {
	Entity *EntityDef
	Parent any
	Fields []Assign
}

type UpdateStmt struct {
	Entity *EntityDef
	ID     any
	Fields []Assign
	Guards []Assign
}

type DeleteStmt struct {
	Entity *EntityDef
	ID     any
	Guards []Assign
}

// ---- raw syntax ----

type rawItem struct {
	star     bool
	word     Token
	fn       string
	fnStar   bool
	fnArg    Token
	distinct bool
	label    string
	hasLabel bool
}

type rawCmp struct {
	field Token
	op    string
	value any
}
type rawNull struct {
	field   Token
	negated bool
}
type rawIn struct {
	field  Token
	values []any
}
type rawAll struct{ members []any }
type rawAny struct{ members []any }
type rawNot struct{ member any }

type rawOrder struct {
	word   Token
	fn     string
	fnStar bool
	fnArg  Token
	isFn   bool
	desc   bool
}

type rawSelect struct {
	items    []rawItem
	distinct bool
	entity   *EntityDef
	where    any
	groupBy  []Token
	orderBy  []rawOrder
	limit    *int
}

type rawAssign struct {
	word  Token
	value any
}

type rawWrite struct {
	kind    string
	entity  *EntityDef
	columns []rawAssign // insert: column words with values
	assigns []rawAssign
	idWord  Token
	id      any
	guards  []rawAssign
}

type nullValue struct{}

var keywords = map[string]bool{}

func init() {
	for _, k := range strings.Fields(`SELECT FROM WHERE AND OR NOT IN IS NULL TRUE FALSE LIKE CONTAINS BETWEEN GROUP ORDER BY
		ASC DESC LIMIT OFFSET AS DISTINCT JOIN INNER LEFT RIGHT FULL CROSS OUTER NATURAL ON HAVING INSERT INTO VALUES UPDATE SET
		DELETE DESCRIBE`) {
		keywords[k] = true
	}
}

func isKeyword(t Token) bool { return t.Kind == tkWord && keywords[strings.ToUpper(t.Value)] }

var functions = map[string]bool{"COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true}

const readFormText = "SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]"
const writeFormsText = "INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...) | UPDATE entity SET field = 'value', ... WHERE id = 'id' | DELETE FROM entity WHERE id = 'id'"

type parser struct {
	toks []Token
	i    int
	text []rune
	src  string
}

func (p *parser) peek() Token { return p.toks[p.i] }
func (p *parser) peekN(n int) Token {
	if p.i+n < len(p.toks) {
		return p.toks[p.i+n]
	}
	return p.toks[len(p.toks)-1]
}
func (p *parser) next() Token {
	t := p.toks[p.i]
	if t.Kind != tkEOF {
		p.i++
	}
	return t
}

func (p *parser) where(t Token) string {
	if t.Kind == tkEOF {
		return "the end of the statement"
	}
	start := t.Pos - 1
	end := start + 24
	if end > len(p.text) {
		end = len(p.text)
	}
	return fmt.Sprintf("position %d (%s)", t.Pos, strings.TrimSpace(string(p.text[start:end])))
}

func (p *parser) hint() string { return apostropheHint(p.src) }

func exampleOf(e *EntityDef) string { return withKey(e.Example) }

func (p *parser) s03(t Token, what string) *AqlError {
	if e := p.special(t); e != nil {
		return e
	}
	return aqlErr("syntax", "cannot read the statement at %s: expected %s.%s Form: %s. Example: %s",
		p.where(t), what, p.hint(), readFormText, exampleOf(entityByName("features")))
}

// special gives the messages for constructs outside the forms.
func (p *parser) special(t Token) *AqlError {
	switch {
	case t.isWord("JOIN") || ((t.isWord("INNER") || t.isWord("LEFT") || t.isWord("RIGHT") || t.isWord("FULL") || t.isWord("CROSS") || t.isWord("OUTER") || t.isWord("NATURAL")) && p.hasJoinAfter()):
		return aqlErr("syntax", "JOIN is not supported. Read one entity per statement and filter by its parent, for example: %s", exampleOf(entityByName("features")))
	case t.isWord("HAVING"):
		return aqlErr("syntax", "HAVING is not supported. Use WHERE, GROUP BY, ORDER BY and LIMIT. Example: SELECT status, COUNT(*) FROM features WHERE product = '%s' GROUP BY status", keyMark)
	case t.isWord("OFFSET"):
		return errS10()
	case t.isPunct("(") && p.selectAfter(t):
		return errS11()
	}
	return nil
}

func (p *parser) hasJoinAfter() bool {
	for k := p.i; k < len(p.toks) && k < p.i+4; k++ {
		if p.toks[k].isWord("JOIN") {
			return true
		}
	}
	return false
}

func (p *parser) selectAfter(t Token) bool {
	for k := range p.toks {
		if p.toks[k].Pos == t.Pos && k+1 < len(p.toks) {
			return p.toks[k+1].isWord("SELECT")
		}
	}
	return false
}

func errS10() *AqlError {
	return aqlErr("syntax", "OFFSET is not supported. Use WHERE to narrow the result and LIMIT to shorten it.")
}

func errS11() *AqlError {
	return aqlErr("syntax", "subqueries are not supported. Send one statement per call. Form: %s", readFormText)
}

func (p *parser) w03(problem string) *AqlError {
	return aqlErr("syntax", "%s%s Forms: %s", problem, p.hint(), writeFormsText)
}

func (p *parser) w03at(format string, t Token) *AqlError {
	return p.w03(fmt.Sprintf(format, p.where(t)))
}

func errW05() *AqlError {
	return aqlErr("syntax", "one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record.")
}

func errS13(w string) *AqlError {
	return aqlErr("syntax", "text goes in single quotes: write '%s' instead of %s", w, w)
}

func changeable() []string {
	var out []string
	for _, e := range catalog {
		if e.HasOperation() {
			out = append(out, e.Name)
		}
	}
	return out
}

// ---- parse-statement ----

type Statement interface{}

func (r *run) parseStatement(text string, allowRead bool) (Statement, *AqlError) {
	if allowRead {
		t := strings.Trim(text, " \t\r\n'\";")
		if k := refKind(t); k != "" {
			return nil, aqlErr("syntax", "%s is an id, not a statement. To read that record: SELECT * FROM %s WHERE id = '%s'", t, entityOfKind(k).Name, t)
		}
	}
	raw, err := parseSyntax(text, allowRead, false)
	if err != nil && err.Kind == "syntax" && hasInnerApostrophe(text) {
		if raw2, err2 := parseSyntax(text, allowRead, true); err2 == nil {
			raw, err = raw2, nil
		}
	}
	if err != nil {
		return nil, err
	}
	switch x := raw.(type) {
	case *rawSelect:
		return r.resolveSelect(x)
	case *DescribeStmt:
		return x, nil
	case *rawWrite:
		return r.resolveWrite(x)
	}
	return nil, aqlErr("syntax", "cannot read the statement")
}

func parseSyntax(text string, allowRead, lenient bool) (any, *AqlError) {
	toks, err := tokenize(text, lenient)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks, text: []rune(text), src: text}
	t := p.peek()
	kind := ""
	switch {
	case t.isWord("SELECT") || t.isWord("FROM"):
		kind = "select"
	case t.Kind == tkWord && functions[strings.ToUpper(t.Value)] && p.peekN(1).isPunct("("):
		kind = "select"
	case t.isPunct("*"):
		kind = "select"
	case t.isWord("DESCRIBE"):
		kind = "describe"
	case t.isWord("INSERT"):
		kind = "insert"
	case t.isWord("UPDATE"):
		kind = "update"
	case t.isWord("DELETE"):
		kind = "delete"
	}
	isRead := kind == "select" || kind == "describe"
	if kind != "" && allowRead && !isRead {
		return nil, aqlErr("syntax", "query only reads. Use the execute tool for INSERT, UPDATE and DELETE.")
	}
	if kind != "" && !allowRead && isRead {
		return nil, aqlErr("syntax", "execute only changes data. Use the query tool for SELECT and DESCRIBE.")
	}
	if kind == "" {
		if allowRead {
			if t.Kind == tkWord && (p.peekN(1).Kind == tkEOF || (p.peekN(1).isPunct(";") && p.peekN(2).Kind == tkEOF)) {
				if e := entityWord(t.Value); e != nil {
					return nil, aqlErr("syntax", "%s is an entity, not a statement. To list its records: %s. To see its fields: DESCRIBE %s", t.Value, exampleOf(e), e.Name)
				}
			}
			if e := p.special(t); e != nil {
				return nil, e
			}
			return nil, aqlErr("syntax", "a statement starts with SELECT or DESCRIBE.%s Form: %s. Example: %s", p.hint(), readFormText, exampleOf(entityByName("features")))
		}
		return nil, aqlErr("syntax", "a statement starts with INSERT, UPDATE or DELETE.%s Forms: %s", p.hint(), writeFormsText)
	}
	switch kind {
	case "select":
		return p.parseSelect()
	case "describe":
		return p.parseDescribe()
	}
	return p.parseWrite(kind)
}

func (p *parser) endOrSemicolon() bool {
	if p.peek().isPunct(";") {
		p.next()
	}
	return p.peek().Kind == tkEOF
}

func (p *parser) parseDescribe() (any, *AqlError) {
	p.next()
	t := p.peek()
	if t.Kind == tkEOF || (t.isPunct(";") && p.peekN(1).Kind == tkEOF) {
		return &DescribeStmt{}, nil
	}
	if t.Kind != tkWord {
		return nil, p.s03(t, "an entity ("+strings.Join(entityNames(), ", ")+")")
	}
	e := entityWord(t.Value)
	if e == nil {
		return nil, errN01(t.Value)
	}
	p.next()
	if !p.endOrSemicolon() {
		return nil, aqlErr("syntax", "DESCRIBE %s lists the fields of %s and takes nothing else. To read records: %s", e.Name, e.Name, exampleOf(e))
	}
	return &DescribeStmt{Entity: e}, nil
}

func errN01(w string) *AqlError {
	return aqlErr("bad_name", "unknown entity %s. Entities: %s. Example: %s", quoteVal(w), strings.Join(entityNames(), ", "), exampleOf(entityByName("features")))
}

func (p *parser) parseSelect() (any, *AqlError) {
	st := &rawSelect{}
	t := p.peek()
	switch {
	case t.isWord("SELECT"):
		p.next()
		if p.peek().isWord("DISTINCT") {
			p.next()
			st.distinct = true
		}
		items, err := p.parseItems()
		if err != nil {
			return nil, err
		}
		st.items = items
	case t.isWord("FROM"):
		st.items = []rawItem{{star: true}}
	default:
		items, err := p.parseItems()
		if err != nil {
			return nil, err
		}
		st.items = items
	}
	if !p.peek().isWord("FROM") {
		return nil, p.s03(p.peek(), "FROM and an entity")
	}
	p.next()
	et := p.peek()
	if et.isPunct("(") && p.peekN(1).isWord("SELECT") {
		return nil, errS11()
	}
	if et.Kind != tkWord || isKeyword(et) {
		return nil, p.s03(et, "an entity ("+strings.Join(entityNames(), ", ")+")")
	}
	p.next()
	st.entity = entityWord(et.Value)
	if st.entity == nil {
		return nil, errN01(et.Value)
	}
	// alias
	if p.peek().isWord("AS") {
		p.next()
		if p.peek().Kind != tkWord || isKeyword(p.peek()) {
			return nil, p.s03(p.peek(), "a name after AS")
		}
		p.next()
	} else if p.peek().Kind == tkWord && !isKeyword(p.peek()) {
		p.next()
	}
	if p.peek().isPunct(",") {
		return nil, aqlErr("syntax", "JOIN is not supported. Read one entity per statement and filter by its parent, for example: %s", exampleOf(entityByName("features")))
	}
	if e := p.special(p.peek()); e != nil {
		return nil, e
	}
	if p.peek().isWord("WHERE") {
		p.next()
		c, err := p.parseOr(st.entity)
		if err != nil {
			return nil, err
		}
		st.where = c
	}
	if p.peek().isWord("GROUP") {
		p.next()
		if !p.peek().isWord("BY") {
			return nil, p.s03(p.peek(), "BY after GROUP")
		}
		p.next()
		for {
			t := p.peek()
			if t.Kind != tkWord || isKeyword(t) {
				return nil, p.s03(t, "a field name")
			}
			p.next()
			st.groupBy = append(st.groupBy, t)
			if !p.peek().isPunct(",") {
				break
			}
			p.next()
		}
	}
	if p.peek().isWord("HAVING") {
		return nil, p.special(p.peek())
	}
	if p.peek().isWord("ORDER") {
		p.next()
		if !p.peek().isWord("BY") {
			return nil, p.s03(p.peek(), "BY after ORDER")
		}
		p.next()
		for {
			t := p.peek()
			if t.Kind != tkWord || isKeyword(t) {
				return nil, p.s03(t, "a field name")
			}
			p.next()
			ok := rawOrder{word: t}
			if p.peek().isPunct("(") {
				fn := strings.ToUpper(t.Value)
				if !functions[fn] {
					return nil, errS12(t.Value)
				}
				p.next()
				ok.isFn, ok.fn = true, fn
				a := p.peek()
				switch {
				case a.isPunct("*"):
					ok.fnStar = true
				case a.Kind == tkWord && !isKeyword(a):
					ok.fnArg = a
				case a.Kind == tkNumber && a.Value == "1" && fn == "COUNT":
					ok.fnStar = true
				default:
					return nil, p.s03(a, "a field or * inside "+fn+"( )")
				}
				p.next()
				if !p.peek().isPunct(")") {
					return nil, p.s03(p.peek(), "')'")
				}
				p.next()
			}
			if p.peek().isWord("ASC") {
				p.next()
			} else if p.peek().isWord("DESC") {
				p.next()
				ok.desc = true
			}
			st.orderBy = append(st.orderBy, ok)
			if !p.peek().isPunct(",") {
				break
			}
			p.next()
		}
	}
	if p.peek().isWord("LIMIT") {
		p.next()
		t := p.peek()
		n, err := strconv.Atoi(t.Value)
		if t.Kind != tkNumber || err != nil || n < 0 || strings.ContainsAny(t.Value, ".-") {
			return nil, p.s03(t, "a number after LIMIT")
		}
		p.next()
		st.limit = &n
		if p.peek().isWord("OFFSET") || p.peek().Kind == tkNumber || (p.peek().isPunct(",") && p.peekN(1).Kind == tkNumber) {
			return nil, errS10()
		}
	}
	if !p.endOrSemicolon() {
		return nil, p.s03(p.peek(), "the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order")
	}
	return st, nil
}

func errS12(name string) *AqlError {
	return aqlErr("syntax", "unknown function %s. Functions: COUNT, SUM, AVG, MIN, MAX. Example: SELECT COUNT(*) FROM features WHERE product = '%s'", strings.ToUpper(name), keyMark)
}

func (p *parser) parseItems() ([]rawItem, *AqlError) {
	var items []rawItem
	first := true
	for {
		it, err := p.parseItem(first)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
		first = false
		if !p.peek().isPunct(",") {
			return items, nil
		}
		p.next()
	}
}

func (p *parser) parseItem(first bool) (rawItem, *AqlError) {
	t := p.peek()
	what := "a field, * or COUNT(*)"
	if first {
		what = "a field, * or COUNT(*) after SELECT"
	}
	var it rawItem
	switch {
	case t.isPunct("*"):
		p.next()
		it.star = true
		return it, nil
	case t.Kind == tkWord && strings.HasSuffix(t.Value, ".") && p.peekN(1).isPunct("*"):
		p.next()
		p.next()
		it.star = true
		return it, nil
	case t.Kind == tkWord && p.peekN(1).isPunct("("):
		fn := strings.ToUpper(t.Value)
		if !functions[fn] {
			return it, errS12(t.Value)
		}
		p.next()
		p.next()
		it.fn = fn
		a := p.peek()
		if a.isWord("SELECT") {
			return it, errS11()
		}
		if a.isWord("DISTINCT") {
			p.next()
			it.distinct = true
			a = p.peek()
		}
		switch {
		case a.isPunct("*") && !it.distinct && fn == "COUNT":
			it.fnStar = true
		case a.Kind == tkNumber && a.Value == "1" && fn == "COUNT" && !it.distinct:
			it.fnStar = true
		case a.Kind == tkWord && !isKeyword(a):
			it.fnArg = a
		default:
			return it, p.s03(a, "a field or * inside "+fn+"( )")
		}
		p.next()
		if !p.peek().isPunct(")") {
			return it, p.s03(p.peek(), "')'")
		}
		p.next()
	case t.Kind == tkWord && !isKeyword(t):
		p.next()
		it.word = t
	default:
		return it, p.s03(t, what)
	}
	if p.peek().isWord("AS") {
		p.next()
		l := p.peek()
		if (l.Kind != tkWord && l.Kind != tkText) || isKeyword(l) {
			return it, p.s03(l, "a name after AS")
		}
		p.next()
		it.label, it.hasLabel = l.Value, true
	}
	return it, nil
}

func (p *parser) parseOr(e *EntityDef) (any, *AqlError) {
	c, err := p.parseAnd(e)
	if err != nil {
		return nil, err
	}
	members := []any{c}
	for p.peek().isWord("OR") {
		p.next()
		c, err := p.parseAnd(e)
		if err != nil {
			return nil, err
		}
		members = append(members, c)
	}
	if len(members) == 1 {
		return c, nil
	}
	return &rawAny{members}, nil
}

func (p *parser) parseAnd(e *EntityDef) (any, *AqlError) {
	var members []any
	for {
		c, err := p.parseNot(e)
		if err != nil {
			return nil, err
		}
		if a, ok := c.(*rawAll); ok && a.members != nil && isBetween(a) {
			members = append(members, a.members...)
		} else {
			members = append(members, c)
		}
		if !p.peek().isWord("AND") {
			break
		}
		p.next()
	}
	if len(members) == 1 {
		return members[0], nil
	}
	return &rawAll{members}, nil
}

// a BETWEEN is an AllOf of two comparisons, marked by its operators.
func isBetween(a *rawAll) bool {
	if len(a.members) != 2 {
		return false
	}
	c1, ok1 := a.members[0].(*rawCmp)
	c2, ok2 := a.members[1].(*rawCmp)
	return ok1 && ok2 && c1.op == ">=" && c2.op == "<=" && c1.field.Pos == c2.field.Pos
}

func (p *parser) parseNot(e *EntityDef) (any, *AqlError) {
	if p.peek().isWord("NOT") {
		p.next()
		c, err := p.parseNot(e)
		if err != nil {
			return nil, err
		}
		return &rawNot{c}, nil
	}
	if p.peek().isPunct("(") {
		if p.peekN(1).isWord("SELECT") {
			return nil, errS11()
		}
		p.next()
		c, err := p.parseOr(e)
		if err != nil {
			return nil, err
		}
		if !p.peek().isPunct(")") {
			return nil, p.s03(p.peek(), "')'")
		}
		p.next()
		return c, nil
	}
	return p.parseComparison(e)
}

func (p *parser) readValue() (any, *AqlError) {
	t := p.peek()
	switch {
	case t.Kind == tkText:
		p.next()
		return t.Value, nil
	case t.Kind == tkNumber:
		p.next()
		return Num(t.Value), nil
	case t.isWord("TRUE"):
		p.next()
		return true, nil
	case t.isWord("FALSE"):
		p.next()
		return false, nil
	case t.isWord("NULL"):
		p.next()
		return nullValue{}, nil
	case t.isPunct("(") && p.peekN(1).isWord("SELECT"):
		return nil, errS11()
	case t.Kind == tkWord && !isKeyword(t):
		return nil, errS13(t.Value)
	}
	return nil, p.s03(t, "a value in single quotes")
}

func (p *parser) parseComparison(e *EntityDef) (any, *AqlError) {
	ft := p.peek()
	if ft.Kind != tkWord || isKeyword(ft) {
		return nil, p.s03(ft, "a field name ("+fieldList(e)+")")
	}
	p.next()
	t := p.peek()
	negated := false
	if t.isWord("NOT") {
		n := p.peekN(1)
		if n.isWord("IN") || n.isWord("CONTAINS") || n.isWord("LIKE") || n.isWord("BETWEEN") {
			p.next()
			negated = true
			t = p.peek()
		}
	}
	var c any
	switch {
	case t.Kind == tkOp && !negated:
		p.next()
		v, err := p.readValue()
		if err != nil {
			return nil, err
		}
		if _, isNull := v.(nullValue); isNull {
			switch t.Value {
			case "=":
				return &rawNull{field: ft}, nil
			case "!=":
				return &rawNull{field: ft, negated: true}, nil
			}
			return nil, p.s03(p.toks[p.i-1], "a value in single quotes")
		}
		return &rawCmp{field: ft, op: t.Value, value: v}, nil
	case t.isWord("IS") && !negated:
		p.next()
		neg := false
		if p.peek().isWord("NOT") {
			p.next()
			neg = true
		}
		if !p.peek().isWord("NULL") {
			return nil, p.s03(p.peek(), "NULL after IS")
		}
		p.next()
		return &rawNull{field: ft, negated: neg}, nil
	case t.isWord("IN"):
		p.next()
		if !p.peek().isPunct("(") {
			return nil, p.s03(p.peek(), "'(' after IN")
		}
		p.next()
		if p.peek().isWord("SELECT") {
			return nil, errS11()
		}
		var vals []any
		for {
			v, err := p.readValue()
			if err != nil {
				return nil, err
			}
			if _, isNull := v.(nullValue); isNull {
				v = nil
			}
			vals = append(vals, v)
			if p.peek().isPunct(",") {
				p.next()
				continue
			}
			break
		}
		if !p.peek().isPunct(")") {
			return nil, p.s03(p.peek(), "')'")
		}
		p.next()
		c = &rawIn{field: ft, values: vals}
	case t.isWord("CONTAINS") || t.isWord("LIKE"):
		p.next()
		v, err := p.readValue()
		if err != nil {
			return nil, err
		}
		if _, isNull := v.(nullValue); isNull {
			return nil, p.s03(p.toks[p.i-1], "a value in single quotes")
		}
		c = &rawCmp{field: ft, op: strings.ToLower(t.Value), value: v}
	case t.isWord("BETWEEN"):
		p.next()
		a, err := p.readValue()
		if err != nil {
			return nil, err
		}
		if !p.peek().isWord("AND") {
			return nil, p.s03(p.peek(), "AND in BETWEEN")
		}
		p.next()
		b, err := p.readValue()
		if err != nil {
			return nil, err
		}
		_, an := a.(nullValue)
		_, bn := b.(nullValue)
		if an || bn {
			return nil, p.s03(p.toks[p.i-1], "a value in single quotes")
		}
		c = &rawAll{[]any{&rawCmp{field: ft, op: ">=", value: a}, &rawCmp{field: ft, op: "<=", value: b}}}
	default:
		return nil, p.s03(t, "a comparison after the field "+ft.Value+" (=, !=, <, >, CONTAINS, LIKE, IN, IS NULL)")
	}
	if negated {
		return &rawNot{c}, nil
	}
	return c, nil
}

// ---- write forms ----

func (p *parser) writeValue() (any, *AqlError) {
	t := p.peek()
	switch {
	case t.Kind == tkText:
		p.next()
		return t.Value, nil
	case t.Kind == tkNumber:
		p.next()
		return Num(t.Value), nil
	case t.isWord("TRUE"):
		p.next()
		return true, nil
	case t.isWord("FALSE"):
		p.next()
		return false, nil
	case t.isWord("NULL"):
		return nil, aqlErr("syntax", "NULL cannot be written. Leave the field out.")
	case t.Kind == tkWord && !isKeyword(t):
		return nil, errS13(t.Value)
	}
	return nil, p.w03at("expected a value in single quotes at %s.", t)
}

func (p *parser) writeEntity(kind string) (*EntityDef, *AqlError) {
	t := p.peek()
	if t.Kind != tkWord || isKeyword(t) {
		switch kind {
		case "insert":
			return nil, p.w03at("expected a field name at %s.", t)
		}
		return nil, p.w03at("unexpected text at %s. One statement, one record.", t)
	}
	p.next()
	e := entityWord(t.Value)
	if e == nil {
		return nil, aqlErr("bad_name", "unknown entity %s. Entities that can be changed: %s", quoteVal(t.Value), strings.Join(changeable(), ", "))
	}
	if !e.HasOperation() {
		return nil, aqlErr("bad_name", "%s cannot be changed. Entities that can be changed: %s", e.Name, strings.Join(changeable(), ", "))
	}
	ok, done := e.CanCreate, "created"
	switch kind {
	case "update":
		ok, done = e.CanUpdate, "changed"
	case "delete":
		ok, done = e.CanDelete, "deleted"
	}
	if !ok {
		var l []string
		for _, x := range catalog {
			if (kind == "insert" && x.CanCreate) || (kind == "update" && x.CanUpdate) || (kind == "delete" && x.CanDelete) {
				l = append(l, x.Name)
			}
		}
		return nil, aqlErr("bad_name", "%s cannot be %s. Entities that can be %s: %s", e.Name, done, done, strings.Join(l, ", "))
	}
	return e, nil
}

func (p *parser) parseWrite(kind string) (any, *AqlError) {
	w := &rawWrite{kind: kind}
	p.next()
	switch kind {
	case "insert":
		if !p.peek().isWord("INTO") {
			return nil, p.w03("expected INTO after INSERT.")
		}
		p.next()
		e, err := p.writeEntity(kind)
		if err != nil {
			return nil, err
		}
		w.entity = e
		if !p.peek().isPunct("(") {
			return nil, p.w03("expected ( and the field names after the entity.")
		}
		p.next()
		var cols []Token
		for {
			t := p.peek()
			if t.Kind != tkWord || isKeyword(t) {
				return nil, p.w03at("expected a field name at %s.", t)
			}
			p.next()
			cols = append(cols, t)
			if p.peek().isPunct(",") {
				p.next()
				continue
			}
			break
		}
		if !p.peek().isPunct(")") {
			return nil, p.w03at("expected ) after the field names at %s.", p.peek())
		}
		p.next()
		if !p.peek().isWord("VALUES") {
			return nil, p.w03at("expected VALUES at %s.", p.peek())
		}
		p.next()
		if !p.peek().isPunct("(") {
			return nil, p.w03("expected ( after VALUES.")
		}
		p.next()
		var vals []any
		for {
			v, err := p.writeValue()
			if err != nil {
				return nil, err
			}
			vals = append(vals, v)
			if p.peek().isPunct(",") {
				p.next()
				continue
			}
			break
		}
		if !p.peek().isPunct(")") {
			return nil, p.w03at("expected ) after the values at %s.", p.peek())
		}
		p.next()
		if p.peek().isPunct(",") && p.peekN(1).isPunct("(") {
			return nil, aqlErr("syntax", "one INSERT creates one record. Send one statement per record.")
		}
		if !p.endOrSemicolon() {
			return nil, p.w03at("unexpected text at %s. One statement, one record.", p.peek())
		}
		if len(cols) != len(vals) {
			return nil, aqlErr("syntax", "%d fields but %d values", len(cols), len(vals))
		}
		for i := range cols {
			w.columns = append(w.columns, rawAssign{word: cols[i], value: vals[i]})
		}
		return w, nil
	case "update":
		e, err := p.writeEntity(kind)
		if err != nil {
			return nil, err
		}
		w.entity = e
		if !p.peek().isWord("SET") {
			return nil, p.w03("expected SET after the entity.")
		}
		p.next()
		for {
			t := p.peek()
			if t.Kind != tkWord || isKeyword(t) {
				return nil, p.w03at("expected a field name at %s.", t)
			}
			p.next()
			if !(p.peek().Kind == tkOp && p.peek().Value == "=") {
				return nil, p.w03(fmt.Sprintf("expected = after the field %s.", t.Value))
			}
			p.next()
			v, err := p.writeValue()
			if err != nil {
				return nil, err
			}
			w.assigns = append(w.assigns, rawAssign{word: t, value: v})
			n := p.peek()
			if n.isPunct(",") {
				p.next()
				continue
			}
			if n.isWord("WHERE") || n.Kind == tkEOF || n.isPunct(";") {
				break
			}
			return nil, aqlErr("syntax", "a field can only be set to a value, not to an expression")
		}
	case "delete":
		if !p.peek().isWord("FROM") {
			return nil, p.w03("expected FROM after DELETE.")
		}
		p.next()
		e, err := p.writeEntity(kind)
		if err != nil {
			return nil, err
		}
		w.entity = e
	}
	if !p.peek().isWord("WHERE") {
		if p.peek().Kind == tkEOF || p.peek().isPunct(";") {
			return nil, aqlErr("syntax", "%s needs WHERE id = '...' with the id of the one record to change.%s Forms: %s", strings.ToUpper(kind), p.hint(), writeFormsText)
		}
		return nil, p.w03at("unexpected text at %s. One statement, one record.", p.peek())
	}
	p.next()
	// first condition: the id field with =
	t := p.peek()
	if t.Kind != tkWord || isKeyword(t) || catalogName(w.entity, t.Value) != w.entity.IDField() || w.entity.IDField() == "" {
		return nil, errW05()
	}
	p.next()
	if !(p.peek().Kind == tkOp && p.peek().Value == "=") {
		return nil, errW05()
	}
	p.next()
	v := p.peek()
	switch {
	case v.Kind == tkText:
		w.id = v.Value
	case v.Kind == tkNumber:
		w.id = v.Value
	case v.isWord("NULL"):
		return nil, aqlErr("syntax", "NULL cannot be written. Leave the field out.")
	case v.Kind == tkWord && !isKeyword(v):
		return nil, errS13(v.Value)
	default:
		return nil, p.w03("WHERE id = needs the id in single quotes.")
	}
	p.next()
	for {
		n := p.peek()
		if n.isWord("OR") {
			return nil, errW05()
		}
		if !n.isWord("AND") {
			break
		}
		p.next()
		g := p.peek()
		if g.isWord("NOT") || g.isPunct("(") {
			return nil, errW05()
		}
		if g.Kind != tkWord || isKeyword(g) {
			return nil, p.w03at("expected a field name at %s.", g)
		}
		p.next()
		op := p.peek()
		if !(op.Kind == tkOp && op.Value == "=") {
			if op.Kind == tkOp || op.isWord("IS") || op.isWord("IN") || op.isWord("LIKE") || op.isWord("CONTAINS") || op.isWord("BETWEEN") || op.isWord("NOT") {
				return nil, aqlErr("syntax", "after WHERE id = '...', further conditions must have the form AND field = 'value'")
			}
			return nil, p.w03(fmt.Sprintf("expected = after the field %s.", g.Value))
		}
		p.next()
		val, err := p.writeValue()
		if err != nil {
			return nil, err
		}
		w.guards = append(w.guards, rawAssign{word: g, value: val})
	}
	if !p.endOrSemicolon() {
		return nil, p.w03at("unexpected text at %s. One statement, one record.", p.peek())
	}
	return w, nil
}

// ---- resolve-names ----

func entityWord(w string) *EntityDef {
	for _, e := range catalog {
		if strings.EqualFold(e.Name, w) || strings.EqualFold(e.Singular, w) {
			return e
		}
		for _, o := range e.OtherNames {
			if strings.EqualFold(o, w) {
				return e
			}
		}
	}
	return nil
}

// splitWord gives the name part of a field word and whether it is a custom
// field word.
func splitWord(word string) (string, bool) {
	i := strings.LastIndex(word, ".")
	if i < 0 {
		return word, false
	}
	name, before := word[i+1:], strings.ToLower(word[:i])
	return name, before == "custom" || strings.HasSuffix(before, ".custom")
}

// labelOf is the word without a qualifier; custom. belongs to the word.
func labelOf(word string) string {
	name, custom := splitWord(word)
	if custom {
		i := strings.LastIndex(word, ".")
		j := strings.LastIndex(word[:i], ".")
		return word[j+1:]
	}
	return name
}

// catalogName resolves a field word by the steps 4 a to 4 d: no request.
func catalogName(e *EntityDef, word string) string {
	name, custom := splitWord(word)
	if custom {
		return ""
	}
	l := strings.ToLower(name)
	if e.HasField(l) {
		return l
	}
	for k, v := range e.OtherFieldNames {
		if strings.EqualFold(k, l) {
			return v
		}
	}
	if e.Name != "products" {
		if v, ok := commonOtherNames[l]; ok && e.HasField(v) {
			return v
		}
	}
	s := strings.ToLower(e.Singular)
	if (l == s || l == s+"_id") && e.IDField() != "" {
		return e.IDField()
	}
	if l == s+"_name" && e.HasField("name") {
		return "name"
	}
	return ""
}

// resolveField is resolve-names for a field word. "" without an error means
// none (only when writing).
func (r *run) resolveField(e *EntityDef, word string, writing bool) (string, *AqlError) {
	name, custom := splitWord(word)
	if custom {
		cf, err := r.customField(e, name, labelOf(word), true)
		if err != nil {
			return "", err
		}
		return cf.Name, nil
	}
	if n := catalogName(e, word); n != "" {
		return n, nil
	}
	cf, err := r.customField(e, name, name, false)
	if err != nil {
		return "", err
	}
	if cf != nil {
		return cf.Name, nil
	}
	if writing {
		return "", nil
	}
	if e.Name == "users" {
		for _, w := range productWords {
			if strings.EqualFold(w, name) {
				return "", aqlErr("bad_name", "users cannot be filtered by product. Fields of users: %s. Example: %s", fieldList(e), exampleOf(e))
			}
		}
	}
	hint := ""
	if e.Name == "features" && (strings.EqualFold(name, "goal") || strings.EqualFold(name, "goal_id")) {
		hint = " Features cannot be filtered by goal."
	}
	return "", aqlErr("bad_name", "unknown field %s on %s.%s Fields of %s: %s.%s Example: %s", quoteVal(name), e.Name, hint, e.Name, fieldList(e), r.customHint(e), exampleOf(e))
}

// ---- resolution of a read statement ----

func (r *run) resolveSelect(raw *rawSelect) (*ReadStmt, *AqlError) {
	e := raw.entity
	st := &ReadStmt{Entity: e, Distinct: raw.distinct, Limit: raw.limit}
	for _, it := range raw.items {
		switch {
		case it.star:
			st.Items = append(st.Items, SelectItem{Star: true})
		case it.fn != "":
			si := SelectItem{Fn: it.fn, Distinct: it.distinct}
			argText := "*"
			if !it.fnStar {
				argText = it.fnArg.Value
				if it.fn == "COUNT" && !it.distinct && catalogName(e, it.fnArg.Value) == "" && entityWord(it.fnArg.Value) != nil {
					argText = "*"
				} else {
					f, err := r.resolveField(e, it.fnArg.Value, false)
					if err != nil {
						return nil, err
					}
					si.Field = f
				}
			}
			if it.distinct {
				argText = "DISTINCT " + argText
			}
			si.Label = it.fn + "(" + argText + ")"
			if it.hasLabel {
				si.Label = it.label
			}
			st.Items = append(st.Items, si)
		default:
			f, err := r.resolveField(e, it.word.Value, false)
			if err != nil {
				return nil, err
			}
			si := SelectItem{Field: f, Label: labelOf(it.word.Value)}
			if it.hasLabel {
				si.Label = it.label
			}
			st.Items = append(st.Items, si)
		}
	}
	if raw.where != nil {
		c, err := r.resolveCond(e, raw.where)
		if err != nil {
			return nil, err
		}
		st.Where = c
	}
	for _, g := range raw.groupBy {
		f, err := r.resolveField(e, g.Value, false)
		if err != nil {
			return nil, err
		}
		st.GroupBy = append(st.GroupBy, f)
	}
	for _, o := range raw.orderBy {
		k := OrderKey{Desc: o.desc}
		if o.isFn {
			agg := &SelectItem{Fn: o.fn}
			if !o.fnStar {
				f, err := r.resolveField(e, o.fnArg.Value, false)
				if err != nil {
					return nil, err
				}
				agg.Field = f
			}
			k.Agg = agg
			for i := range st.Items {
				it := st.Items[i]
				if it.Fn == agg.Fn && it.Field == agg.Field && !it.Distinct {
					k.Label = it.Label
				}
			}
		} else {
			found := false
			for _, it := range st.Items {
				if !it.Star && it.Label != "" && strings.EqualFold(it.Label, o.word.Value) && (it.Fn != "" || it.Label != labelOf(it.Field)) {
					if it.Fn != "" {
						k.Label = it.Label
						cp := it
						k.Agg = &cp
					} else {
						k.Field = it.Field
					}
					found = true
					break
				}
			}
			if !found {
				f, err := r.resolveField(e, o.word.Value, false)
				if err != nil {
					return nil, err
				}
				k.Field = f
			}
		}
		st.OrderBy = append(st.OrderBy, k)
	}
	return st, nil
}

func (r *run) resolveCond(e *EntityDef, c any) (Cond, *AqlError) {
	switch x := c.(type) {
	case *rawCmp:
		f, err := r.resolveField(e, x.field.Value, false)
		if err != nil {
			return nil, err
		}
		return &Cmp{Field: f, Op: x.op, Value: x.value}, nil
	case *rawNull:
		f, err := r.resolveField(e, x.field.Value, false)
		if err != nil {
			return nil, err
		}
		return &NullTest{Field: f, Negated: x.negated}, nil
	case *rawIn:
		f, err := r.resolveField(e, x.field.Value, false)
		if err != nil {
			return nil, err
		}
		return &InList{Field: f, Values: x.values}, nil
	case *rawAll:
		out := &AllOf{}
		for _, m := range x.members {
			rc, err := r.resolveCond(e, m)
			if err != nil {
				return nil, err
			}
			out.Members = append(out.Members, rc)
		}
		return out, nil
	case *rawAny:
		out := &AnyOf{}
		for _, m := range x.members {
			rc, err := r.resolveCond(e, m)
			if err != nil {
				return nil, err
			}
			out.Members = append(out.Members, rc)
		}
		return out, nil
	case *rawNot:
		rc, err := r.resolveCond(e, x.member)
		if err != nil {
			return nil, err
		}
		return &Negation{rc}, nil
	}
	return nil, aqlErr("syntax", "cannot read the condition")
}

// ---- resolution of a write statement ----

func writableList(e *EntityDef) string {
	var l []string
	for _, f := range e.Fields {
		switch f.Writable {
		case wYes:
			l = append(l, f.Name)
		case wInsertOnly:
			l = append(l, f.Name+" (INSERT only)")
		}
	}
	return strings.Join(l, ", ")
}

func (r *run) errW16(e *EntityDef, opening string) *AqlError {
	note := ""
	if e.CanCreate {
		note = ", and parent in INSERT"
	}
	hint := ""
	if l := r.customNames(e, true); len(l) > 0 {
		hint = ". Custom fields, with UPDATE only: " + listMax(l)
	}
	return aqlErr("bad_name", "%s Fields of %s that can be set: %s%s%s", opening, e.Name, writableList(e), note, hint)
}

func errW15(e *EntityDef) *AqlError {
	return aqlErr("bad_name", "the id of a record cannot be set. Fields of %s that can be set: %s", e.Name, writableList(e))
}

// assignedField checks one column of INSERT or one assignment of UPDATE.
func (r *run) assignedField(e *EntityDef, word string, insert bool) (string, *AqlError) {
	f, err := r.resolveField(e, word, true)
	if err != nil {
		return "", err
	}
	if f == "" {
		return "", r.errW16(e, fmt.Sprintf("unknown field %s on %s.", quoteVal(labelOf(word)), e.Name))
	}
	if f == e.IDField() {
		return "", errW15(e)
	}
	if cf := r.custom[f]; cf != nil && strings.HasPrefix(f, "custom.") {
		if cf.Kind == "scorecard" {
			return "", r.errW16(e, fmt.Sprintf("the field %s of %s cannot be set.", f, e.Name))
		}
		if insert {
			return "", aqlErr("bad_name", "custom fields cannot be set in INSERT. Leave out %s, create the record, and then send: UPDATE %s SET %s = '...' WHERE id = '...'", f, e.Name, f)
		}
		return f, nil
	}
	fd := e.Field(f)
	switch {
	case fd == nil || fd.Writable == wNo:
		return "", r.errW16(e, fmt.Sprintf("the field %s of %s cannot be set.", f, e.Name))
	case fd.Writable == wInsertOnly && !insert:
		return "", r.errW16(e, fmt.Sprintf("the field %s of %s can be set in INSERT only.", f, e.Name))
	}
	return f, nil
}

func (r *run) isParentColumn(e *EntityDef, word string) bool {
	name, custom := splitWord(word)
	if custom {
		return false
	}
	l := strings.ToLower(name)
	if l == "parent" || (e.ParentKind != "any" && l == e.ParentKind) {
		return true
	}
	return catalogName(e, word) == e.ParentField
}

func (r *run) resolveWrite(w *rawWrite) (Statement, *AqlError) {
	e := w.entity
	switch w.kind {
	case "insert":
		st := &InsertStmt{Entity: e}
		hasParent := false
		for _, c := range w.columns {
			if r.isParentColumn(e, c.word.Value) {
				if hasParent && !strings.EqualFold(textOf(st.Parent), textOf(c.value)) {
					return nil, aqlErr("syntax", "the parent is given twice with different values")
				}
				hasParent = true
				st.Parent = c.value
				continue
			}
			f, err := r.assignedField(e, c.word.Value, true)
			if err != nil {
				return nil, err
			}
			st.Fields = append(st.Fields, Assign{Field: f, Value: c.value, Word: c.word.Value})
		}
		return st, nil
	case "update":
		st := &UpdateStmt{Entity: e, ID: w.id}
		for _, a := range w.assigns {
			f, err := r.assignedField(e, a.word.Value, false)
			if err != nil {
				return nil, err
			}
			st.Fields = append(st.Fields, Assign{Field: f, Value: a.value, Word: a.word.Value})
		}
		gs, err := r.resolveGuards(e, w.guards)
		if err != nil {
			return nil, err
		}
		st.Guards = gs
		return st, nil
	}
	st := &DeleteStmt{Entity: e, ID: w.id}
	gs, err := r.resolveGuards(e, w.guards)
	if err != nil {
		return nil, err
	}
	st.Guards = gs
	return st, nil
}

func (r *run) resolveGuards(e *EntityDef, raw []rawAssign) ([]Assign, *AqlError) {
	var out []Assign
	for _, g := range raw {
		f, err := r.resolveField(e, g.word.Value, true)
		if err != nil {
			return nil, err
		}
		if f == "" {
			hint := ""
			if l := r.customNames(e, false); len(l) > 0 {
				hint = ". Custom fields of " + e.Name + ": " + listMax(l)
			}
			return nil, aqlErr("bad_name", "unknown field %s on %s. Fields of %s: %s%s", quoteVal(labelOf(g.word.Value)), e.Name, e.Name, fieldList(e), hint)
		}
		out = append(out, Assign{Field: f, Value: g.value, Word: g.word.Value})
	}
	return out, nil
}
