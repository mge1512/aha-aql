// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e

package main

import "strings"

// typeText is the text of `type` in DESCRIBE.
func typeText(fd *FieldDef) string {
	switch fd.Type {
	case tString:
		return "string"
	case tText:
		return "text"
	case tNumber:
		return "number"
	case tBoolean:
		return "boolean"
	case tKey:
		return "product key"
	case tUser:
		if fd.AsName {
			return "user (name)"
		}
		return "user (e-mail address)"
	case tStatus:
		return "status name"
	case tDate:
		return "date (YYYY-MM-DD)"
	case tDatetime:
		return "timestamp"
	case tReference:
		return "id of " + article(fd.RefKind) + " " + fd.RefKind
	}
	return "string"
}

func parentWhat(e *EntityDef) string {
	switch e.ParentKind {
	case "product":
		return "a product key"
	case "any":
		return "the id of the commented record"
	}
	return "the id of " + article(e.ParentKind) + " " + e.ParentKind
}

// runDescribe is describe-statement.
func (sc *stmtCtx) runDescribe(st *DescribeStmt) (omap, *AqlError) {
	if st.E == nil {
		var ents []any
		for _, e := range catalog {
			ents = append(ents, omap{{"entity", e.Name}, {"about", sc.aboutOf(e)}})
		}
		return omap{{"entities", ents}, {"note", "DESCRIBE entity lists the fields of one entity, for example: DESCRIBE features. Most lists need a product key: SELECT key, name FROM products"}}, nil
	}
	e := st.E
	out := omap{{"entity", e.Name}, {"about", sc.aboutOf(e)}}
	var fields []any
	for _, fd := range e.Fields {
		o := omap{{"name", fd.Name}, {"type", typeText(fd)}}
		switch fd.Writable {
		case wYes:
			o = append(o, kv{"writable", true})
		case wInsertOnly:
			o = append(o, kv{"writable", "INSERT only"})
		}
		fields = append(fields, o)
	}
	out = append(out, kv{"fields", fields})
	cfs, err := sc.customFieldsOf(e)
	if err != nil {
		return nil, err
	}
	if len(cfs) > 0 {
		var list []any
		for _, c := range cfs {
			info := kindTable[c.TypeKey]
			o := omap{{"name", c.Name}, {"label", c.Label}, {"type", info.desc}}
			if info.settable && e.CanUpdate {
				o = append(o, kv{"writable", true})
			}
			list = append(list, o)
		}
		out = append(out, kv{"custom_fields", list})
		idf := e.idField()
		out = append(out, kv{"custom_fields_note", "* shows custom fields only with " + idf + " = '...', and only those that have a value. To select one by name: SELECT " +
			idf + ", " + cfs[0].Name + " FROM " + e.Name + " WHERE " + idf + " = '" + sc.exampleID(e.Kind) + "'"})
	}
	if len(e.ParentFilters) > 0 {
		var pf []any
		for _, p := range e.ParentFilters {
			pf = append(pf, p+" = '...'")
		}
		out = append(out, kv{"filter_by_parent", pf})
	}
	if e.field("status") != nil {
		out = append(out, kv{"status_values", "SELECT name FROM statuses WHERE product = '" + sc.key() + "' AND entity = '" + e.Name + "'"})
	}
	if !e.hasOps() {
		out = append(out, kv{"change_with_execute", "read-only"})
	} else {
		var ops []any
		if e.CanCreate {
			ops = append(ops, "INSERT INTO "+e.Name+" (parent, ...) VALUES (...): parent is "+parentWhat(e))
		}
		if e.CanUpdate {
			ops = append(ops, "UPDATE "+e.Name+" SET field = 'value' WHERE id = '...'")
		}
		if e.CanDelete {
			ops = append(ops, "DELETE FROM "+e.Name+" WHERE id = '...'")
		}
		out = append(out, kv{"change_with_execute", ops})
	}
	out = append(out, kv{"example", strings.ReplaceAll(e.Example, "{KEY}", sc.key())})
	return out, nil
}
