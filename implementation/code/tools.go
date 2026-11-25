// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268

package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const queryDescription = `Read from Aha! with one AQL statement.
Start with: SELECT key, name FROM products. Most other lists need a product key.
DESCRIBE lists the entities; DESCRIBE features lists the fields of features.
A workspace or project is a product; a milestone or version is a release; an objective is a goal.
Search by word: SELECT id, name FROM features WHERE product = '{KEY}' AND name CONTAINS 'word'
In name and description: SELECT id, name FROM features WHERE release = '{KEY}-R-3' AND (name CONTAINS 'word' OR description CONTAINS 'word')
SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]
DESCRIBE entity
Entities: {ENTITIES}.
id is the reference number of a record, for example '{KEY}-23'.
Filter by parent: product = '{KEY}', release = '{KEY}-R-3', feature = '{KEY}-23', initiative = '{KEY}-S-1'.
Other conditions: !=, <, >, CONTAINS 'text', IS NULL. Count with COUNT(*).
Custom fields are written custom.key. DESCRIBE entity lists them.
Text goes in single quotes. Write an apostrophe inside text twice: 'Year''s end'.
Examples:
SELECT id, name, status FROM features WHERE release = '{KEY}-R-3'
SELECT status, COUNT(*) FROM features WHERE product = '{KEY}' GROUP BY status
SELECT id, name, release_date FROM releases WHERE product = '{KEY}' AND parking_lot = false ORDER BY release_date
SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features'
SELECT email FROM me
`

const executeDescription = `Change ONE record in Aha! with one AQL statement. Three forms:
INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)
UPDATE entity SET field = 'value', ... WHERE id = 'id'
DELETE FROM entity WHERE id = 'id'
parent is the record the new one belongs to: a release for features and epics, a product for releases, initiatives, goals and ideas, a feature for requirements, any record for comments.
Text and apostrophes as in query. DESCRIBE entity in query lists the field names.
Examples:
INSERT INTO features (parent, name) VALUES ('{KEY}-R-3', 'Export to CSV')
UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = '{KEY}-23'
INSERT INTO comments (parent, body) VALUES ('{KEY}-23', 'It''s ready.')
`

// serverInstructions is the text of the server instructions of
// tool-registration. It is the same for every Account and has no {KEY}.
const serverInstructions = `This server reads and changes records of one Aha! account with AQL, a small SQL dialect.
The tool query reads (SELECT, DESCRIBE). The tool execute, when the tool list has it, changes one record (INSERT, UPDATE, DELETE).
Start with SELECT key, name FROM products, then DESCRIBE an entity for its field names.
Most error messages show what to write instead; then send the corrected statement.
`

// startPromptText is the prompt text of the MCP prompt start.
const startPromptText = `Use the tool query of the Aha! server, with one AQL statement in each call.
1. SELECT key, name FROM products
2. DESCRIBE features lists the fields of features; DESCRIBE alone lists the entities.
3. SELECT id, name, status FROM features WHERE product = '{KEY}'
Search by word: SELECT id, name FROM features WHERE product = '{KEY}' AND name CONTAINS 'word'
A workspace or project is a product; a milestone or version is a release; an objective is a goal.
Most error messages show what to write instead; then send the corrected statement.
`

// app is the server process.
type app struct {
	mode     string
	settings Settings
	acct     *Account    // stdio and manual mode
	ek       *exampleKey // stdio and manual mode
	client   *ahaClient
	errMu    sync.Mutex
}

func (a *app) diag(line string) {
	a.errMu.Lock()
	defer a.errMu.Unlock()
	fmt.Fprintln(stderr, line)
}

type ctxKey struct{}

// reqState is the Account and the ExampleKey of one HTTP request.
type reqState struct {
	acct Account
	ek   *exampleKey
}

func (a *app) newExampleKey() *exampleKey {
	return &exampleKey{defaultProduct: a.settings.DefaultProduct, warn: a.diag}
}

// state is request-account: the Account that the call runs with.
func (a *app) state(ctx context.Context) (Account, *exampleKey) {
	if rs, ok := ctx.Value(ctxKey{}).(*reqState); ok {
		return rs.acct, rs.ek
	}
	return *a.acct, a.ek
}

func (a *app) stmt(ctx context.Context) *stmtCtx {
	acct, ek := a.state(ctx)
	if a.mode != "http" {
		ctx = context.WithoutCancel(ctx)
	}
	return newStmt(ctx, acct, a.client, ek)
}

func unknownParams(args map[string]any, param string) []string {
	var out []string
	for k := range args {
		if k != param {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// queryTool is the tool query. It gives the JSON text of the result, or
// the message of an AqlError.
func (a *app) queryTool(ctx context.Context, args map[string]any) (string, *AqlError) {
	sc := a.stmt(ctx)
	if u := unknownParams(args, "query"); len(u) > 0 {
		return "", sc.errA02("query", "query", u)
	}
	q, ok := args["query"].(string)
	if !ok || strings.TrimSpace(q) == "" {
		return "", sc.errA01()
	}
	st, err := sc.parseStatement(q, true, false)
	if err != nil {
		return "", err
	}
	var out omap
	switch x := st.(type) {
	case *DescribeStmt:
		out, err = sc.runDescribe(x)
	case *ReadStmt:
		out, err = sc.runRead(x)
	}
	if err != nil {
		return "", err
	}
	return encodeJSON(out), nil
}

// executeTool is the tool execute.
func (a *app) executeTool(ctx context.Context, args map[string]any, dryRun, logLine bool) (string, *AqlError) {
	sc := a.stmt(ctx)
	info := &writeInfo{op: "none", entity: "none", id: "none"}
	text, err := func() (string, *AqlError) {
		if u := unknownParams(args, "statement"); len(u) > 0 {
			return "", sc.errA02("execute", "statement", u)
		}
		s, ok := args["statement"].(string)
		if !ok || strings.TrimSpace(s) == "" {
			return "", errA03()
		}
		st, err := sc.parseStatement(s, false, true)
		if err != nil {
			return "", err
		}
		ws := st.(*WriteStmt)
		info = &writeInfo{op: ws.Kind, entity: ws.E.Name, id: "none"}
		out, wi, err := sc.runWrite(ws, dryRun)
		if wi != nil {
			info = wi
		}
		if err != nil {
			return "", err
		}
		return encodeJSON(out), nil
	}()
	if logLine {
		result := "ok"
		if err != nil {
			result = err.Kind
		}
		a.diag(fmt.Sprintf("%s execute account=%s op=%s entity=%s id=%s result=%s",
			time.Now().UTC().Format("2006-01-02T15:04:05Z"), sc.acct.Subdomain, info.op, info.entity, info.id, result))
	}
	return text, err
}

func toolResult(text string, err *AqlError) *mcp.CallToolResult {
	if err != nil {
		return mcp.NewToolResultError(err.Msg)
	}
	return mcp.NewToolResultText(text)
}

// newMCPServer is tool-registration.
func (a *app) newMCPServer() *server.MCPServer {
	hooks := &server.Hooks{}
	hooks.AddAfterListTools(func(ctx context.Context, id any, msg *mcp.ListToolsRequest, res *mcp.ListToolsResult) {
		sc := a.stmt(ctx)
		key := sc.key()
		for i := range res.Tools {
			res.Tools[i].Description = strings.ReplaceAll(res.Tools[i].Description, "{KEY}", key)
		}
	})
	s := server.NewMCPServer("mcp-server-aha-aql", serverVersion,
		server.WithToolCapabilities(false),
		server.WithPromptCapabilities(false),
		server.WithInstructions(serverInstructions),
		server.WithHooks(hooks),
		server.WithRecovery(),
	)
	qd := strings.ReplaceAll(queryDescription, "{ENTITIES}", strings.Join(entityNames(), ", "))
	s.AddTool(mcp.NewTool("query",
		mcp.WithDescription(qd),
		mcp.WithString("query", mcp.Required(), mcp.Description("One AQL statement")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolResult(a.queryTool(ctx, req.GetArguments())), nil
	})
	if !a.settings.ReadOnly {
		s.AddTool(mcp.NewTool("execute",
			mcp.WithDescription(executeDescription),
			mcp.WithString("statement", mcp.Required(), mcp.Description("One AQL statement")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return toolResult(a.executeTool(ctx, req.GetArguments(), false, true)), nil
		})
	}
	// Step 7: the one MCP prompt, start.
	s.AddPrompt(mcp.NewPrompt("start", mcp.WithPromptDescription("How to start with AQL")),
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			key := a.stmt(ctx).key()
			text := strings.ReplaceAll(startPromptText, "{KEY}", key)
			return mcp.NewGetPromptResult("How to start with AQL", []mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(text)),
			}), nil
		})
	return s
}

// refuse writes the answer of request-account for a refused request.
func refuse(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprint(w, encodeJSON(omap{{"jsonrpc", "2.0"}, {"id", nil}, {"error", omap{{"code", -32600}, {"message", msg}}}}))
}

// accountMiddleware is request-account in http mode.
func (a *app) accountMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["Origin"]; ok {
			refuse(w, http.StatusForbidden, "requests from a web page are not accepted")
			return
		}
		key := r.Header.Get("Aha-Api-Key")
		if !reCredential.MatchString(key) {
			refuse(w, http.StatusBadRequest, "no credential. Send the header Aha-Api-Key with an Aha! API key")
			return
		}
		sub := ""
		if _, ok := r.Header["Aha-Subdomain"]; ok {
			sub = strings.ToLower(r.Header.Get("Aha-Subdomain"))
			if !reSubdomain.MatchString(sub) {
				refuse(w, http.StatusBadRequest, "Aha-Subdomain must be the account name before .aha.io")
				return
			}
		}
		if a.settings.Subdomain != "" && sub != "" && sub != a.settings.Subdomain {
			refuse(w, http.StatusForbidden, "this server works with one Aha! account only")
			return
		}
		if sub == "" {
			sub = a.settings.Subdomain
		}
		if sub == "" {
			refuse(w, http.StatusBadRequest, "no Aha! account. Send the header Aha-Subdomain")
			return
		}
		rs := &reqState{acct: Account{Subdomain: sub, Credential: key}, ek: a.newExampleKey()}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, rs)))
	})
}
