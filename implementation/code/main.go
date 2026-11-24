// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

// mcp-server-aha-aql: an MCP server for Aha! with one query language, AQL,
// and two tools: query reads, execute changes one record.
//
// Invocation:
//
//	mcp-server-aha-aql stdio   [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
//	mcp-server-aha-aql http    [listen=HOST:PORT] [subdomain=NAME] [default-product=PRODUCT] [read-only=true]
//	mcp-server-aha-aql query   query=STATEMENT ...
//	mcp-server-aha-aql execute statement=STATEMENT [dry-run=true] ...
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// specSHA256 is the SHA256 of the specification this binary is generated from.
const specSHA256 = "9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262"

// serverVersion is the version of the specification; the build may set it.
var serverVersion = "0.13.0"

// ahaVariables are the variables of the server's own settings. The proxy
// and trust store variables are read by the standard library.
var ahaVariables = map[string]bool{"AHA_SUBDOMAIN": true, "AHA_API_KEY": true, "AHA_DEFAULT_PRODUCT": true, "AHA_READ_ONLY": true}

func getenv(name string) string {
	if !ahaVariables[name] {
		return ""
	}
	return os.Getenv(name)
}

func main() {
	os.Exit(realMain(os.Args[1:]))
}

func realMain(args []string) int {
	if len(args) > 0 && args[0] == "--version" {
		fmt.Printf("mcp-server-aha-aql %s spec:%s\n", serverVersion, specSHA256)
		return 0
	}
	cfg, err := loadConfiguration(args, getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	switch cfg.mode {
	case modeStdio:
		return serveStdio(cfg)
	case modeHTTP:
		return serveHTTP(cfg)
	}
	return manualMode(cfg)
}

func serveStdio(cfg *startConfig) int {
	process := &session{mode: modeStdio, settings: cfg.settings, acct: cfg.acct}
	srv := newMCPServer(process, cfg.settings)
	if err := server.ServeStdio(srv); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "stdio error: %v\n", err)
		return 1
	}
	return 0
}

// refuse answers a request that request-account does not accept.
func refuse(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, toJSON(Obj{{"jsonrpc", "2.0"}, {"id", nil}, {"error", Obj{{"code", -32600}, {"message", text}}}}))
}

// requestAccount is request-account in http mode: it runs before the MCP
// handler for every request to the endpoint.
func requestAccount(settings *Settings, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" {
			refuse(w, 403, "requests from a web page are not accepted")
			return
		}
		cred := r.Header.Get("Aha-Api-Key")
		if !reCred.MatchString(cred) {
			refuse(w, 400, "no credential. Send the header Aha-Api-Key with an Aha! API key")
			return
		}
		sub := ""
		if _, has := r.Header["Aha-Subdomain"]; has {
			sub = strings.ToLower(r.Header.Get("Aha-Subdomain"))
			if !reSubdomain.MatchString(sub) {
				refuse(w, 400, "Aha-Subdomain must be the account name before .aha.io")
				return
			}
		}
		if settings.Subdomain != "" && sub != "" && sub != settings.Subdomain {
			refuse(w, 403, "this server works with one Aha! account only")
			return
		}
		if sub == "" {
			sub = settings.Subdomain
		}
		if sub == "" {
			refuse(w, 400, "no Aha! account. Send the header Aha-Subdomain")
			return
		}
		s := &session{mode: modeHTTP, settings: settings, acct: Account{Subdomain: sub, Credential: cred}}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, s)))
	})
}

func serveHTTP(cfg *startConfig) int {
	ln, err := net.Listen("tcp", cfg.settings.Listen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot listen on %s: %v\n", cfg.settings.Listen, err)
		return 1
	}
	srv := newMCPServer(nil, cfg.settings)
	streamable := server.NewStreamableHTTPServer(srv,
		server.WithStateLess(true),
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			if s, ok := r.Context().Value(ctxKey{}).(*session); ok {
				return context.WithValue(ctx, ctxKey{}, s)
			}
			return ctx
		}),
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", requestAccount(cfg.settings, streamable))
	hs := &http.Server{Handler: mux, ReadHeaderTimeout: 30 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- hs.Serve(ln) }()
	fmt.Fprintf(os.Stderr, "mcp-server-aha-aql %s: MCP over HTTP on http://%s/mcp\n", serverVersion, ln.Addr())
	select {
	case <-ctx.Done():
	case err := <-errc:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "error: http server: %v\n", err)
			return 1
		}
	}
	sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = hs.Shutdown(sctx)
	return 0
}

func manualMode(cfg *startConfig) int {
	key := "query"
	if cfg.word == "execute" {
		key = "statement"
	}
	stmt, ok := cfg.opts[key]
	if !ok {
		fmt.Fprintf(os.Stderr, "error: %s needs %s=STATEMENT, or %s=- to read the statement from standard input\n", cfg.word, key, key)
		return 2
	}
	if stmt == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: standard input cannot be read\n")
			return 2
		}
		stmt = string(data)
	}
	if cfg.word == "execute" && cfg.settings.ReadOnly {
		fmt.Fprintln(os.Stderr, "error: execute is not available, because read-only is true (option read-only or variable AHA_READ_ONLY)")
		return 2
	}
	process := &session{mode: modeManual, settings: cfg.settings, acct: cfg.acct}
	var text string
	if cfg.word == "query" {
		text, ok = process.runQuery(map[string]any{"query": stmt})
	} else {
		text, ok, _, _ = process.runExecute(map[string]any{"statement": stmt}, cfg.settings.DryRun)
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "error: %s\n", text)
		return 1
	}
	fmt.Println(text)
	return 0
}
