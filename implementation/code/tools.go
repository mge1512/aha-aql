// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// The tools query and execute, and tool-registration.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func queryDescription() string {
	return "Read from Aha! with one AQL statement.\n" +
		"SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]\n" +
		"DESCRIBE entity\n" +
		"Entities: " + strings.Join(entityNames(), ", ") + ".\n" +
		"id is the reference number of a record, for example '{KEY}-23'.\n" +
		"Filter by parent: product = '{KEY}', release = '{KEY}-R-3', feature = '{KEY}-23', initiative = '{KEY}-S-1'.\n" +
		"Other conditions: !=, <, >, CONTAINS 'text', IS NULL. Count with COUNT(*).\n" +
		"Custom fields are written custom.key. DESCRIBE entity lists them.\n" +
		"Text goes in single quotes. Write an apostrophe inside text twice: 'Year''s end'.\n" +
		"Examples:\n" +
		"SELECT id, name, status FROM features WHERE release = '{KEY}-R-3'\n" +
		"SELECT status, COUNT(*) FROM features WHERE product = '{KEY}' GROUP BY status\n" +
		"SELECT id, name, release_date FROM releases WHERE product = '{KEY}' AND parking_lot = false ORDER BY release_date\n" +
		"SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features'\n" +
		"SELECT email FROM me\n"
}

const executeDescription = "Change ONE record in Aha! with one AQL statement. Three forms:\n" +
	"INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)\n" +
	"UPDATE entity SET field = 'value', ... WHERE id = 'id'\n" +
	"DELETE FROM entity WHERE id = 'id'\n" +
	"parent is the record the new one belongs to: a release for features and epics, a product for releases, initiatives, goals and ideas, a feature for requirements, any record for comments.\n" +
	"Text and apostrophes as in query. DESCRIBE entity in query lists the field names.\n" +
	"Examples:\n" +
	"INSERT INTO features (parent, name) VALUES ('{KEY}-R-3', 'Export to CSV')\n" +
	"UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = '{KEY}-23'\n" +
	"INSERT INTO comments (parent, body) VALUES ('{KEY}-23', 'It''s ready.')\n"

// errA02 is the message for parameters that the tool does not declare.
func errA02(tool, param string, unknown []string) *AqlError {
	sort.Strings(unknown)
	hint := ""
	has := func(n ...string) bool {
		for _, u := range unknown {
			for _, x := range n {
				if u == x {
					return true
				}
			}
		}
		return false
	}
	switch {
	case tool == "query" && has("product"):
		hint = " Name the product inside the statement: ... WHERE product = '" + keyMark + "'"
	case has("query", "statement", "aql", "sql"):
		hint = " Pass the statement as " + param + "."
	}
	return aqlErr("bad_args", "tool %s has one parameter: %s (one AQL statement). It does not accept: %s.%s", tool, param, strings.Join(unknown, ", "), hint)
}

func unknownParams(args map[string]any, param string) []string {
	var out []string
	for k := range args {
		if k != param {
			out = append(out, k)
		}
	}
	return out
}

// runQuery is the tool query. The text is the result object or the message.
func (s *session) runQuery(args map[string]any) (string, bool) {
	r := newRun(s)
	obj, err := r.query(args)
	if err != nil {
		return r.show(err.Message), false
	}
	return r.show(toJSON(obj)), true
}

func (r *run) query(args map[string]any) (Obj, *AqlError) {
	if u := unknownParams(args, "query"); len(u) > 0 {
		return nil, errA02("query", "query", u)
	}
	q, ok := args["query"].(string)
	if !ok || strings.TrimSpace(q) == "" {
		return nil, aqlErr("bad_args", "query parameter is required: one AQL statement. Example: %s", exampleOf(entityByName("features")))
	}
	st, err := r.parseStatement(q, true)
	if err != nil {
		return nil, err
	}
	switch x := st.(type) {
	case *DescribeStmt:
		return r.describe(x)
	case *ReadStmt:
		return r.readStatement(x)
	}
	return nil, aqlErr("syntax", "query only reads. Use the execute tool for INSERT, UPDATE and DELETE.")
}

// runExecute is the tool execute. It returns the text, whether it is a
// result, and the parts of the log line.
func (s *session) runExecute(args map[string]any, dryRun bool) (string, bool, writeOutcome, string) {
	r := newRun(s)
	out := writeOutcome{op: "none", entity: "none", id: "none"}
	obj, err := r.execute(args, dryRun, &out)
	if err != nil {
		return r.show(err.Message), false, out, err.Kind
	}
	return r.show(toJSON(obj)), true, out, "ok"
}

func isRecordIDForm(s string) bool {
	return isRefNum(s) || isNumericID(s) || reProdKey.MatchString(s)
}

func (r *run) execute(args map[string]any, dryRun bool, out *writeOutcome) (Obj, *AqlError) {
	if u := unknownParams(args, "statement"); len(u) > 0 {
		return nil, errA02("execute", "statement", u)
	}
	q, ok := args["statement"].(string)
	if !ok || strings.TrimSpace(q) == "" {
		return nil, aqlErr("bad_args", "statement parameter is required: one INSERT, UPDATE or DELETE statement")
	}
	st, err := r.parseStatement(q, false)
	if err != nil {
		return nil, err
	}
	setID := func(id any) {
		if s := strings.ToUpper(textOf(id)); isRecordIDForm(s) {
			out.id = s
		}
	}
	switch x := st.(type) {
	case *InsertStmt:
		out.op, out.entity = "insert", x.Entity.Name
		return r.insert(x, dryRun, out)
	case *UpdateStmt:
		out.op, out.entity = "update", x.Entity.Name
		setID(x.ID)
		return r.update(x, dryRun, out)
	case *DeleteStmt:
		out.op, out.entity = "delete", x.Entity.Name
		setID(x.ID)
		return r.delete(x, dryRun, out)
	}
	return nil, aqlErr("syntax", "execute only changes data. Use the query tool for SELECT and DESCRIBE.")
}

var logMu sync.Mutex

func logExecute(acct Account, out writeOutcome, result string) {
	logMu.Lock()
	defer logMu.Unlock()
	fmt.Fprintf(os.Stderr, "%s execute account=%s op=%s entity=%s id=%s result=%s\n",
		clock.Now().UTC().Format("2006-01-02T15:04:05Z"), acct.Subdomain, out.op, out.entity, out.id, result)
}

// ---- tool-registration ----

type ctxKey struct{}

// sessionFrom gives the session of an MCP request: the one of the HTTP
// request in http mode, the one of the process in stdio mode.
func sessionFrom(ctx context.Context, fallback *session) *session {
	if s, ok := ctx.Value(ctxKey{}).(*session); ok && s != nil {
		return s
	}
	return fallback
}

func logInternal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s internal error: %s\n", clock.Now().UTC().Format("2006-01-02T15:04:05Z"), fmt.Sprintf(format, args...))
}

func guarded(h server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (res *mcp.CallToolResult, err error) {
		defer func() {
			if p := recover(); p != nil {
				logInternal("tool %s: %v", req.Params.Name, p)
				res, err = nil, fmt.Errorf("internal error of the server")
			}
		}()
		return h(ctx, req)
	}
}

func newMCPServer(process *session, settings *Settings) *server.MCPServer {
	filter := func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
		s := sessionFrom(ctx, process)
		out := make([]mcp.Tool, 0, len(tools))
		var key string
		for _, t := range tools {
			if strings.Contains(t.Description, "{KEY}") {
				if key == "" {
					key = s.exampleKey(nil)
				}
				t.Description = strings.ReplaceAll(t.Description, "{KEY}", key)
			}
			out = append(out, t)
		}
		return out
	}
	srv := server.NewMCPServer("mcp-server-aha-aql", serverVersion,
		server.WithToolCapabilities(false),
		server.WithToolFilter(filter),
	)
	srv.AddTool(mcp.NewTool("query",
		mcp.WithDescription(queryDescription()),
		mcp.WithString("query", mcp.Required(), mcp.Description("One AQL statement")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	), guarded(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s := sessionFrom(ctx, process)
		text, ok := s.runQuery(req.GetArguments())
		if !ok {
			return mcp.NewToolResultError(text), nil
		}
		return mcp.NewToolResultText(text), nil
	}))
	if !settings.ReadOnly {
		srv.AddTool(mcp.NewTool("execute",
			mcp.WithDescription(executeDescription),
			mcp.WithString("statement", mcp.Required(), mcp.Description("One AQL statement")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), guarded(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			s := sessionFrom(ctx, process)
			text, ok, out, result := s.runExecute(req.GetArguments(), false)
			logExecute(s.acct, out, result)
			if !ok {
				return mcp.NewToolResultError(text), nil
			}
			return mcp.NewToolResultText(text), nil
		}))
	}
	return srv
}
