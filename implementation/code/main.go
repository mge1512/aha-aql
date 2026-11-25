// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e

// Command mcp-server-aha-aql is an MCP server for the Aha! product
// management service. A language model reads and changes Aha! records
// through one query language, AQL, and two tools: query and execute.
//
// Invocation:
//
//	mcp-server-aha-aql stdio   [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
//	mcp-server-aha-aql http    [listen=HOST:PORT] [subdomain=NAME] [default-product=PRODUCT] [read-only=true]
//	mcp-server-aha-aql query   query=STATEMENT [options]
//	mcp-server-aha-aql execute statement=STATEMENT [dry-run=true] [options]
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// serverVersion is the version of the spec; the build can set it with
// -ldflags "-X main.serverVersion=...".
var serverVersion = "0.24.0"

// specSHA256 is the SHA256 of the specification this binary was translated from.
const specSHA256 = "caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e"

var permissionBits = runtime.GOOS != "windows"

var stderr io.Writer = os.Stderr

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 && args[0] == "--version" {
		fmt.Printf("mcp-server-aha-aql %s spec:%s\n", serverVersion, specSHA256)
		return 0
	}
	a := &app{client: newAhaClient()}
	mode, settings, acct, serr := loadConfiguration(args, os.Getenv, a.diag)
	if serr != nil {
		a.diag(serr.msg)
		return 2
	}
	a.mode, a.settings, a.acct = mode, settings, acct
	a.ek = a.newExampleKey()
	switch mode {
	case "stdio":
		return a.runStdio()
	case "http":
		return a.runHTTP()
	}
	return a.runManual(args[0])
}

// runManual is manual-mode.
func (a *app) runManual(word string) int {
	key := "query"
	if word == "execute" {
		key = "statement"
	}
	if !a.settings.HasStatement {
		a.diag(fmt.Sprintf("error: %s needs %s=STATEMENT, or %s=- to read the statement from standard input", word, key, key))
		return 2
	}
	stmt := a.settings.Statement
	if stmt == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			a.diag("error: standard input cannot be read")
			return 2
		}
		stmt = string(data)
	}
	if word == "execute" && a.settings.ReadOnly {
		a.diag("error: execute is not available, because read-only is true (option read-only or variable AHA_READ_ONLY)")
		return 2
	}
	ctx := context.Background()
	var text string
	var err *AqlError
	if word == "query" {
		text, err = a.queryTool(ctx, map[string]any{"query": stmt})
	} else {
		text, err = a.executeTool(ctx, map[string]any{"statement": stmt}, a.settings.DryRun, false)
	}
	if err != nil {
		a.diag("error: " + err.Msg)
		return 1
	}
	fmt.Fprintln(os.Stdout, text)
	return 0
}

// runStdio is the stdio transport of the deployment template.
func (a *app) runStdio() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	ss := server.NewStdioServer(a.newMCPServer())
	ss.SetErrorLogger(log.New(&diagWriter{a}, "", log.LstdFlags))
	if err := ss.Listen(ctx, os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
		a.diag("error: stdio: " + err.Error())
		return 1
	}
	return 0
}

type diagWriter struct{ a *app }

func (d *diagWriter) Write(p []byte) (int, error) {
	d.a.errMu.Lock()
	defer d.a.errMu.Unlock()
	return stderr.Write(p)
}

// runHTTP is the streamable-http transport of the deployment template.
func (a *app) runHTTP() int {
	ln, err := net.Listen("tcp", a.settings.Listen)
	if err != nil {
		a.diag(fmt.Sprintf("error: cannot listen on %s: %v", a.settings.Listen, err))
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	hs := server.NewStreamableHTTPServer(a.newMCPServer(), server.WithEndpointPath("/mcp"),
		// The server keeps no MCP session: no Mcp-Session-Id is given, and
		// one in a request is ignored.
		server.WithStateLess(true))
	mux := http.NewServeMux()
	mux.Handle("/mcp", a.accountMiddleware(hs))
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 30 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	a.diag("mcp-server-aha-aql " + serverVersion + ": listening on http://" + ln.Addr().String() + "/mcp")
	select {
	case <-ctx.Done():
	case err := <-errc:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.diag("error: http: " + err.Error())
			return 1
		}
	}
	sctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(sctx)
	_ = hs.Shutdown(sctx)
	return 0
}
