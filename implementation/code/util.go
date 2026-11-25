// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268

package main

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	reSubdomain  = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	reProductKey = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	reNumericID  = regexp.MustCompile(`^[0-9]{6,}$`)
	reRefNum     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*-([A-Za-z]-)?[0-9]+(-[0-9]+)?$`)
	reDateText   = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	reTimestamp  = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})[T ]([0-9]{2}):([0-9]{2})(:([0-9]{2})(\.[0-9]+)?)?(Z|[+-][0-9]{2}:[0-9]{2})?$`)
	reEmail      = regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
)

// reCredential matches Credential: 1 to 4096 printable ASCII characters
// without a space.
var reCredential = credentialMatcher{}

type credentialMatcher struct{}

func (credentialMatcher) MatchString(s string) bool {
	if len(s) < 1 || len(s) > 4096 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '!' || s[i] > '~' {
			return false
		}
	}
	return true
}

// refKind gives the kind of a ReferenceNumber, or "" when the text is no
// ReferenceNumber or has no kind (TYPES: RecordId).
func refKind(s string) string {
	if !reRefNum.MatchString(s) {
		return ""
	}
	parts := strings.Split(s, "-")
	switch len(parts) {
	case 2:
		return "feature"
	case 3:
		if isDigits(parts[1]) {
			return "requirement"
		}
		switch strings.ToUpper(parts[1]) {
		case "R":
			return "release"
		case "E":
			return "epic"
		case "S":
			return "initiative"
		case "G":
			return "goal"
		case "I":
			return "idea"
		}
	}
	return ""
}

func isRefNum(s string) bool    { return reRefNum.MatchString(s) }
func isNumericID(s string) bool { return reNumericID.MatchString(s) }

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isRecordID reports whether s has the form of a RecordId.
func isRecordID(s string) bool {
	return isRefNum(s) || isNumericID(s) || reProductKey.MatchString(s)
}

// validDay reports whether s is a DateText that is a day of the calendar.
func validDay(s string) bool {
	if !reDateText.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// parseTimestamp reads a timestamp in the forms of parse-statement and of
// the API (ISO 8601). Without a zone it is UTC.
func parseTimestamp(s string) (time.Time, bool) {
	m := reTimestamp.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil || !validDay(m[1]) {
		return time.Time{}, false
	}
	h, _ := strconv.Atoi(m[2])
	mi, _ := strconv.Atoi(m[3])
	sec := 0
	if m[5] != "" {
		sec, _ = strconv.Atoi(m[5])
	}
	if h > 23 || mi > 59 || sec > 60 {
		return time.Time{}, false
	}
	var nsec int
	if m[6] != "" {
		frac := m[6][1:]
		for len(frac) < 9 {
			frac += "0"
		}
		nsec, _ = strconv.Atoi(frac[:9])
	}
	d, _ := time.Parse("2006-01-02", m[1])
	loc := time.UTC
	if z := m[7]; z != "" && z != "Z" {
		zh, _ := strconv.Atoi(z[1:3])
		zm, _ := strconv.Atoi(z[4:6])
		off := zh*3600 + zm*60
		if z[0] == '-' {
			off = -off
		}
		loc = time.FixedZone("", off)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), h, mi, sec, nsec, loc), true
}

func isTimestamp(s string) bool { _, ok := parseTimestamp(s); return ok }

// compForm is the comparison form of a text (evaluate-condition, step 5).
func compForm(s string) string {
	r := strings.NewReplacer("\u2018", "'", "\u2019", "'", "\u201a", "'", "\u201b", "'",
		"\u201c", "\"", "\u201d", "\"", "\u201e", "\"", "\u201f", "\"", "\u00a0", " ")
	return strings.ToLower(strings.TrimSpace(r.Replace(s)))
}

func textEq(a, b string) bool { return compForm(a) == compForm(b) }

// quote writes a value as a statement has to write it.
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// article gives "an" before a vowel, "a" otherwise.
func article(word string) string {
	if word != "" && strings.ContainsRune("aeiouAEIOU", rune(word[0])) {
		return "an"
	}
	return "a"
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// ---- values ----

// toNumber reads a Number or a text that reads as a number.
func toNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

// truth reads a truth value: a Boolean, the texts true and false, the
// numbers 1 and 0.
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
		case "1":
			return true, true
		case "0":
			return false, true
		}
	case float64:
		if x == 1 {
			return true, true
		}
		if x == 0 {
			return false, true
		}
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return truth(f)
		}
	}
	return false, false
}

// textOf gives the text form of a value.
func textOf(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return fmtNum(x)
	case int:
		return strconv.Itoa(x)
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return fmtNum(f)
		}
		return x.String()
	case []string:
		return strings.Join(x, ", ")
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func fmtNum(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// noValue reports a field without a value: null or empty text.
func noValue(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok && s == "" {
		return true
	}
	return false
}

// ---- ordered JSON ----

type kv struct {
	k string
	v any
}

// omap is a JSON object that keeps the order of its members.
type omap []kv

func (o *omap) set(k string, v any) {
	for i := range *o {
		if (*o)[i].k == k {
			(*o)[i].v = v
			return
		}
	}
	*o = append(*o, kv{k, v})
}

func (o omap) get(k string) (any, bool) {
	for _, e := range o {
		if e.k == k {
			return e.v, true
		}
	}
	return nil, false
}

func (o omap) has(k string) bool { _, ok := o.get(k); return ok }

// encodeJSON writes v as JSON text without white space outside strings
// and without HTML escaping.
func encodeJSON(v any) string {
	var b bytes.Buffer
	writeJSON(&b, v)
	return b.String()
}

func writeString(b *bytes.Buffer, s string) {
	var t bytes.Buffer
	enc := json.NewEncoder(&t)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	b.Write(bytes.TrimRight(t.Bytes(), "\n"))
}

func writeJSON(b *bytes.Buffer, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case string:
		writeString(b, x)
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case float64:
		b.WriteString(fmtNum(x))
	case int:
		b.WriteString(strconv.Itoa(x))
	case json.Number:
		if f, err := x.Float64(); err == nil {
			b.WriteString(fmtNum(f))
		} else {
			b.WriteString(x.String())
		}
	case omap:
		b.WriteByte('{')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, e.k)
			b.WriteByte(':')
			writeJSON(b, e.v)
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJSON(b, e)
		}
		b.WriteByte(']')
	case []string:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, e)
		}
		b.WriteByte(']')
	case []omap:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJSON(b, e)
		}
		b.WriteByte(']')
	case map[string]any:
		raw, _ := json.Marshal(x)
		b.Write(raw)
	default:
		raw, _ := json.Marshal(x)
		b.Write(raw)
	}
}

// ---- API records ----

type apiRec = map[string]any

// lookup takes the value at a source "a.b.c" of an API record.
func lookup(r apiRec, src string) any {
	var cur any = r
	for _, p := range strings.Split(src, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[p]
		if !ok {
			return nil
		}
	}
	return cur
}

func strOf(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return fmtNum(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// sameDigits compares two ids that Aha! writes as numbers or as texts.
func sameDigits(a, b any) bool {
	x, y := strOf(a), strOf(b)
	return x != "" && x == y
}
