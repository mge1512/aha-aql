// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e

package main

import (
	"fmt"
	"strings"
	"unicode"
)

type tokKind int

const (
	tkWord tokKind = iota
	tkText
	tkNumber
	tkOp
	tkPunct
	tkEOF
)

// token is one token of a statement (tokenize).
type token struct {
	kind tokKind
	val  string
	pos  int // 1-based position of the first character
}

func (t token) is(word string) bool {
	return t.kind == tkWord && strings.EqualFold(t.val, word)
}

func (t token) punct(p string) bool { return t.kind == tkPunct && t.val == p }

// tokenize splits the text of a statement into tokens.
func tokenize(src []rune, lenient bool) ([]token, *AqlError) {
	var out []token
	n := len(src)
	i := 0
	isWordChar := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' }
	for i < n {
		r := src[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}
		start := i
		switch {
		case r == '\'' || r == '"':
			q := r
			var b strings.Builder
			i++
			closed := false
			for i < n {
				c := src[i]
				if c == q {
					if i+1 < n && src[i+1] == q {
						b.WriteRune(q)
						i += 2
						continue
					}
					if lenient && q == '\'' && i > start+1 && isLetterDigit(src[i-1]) && i+1 < n && isLetter(src[i+1]) {
						b.WriteRune(c)
						i++
						continue
					}
					closed = true
					i++
					break
				}
				if c == '\\' && i+1 < n {
					switch src[i+1] {
					case 'n':
						b.WriteRune('\n')
					case 't':
						b.WriteRune('\t')
					default:
						b.WriteRune(src[i+1])
					}
					i += 2
					continue
				}
				b.WriteRune(c)
				i++
			}
			if !closed {
				name := "apostrophe"
				hint := apostropheHint(src)
				if hint == "" {
					if q == '\'' {
						hint = " Text goes in single quotes: 'like this'."
					} else {
						hint = " Use single quotes for text: 'like this'."
					}
				}
				if q == '"' {
					name = "double quote"
				}
				return nil, aqlErr("syntax", "the text that starts with the %s at position %d is not closed.%s", name, start+1, hint)
			}
			out = append(out, token{kind: tkText, val: b.String(), pos: start + 1})
		case r == '=':
			out = append(out, token{kind: tkOp, val: "=", pos: start + 1})
			i++
		case r == '!' && i+1 < n && src[i+1] == '=':
			out = append(out, token{kind: tkOp, val: "!=", pos: start + 1})
			i += 2
		case r == '<':
			if i+1 < n && src[i+1] == '>' {
				out = append(out, token{kind: tkOp, val: "!=", pos: start + 1})
				i += 2
			} else if i+1 < n && src[i+1] == '=' {
				out = append(out, token{kind: tkOp, val: "<=", pos: start + 1})
				i += 2
			} else {
				out = append(out, token{kind: tkOp, val: "<", pos: start + 1})
				i++
			}
		case r == '>':
			if i+1 < n && src[i+1] == '=' {
				out = append(out, token{kind: tkOp, val: ">=", pos: start + 1})
				i += 2
			} else {
				out = append(out, token{kind: tkOp, val: ">", pos: start + 1})
				i++
			}
		case strings.ContainsRune("(),;*", r):
			out = append(out, token{kind: tkPunct, val: string(r), pos: start + 1})
			i++
		case unicode.IsDigit(r) || (r == '-' && i+1 < n && unicode.IsDigit(src[i+1])):
			if d, ok := dateAt(src, i); ok {
				out = append(out, token{kind: tkText, val: d, pos: start + 1})
				i += 10
				continue
			}
			j := i
			if src[j] == '-' {
				j++
			}
			for j < n && unicode.IsDigit(src[j]) {
				j++
			}
			if j+1 < n && src[j] == '.' && unicode.IsDigit(src[j+1]) {
				j++
				for j < n && unicode.IsDigit(src[j]) {
					j++
				}
			}
			if j < n && (unicode.IsLetter(src[j]) || unicode.IsDigit(src[j]) || src[j] == '_' || src[j] == '.') {
				return nil, errS02(src, i)
			}
			out = append(out, token{kind: tkNumber, val: string(src[i:j]), pos: start + 1})
			i = j
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < n && isWordChar(src[j]) {
				j++
			}
			out = append(out, token{kind: tkWord, val: string(src[i:j]), pos: start + 1})
			i = j
		default:
			return nil, errS02(src, i)
		}
	}
	out = append(out, token{kind: tkEOF, pos: 0})
	return out, nil
}

func errS02(src []rune, i int) *AqlError {
	return aqlErr("syntax", "unexpected character %s at position %d.%s", string(src[i]), i+1, apostropheHint(src))
}

// dateAt reads a date without quotes at i (tokenize, step 5).
func dateAt(src []rune, i int) (string, bool) {
	if i+10 > len(src) {
		return "", false
	}
	for k := 0; k < 10; k++ {
		c := src[i+k]
		if k == 4 || k == 7 {
			if c != '-' {
				return "", false
			}
		} else if !unicode.IsDigit(c) {
			return "", false
		}
	}
	if i+10 < len(src) {
		c := src[i+10]
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '-' {
			return "", false
		}
	}
	return string(src[i : i+10]), true
}

var _ = fmt.Sprintf
