// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// Values and the JSON text of results: members in order, no white space
// outside strings, no line break.

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

// Num is a number as a statement writes it.
type Num string

func (n Num) Float() (float64, bool) {
	f, err := strconv.ParseFloat(string(n), 64)
	return f, err == nil
}

// KV is one member of an ordered JSON object.
type KV struct {
	K string
	V any
}

// Obj is a JSON object with its members in order.
type Obj []KV

func (o *Obj) Set(k string, v any) {
	for i := range *o {
		if (*o)[i].K == k {
			(*o)[i].V = v
			return
		}
	}
	*o = append(*o, KV{k, v})
}

func (o Obj) Get(k string) (any, bool) {
	for _, kv := range o {
		if kv.K == k {
			return kv.V, true
		}
	}
	return nil, false
}

func fmtFloat(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func encodeString(buf *bytes.Buffer, s string) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	buf.Write(bytes.TrimRight(b.Bytes(), "\n"))
}

func encodeJSON(buf *bytes.Buffer, v any) {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case string:
		encodeString(buf, x)
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case float64:
		buf.WriteString(fmtFloat(x))
	case int:
		buf.WriteString(strconv.Itoa(x))
	case Num:
		if f, ok := x.Float(); ok {
			if strings.ContainsAny(string(x), ".eE") {
				buf.WriteString(fmtFloat(f))
			} else {
				buf.WriteString(strings.TrimPrefix(string(x), "+"))
			}
		} else {
			encodeString(buf, string(x))
		}
	case json.Number:
		encodeJSON(buf, Num(x))
	case Obj:
		buf.WriteByte('{')
		for i, kv := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodeString(buf, kv.K)
			buf.WriteByte(':')
			encodeJSON(buf, kv.V)
		}
		buf.WriteByte('}')
	case []Obj:
		buf.WriteByte('[')
		for i, o := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodeJSON(buf, o)
		}
		buf.WriteByte(']')
	case []string:
		buf.WriteByte('[')
		for i, s := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodeString(buf, s)
		}
		buf.WriteByte(']')
	case []any:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodeJSON(buf, e)
		}
		buf.WriteByte(']')
	case map[string]any:
		b, _ := json.Marshal(x)
		buf.Write(b)
	default:
		b, _ := json.Marshal(x)
		buf.Write(b)
	}
}

func toJSON(v any) string {
	var buf bytes.Buffer
	encodeJSON(&buf, v)
	return buf.String()
}

// textOf gives the text of a value: a text as it is, a number in its JSON
// form, a Boolean as true or false.
func textOf(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return fmtFloat(x)
	case int:
		return strconv.Itoa(x)
	case Num:
		return string(x)
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "true"
		}
		return "false"
	case []string:
		return strings.Join(x, ", ")
	}
	return toJSON(v)
}

// numberOf reads a value as a number.
func numberOf(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case Num:
		return x.Float()
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	}
	return 0, false
}

// quoteVal writes a value as a statement has to write it: in apostrophes,
// with every apostrophe written twice.
func quoteVal(v any) string {
	return "'" + strings.ReplaceAll(textOf(v), "'", "''") + "'"
}
