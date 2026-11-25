// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

// Settings is the configuration of the process (TYPES: Settings).
type Settings struct {
	Subdomain      string
	KeyFile        string
	DefaultProduct string
	ReadOnly       bool
	Listen         string
	DryRun         bool
	Statement      string
	HasStatement   bool
}

type option struct {
	key      string
	words    []string
	variable string
}

// options is the first table of load-configuration, in its order.
var options = []option{
	{"subdomain", []string{"stdio", "http", "query", "execute"}, "AHA_SUBDOMAIN"},
	{"api-key-file", []string{"stdio", "query", "execute"}, ""},
	{"default-product", []string{"stdio", "http", "query", "execute"}, "AHA_DEFAULT_PRODUCT"},
	{"read-only", []string{"stdio", "http", "query", "execute"}, "AHA_READ_ONLY"},
	{"listen", []string{"http"}, ""},
	{"dry-run", []string{"execute"}, ""},
	{"query", []string{"query"}, ""},
	{"statement", []string{"execute"}, ""},
}

func optionFor(key, word string) bool {
	for _, o := range options {
		if o.key == key {
			for _, w := range o.words {
				if w == word {
					return true
				}
			}
		}
	}
	return false
}

func optionsOfWord(word string) string {
	var out []string
	for _, o := range options {
		if optionFor(o.key, word) {
			out = append(out, o.key)
		}
	}
	return strings.Join(out, ", ")
}

// startError is a start message with exit code 2.
type startError struct{ msg string }

func startErr(format string, args ...any) *startError {
	return &startError{msg: "error: " + fmt.Sprintf(format, args...)}
}

func isLoopbackHost(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		host = listen
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// loadConfiguration is load-configuration. warn writes a warning line.
func loadConfiguration(args []string, getenv func(string) string, warn func(string)) (string, Settings, *Account, *startError) {
	var s Settings
	s.Listen = "127.0.0.1:8080"
	if len(args) == 0 {
		return "", s, nil, startErr("unknown transport ''. Valid: stdio, http, query, execute")
	}
	word := args[0]
	mode := ""
	switch word {
	case "stdio", "http":
		mode = word
	case "query", "execute":
		mode = "manual"
	default:
		return "", s, nil, startErr("unknown transport '%s'. Valid: stdio, http, query, execute", word)
	}
	type pair struct{ k, v string }
	var pairs []pair
	for i, a := range args[1:] {
		k, v, ok := strings.Cut(a, "=")
		if !ok || k == "" {
			return "", s, nil, startErr("argument %d is not key=value", i+1)
		}
		pairs = append(pairs, pair{k, v})
	}
	if mode == "http" {
		for _, p := range pairs {
			if p.k == "api-key-file" {
				return "", s, nil, startErr("http mode has no option api-key-file. Each request brings its credential in the header Aha-Api-Key")
			}
		}
	}
	given := map[string]string{}
	for _, p := range pairs {
		if !optionFor(p.k, word) {
			return "", s, nil, startErr("unknown option '%s' for %s. Options: %s", p.k, word, optionsOfWord(word))
		}
		if _, dup := given[p.k]; dup {
			return "", s, nil, startErr("option '%s' is given twice", p.k)
		}
		given[p.k] = p.v
	}
	// values and their source names (the variable stands for the key)
	src := map[string]string{}
	val := map[string]string{}
	for _, o := range options {
		if v, ok := given[o.key]; ok {
			val[o.key], src[o.key] = v, o.key
		} else if o.variable != "" {
			if v := getenv(o.variable); v != "" {
				val[o.key], src[o.key] = v, o.variable
			}
		}
	}
	for _, k := range []string{"read-only", "dry-run"} {
		if v, ok := val[k]; ok {
			switch v {
			case "true":
				if k == "read-only" {
					s.ReadOnly = true
				} else {
					s.DryRun = true
				}
			case "false":
			default:
				return "", s, nil, startErr("%s must be true or false", src[k])
			}
		}
	}
	if v, ok := val["subdomain"]; ok {
		v = strings.ToLower(v)
		if !reSubdomain.MatchString(v) {
			return "", s, nil, startErr("%s must be the account name before .aha.io, for example mycompany", src["subdomain"])
		}
		s.Subdomain = v
	}
	if v, ok := val["default-product"]; ok {
		if v == "" {
			return "", s, nil, startErr("default-product must be a product key, id or name")
		}
		s.DefaultProduct = v
	}
	if v, ok := val["api-key-file"]; ok {
		if v == "" {
			return "", s, nil, startErr("api-key-file must be a file path")
		}
		s.KeyFile = v
	}
	if v, ok := val["listen"]; ok {
		s.Listen = v
	}
	if mode == "http" && !isLoopbackHost(s.Listen) {
		warn("warning: the listen address is not a loopback address. Every request has a credential: put a TLS endpoint in front of the server")
	}
	switch word {
	case "query":
		s.Statement, s.HasStatement = val["query"], has(given, "query")
	case "execute":
		s.Statement, s.HasStatement = val["statement"], has(given, "statement")
	}
	if mode == "http" {
		if getenv("AHA_API_KEY") != "" {
			warn("warning: AHA_API_KEY is not read in http mode. Each request brings its credential in the header Aha-Api-Key")
		}
		return mode, s, nil, nil
	}
	if s.Subdomain == "" {
		return "", s, nil, startErr("no Aha! account. Use subdomain=NAME or set AHA_SUBDOMAIN")
	}
	var cred string
	if s.KeyFile != "" {
		c, err := readKeyFile(s.KeyFile)
		if err != nil {
			return "", s, nil, err
		}
		cred = c
	} else {
		raw := getenv("AHA_API_KEY")
		if raw == "" {
			return "", s, nil, startErr("no credential. Use api-key-file=PATH or set AHA_API_KEY")
		}
		cred = strings.TrimSpace(raw)
		if cred == "" {
			return "", s, nil, startErr("no usable credential in AHA_API_KEY: empty")
		}
		if !reCredential.MatchString(cred) {
			return "", s, nil, startErr("no usable credential in AHA_API_KEY: not a single value of at most 4096 printable characters")
		}
	}
	return mode, s, &Account{Subdomain: s.Subdomain, Credential: cred}, nil
}

func has(m map[string]string, k string) bool { _, ok := m[k]; return ok }

// readKeyFile is step 9 of load-configuration.
func readKeyFile(path string) (string, *startError) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", startErr("no usable credential in %s: no such file", path)
		}
		return "", startErr("no usable credential in %s: cannot be read", path)
	}
	if !fi.Mode().IsRegular() {
		return "", startErr("no usable credential in %s: not a regular file", path)
	}
	fh, err := os.Open(path)
	if err != nil {
		return "", startErr("no usable credential in %s: cannot be read", path)
	}
	defer fh.Close()
	if permissionBits && fi.Mode().Perm()&0o077 != 0 {
		return "", startErr("%s can be read by other users. Run: chmod 600 %s", path, path)
	}
	data, err := io.ReadAll(io.LimitReader(fh, 4097))
	if err != nil {
		return "", startErr("no usable credential in %s: cannot be read", path)
	}
	cred := strings.TrimSpace(string(data))
	if cred == "" {
		return "", startErr("no usable credential in %s: empty", path)
	}
	if !reCredential.MatchString(cred) {
		return "", startErr("no usable credential in %s: not a single value of at most 4096 printable characters", path)
	}
	return cred, nil
}
