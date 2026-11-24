// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// load-configuration: the first word, the options and the environment
// variables AHA_SUBDOMAIN, AHA_API_KEY, AHA_DEFAULT_PRODUCT and
// AHA_READ_ONLY. The proxy and trust store variables are read by the
// standard library (see aha-request).

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"runtime"
	"strings"
)

const defaultListen = "127.0.0.1:8080"

var optionOrder = []string{"subdomain", "api-key-file", "default-product", "read-only", "listen", "dry-run", "query", "statement"}

var optionsOf = map[string]map[string]bool{
	"stdio":   {"subdomain": true, "api-key-file": true, "default-product": true, "read-only": true},
	"http":    {"subdomain": true, "default-product": true, "read-only": true, "listen": true},
	"query":   {"subdomain": true, "api-key-file": true, "default-product": true, "read-only": true, "query": true},
	"execute": {"subdomain": true, "api-key-file": true, "default-product": true, "read-only": true, "dry-run": true, "statement": true},
}

type startConfig struct {
	word     string
	mode     Mode
	settings *Settings
	acct     Account
	opts     map[string]string
}

type startError struct{ msg string }

func (e *startError) Error() string { return e.msg }

func startErr(format string, args ...any) error { return &startError{fmt.Sprintf(format, args...)} }

func optionKeys(word string) string {
	var l []string
	for _, k := range optionOrder {
		if optionsOf[word][k] {
			l = append(l, k)
		}
	}
	return strings.Join(l, ", ")
}

func isLoopback(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// loadConfiguration returns the configuration or a start error (G01 to G09,
// G12). getenv gives the value of one of the four variables of the server.
func loadConfiguration(args []string, getenv func(string) string) (*startConfig, error) {
	word := ""
	if len(args) > 0 {
		word = args[0]
	}
	cfg := &startConfig{word: word, settings: &Settings{Listen: defaultListen}, opts: map[string]string{}}
	switch word {
	case "stdio":
		cfg.mode = modeStdio
	case "http":
		cfg.mode = modeHTTP
	case "query", "execute":
		cfg.mode = modeManual
	default:
		return nil, startErr("error: unknown transport '%s'. Valid: stdio, http, query, execute", word)
	}
	rest := args[1:]
	type kv struct{ k, v string }
	var pairs []kv
	for i, a := range rest {
		eq := strings.Index(a, "=")
		if eq <= 0 {
			return nil, startErr("error: argument %d is not key=value", i+1)
		}
		pairs = append(pairs, kv{a[:eq], a[eq+1:]})
	}
	if cfg.mode == modeHTTP {
		for _, p := range pairs {
			if p.k == "api-key-file" {
				return nil, startErr("error: http mode has no option api-key-file. Each request brings its credential in the header Aha-Api-Key")
			}
		}
	}
	for _, p := range pairs {
		if !optionsOf[word][p.k] {
			return nil, startErr("error: unknown option '%s' for %s. Options: %s", p.k, word, optionKeys(word))
		}
		if _, dup := cfg.opts[p.k]; dup {
			return nil, startErr("error: option '%s' is given twice", p.k)
		}
		cfg.opts[p.k] = p.v
	}
	// step 5: the variables, where the option is absent
	type src struct {
		value string
		name  string // the key, or the variable
		set   bool
	}
	take := func(key, variable string) src {
		if v, ok := cfg.opts[key]; ok {
			return src{v, key, true}
		}
		if variable != "" {
			if v := getenv(variable); v != "" {
				return src{v, variable, true}
			}
		}
		return src{}
	}
	readOnly := take("read-only", "AHA_READ_ONLY")
	dryRun := take("dry-run", "")
	sub := take("subdomain", "AHA_SUBDOMAIN")
	dp := take("default-product", "AHA_DEFAULT_PRODUCT")
	// step 6 a
	for _, b := range []struct {
		s   src
		dst *bool
	}{{readOnly, &cfg.settings.ReadOnly}, {dryRun, &cfg.settings.DryRun}} {
		if !b.s.set {
			continue
		}
		switch b.s.value {
		case "true":
			*b.dst = true
		case "false":
			*b.dst = false
		default:
			return nil, startErr("error: %s must be true or false", b.s.name)
		}
	}
	// step 6 b
	if sub.set {
		v := strings.ToLower(sub.value)
		if !reSubdomain.MatchString(v) {
			return nil, startErr("error: %s must be the account name before .aha.io, for example mycompany", sub.name)
		}
		cfg.settings.Subdomain = v
	}
	// step 6 c
	if dp.set {
		if dp.value == "" {
			return nil, startErr("error: default-product must be a product key, id or name")
		}
		cfg.settings.DefaultProduct = dp.value
	}
	if v, ok := cfg.opts["api-key-file"]; ok {
		if v == "" {
			return nil, startErr("error: api-key-file must be a file path")
		}
		cfg.settings.APIKeyFile = v
	}
	// step 6 d
	if v, ok := cfg.opts["listen"]; ok {
		cfg.settings.Listen = v
	}
	if cfg.mode == modeHTTP && !isLoopback(cfg.settings.Listen) {
		warn("listen=%s is not a loopback address. Every request has a credential; put a TLS endpoint in front of the server", cfg.settings.Listen)
	}
	// step 7
	if cfg.mode == modeHTTP {
		if getenv("AHA_API_KEY") != "" {
			warn("AHA_API_KEY is not read in http mode. Each request brings its credential in the header Aha-Api-Key")
		}
		return cfg, nil
	}
	// step 8
	if cfg.settings.Subdomain == "" {
		return nil, startErr("error: no Aha! account. Use subdomain=NAME or set AHA_SUBDOMAIN")
	}
	cfg.acct.Subdomain = cfg.settings.Subdomain
	// step 9
	if path := cfg.settings.APIKeyFile; path != "" {
		cred, err := readKeyFile(path)
		if err != nil {
			return nil, err
		}
		cfg.acct.Credential = cred
		return cfg, nil
	}
	// step 10
	raw := getenv("AHA_API_KEY")
	if raw == "" {
		return nil, startErr("error: no credential. Use api-key-file=PATH or set AHA_API_KEY")
	}
	cred := strings.TrimSpace(raw)
	if cred == "" {
		return nil, startErr("error: no usable credential in AHA_API_KEY: empty")
	}
	if !reCred.MatchString(cred) {
		return nil, startErr("error: no usable credential in AHA_API_KEY: not a single value of at most 4096 printable characters")
	}
	cfg.acct.Credential = cred
	return cfg, nil
}

func readKeyFile(path string) (string, error) {
	g07 := func(reason string) error {
		return startErr("error: no usable credential in %s: %s", path, reason)
	}
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", g07("no such file")
		}
		return "", g07("cannot be read")
	}
	if !st.Mode().IsRegular() {
		return "", g07("not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", g07("cannot be read")
	}
	defer f.Close()
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if st.Mode().Perm()&0o077 != 0 {
			return "", startErr("error: %s can be read by other users. Run: chmod 600 %s", path, path)
		}
	}
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return "", g07("cannot be read")
	}
	cred := strings.TrimSpace(string(data))
	if cred == "" {
		return "", g07("empty")
	}
	if !reCred.MatchString(cred) {
		return "", g07("not a single value of at most 4096 printable characters")
	}
	return cred, nil
}
