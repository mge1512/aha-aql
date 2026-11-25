// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268

package main

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// cfDef is one custom field definition of the account.
type cfDef struct {
	id, key, name, typeName, fieldable string
}

// CustomField is one custom field of an entity (TYPES: CustomField).
type CustomField struct {
	Name     string // "custom." and the key in lower case
	Key      string // as the definition writes it
	Label    string
	DefID    string
	TypeKey  string // type name in lower case, a key of kindTable
	TypeName string // type name as the definition writes it
}

type kindInfo struct {
	kind     string
	ftype    FieldType
	desc     string
	settable bool
}

// kindTable is the table of kinds of custom-fields.
var kindTable = map[string]kindInfo{
	"textfield":              {"text", tString, "string", true},
	"urlfield":               {"url", tString, "string (web address)", true},
	"notefield":              {"note", tText, "text", true},
	"numberfield":            {"number", tNumber, "number", true},
	"datefield":              {"date", tDate, "date (YYYY-MM-DD)", true},
	"selecteditable":         {"choice", tString, "string", true},
	"selectconstant":         {"fixed_choice", tString, "one option of a list", true},
	"selectmultipleeditable": {"tags", tString, "tags, separated by commas", true},
	"selectmultipleconstant": {"fixed_tags", tString, "options of a list, separated by commas", true},
	"usersfield":             {"users", tUser, "users (e-mail addresses), separated by commas", true},
	"scorecardfield":         {"scorecard", tNumber, "number", false},
}

func (c *CustomField) kind() string { return kindTable[c.TypeKey].kind }

func afterColons(s string) string {
	if i := strings.LastIndex(s, "::"); i >= 0 {
		return s[i+2:]
	}
	return s
}

func customNames(list []*CustomField) []string {
	var out []string
	for _, c := range list {
		out = append(out, c.Name)
	}
	return out
}

func sortStrings(s []string) { sort.Strings(s) }

// loadDefs reads the definitions at most once per statement
// (custom-fields, step 2).
func (sc *stmtCtx) loadDefs() *AqlError {
	if sc.defsLoaded {
		return nil
	}
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(pageSize))
	var defs []cfDef
	for page := 1; ; page++ {
		params.Set("page", strconv.Itoa(page))
		lp, f := sc.c.apiList(sc.ctx, sc.acct, "/custom_field_definitions", params)
		if f != nil {
			if f.kind == fNotAuthorized || f.kind == fNotFound {
				sc.defsLoaded = true
				sc.defsStatus = f.status
				sc.defs = nil
				return nil
			}
			return sc.failErr(f)
		}
		for _, r := range lp.records {
			defs = append(defs, cfDef{id: strOf(r["id"]), key: strOf(r["key"]), name: strOf(r["name"]),
				typeName: afterColons(strOf(r["type"])), fieldable: afterColons(strOf(r["custom_fieldable_type"]))})
		}
		if page >= lp.totalPages || page >= 1000 {
			break
		}
	}
	sc.defs, sc.defsLoaded = defs, true
	return nil
}

// defsOf chooses the definitions of the entity (step 3), first per key.
func (sc *stmtCtx) defsOf(e *EntityDef) []cfDef {
	var out []cfDef
	seen := map[string]bool{}
	for _, d := range sc.defs {
		ok := false
		for _, t := range e.CustomTypes {
			if strings.EqualFold(t, d.fieldable) {
				ok = true
			}
		}
		if !ok || seen[strings.ToLower(d.key)] {
			continue
		}
		seen[strings.ToLower(d.key)] = true
		out = append(out, d)
	}
	return out
}

func toCustom(d cfDef) *CustomField {
	return &CustomField{Name: "custom." + strings.ToLower(d.key), Key: d.key, Label: d.name, DefID: d.id,
		TypeKey: strings.ToLower(d.typeName), TypeName: d.typeName}
}

// customFieldsOf is custom-fields without a key.
func (sc *stmtCtx) customFieldsOf(e *EntityDef) ([]*CustomField, *AqlError) {
	if len(e.CustomTypes) == 0 {
		return nil, nil
	}
	if err := sc.loadDefs(); err != nil {
		return nil, err
	}
	var out []*CustomField
	for _, d := range sc.defsOf(e) {
		if _, ok := kindTable[strings.ToLower(d.typeName)]; ok {
			out = append(out, toCustom(d))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// customLookup is custom-fields with a key. It gives nil and no error only
// when explicit is false.
func (sc *stmtCtx) customLookup(e *EntityDef, key string, explicit bool) (*CustomField, *AqlError) {
	if len(e.CustomTypes) == 0 {
		if explicit {
			return nil, sc.errN06(e)
		}
		return nil, nil
	}
	if err := sc.loadDefs(); err != nil {
		return nil, err
	}
	if sc.defsStatus != 0 {
		if explicit {
			return nil, sc.errN07(sc.defsStatus, e)
		}
		return nil, nil
	}
	var supported int
	for _, d := range sc.defsOf(e) {
		_, ok := kindTable[strings.ToLower(d.typeName)]
		if ok {
			supported++
		}
		if strings.EqualFold(d.key, key) {
			cf := toCustom(d)
			if !ok {
				hint, err := sc.customHint(e)
				if err != nil {
					return nil, err
				}
				return nil, aqlErr("bad_name", "%s has the type %s, and the server cannot read or set custom fields of that type.%s", cf.Name, d.typeName, hint)
			}
			sc.custom[cf.Name] = cf
			return cf, nil
		}
	}
	if !explicit {
		return nil, nil
	}
	word := "custom." + key
	if supported > 0 {
		list, _ := sc.customFieldsOf(e)
		return nil, aqlErr("bad_name", "unknown custom field '%s' on %s. Custom fields of %s: %s. To see their types: DESCRIBE %s", word, e.Name, e.Name, listMax(customNames(list)), e.Name)
	}
	return nil, sc.errN06(e)
}

// options reads the options of a definition at most once per statement.
func (sc *stmtCtx) optionsOf(cf *CustomField) ([]string, *AqlError) {
	if o, ok := sc.options[cf.DefID]; ok {
		return o, nil
	}
	list, f := sc.c.apiGetList(sc.ctx, sc.acct, "/custom_field_definitions/"+esc(cf.DefID)+"/options", "options")
	if f != nil {
		return nil, sc.failErr(f)
	}
	var out []string
	for _, o := range list {
		out = append(out, strOf(o["text"]))
	}
	sc.options[cf.DefID] = out
	return out, nil
}

// fieldType gives the field type of a catalog field or a custom field.
func (sc *stmtCtx) fieldType(e *EntityDef, name string) FieldType {
	if strings.HasPrefix(name, "custom.") {
		if c := sc.custom[name]; c != nil {
			return kindTable[c.TypeKey].ftype
		}
		return tString
	}
	if fd := e.field(name); fd != nil {
		return fd.Type
	}
	return tString
}

// multiValued reports a field with several values (evaluate-condition, step 11).
func (sc *stmtCtx) multiValued(e *EntityDef, name string) bool {
	if e.Name == "features" && name == "tags" {
		return true
	}
	if c := sc.custom[name]; c != nil {
		k := c.kind()
		return k == "tags" || k == "fixed_tags" || k == "users"
	}
	return false
}
