// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// tokenize: splits the text of a statement into tokens.

import (
	"fmt"
	"strings"
	"unicode"
)

type TokKind int

const (
	tkWord TokKind = iota
	tkText
	tkNumber
	tkOp
	tkPunct
	tkEOF
)

type Token struct {
	Kind  TokKind
	Value string
	Pos   int // 1-based position of the first character
	Quote rune
}

func (t Token) isWord(w string) bool  { return t.Kind == tkWord && strings.EqualFold(t.Value, w) }
func (t Token) isPunct(p string) bool { return t.Kind == tkPunct && t.Value == p }

func isLetter(r rune) bool { return unicode.IsLetter(r) }
func isDigit(r rune) bool  { return r >= '0' && r <= '9' }

func tokenize(text string, lenient bool) ([]Token, *AqlError) {
	rs := []rune(text)
	n := len(rs)
	var toks []Token
	i := 0
	for i < n {
		r := rs[i]
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
				c := rs[i]
				if c == q {
					if i+1 < n && rs[i+1] == q {
						b.WriteRune(q)
						i += 2
						continue
					}
					if lenient && q == '\'' && i > start+1 && (isLetter(rs[i-1]) || isDigit(rs[i-1])) && i+1 < n && isLetter(rs[i+1]) {
						b.WriteRune(c)
						i++
						continue
					}
					closed = true
					i++
					break
				}
				if c == '\\' && i+1 < n {
					switch rs[i+1] {
					case 'n':
						b.WriteRune('\n')
					case 't':
						b.WriteRune('\t')
					default:
						b.WriteRune(rs[i+1])
					}
					i += 2
					continue
				}
				b.WriteRune(c)
				i++
			}
			if !closed {
				return nil, errS01(text, q, start+1)
			}
			toks = append(toks, Token{Kind: tkText, Value: b.String(), Pos: start + 1, Quote: q})
		case r == '=' || r == '<' || r == '>' || r == '!':
			op := string(r)
			if i+1 < n {
				two := string(rs[i : i+2])
				if two == "!=" || two == "<>" || two == "<=" || two == ">=" {
					op = two
				}
			}
			if op == "!" {
				return nil, errS02(text, r, start+1)
			}
			i += len(op)
			if op == "<>" {
				op = "!="
			}
			toks = append(toks, Token{Kind: tkOp, Value: op, Pos: start + 1})
		case r == '(' || r == ')' || r == ',' || r == ';' || r == '*':
			toks = append(toks, Token{Kind: tkPunct, Value: string(r), Pos: start + 1})
			i++
		case isDigit(r) || (r == '-' && i+1 < n && isDigit(rs[i+1])):
			// date without quotes
			if isDateAt(rs, i) {
				toks = append(toks, Token{Kind: tkText, Value: string(rs[i : i+10]), Pos: start + 1})
				i += 10
				continue
			}
			j := i
			if rs[j] == '-' {
				j++
			}
			for j < n && isDigit(rs[j]) {
				j++
			}
			if j+1 < n && rs[j] == '.' && isDigit(rs[j+1]) {
				j++
				for j < n && isDigit(rs[j]) {
					j++
				}
			}
			if j < n && (isLetter(rs[j]) || isDigit(rs[j]) || rs[j] == '_' || rs[j] == '.') {
				return nil, errS02(text, r, start+1)
			}
			toks = append(toks, Token{Kind: tkNumber, Value: string(rs[i:j]), Pos: start + 1})
			i = j
		case isLetter(r) || r == '_':
			j := i + 1
			for j < n && (isLetter(rs[j]) || isDigit(rs[j]) || rs[j] == '_' || rs[j] == '.') {
				j++
			}
			toks = append(toks, Token{Kind: tkWord, Value: string(rs[i:j]), Pos: start + 1})
			i = j
		default:
			return nil, errS02(text, r, start+1)
		}
	}
	toks = append(toks, Token{Kind: tkEOF, Pos: n + 1})
	return toks, nil
}

func isDateAt(rs []rune, i int) bool {
	if i+10 > len(rs) {
		return false
	}
	for k := 0; k < 10; k++ {
		c := rs[i+k]
		if k == 4 || k == 7 {
			if c != '-' {
				return false
			}
		} else if !isDigit(c) {
			return false
		}
	}
	if i+10 < len(rs) {
		c := rs[i+10]
		if isLetter(c) || isDigit(c) || c == '_' || c == '-' {
			return false
		}
	}
	return true
}

// hasInnerApostrophe: an apostrophe directly between a letter or digit and
// a letter.
func hasInnerApostrophe(text string) bool {
	_, ok := innerApostropheWord(text)
	return ok
}

// innerApostropheWord gives the word of the apostrophe hint.
func innerApostropheWord(text string) (string, bool) {
	rs := []rune(text)
	for i := 1; i+1 < len(rs); i++ {
		if rs[i] == '\'' && (isLetter(rs[i-1]) || isDigit(rs[i-1])) && isLetter(rs[i+1]) {
			a := i
			for a > 0 && !unicode.IsSpace(rs[a-1]) {
				a--
			}
			b := i + 1
			for b < len(rs) && !unicode.IsSpace(rs[b]) {
				b++
			}
			w := strings.Trim(string(rs[a:b]), `'"%(),;`)
			return w, true
		}
	}
	return "", false
}

func apostropheHint(text string) string {
	w, ok := innerApostropheWord(text)
	if !ok {
		return ""
	}
	return fmt.Sprintf(" The text contains an apostrophe (%s). Inside text, write an apostrophe twice: %s", w, strings.ReplaceAll(w, "'", "''"))
}

func errS01(text string, q rune, pos int) *AqlError {
	name := "apostrophe"
	hint := apostropheHint(text)
	if q == '"' {
		name = "double quote"
	}
	if hint == "" {
		if q == '\'' {
			hint = " Text goes in single quotes: 'like this'."
		} else {
			hint = " Use single quotes for text: 'like this'."
		}
	}
	return &AqlError{Kind: "syntax", Message: fmt.Sprintf("the text that starts with the %s at position %d is not closed.%s", name, pos, hint)}
}

func errS02(text string, c rune, pos int) *AqlError {
	return &AqlError{Kind: "syntax", Message: fmt.Sprintf("unexpected character %c at position %d.%s", c, pos, apostropheHint(text))}
}
