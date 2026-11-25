// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e

package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// AqlError is the error of a statement (TYPES: AqlError).
type AqlError struct {
	Kind string
	Msg  string
}

func (e *AqlError) Error() string { return e.Msg }

func aqlErr(kind, format string, args ...any) *AqlError {
	return &AqlError{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

const (
	maxListed     = 30
	defaultLimit  = 50
	maxLimit      = 200
	maxPages      = 15
	pageSize      = 200
	maxInValues   = 20
	readFormText  = "SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]"
	writeFormText = "INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...) | UPDATE entity SET field = 'value', ... WHERE id = 'id' | DELETE FROM entity WHERE id = 'id'"
)

// listMax shows at most MaxListed entries (error-messages, step 5).
func listMax(items []string) string {
	if len(items) <= maxListed {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:maxListed], ", ") + fmt.Sprintf(", and %d more", len(items)-maxListed)
}

func (sc *stmtCtx) exampleOf(e *EntityDef) string {
	return strings.ReplaceAll(e.Example, "{KEY}", sc.key())
}

func (sc *stmtCtx) aboutOf(e *EntityDef) string {
	return strings.ReplaceAll(e.About, "{KEY}", sc.key())
}

// exampleID is <example id> of error-messages.
func (sc *stmtCtx) exampleID(kind string) string {
	k := sc.key()
	switch kind {
	case "feature":
		return k + "-23"
	case "release":
		return k + "-R-3"
	case "initiative":
		return k + "-S-1"
	case "epic":
		return k + "-E-1"
	case "goal":
		return k + "-G-1"
	case "idea":
		return k + "-I-1"
	case "requirement":
		return k + "-23-1"
	case "product":
		return k
	}
	return k + "-23"
}

func singularOf(e *EntityDef) string {
	if e.Name == "me" {
		return "user"
	}
	return e.Singular
}

// whereText is <where>: the end of the statement, or the position of a
// token and the next 24 characters of the statement from there.
func whereText(src []rune, pos int) string {
	if pos <= 0 || pos > len(src) {
		return "the end of the statement"
	}
	end := pos - 1 + 24
	if end > len(src) {
		end = len(src)
	}
	return fmt.Sprintf("position %d (%s)", pos, strings.TrimSpace(string(src[pos-1:end])))
}

func isLetter(r rune) bool      { return unicode.IsLetter(r) }
func isLetterDigit(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
func hasInnerApostrophe(src []rune) bool {
	for i := 1; i+1 < len(src); i++ {
		if src[i] == '\'' && isLetterDigit(src[i-1]) && isLetter(src[i+1]) {
			return true
		}
	}
	return false
}

// apostropheHint is <apostrophe hint> of error-messages.
func apostropheHint(src []rune) string {
	for i := 1; i+1 < len(src); i++ {
		if src[i] == '\'' && isLetterDigit(src[i-1]) && isLetter(src[i+1]) {
			a, b := i, i
			for a > 0 && !unicode.IsSpace(src[a-1]) {
				a--
			}
			for b+1 < len(src) && !unicode.IsSpace(src[b+1]) {
				b++
			}
			word := string(src[a : b+1])
			word = strings.Trim(word, "'\"%(),;")
			return fmt.Sprintf(" The text contains an apostrophe (%s). Inside text, write an apostrophe twice: %s", word, strings.ReplaceAll(word, "'", "''"))
		}
	}
	return ""
}

// ---- messages of the tool calls ----

func (sc *stmtCtx) errA01() *AqlError {
	return aqlErr("bad_args", "query parameter is required: one AQL statement. Example: %s", sc.exampleOf(entityByName("features")))
}

func (sc *stmtCtx) errA02(tool, param string, unknown []string) *AqlError {
	sort.Strings(unknown)
	hint := ""
	for _, n := range unknown {
		if tool == "query" && n == "product" {
			hint = " Name the product inside the statement: ... WHERE product = '" + sc.key() + "'"
			break
		}
	}
	if hint == "" {
		for _, n := range unknown {
			switch n {
			case "query", "statement", "aql", "sql":
				hint = " Pass the statement as " + param + "."
			}
			if hint != "" {
				break
			}
		}
	}
	return aqlErr("bad_args", "tool %s has one parameter: %s (one AQL statement). It does not accept: %s.%s", tool, param, strings.Join(unknown, ", "), hint)
}

func errA03() *AqlError {
	return aqlErr("bad_args", "statement parameter is required: one INSERT, UPDATE or DELETE statement")
}

// ---- syntax ----

func (sc *stmtCtx) errS03(src []rune, pos int, what string) *AqlError {
	return aqlErr("syntax", "cannot read the statement at %s: expected %s.%s Form: %s. Example: %s",
		whereText(src, pos), what, apostropheHint(src), readFormText, sc.exampleOf(entityByName("features")))
}

func (sc *stmtCtx) errS04(src []rune) *AqlError {
	return aqlErr("syntax", "a statement starts with SELECT or DESCRIBE.%s Form: %s. Example: %s", apostropheHint(src), readFormText, sc.exampleOf(entityByName("features")))
}

func (sc *stmtCtx) errS08() *AqlError {
	return aqlErr("syntax", "JOIN is not supported. Read one entity per statement and filter by its parent, for example: %s", sc.exampleOf(entityByName("features")))
}

func (sc *stmtCtx) errS09() *AqlError {
	return aqlErr("syntax", "HAVING is not supported. Use WHERE, GROUP BY, ORDER BY and LIMIT. Example: SELECT status, COUNT(*) FROM features WHERE product = '%s' GROUP BY status", sc.key())
}

func errS10() *AqlError {
	return aqlErr("syntax", "OFFSET is not supported. Use WHERE to narrow the result and LIMIT to shorten it.")
}

func errS11() *AqlError {
	return aqlErr("syntax", "subqueries are not supported. Send one statement per call. Form: %s", readFormText)
}

func (sc *stmtCtx) errS12(name string) *AqlError {
	return aqlErr("syntax", "unknown function %s. Functions: COUNT, SUM, AVG, MIN, MAX. Example: SELECT COUNT(*) FROM features WHERE product = '%s'", strings.ToUpper(name), sc.key())
}

func errS13(word string) *AqlError {
	return aqlErr("syntax", "text goes in single quotes: write '%s' instead of %s", word, word)
}

func (sc *stmtCtx) errS14(e *EntityDef) *AqlError {
	return aqlErr("syntax", "DESCRIBE %s lists the fields of %s and takes nothing else. To read records: %s", e.Name, e.Name, sc.exampleOf(e))
}

func errS15(label string) *AqlError {
	return aqlErr("syntax", "two select items have the label '%s'. Give one of them another label, in the form: field AS label", label)
}

func errS16(label string, e *EntityDef) *AqlError {
	return aqlErr("syntax", "the label '%s' is also the name of a field of %s. Choose another label, in the form: field AS label", label, e.Name)
}

func errW03(problem string, src []rune) *AqlError {
	return aqlErr("syntax", "%s%s Forms: %s", problem, apostropheHint(src), writeFormText)
}

// ---- names ----

func (sc *stmtCtx) customHint(e *EntityDef) (string, *AqlError) {
	list, err := sc.customFieldsOf(e)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", nil
	}
	return " Custom fields of " + e.Name + ": " + listMax(customNames(list)) + ".", nil
}

func (sc *stmtCtx) errN01(word string) *AqlError {
	return aqlErr("bad_name", "unknown entity '%s'. Entities: %s. Example: %s", word, strings.Join(entityNames(), ", "), sc.exampleOf(entityByName("features")))
}

func (sc *stmtCtx) errN02(word string, e *EntityDef) *AqlError {
	hint := ""
	if e.Name == "features" && (strings.EqualFold(word, "goal") || strings.EqualFold(word, "goal_id")) {
		hint = " Features cannot be filtered by goal."
	}
	ch, err := sc.customHint(e)
	if err != nil {
		return err
	}
	return aqlErr("bad_name", "unknown field '%s' on %s.%s Fields of %s: %s.%s Example: %s", word, e.Name, hint, e.Name, strings.Join(fieldNames(e), ", "), ch, sc.exampleOf(e))
}

func (sc *stmtCtx) errN03() *AqlError {
	u := entityByName("users")
	return aqlErr("bad_name", "users cannot be filtered by product. Fields of users: %s. Example: %s", strings.Join(fieldNames(u), ", "), sc.exampleOf(u))
}

func (sc *stmtCtx) errN06(e *EntityDef) *AqlError {
	return aqlErr("bad_name", "%s have no custom fields that can be used here. Fields of %s: %s. Example: %s", e.Name, e.Name, strings.Join(fieldNames(e), ", "), sc.exampleOf(e))
}

func (sc *stmtCtx) errN07(status int, e *EntityDef) *AqlError {
	return aqlErr("bad_name", "the custom fields of the account cannot be read with this API key (status %d). Fields of %s: %s. Example: %s", status, e.Name, strings.Join(fieldNames(e), ", "), sc.exampleOf(e))
}

// ---- scope ----

func (sc *stmtCtx) errC01(e *EntityDef) *AqlError {
	if e.Name == "comments" {
		return aqlErr("scope", "comments are listed per record: add parent = '...' to WHERE. Example: %s", sc.exampleOf(e))
	}
	parents := routeFields(e)
	if e.Name == "statuses" {
		parents = []string{"product"}
	}
	var conds []string
	for _, p := range parents {
		conds = append(conds, p+" = '...'")
	}
	return aqlErr("scope", "%s are listed per %s: add %s to WHERE. Example: %s", e.Name, strings.Join(parents, " or "), strings.Join(conds, " or "), sc.exampleOf(e))
}

// ---- products and ids ----

func errP01(value string, keys []string) *AqlError {
	sort.Strings(keys)
	return aqlErr("bad_product", "unknown product %s. Product keys: %s. To see their names: SELECT key, name FROM products", quote(value), listMax(keys))
}

func (sc *stmtCtx) errK04(value string, e *EntityDef) *AqlError {
	if e.Name == "comments" {
		return aqlErr("wrong_kind", "%s is not the id of a comment. To find the id: SELECT id, body FROM comments WHERE parent = '%s-23'", quote(value), sc.key())
	}
	return aqlErr("wrong_kind", "%s is not an id. An id looks like '%s'. To search by name: SELECT * FROM %s WHERE name CONTAINS %s", quote(value), sc.exampleID(e.Kind), e.Name, quote(value))
}

func (sc *stmtCtx) errK07(kind, value string) *AqlError {
	ke := entityOfKind(kind)
	if ke == nil {
		ke = entityByName("products")
	}
	return aqlErr("not_found", "there is no %s with the id %s. To find it: %s", kind, quote(value), sc.exampleOf(ke))
}

func (sc *stmtCtx) errK08(value string) *AqlError {
	return aqlErr("wrong_kind", "parent = needs the id of the commented record, for example '%s-23'. Got %s.", sc.key(), quote(value))
}

// ---- values ----

func errV01(field string, t FieldType, value string) *AqlError {
	var typeText, form string
	switch t {
	case tNumber:
		typeText, form = "a number", "5 or 2.5"
	case tBoolean:
		typeText, form = "true or false", "TRUE or FALSE"
	case tDate:
		typeText, form = "a date", "'YYYY-MM-DD'"
	default:
		typeText, form = "a timestamp", "'YYYY-MM-DD' or 'YYYY-MM-DDThh:mm:ssZ'"
	}
	return aqlErr("bad_value", "%s is %s: write the value as %s. Got %s.", field, typeText, form, quote(value))
}

func errV02(field string) *AqlError {
	return aqlErr("bad_value", "%s is true or false: compare it with = or !=, for example %s = TRUE", field, field)
}

// ---- writes ----

func errW01() *AqlError {
	return aqlErr("syntax", "execute only changes data. Use the query tool for SELECT and DESCRIBE.")
}

func errW04(keyword string, src []rune) *AqlError {
	return aqlErr("syntax", "%s needs WHERE id = '...' with the id of the one record to change.%s Forms: %s", keyword, apostropheHint(src), writeFormText)
}

func errW05() *AqlError {
	return aqlErr("syntax", "one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record.")
}

func errW06() *AqlError {
	return aqlErr("syntax", "after WHERE id = '...', further conditions must have the form AND field = 'value'. For a field without a value write: AND field = ''")
}

func errW13(e *EntityDef) *AqlError {
	return aqlErr("bad_name", "%s cannot be changed. Entities that can be changed: %s", e.Name, changeableList())
}

func errW14(e *EntityDef, kind string) *AqlError {
	done := map[string]string{"insert": "created", "update": "changed", "delete": "deleted"}[kind]
	var list []string
	for _, x := range catalog {
		if (kind == "insert" && x.CanCreate) || (kind == "update" && x.CanUpdate) || (kind == "delete" && x.CanDelete) {
			list = append(list, x.Name)
		}
	}
	return aqlErr("bad_name", "%s cannot be %s. Entities that can be %s: %s", e.Name, done, done, strings.Join(list, ", "))
}

// errW16 builds W16 with its opening.
func (sc *stmtCtx) errW16(opening string, e *EntityDef) *AqlError {
	parentNote := ""
	if e.CanCreate {
		parentNote = ", and parent in INSERT"
	}
	hint := ""
	list, err := sc.customFieldsOf(e)
	if err != nil {
		return err
	}
	var settable []string
	for _, c := range list {
		if kindTable[c.TypeKey].settable {
			settable = append(settable, c.Name)
		}
	}
	if len(settable) > 0 {
		hint = ". Custom fields, with UPDATE only: " + listMax(settable)
	}
	return aqlErr("bad_name", "%s Fields of %s that can be set: %s%s%s", opening, e.Name, writableList(e), parentNote, hint)
}

func (sc *stmtCtx) errW17(word string, e *EntityDef) *AqlError {
	list, err := sc.customFieldsOf(e)
	if err != nil {
		return err
	}
	hint := ""
	if len(list) > 0 {
		hint = ". Custom fields of " + e.Name + ": " + listMax(customNames(list))
	}
	return aqlErr("bad_name", "unknown field '%s' on %s. Fields of %s: %s%s", word, e.Name, e.Name, strings.Join(fieldNames(e), ", "), hint)
}

// errW25 is the message for a referenced record that does not exist.
// product is the key of the product of the changed record, or "".
func (sc *stmtCtx) errW25(kind, value, product string) *AqlError {
	value = strings.ToUpper(value)
	if kind == "requirement" {
		feat := value
		if i := strings.LastIndex(feat, "-"); i > 0 {
			feat = feat[:i]
		}
		return aqlErr("not_found", "there is no requirement with the id %s. To find the id, use the query tool: SELECT id, name FROM requirements WHERE feature = '%s'", quote(value), feat)
	}
	if product == "" {
		if i := strings.Index(value, "-"); i > 0 {
			if p, _ := sc.findProductQuiet(value[:i]); p != nil && strings.EqualFold(p.Key, value[:i]) {
				product = p.Key
			}
		}
	}
	if product == "" {
		product = sc.key()
	}
	ke := entityOfKind(kind)
	name := kind + "s"
	if ke != nil {
		name = ke.Name
	}
	return aqlErr("not_found", "there is no %s with the id %s. To find the id, use the query tool: SELECT id, name FROM %s WHERE product = '%s'", kind, quote(value), name, product)
}

// ---- failed requests ----

func answerText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(no text)"
	}
	r := []rune(s)
	if len(r) > 300 {
		r = r[:300]
	}
	return string(r)
}

// failErr maps a failed request to X04, X01, X02 or X03.
func (sc *stmtCtx) failErr(f *apiFailure) *AqlError {
	switch f.kind {
	case fNotAuthorized:
		return aqlErr("not_authorized", "Aha! refused the request (status %d): the API key is not valid for the account %s, or it has no permission for this record. Repeating the statement does not help.", f.status, sc.acct.Subdomain)
	case fRateLimited:
		return aqlErr("rate_limited", "aha: API error (status 429): too many requests")
	case fUnreachable:
		host := sc.acct.Subdomain + ".aha.io"
		var reason string
		hint := ""
		switch f.cause {
		case "no_answer":
			reason = "no answer within 60 seconds"
		case "certificate":
			reason = "the certificate of " + host + " could not be verified"
		case "proxy":
			reason = "the proxy did not open a connection to " + host
		default:
			reason = "the connection to " + host + " failed"
		}
		if f.write && f.cause != "certificate" && f.cause != "proxy" {
			hint = " The change may have been made. Read the record before you send the statement again."
		}
		return aqlErr("network", "cannot reach Aha!: %s.%s", reason, hint)
	}
	return aqlErr("api_error", "aha: API error (status %d) on %s %s: %s", f.status, f.method, f.route, answerText(f.text))
}
