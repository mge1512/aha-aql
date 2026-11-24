// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
// tests by: claude-opus-5-5

package independenttests

// FakeAha: the test double of the INTERFACES section, seen from outside the
// binary. The binary under test reaches it the way a sandbox lets it reach
// Aha!: through the HTTPS proxy of the environment (HTTPS_PROXY), with a
// certificate authority in the trust store (SSL_CERT_FILE). FakeAha is that
// proxy. It opens the tunnel, ends TLS with a certificate for *.aha.io, and
// answers the requests from memory with the fixture account. It never opens
// a network connection itself.

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Call is one request that FakeAha received.
type Call struct {
	Method string
	Host   string
	Path   string
	Query  url.Values
	Header http.Header
	Body   string
	Proxy  string // the proxy that the server chose: the address of FakeAha
}

// rule changes the answer of FakeAha for one route.
type rule struct {
	method   string // "" = every method
	path     string
	status   int
	header   map[string]string
	body     string
	times    int // 0 = every time
	noAnswer bool
	used     int
}

type rec = map[string]any

type FakeAha struct {
	t        *testing.T
	ln       net.Listener
	addr     string
	tlsConf  *tls.Config
	mu       sync.Mutex
	calls    []Call
	connects []string
	rules    []*rule
	failures []string
	closed   chan struct{}

	refuseTunnel       bool
	noCustomInLists    bool
	products           []rec
	releases           []rec
	features           []rec
	epics              []rec
	initiatives        []rec
	goals              []rec
	requirements       []rec
	ideas              []rec
	comments           []rec
	users              []rec
	workflows          []rec
	definitions        []rec
	options            map[string][]string
	nextNumeric        int64
	srv                *http.Server
	connCh             chan net.Conn
	stopOnce           sync.Once
	blockedRequestDone chan struct{}
}

const (
	demoID  = "7000000000000000001"
	otherID = "7000000000000000002"
	user1ID = "7000000000000000900"
	user2ID = "7000000000000000901"
)

func newFakeAha(t *testing.T) *FakeAha {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("FakeAha: listen: %v", err)
	}
	f := &FakeAha{
		t:                  t,
		ln:                 ln,
		addr:               ln.Addr().String(),
		tlsConf:            &tls.Config{Certificates: []tls.Certificate{testPKI.leaf}},
		closed:             make(chan struct{}),
		options:            map[string][]string{},
		nextNumeric:        7000000000000005000,
		connCh:             make(chan net.Conn),
		blockedRequestDone: make(chan struct{}),
	}
	f.loadFixture()
	f.srv = &http.Server{Handler: http.HandlerFunc(f.serve), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = f.srv.Serve(&chanListener{ch: f.connCh, closed: f.closed, addr: ln.Addr()}) }()
	go f.acceptLoop()
	t.Cleanup(f.Close)
	return f
}

// ProxyURL is the value of HTTPS_PROXY for the binary under test.
func (f *FakeAha) ProxyURL() string { return "http://" + f.addr }

func (f *FakeAha) Close() {
	f.stopOnce.Do(func() {
		close(f.closed)
		_ = f.ln.Close()
		_ = f.srv.Close()
		f.mu.Lock()
		failures := append([]string(nil), f.failures...)
		f.mu.Unlock()
		for _, m := range failures {
			f.t.Errorf("FakeAha: %s", m)
		}
	})
}

type chanListener struct {
	ch     chan net.Conn
	closed chan struct{}
	addr   net.Addr
}

func (l *chanListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}
func (l *chanListener) Close() error   { return nil }
func (l *chanListener) Addr() net.Addr { return l.addr }

type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufConn) Read(p []byte) (int, error) { return c.r.Read(p) }

func (f *FakeAha) acceptLoop() {
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handleProxyConn(c)
	}
}

func (f *FakeAha) handleProxyConn(c net.Conn) {
	br := bufio.NewReader(c)
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	req, err := http.ReadRequest(br)
	if err != nil {
		c.Close()
		return
	}
	_ = c.SetReadDeadline(time.Time{})
	if req.Method != http.MethodConnect {
		f.fail("the server sent a %s request to the proxy instead of CONNECT", req.Method)
		_, _ = io.WriteString(c, "HTTP/1.1 405 Method Not Allowed\r\nContent-Length: 0\r\n\r\n")
		c.Close()
		return
	}
	f.mu.Lock()
	f.connects = append(f.connects, req.Host)
	refuse := f.refuseTunnel
	f.mu.Unlock()
	if req.Host != "example.aha.io:443" && req.Host != "other.aha.io:443" {
		f.fail("CONNECT for another host: %s", req.Host)
		refuse = true
	}
	if refuse {
		_, _ = io.WriteString(c, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
		c.Close()
		return
	}
	_, _ = io.WriteString(c, "HTTP/1.1 200 Connection established\r\n\r\n")
	tc := tls.Server(&bufConn{Conn: c, r: br}, f.tlsConf)
	if err := tc.Handshake(); err != nil {
		c.Close()
		return
	}
	select {
	case f.connCh <- tc:
	case <-f.closed:
		c.Close()
	}
}

func (f *FakeAha) fail(format string, args ...any) {
	f.mu.Lock()
	f.failures = append(f.failures, fmt.Sprintf(format, args...))
	f.mu.Unlock()
}

// ---- control of FakeAha by a test ----

func (f *FakeAha) Reset() {
	f.mu.Lock()
	f.calls = nil
	f.connects = nil
	f.mu.Unlock()
}

func (f *FakeAha) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Call(nil), f.calls...)
}

func (f *FakeAha) Connects() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.connects...)
}

func (f *FakeAha) CallsTo(method, path string) []Call {
	var out []Call
	for _, c := range f.Calls() {
		if (method == "" || c.Method == method) && c.Path == path {
			out = append(out, c)
		}
	}
	return out
}

func (f *FakeAha) WriteCalls() []Call {
	var out []Call
	for _, c := range f.Calls() {
		if c.Method != http.MethodGet {
			out = append(out, c)
		}
	}
	return out
}

func (f *FakeAha) Answer(method, path string, status int, header map[string]string, body string, times int) {
	f.mu.Lock()
	f.rules = append(f.rules, &rule{method: method, path: path, status: status, header: header, body: body, times: times})
	f.mu.Unlock()
}

func (f *FakeAha) NoAnswer(method, path string) {
	f.mu.Lock()
	f.rules = append(f.rules, &rule{method: method, path: path, noAnswer: true})
	f.mu.Unlock()
}

func (f *FakeAha) RefuseTunnel() {
	f.mu.Lock()
	f.refuseTunnel = true
	f.mu.Unlock()
}

func (f *FakeAha) LeaveOutCustomFieldsOfLists() {
	f.mu.Lock()
	f.noCustomInLists = true
	f.mu.Unlock()
}

func (f *FakeAha) AddDefinition(id, key, name, typ, recordType string) {
	f.mu.Lock()
	f.definitions = append(f.definitions, rec{"id": id, "key": key, "name": name, "type": typ, "custom_fieldable_type": recordType})
	f.mu.Unlock()
}

func (f *FakeAha) SetCustomValue(entity, ref string, entry rec) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.findIn(f.listOf(entity), ref)
	if r == nil {
		f.t.Fatalf("FakeAha: no %s %s", entity, ref)
	}
	cf, _ := r["custom_fields"].([]any)
	r["custom_fields"] = append(cf, entry)
}

// AddProduct adds a product with one release and n features in it.
func (f *FakeAha) AddProduct(key, id string, features int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.products = append(f.products, f.productRecord(key, id, key+" Product", nil))
	if features == 0 {
		return
	}
	relRef := key + "-R-1"
	f.releases = append(f.releases, rec{"id": f.numeric(), "reference_num": relRef, "name": key + " release",
		"release_date": "2026-09-30", "product_id": id, "workflow_status": rec{"name": "Planned"},
		"created_at": "2026-01-01T00:00:00.000Z", "url": "https://example.aha.io/releases/" + relRef})
	for i := 1; i <= features; i++ {
		ref := fmt.Sprintf("%s-%d", key, i)
		f.features = append(f.features, rec{"id": f.numeric(), "reference_num": ref,
			"name": fmt.Sprintf("%s feature %04d", key, i), "workflow_status": rec{"name": "Under consideration"},
			"release_reference_num": relRef, "product_id": id, "custom_fields": []any{},
			"created_at": "2026-01-01T00:00:00.000Z", "url": "https://example.aha.io/features/" + ref})
	}
}

func (f *FakeAha) numeric() string {
	f.nextNumeric++
	return strconv.FormatInt(f.nextNumeric, 10)
}

// ---- the fixture account ----

func textEntry(key, name, value string) rec {
	return rec{"key": key, "name": name, "type": "string", "value": value}
}

func usersEntry(ids ...string) rec {
	var emails, full []any
	var vals []any
	for _, id := range ids {
		vals = append(vals, id)
		if id == user1ID {
			emails = append(emails, "test.user@example.com")
			full = append(full, rec{"id": user1ID, "name": "Test User", "email": "test.user@example.com"})
		} else {
			emails = append(emails, "second.user@example.com")
			full = append(full, rec{"id": user2ID, "name": "Second User", "email": "second.user@example.com"})
		}
	}
	return rec{"key": "reviewers", "name": "Reviewers", "type": "array", "value": vals, "email_value": emails, "full_value": full}
}

func (f *FakeAha) productRecord(key, id, name string, custom []any) rec {
	if custom == nil {
		custom = []any{}
	}
	return rec{
		"id": id, "reference_prefix": key, "name": name, "product_line": false,
		"created_at": "2025-01-10T10:00:00.000Z", "url": "https://example.aha.io/products/" + key,
		"feature_workflow": rec{"id": "401"}, "epic_workflow": rec{"id": "401"}, "requirement_workflow": rec{"id": "401"},
		"release_workflow": rec{"id": "402"}, "initiative_workflow": rec{"id": "403"},
		"strategic_imperative_workflow": rec{"id": "404"}, "idea_workflow": rec{"id": "405"},
		"custom_fields": custom,
	}
}

func user(id string) rec {
	if id == user1ID {
		return rec{"id": user1ID, "name": "Test User", "email": "test.user@example.com"}
	}
	return rec{"id": user2ID, "name": "Second User", "email": "second.user@example.com"}
}

func (f *FakeAha) loadFixture() {
	f.products = []rec{
		f.productRecord("DEMO", demoID, "Demo Product", []any{textEntry("cost_center", "Cost center", "CC-100")}),
		f.productRecord("OTHER", otherID, "Other Product", nil),
	}
	rel := func(n int, name string, date any, parking bool) rec {
		ref := fmt.Sprintf("DEMO-R-%d", n)
		r := rec{"id": fmt.Sprintf("70000000000000001%02d", n), "reference_num": ref, "name": name,
			"release_date": date, "product_id": demoID, "workflow_status": rec{"name": "Planned"},
			"progress": 0, "created_at": "2026-01-01T00:00:00.000Z", "url": "https://example.aha.io/releases/" + ref,
			"theme": rec{"body": ""}, "custom_fields": []any{}}
		if parking {
			r["parking_lot"] = true
		}
		return r
	}
	f.releases = []rec{
		rel(1, "What's next", nil, true),
		rel(2, "Spring release", "2026-03-31", false),
		rel(3, "Summer release", "2026-06-30", false),
		rel(4, "Rider's cup release", "2026-12-15", false),
	}
	f.releases[2]["custom_fields"] = []any{textEntry("budget_owner", "Budget owner", "Finance")}
	feat := func(n int, name, status, release, initiative, assigned string, custom []any) rec {
		ref := fmt.Sprintf("DEMO-%d", n)
		if custom == nil {
			custom = []any{}
		}
		r := rec{"id": fmt.Sprintf("70000000000000002%02d", n), "reference_num": ref, "name": name,
			"workflow_status":       rec{"id": "70000000000000040" + strconv.Itoa(n), "name": status},
			"release_reference_num": release, "product_id": demoID, "score": 0, "progress": 0,
			"created_at": "2026-02-01T08:00:00.000Z", "updated_at": "2026-02-02T08:00:00.000Z",
			"url": "https://example.aha.io/features/" + ref, "custom_fields": custom,
			"description": rec{"id": "70000000000000006" + fmt.Sprintf("%02d", n), "body": ""}}
		if initiative != "" {
			r["initiative_reference_num"] = initiative
		}
		if assigned != "" {
			r["assigned_to_user"] = user(user1ID)
		}
		return r
	}
	f.features = []rec{
		feat(1, "CSV export", "In design", "DEMO-R-2", "", "x", []any{
			rec{"key": "priority", "name": "Priority", "type": "string", "value": "High"},
			rec{"key": "customer", "name": "Customer", "type": "string", "value": "Acme"},
			rec{"key": "go_live", "name": "Go-live date", "type": "date", "value": "2026-04-15"},
			rec{"key": "effort_days", "name": "Effort in days", "type": "number", "value": "12.5"},
			rec{"key": "markets", "name": "Markets", "type": "array", "value": []any{"EMEA", "APAC"}},
			usersEntry(user2ID),
			rec{"key": "release_notes", "name": "Release notes", "type": "note", "value": "<p>Adds the export.</p><p>See the handbook.</p>"},
			rec{"key": "value_score", "name": "Value score", "type": "scorecard", "value": 31},
			rec{"key": "business_case", "name": "Business case", "type": "equation", "value": rec{"result": 7, "formula": "a*b"}},
		}),
		feat(2, "Offline maps", "Under consideration", "DEMO-R-2", "", "", []any{
			rec{"key": "priority", "name": "Priority", "type": "string", "value": "Medium"},
			rec{"key": "effort_days", "name": "Effort in days", "type": "number", "value": "8.0"},
		}),
		feat(3, "Live dashboard", "In development", "DEMO-R-3", "DEMO-S-1", "", []any{
			rec{"key": "priority", "name": "Priority", "type": "string", "value": "High"},
			rec{"key": "customer", "name": "Customer", "type": "string", "value": "Globex"},
			rec{"key": "go_live", "name": "Go-live date", "type": "date", "value": "2026-07-01"},
			rec{"key": "effort_days", "name": "Effort in days", "type": "number", "value": "3.0"},
			rec{"key": "markets", "name": "Markets", "type": "array", "value": []any{"EMEA"}},
			usersEntry(user1ID, user2ID),
		}),
		feat(4, "Live video feed", "Under consideration", "DEMO-R-3", "DEMO-S-1", "", nil),
		feat(5, "Rider's handbook", "Shipped", "DEMO-R-3", "", "", nil),
		feat(6, "Route planner", "Under consideration", "DEMO-R-1", "", "", nil),
	}
	f.features[0]["description"] = rec{"id": "7000000000000000601", "body": "<p>Export <b>all</b> columns.</p><p>Keep the order.</p>"}
	f.initiatives = []rec{{"id": "7000000000000000301", "reference_num": "DEMO-S-1", "name": "Live coverage",
		"workflow_status": rec{"name": "In progress"}, "product_id": demoID, "custom_fields": []any{},
		"url": "https://example.aha.io/initiatives/DEMO-S-1"}}
	f.goals = []rec{
		{"id": "7000000000000000351", "reference_num": "DEMO-G-1", "name": "Grow the audience", "product_id": demoID,
			"success_metric": rec{"name": "Monthly viewers", "workflow_status": rec{"name": "On track"}}, "custom_fields": []any{},
			"url": "https://example.aha.io/strategic_imperatives/DEMO-G-1"},
		{"id": "7000000000000000352", "reference_num": "OTHER-G-1", "name": "Other goal", "product_id": otherID,
			"success_metric": rec{"name": "Not defined yet"}, "custom_fields": []any{},
			"url": "https://example.aha.io/strategic_imperatives/OTHER-G-1"},
	}
	f.requirements = []rec{{"id": "7000000000000000501", "reference_num": "DEMO-1-1", "name": "Column selection",
		"workflow_status": rec{"name": "Under consideration"}, "product_id": demoID, "custom_fields": []any{},
		"feature": rec{"id": "7000000000000000201", "reference_num": "DEMO-1", "name": "CSV export"}}}
	f.epics = []rec{{"id": "7000000000000000551", "reference_num": "DEMO-E-1", "name": "Race day",
		"workflow_status": rec{"name": "In design"}, "product_id": demoID, "custom_fields": []any{},
		"release": rec{"id": "7000000000000000103", "reference_num": "DEMO-R-3", "name": "Summer release"}}}
	f.comments = []rec{{"id": "7000000000000000701", "body": "Verified with the provider.", "user": user(user1ID),
		"created_at": "2026-03-01T12:00:00.000Z", "url": "https://example.aha.io/comments/7000000000000000701",
		"commentable": rec{"id": "7000000000000000203", "url": "https://example.aha.io/features/DEMO-3"}, "_parent": "DEMO-3"}}
	f.users = []rec{
		{"id": user1ID, "name": "Test User", "email": "test.user@example.com", "created_at": "2025-01-01T00:00:00.000Z"},
		{"id": user2ID, "name": "Second User", "email": "second.user@example.com", "created_at": "2025-01-02T00:00:00.000Z"},
	}
	ws := func(names ...string) []any {
		var out []any
		for i, n := range names {
			out = append(out, rec{"id": 9000 + i, "name": n, "position": i + 1, "complete": i == len(names)-1})
		}
		return out
	}
	f.workflows = []rec{
		{"id": 401, "name": "Feature workflow", "statusable_type": "Feature", "workflow_statuses": ws("Under consideration", "In design", "Ready to develop", "In development", "Shipped", "Will not implement")},
		{"id": 402, "name": "Release workflow", "statusable_type": "Release", "workflow_statuses": ws("Planned", "In progress", "Shipped")},
		{"id": 403, "name": "Initiative workflow", "statusable_type": "Initiative", "workflow_statuses": ws("Not started", "In progress", "Done")},
		{"id": 404, "name": "Goal workflow", "statusable_type": "StrategicImperative", "workflow_statuses": ws("Not started", "On track", "At risk", "Achieved")},
		{"id": 405, "name": "Idea workflow", "statusable_type": "Ideas::Idea", "workflow_statuses": ws("Needs review", "Planned", "Shipped")},
	}
	def := func(n int, key, name, typ, rt string) rec {
		return rec{"id": fmt.Sprintf("70000000000000008%02d", n), "key": key, "name": name,
			"type": "CustomFieldDefinitions::" + typ, "custom_fieldable_type": rt}
	}
	f.definitions = []rec{
		def(1, "priority", "Priority", "SelectConstant", "Feature"),
		def(2, "customer", "Customer", "SelectEditable", "Feature"),
		def(3, "go_live", "Go-live date", "DateField", "Feature"),
		def(4, "effort_days", "Effort in days", "NumberField", "Feature"),
		def(5, "markets", "Markets", "SelectMultipleEditable", "Feature"),
		def(6, "platforms", "Platforms", "SelectMultipleConstant", "Feature"),
		def(7, "reviewers", "Reviewers", "Records::UsersField", "Feature"),
		def(8, "release_notes", "Release notes", "NoteField", "Feature"),
		def(9, "value_score", "Value score", "ScorecardField", "Feature"),
		def(10, "business_case", "Business case", "EquationField", "Feature"),
		def(11, "budget_owner", "Budget owner", "TextField", "Release"),
		def(12, "cost_center", "Cost center", "TextField", "Project"),
		def(13, "segment", "Segment", "SelectConstant", "Ideas::Idea"),
	}
	f.options = map[string][]string{
		"7000000000000000801": {"High", "Medium", "Low"},
		"7000000000000000806": {"Web", "iOS", "Android"},
		"7000000000000000802": {"Acme", "Globex"},
		"7000000000000000813": {"Enterprise", "Midmarket"},
	}
}

// ---- serving ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func (f *FakeAha) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c := Call{Method: r.Method, Host: r.Host, Path: r.URL.Path, Query: r.URL.Query(), Header: r.Header.Clone(), Body: string(body), Proxy: f.ProxyURL()}
	f.mu.Lock()
	f.calls = append(f.calls, c)
	f.mu.Unlock()
	if r.Host != "example.aha.io" {
		f.fail("request for another host: %s %s", r.Host, r.URL.Path)
		writeJSON(w, 403, rec{"error": "Forbidden"})
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		f.fail("request outside /api/v1: %s", r.URL.Path)
		writeJSON(w, 404, rec{"error": "Not found"})
		return
	}
	route := strings.TrimPrefix(r.URL.Path, "/api/v1")
	f.mu.Lock()
	var hit *rule
	for _, ru := range f.rules {
		if (ru.method == "" || ru.method == r.Method) && ru.path == route && (ru.times == 0 || ru.used < ru.times) {
			ru.used++
			hit = ru
			break
		}
	}
	f.mu.Unlock()
	if hit != nil {
		if hit.noAnswer {
			select {
			case <-r.Context().Done():
			case <-f.closed:
			case <-time.After(120 * time.Second):
			}
			return
		}
		for k, v := range hit.header {
			w.Header().Set(k, v)
		}
		w.WriteHeader(hit.status)
		_, _ = io.WriteString(w, hit.body)
		return
	}
	if r.Header.Get("Authorization") != "Bearer fixture-key" {
		writeJSON(w, 403, rec{"error": "Forbidden"})
		return
	}
	f.mu.Lock()
	status, answer := f.route(r.Method, route, r.URL.Query(), body)
	f.mu.Unlock()
	if status == 204 {
		w.WriteHeader(204)
		return
	}
	writeJSON(w, status, answer)
}

func notFound() (int, any) { return 404, rec{"error": "Not found"} }

func (f *FakeAha) listOf(entity string) []rec {
	switch entity {
	case "products":
		return f.products
	case "releases":
		return f.releases
	case "features":
		return f.features
	case "epics":
		return f.epics
	case "initiatives":
		return f.initiatives
	case "goals":
		return f.goals
	case "requirements":
		return f.requirements
	case "ideas":
		return f.ideas
	case "comments":
		return f.comments
	case "users":
		return f.users
	}
	return nil
}

func (f *FakeAha) setList(entity string, l []rec) {
	switch entity {
	case "products":
		f.products = l
	case "releases":
		f.releases = l
	case "features":
		f.features = l
	case "epics":
		f.epics = l
	case "initiatives":
		f.initiatives = l
	case "goals":
		f.goals = l
	case "requirements":
		f.requirements = l
	case "ideas":
		f.ideas = l
	case "comments":
		f.comments = l
	}
}

func (f *FakeAha) findIn(l []rec, ref string) rec {
	for _, r := range l {
		if r["reference_num"] == ref || r["id"] == ref || r["reference_prefix"] == ref {
			return r
		}
	}
	return nil
}

var singular = map[string]string{"products": "product", "releases": "release", "features": "feature", "epics": "epic",
	"initiatives": "initiative", "goals": "goal", "requirements": "requirement", "ideas": "idea", "comments": "comment"}

func str(v any) string { s, _ := v.(string); return s }

func (f *FakeAha) productOf(r rec) rec {
	for _, p := range f.products {
		if p["id"] == r["product_id"] {
			return p
		}
	}
	return nil
}

// children of a parent route.
func (f *FakeAha) children(parentEntity string, parent rec, child string) ([]rec, bool) {
	var out []rec
	switch {
	case parentEntity == "products":
		if child == "comments" {
			return nil, true
		}
		for _, r := range f.listOf(child) {
			if r["product_id"] == parent["id"] {
				out = append(out, r)
			}
		}
		return out, child != "workflows"
	case child == "comments":
		for _, r := range f.comments {
			if r["_parent"] == parent["reference_num"] {
				out = append(out, r)
			}
		}
		return out, true
	case parentEntity == "releases" && child == "features":
		for _, r := range f.features {
			if r["release_reference_num"] == parent["reference_num"] {
				out = append(out, r)
			}
		}
		return out, true
	case parentEntity == "releases" && child == "epics":
		for _, r := range f.epics {
			if rr, ok := r["release"].(rec); ok && rr["reference_num"] == parent["reference_num"] {
				out = append(out, r)
			}
		}
		return out, true
	case parentEntity == "epics" && child == "features":
		for _, r := range f.features {
			if r["epic_reference_num"] == parent["reference_num"] {
				out = append(out, r)
			}
		}
		return out, true
	case parentEntity == "initiatives" && (child == "features" || child == "epics"):
		for _, r := range f.listOf(child) {
			if r["initiative_reference_num"] == parent["reference_num"] {
				out = append(out, r)
			}
		}
		return out, true
	case parentEntity == "features" && child == "requirements":
		for _, r := range f.requirements {
			if fr, ok := r["feature"].(rec); ok && fr["reference_num"] == parent["reference_num"] {
				out = append(out, r)
			}
		}
		return out, true
	}
	return nil, false
}

func copyRec(r rec) rec {
	b, _ := json.Marshal(r)
	var out rec
	_ = json.Unmarshal(b, &out)
	delete(out, "_parent")
	return out
}

func (f *FakeAha) listAnswer(member string, l []rec, q url.Values) (int, any) {
	if q.Get("q") != "" {
		var kept []rec
		for _, r := range l {
			if strings.Contains(strings.ToLower(str(r["name"])), strings.ToLower(q.Get("q"))) {
				kept = append(kept, r)
			}
		}
		l = kept
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	per, _ := strconv.Atoi(q.Get("per_page"))
	if per < 1 {
		per = 30
	}
	var fields map[string]bool
	if q.Get("fields") != "" {
		fields = map[string]bool{"id": true}
		for _, n := range strings.Split(q.Get("fields"), ",") {
			fields[n] = true
		}
	}
	total := len(l)
	pages := (total + per - 1) / per
	var recs []any
	for i := (page - 1) * per; i < total && i < page*per; i++ {
		full := copyRec(l[i])
		if fields != nil {
			out := rec{}
			for k, v := range full {
				if fields[k] {
					out[k] = v
				}
			}
			if !fields["custom_fields"] || f.noCustomInLists {
				delete(out, "custom_fields")
			}
			full = out
		} else {
			delete(full, "custom_fields")
		}
		recs = append(recs, full)
	}
	if recs == nil {
		recs = []any{}
	}
	return 200, rec{member: recs, "pagination": rec{"total_records": total, "total_pages": pages, "current_page": page}}
}

func (f *FakeAha) route(method, route string, q url.Values, body []byte) (int, any) {
	parts := strings.Split(strings.Trim(route, "/"), "/")
	switch method {
	case http.MethodGet:
		return f.get(parts, q)
	case http.MethodPost:
		return f.create(parts, body)
	case http.MethodPut:
		return f.update(parts, body)
	case http.MethodDelete:
		return f.remove(parts)
	}
	return 405, rec{"error": "method"}
}

func (f *FakeAha) get(p []string, q url.Values) (int, any) {
	switch {
	case len(p) == 1 && p[0] == "me":
		return 200, rec{"user": copyRec(f.users[0])}
	case len(p) == 1 && p[0] == "users":
		return f.listAnswer("users", f.users, q)
	case len(p) == 1 && p[0] == "custom_field_definitions":
		var l []any
		for _, d := range f.definitions {
			l = append(l, d)
		}
		return 200, rec{"custom_field_definitions": l}
	case len(p) == 3 && p[0] == "custom_field_definitions" && p[2] == "options":
		opts, ok := f.options[p[1]]
		found := false
		for _, d := range f.definitions {
			if d["id"] == p[1] {
				found = true
			}
		}
		if !found {
			return notFound()
		}
		l := []any{}
		for i, o := range opts {
			l = append(l, rec{"id": strconv.Itoa(i + 1), "text": o})
		}
		_ = ok
		return 200, rec{"options": l}
	case len(p) == 1 && p[0] == "products":
		return f.listAnswer("products", f.products, q)
	case len(p) == 1 && singular[p[0]] != "":
		return f.listAnswer(p[0], f.listOf(p[0]), q)
	case len(p) == 2 && singular[p[0]] != "":
		r := f.findIn(f.listOf(p[0]), p[1])
		if r == nil {
			return notFound()
		}
		return 200, rec{singular[p[0]]: copyRec(r)}
	case len(p) == 3 && p[0] == "products" && p[2] == "workflows":
		if f.findIn(f.products, p[1]) == nil {
			return notFound()
		}
		var l []any
		for _, w := range f.workflows {
			l = append(l, w)
		}
		return 200, rec{"workflows": l}
	case len(p) == 3 && singular[p[0]] != "":
		parent := f.findIn(f.listOf(p[0]), p[1])
		if parent == nil {
			return notFound()
		}
		l, ok := f.children(p[0], parent, p[2])
		if !ok {
			return notFound()
		}
		return f.listAnswer(p[2], l, q)
	}
	return notFound()
}

func (f *FakeAha) nextRef(entity, prefix string, product rec) string {
	key := str(product["reference_prefix"])
	max := 0
	for _, r := range f.listOf(entity) {
		ref := str(r["reference_num"])
		if !strings.HasPrefix(ref, key+"-"+prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(ref, key+"-"+prefix))
		if err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s-%s%d", key, prefix, max+1)
}

func firstStatus(workflow int) rec {
	return rec{"name": map[int]string{401: "Under consideration", 402: "Planned", 403: "Not started", 404: "Not started", 405: "Needs review"}[workflow]}
}

func (f *FakeAha) create(p []string, body []byte) (int, any) {
	if len(p) != 3 {
		return notFound()
	}
	parent := f.findIn(f.listOf(p[0]), p[1])
	if parent == nil {
		return notFound()
	}
	child := p[2]
	sing := singular[child]
	var wrapper map[string]rec
	if err := json.Unmarshal(body, &wrapper); err != nil || wrapper[sing] == nil {
		return 400, rec{"error": "Required parameter missing: " + sing}
	}
	in := wrapper[sing]
	product := parent
	if p[0] != "products" {
		product = f.productOf(parent)
	}
	nr := rec{"id": f.numeric(), "created_at": "2026-11-13T09:30:00.000Z", "custom_fields": []any{}}
	if product != nil {
		nr["product_id"] = product["id"]
	}
	switch child {
	case "features":
		nr["reference_num"] = f.nextRef("features", "", product)
		nr["release_reference_num"] = parent["reference_num"]
		nr["workflow_status"] = firstStatus(401)
	case "epics":
		nr["reference_num"] = f.nextRef("epics", "E-", product)
		nr["release"] = rec{"id": parent["id"], "reference_num": parent["reference_num"], "name": parent["name"]}
		nr["workflow_status"] = firstStatus(401)
	case "releases":
		nr["reference_num"] = f.nextRef("releases", "R-", product)
		nr["workflow_status"] = firstStatus(402)
	case "initiatives":
		nr["reference_num"] = f.nextRef("initiatives", "S-", product)
		nr["workflow_status"] = firstStatus(403)
	case "goals":
		sm, ok := in["success_metric"].(map[string]any)
		if !ok {
			return 400, rec{"error": "Required parameter missing: success_metric"}
		}
		if strings.TrimSpace(str(sm["name"])) == "" {
			return 400, rec{"error": "Validation failed", "errors": rec{"message": "Success metric cannot be blank"}}
		}
		nr["reference_num"] = f.nextRef("goals", "G-", product)
		st := firstStatus(404)
		if ws, ok := sm["workflow_status"].(string); ok {
			st = rec{"name": ws}
		}
		nr["success_metric"] = rec{"name": sm["name"], "workflow_status": st}
		delete(in, "success_metric")
	case "ideas":
		nr["reference_num"] = f.nextRef("ideas", "I-", product)
		nr["workflow_status"] = firstStatus(405)
	case "requirements":
		max := 0
		for _, r := range f.requirements {
			ref := str(r["reference_num"])
			if strings.HasPrefix(ref, str(parent["reference_num"])+"-") {
				n, _ := strconv.Atoi(strings.TrimPrefix(ref, str(parent["reference_num"])+"-"))
				if n > max {
					max = n
				}
			}
		}
		nr["reference_num"] = fmt.Sprintf("%s-%d", parent["reference_num"], max+1)
		nr["feature"] = rec{"id": parent["id"], "reference_num": parent["reference_num"], "name": parent["name"]}
		nr["workflow_status"] = firstStatus(401)
	case "comments":
		nr["user"] = user(user1ID)
		nr["commentable"] = rec{"id": parent["id"], "url": "https://example.aha.io/" + p[0] + "/" + str(parent["reference_num"])}
		nr["_parent"] = parent["reference_num"]
		delete(nr, "product_id")
	default:
		return notFound()
	}
	f.apply(child, nr, in)
	f.setList(child, append(f.listOf(child), nr))
	return 200, rec{sing: copyRec(nr)}
}

func (f *FakeAha) apply(entity string, r rec, in rec) {
	for k, v := range in {
		switch k {
		case "workflow_status":
			if entity == "goals" {
				sm, _ := r["success_metric"].(rec)
				if sm == nil {
					sm = rec{}
				}
				sm["workflow_status"] = rec{"name": v}
				r["success_metric"] = sm
			} else {
				r["workflow_status"] = rec{"name": v}
			}
		case "success_metric":
			// ignored in an update, as Aha! does
		case "assigned_to_user":
			s := str(v)
			for _, u := range f.users {
				if u["email"] == s || u["id"] == s {
					r["assigned_to_user"] = copyRec(u)
				}
			}
		case "description", "theme":
			r[k] = rec{"body": v}
		case "tags":
			var tags []any
			for _, t := range strings.Split(str(v), ",") {
				if strings.TrimSpace(t) != "" {
					tags = append(tags, strings.TrimSpace(t))
				}
			}
			r["tags"] = tags
		case "release":
			r["release_reference_num"] = v
		case "initiative":
			r["initiative_reference_num"] = v
		case "custom_fields":
			m, _ := v.(map[string]any)
			f.applyCustom(r, m)
		default:
			r[k] = v
		}
	}
}

func (f *FakeAha) applyCustom(r rec, m map[string]any) {
	cf, _ := r["custom_fields"].([]any)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		v := m[key]
		var def rec
		for _, d := range f.definitions {
			if d["key"] == key {
				def = d
				break
			}
		}
		entry := rec{"key": key, "name": str(def["name"]), "type": "string", "value": v}
		typ := str(def["type"])
		switch {
		case strings.HasSuffix(typ, "NumberField"):
			entry["type"] = "number"
			if n, ok := v.(float64); ok {
				entry["value"] = strconv.FormatFloat(n, 'f', 1, 64)
			}
		case strings.HasSuffix(typ, "UsersField"):
			var ids []any
			for _, x := range v.([]any) {
				s := str(x)
				for _, u := range f.users {
					if u["email"] == s || u["id"] == s {
						ids = append(ids, u["id"])
					}
				}
			}
			var idStrs []string
			for _, id := range ids {
				idStrs = append(idStrs, str(id))
			}
			entry = usersEntry(idStrs...)
			entry["name"] = str(def["name"])
			entry["key"] = key
		case strings.HasSuffix(typ, "SelectMultipleEditable"), strings.HasSuffix(typ, "SelectMultipleConstant"):
			entry["type"] = "array"
		}
		replaced := false
		for i, e := range cf {
			if em, ok := e.(rec); ok && em["key"] == key {
				cf[i] = entry
				replaced = true
			} else if em, ok := e.(map[string]any); ok && em["key"] == key {
				cf[i] = entry
				replaced = true
			}
		}
		if !replaced {
			cf = append(cf, entry)
		}
	}
	r["custom_fields"] = cf
}

func (f *FakeAha) update(p []string, body []byte) (int, any) {
	if len(p) != 2 || singular[p[0]] == "" || p[0] == "products" {
		return notFound()
	}
	r := f.findIn(f.listOf(p[0]), p[1])
	if r == nil {
		return notFound()
	}
	var wrapper map[string]rec
	if err := json.Unmarshal(body, &wrapper); err != nil || wrapper[singular[p[0]]] == nil {
		return 400, rec{"error": "Required parameter missing: " + singular[p[0]]}
	}
	f.apply(p[0], r, wrapper[singular[p[0]]])
	return 200, rec{singular[p[0]]: copyRec(r)}
}

func (f *FakeAha) remove(p []string) (int, any) {
	var entity, ref string
	switch {
	case len(p) == 2:
		entity, ref = p[0], p[1]
	case len(p) == 4 && p[0] == "products":
		if f.findIn(f.products, p[1]) == nil {
			return notFound()
		}
		entity, ref = p[2], p[3]
	default:
		return notFound()
	}
	l := f.listOf(entity)
	for i, r := range l {
		if r["reference_num"] == ref || r["id"] == ref {
			f.setList(entity, append(l[:i:i], l[i+1:]...))
			return 204, nil
		}
	}
	return notFound()
}
