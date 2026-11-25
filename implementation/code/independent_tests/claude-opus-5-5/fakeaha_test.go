// generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268
// tests by: claude-opus-5-5

// FakeAha is the test double that the INTERFACES section of the spec
// declares. It is an HTTP proxy outside the server process. The server
// under test reaches it through HTTPS_PROXY, asks it with CONNECT for a
// tunnel to example.aha.io:443, and FakeAha ends the TLS connection inside
// the tunnel with a certificate that the certificate authority of the test
// has signed. It answers from memory with the fixture account of the spec
// and records every request. It opens no network connection of its own.
package aqltests

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type obj = map[string]any

// Call is one request that arrived inside a tunnel.
type Call struct {
	Method string
	Host   string
	Path   string // without /api/v1
	Query  url.Values
	Header http.Header
	Body   []byte
	Time   time.Time
}

// Connect is one CONNECT request that arrived at the proxy.
type Connect struct {
	Host   string
	Header http.Header
}

type override struct {
	method   string
	path     string
	status   int
	header   map[string]string
	resetIn  int // seconds added to now for X-Ratelimit-Reset; <0: no such header
	times    int // <0: every time
	noAnswer bool
	body     string
}

// FakeAha is the fake Aha! account behind a fake proxy.
type FakeAha struct {
	t           *testing.T
	dir         string
	CAFile      string
	ProxyURL    string
	proxyLn     net.Listener
	proxySrv    *http.Server
	innerSrv    *http.Server
	inner       *chanListener
	tlsCfg      *tls.Config
	done        chan struct{}
	closeOnce   sync.Once
	mu          sync.Mutex
	calls       []Call
	connects    []Connect
	badConnects []string
	refuse      bool
	omitCF      bool
	overrides   []*override
	data        *fixture
}

type chanListener struct {
	ch     chan net.Conn
	done   chan struct{}
	once   sync.Once
	addr   net.Addr
	closed bool
}

func (l *chanListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}
func (l *chanListener) Close() error   { l.once.Do(func() { close(l.done) }); return nil }
func (l *chanListener) Addr() net.Addr { return l.addr }

// assertInSandbox fails the test when path is outside dir (hermetic rule 5).
func assertInSandbox(t *testing.T, dir, path string) {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	d, _ := filepath.Abs(dir)
	if !strings.HasPrefix(abs, d+string(os.PathSeparator)) {
		t.Fatalf("refusing to write outside the sandbox: %s", abs)
	}
}

func writeSandboxFile(t *testing.T, dir, name string, data []byte, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(dir, name)
	assertInSandbox(t, dir, p)
	if err := os.WriteFile(p, data, mode); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatalf("chmod %s: %v", p, err)
	}
	return p
}

func newCA(t *testing.T, cn string) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	return cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// NewFakeAha starts FakeAha with the unchanged fixture account. dir is the
// sandbox of the test.
func NewFakeAha(t *testing.T, dir string) *FakeAha {
	t.Helper()
	ca, caKey, caPEM := newCA(t, "aha-aql test CA")
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano() + 1),
		Subject:      pkix.Name{CommonName: "example.aha.io"},
		DNSNames:     []string{"example.aha.io"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	f := &FakeAha{t: t, dir: dir, done: make(chan struct{}), data: newFixture()}
	f.CAFile = writeSandboxFile(t, dir, "test-ca.pem", caPEM, 0o600)
	f.tlsCfg = &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}},
		NextProtos:   []string{"http/1.1"},
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f.proxyLn = ln
	f.ProxyURL = "http://" + ln.Addr().String()
	f.inner = &chanListener{ch: make(chan net.Conn), done: make(chan struct{}), addr: ln.Addr()}
	f.innerSrv = &http.Server{Handler: http.HandlerFunc(f.serveAPI), ErrorLog: nil}
	f.proxySrv = &http.Server{Handler: http.HandlerFunc(f.serveProxy)}
	go func() { _ = f.innerSrv.Serve(f.inner) }()
	go func() { _ = f.proxySrv.Serve(ln) }()
	t.Cleanup(f.Close)
	return f
}

// Close stops FakeAha and releases every request that waits for no answer.
func (f *FakeAha) Close() {
	f.closeOnce.Do(func() {
		close(f.done)
		_ = f.proxySrv.Close()
		_ = f.inner.Close()
		_ = f.innerSrv.Close()
		f.mu.Lock()
		bad := append([]string(nil), f.badConnects...)
		f.mu.Unlock()
		if len(bad) > 0 {
			f.t.Errorf("CONNECT for another host: %v", bad)
		}
	})
}

func (f *FakeAha) serveProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		f.mu.Lock()
		f.badConnects = append(f.badConnects, "non-CONNECT "+r.Method+" "+r.URL.String())
		f.mu.Unlock()
		http.Error(w, "proxy accepts CONNECT only", http.StatusMethodNotAllowed)
		return
	}
	f.mu.Lock()
	f.connects = append(f.connects, Connect{Host: r.Host, Header: r.Header.Clone()})
	refuse := f.refuse
	if r.Host != "example.aha.io:443" {
		f.badConnects = append(f.badConnects, r.Host)
		refuse = true
	}
	f.mu.Unlock()
	if refuse {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "no hijack", 500)
		return
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		return
	}
	if _, err := conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n")); err != nil {
		conn.Close()
		return
	}
	var c net.Conn = conn
	if buf != nil && buf.Reader.Buffered() > 0 {
		c = &prefixConn{Conn: conn, r: io.MultiReader(buf.Reader, conn)}
	}
	tc := tls.Server(c, f.tlsCfg)
	select {
	case f.inner.ch <- tc:
	case <-f.done:
		conn.Close()
	}
}

type prefixConn struct {
	net.Conn
	r io.Reader
}

func (p *prefixConn) Read(b []byte) (int, error) { return p.r.Read(b) }

// ---- controls of the test ----

// Reset forgets the recorded calls and CONNECT requests.
func (f *FakeAha) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
	f.connects = nil
}

// ResetData restores the unchanged fixture account.
func (f *FakeAha) ResetData() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = newFixture()
}

func (f *FakeAha) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Call(nil), f.calls...)
}

func (f *FakeAha) Connects() []Connect {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Connect(nil), f.connects...)
}

func (f *FakeAha) RefuseConnect(v bool) { f.mu.Lock(); f.refuse = v; f.mu.Unlock() }

// OmitCustomFieldsInLists leaves the member custom_fields out of every
// record of a list.
func (f *FakeAha) OmitCustomFieldsInLists(v bool) { f.mu.Lock(); f.omitCF = v; f.mu.Unlock() }

// Answer tells FakeAha to answer method+path with status (and a header)
// the given number of times (<0: always).
func (f *FakeAha) Answer(method, path string, status int, times int, header map[string]string, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.overrides = append(f.overrides, &override{method: method, path: path, status: status, times: times, header: header, resetIn: -1, body: body})
}

// AnswerRateLimited answers with 429 and X-Ratelimit-Reset = now + secs.
func (f *FakeAha) AnswerRateLimited(method, path string, times, secs int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.overrides = append(f.overrides, &override{method: method, path: path, status: 429, times: times, resetIn: secs})
}

// NoAnswer tells FakeAha to give no answer to method+path.
func (f *FakeAha) NoAnswer(method, path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.overrides = append(f.overrides, &override{method: method, path: path, noAnswer: true, times: -1, resetIn: -1})
}

// AddDefinition adds a custom field definition to the fixture account.
func (f *FakeAha) AddDefinition(id, key, name, typ, fieldable string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data.defs = append(f.data.defs, obj{"id": id, "key": key, "name": name, "type": typ, "custom_fieldable_type": fieldable})
}

// SetCustomValue sets a custom entry of a feature.
func (f *FakeAha) SetCustomValue(ref, key string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.data.find("features", ref)
	if r == nil {
		f.t.Fatalf("no feature %s", ref)
	}
	setEntry(f.data, r, key, value)
}

// AddProduct adds a product with features. With release != "" the
// features are in that release.
func (f *FakeAha) AddProduct(key, id string, nFeatures int, release string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data.addProduct(key, id, key+" product")
	if release != "" {
		f.data.releases = append(f.data.releases, obj{
			"id": id + "1", "reference_num": release, "name": release, "release_date": nil, "start_date": nil,
			"workflow_status": obj{"name": "Planned"}, "released": false, "product_id": id, "progress": 0,
			"theme": obj{"body": ""}, "created_at": "2026-01-01T08:00:00.000Z", "url": "https://example.aha.io/releases/" + release,
			"_product": key,
		})
	}
	for i := 1; i <= nFeatures; i++ {
		ref := fmt.Sprintf("%s-%d", key, i)
		var rel any
		if release != "" {
			rel = release
		}
		f.data.features = append(f.data.features, obj{
			"id": fmt.Sprintf("%s%05d", id[:len(id)-5], 10000+i), "reference_num": ref, "name": fmt.Sprintf("Bulk feature %d", i),
			"workflow_status":       obj{"name": "Under consideration", "position": 1, "complete": false},
			"release_reference_num": rel, "initiative_reference_num": nil, "epic_reference_num": nil,
			"assigned_to_user": nil, "product_id": id, "start_date": nil, "due_date": nil, "tags": []any{},
			"score": 0, "progress": 0, "description": obj{"body": ""},
			"created_at": "2026-02-01T08:00:00.000Z", "updated_at": "2026-02-03T23:30:00.000Z",
			"url": "https://example.aha.io/features/" + ref, "_product": key,
		})
	}
}

// ---- recorded calls, helpers for assertions ----

func (f *FakeAha) count(method, path string) int {
	n := 0
	for _, c := range f.Calls() {
		if c.Method == method && c.Path == path {
			n++
		}
	}
	return n
}

func (f *FakeAha) writeCalls() []Call {
	var out []Call
	for _, c := range f.Calls() {
		if c.Method == "POST" || c.Method == "PUT" || c.Method == "DELETE" {
			out = append(out, c)
		}
	}
	return out
}

func (f *FakeAha) describeCalls() string {
	var parts []string
	for _, c := range f.Calls() {
		s := c.Method + " " + c.Path
		if q := c.Query.Encode(); q != "" {
			s += "?" + q
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "; ")
}

// ---- the API ----

func (f *FakeAha) serveAPI(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	f.mu.Lock()
	f.calls = append(f.calls, Call{Method: r.Method, Host: r.Host, Path: path, Query: r.URL.Query(), Header: r.Header.Clone(), Body: body, Time: time.Now()})
	var ov *override
	for _, o := range f.overrides {
		if o.method == r.Method && o.path == path && o.times != 0 {
			ov = o
			if o.times > 0 {
				o.times--
			}
			break
		}
	}
	f.mu.Unlock()
	if r.Host != "example.aha.io" && r.Host != "example.aha.io:443" {
		f.mu.Lock()
		f.badConnects = append(f.badConnects, "request for host "+r.Host)
		f.mu.Unlock()
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		writeJSON(w, 404, obj{"error": "Not found"})
		return
	}
	if ov != nil {
		if ov.noAnswer {
			select {
			case <-r.Context().Done():
			case <-f.done:
			}
			return
		}
		for k, v := range ov.header {
			w.Header().Set(k, v)
		}
		if ov.resetIn >= 0 {
			w.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().UTC().Unix()+int64(ov.resetIn), 10))
		}
		if ov.body != "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(ov.status)
		_, _ = io.WriteString(w, ov.body)
		return
	}
	if r.Header.Get("Authorization") != "Bearer fixture-key" {
		writeJSON(w, 403, obj{"error": "Forbidden"})
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	status, out := f.data.route(r.Method, path, r.URL.Query(), body, f.omitCF)
	if status == 204 {
		w.WriteHeader(204)
		return
	}
	writeJSON(w, status, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ---- the fixture account ----

type fixture struct {
	products, users, releases, features, epics, requirements, initiatives, goals, ideas, comments []obj
	defs                                                                                          []obj
	options                                                                                       map[string][]string
	nextID                                                                                        int64
}

const (
	idDEMO  = "7000000000000000001"
	idOTHER = "7000000000000000002"
	idTest  = "7000000000000000900"
	idSec   = "7000000000000000901"
)

var workflowStatuses = map[string][]string{
	"401": {"Under consideration", "In design", "Ready to develop", "In development", "Shipped", "Will not implement"},
	"402": {"Planned", "In progress", "Shipped"},
	"403": {"Not started", "In progress", "Done"},
	"404": {"Not started", "On track", "At risk", "Achieved"},
	"405": {"Needs review", "Planned", "Shipped"},
}

var workflowOfType = map[string]string{
	"features": "401", "epics": "401", "requirements": "401", "releases": "402",
	"initiatives": "403", "goals": "404", "ideas": "405",
}

func userObj(id string) obj {
	switch id {
	case idTest:
		return obj{"id": idTest, "name": "Test User", "email": "test.user@example.com", "created_at": "2025-01-01T08:00:00.000Z"}
	case idSec:
		return obj{"id": idSec, "name": "Second User", "email": "second.user@example.com", "created_at": "2025-01-02T08:00:00.000Z"}
	}
	return nil
}

func statusObj(typ, name string) obj {
	wf := workflowOfType[typ]
	for i, s := range workflowStatuses[wf] {
		if strings.EqualFold(s, name) {
			return obj{"id": fmt.Sprintf("%s%02d", wf, i+1), "name": s, "position": i + 1, "complete": s == "Shipped" || s == "Done" || s == "Achieved"}
		}
	}
	return obj{"name": name}
}

func (fx *fixture) addProduct(key, id, name string) {
	fx.products = append(fx.products, obj{
		"id": id, "reference_prefix": key, "name": name, "product_line": false, "parent_id": nil,
		"description": obj{"id": id + "9", "body": "<p>" + name + ".</p>"},
		"created_at":  "2025-06-01T08:00:00.000Z", "url": "https://example.aha.io/products/" + key,
		"feature_workflow": obj{"id": "401"}, "epic_workflow": obj{"id": "401"}, "requirement_workflow": obj{"id": "401"},
		"release_workflow": obj{"id": "402"}, "initiative_workflow": obj{"id": "403"},
		"strategic_imperative_workflow": obj{"id": "404"}, "idea_workflow": obj{"id": "405"},
		"_product": key,
	})
}

func newFixture() *fixture {
	fx := &fixture{nextID: 7000000000000009000, options: map[string][]string{
		"7000000000000000801": {"High", "Medium", "Low"},
		"7000000000000000802": {"Acme", "Globex"},
		"7000000000000000806": {"Web", "iOS", "Android"},
		"7000000000000000813": {"Enterprise", "Midmarket"},
	}}
	fx.addProduct("DEMO", idDEMO, "Demo Product")
	fx.addProduct("OTHER", idOTHER, "Other Product")
	fx.users = []obj{userObj(idTest), userObj(idSec)}
	rel := func(n int, name string, date any, parking bool) obj {
		o := obj{
			"id": fmt.Sprintf("70000000000000001%02d", n), "reference_num": fmt.Sprintf("DEMO-R-%d", n), "name": name,
			"release_date": date, "start_date": nil, "workflow_status": statusObj("releases", "Planned"), "released": false,
			"product_id": idDEMO, "progress": 0, "theme": obj{"body": ""}, "created_at": "2026-01-01T08:00:00.000Z",
			"url": fmt.Sprintf("https://example.aha.io/releases/DEMO-R-%d", n), "_product": "DEMO",
		}
		if parking {
			o["parking_lot"] = true
		}
		return o
	}
	fx.releases = []obj{
		rel(1, "What's next", nil, true),
		rel(2, "Spring release", "2026-03-31", false),
		rel(3, "Summer release", "2026-06-30", false),
		rel(4, "Rider's cup release", "2026-12-15", false),
	}
	feat := func(n int, name, status, release string, initiative any, assigned string) obj {
		var au any
		if assigned != "" {
			au = userObj(assigned)
		}
		updated := "2026-02-03T23:30:00.000Z"
		if n == 1 {
			updated = "2026-02-02T08:00:00.000Z"
		}
		desc := obj{"id": fmt.Sprintf("70000000000000006%02d", n), "body": ""}
		if n == 1 {
			desc["body"] = "<p>Export <b>all</b> columns.</p><p>Keep the order.</p>"
			desc["id"] = "7000000000000000601"
		}
		return obj{
			"id": fmt.Sprintf("70000000000000002%02d", n), "reference_num": fmt.Sprintf("DEMO-%d", n), "name": name,
			"workflow_status": statusObj("features", status), "release_reference_num": release,
			"initiative_reference_num": initiative, "epic_reference_num": nil, "assigned_to_user": au,
			"product_id": idDEMO, "start_date": nil, "due_date": nil, "tags": []any{}, "score": 0, "progress": 0,
			"description": desc, "created_at": "2026-02-01T08:00:00.000Z", "updated_at": updated,
			"url": fmt.Sprintf("https://example.aha.io/features/DEMO-%d", n), "_product": "DEMO",
		}
	}
	fx.features = []obj{
		feat(1, "CSV export", "In design", "DEMO-R-2", nil, idTest),
		feat(2, "Offline maps", "Under consideration", "DEMO-R-2", nil, ""),
		feat(3, "Live dashboard", "In development", "DEMO-R-3", "DEMO-S-1", ""),
		feat(4, "Live video feed", "Under consideration", "DEMO-R-3", "DEMO-S-1", ""),
		feat(5, "Rider's handbook", "Shipped", "DEMO-R-3", nil, ""),
		feat(6, "Route planner", "Under consideration", "DEMO-R-1", nil, ""),
	}
	fx.features[2]["tags"] = []any{"live", "dashboard"}
	fx.features[3]["tags"] = []any{"live"}
	fx.initiatives = []obj{{
		"id": "7000000000000000501", "reference_num": "DEMO-S-1", "name": "Live coverage",
		"workflow_status": statusObj("initiatives", "In progress"), "product_id": idDEMO, "start_date": nil, "end_date": nil,
		"progress": 0, "value": nil, "effort": nil, "color": "#00aa00", "presented": false, "description": obj{"body": ""},
		"created_at": "2026-01-05T08:00:00.000Z", "updated_at": "2026-01-06T08:00:00.000Z",
		"url": "https://example.aha.io/initiatives/DEMO-S-1", "_product": "DEMO",
	}}
	fx.goals = []obj{
		{"id": "7000000000000000551", "reference_num": "DEMO-G-1", "name": "Grow the audience",
			"success_metric": obj{"name": "Monthly viewers", "workflow_status": statusObj("goals", "On track")},
			"product_id":     idDEMO, "progress": 0, "value": nil, "effort": nil, "description": obj{"body": ""},
			"created_at": "2026-01-05T08:00:00.000Z", "updated_at": "2026-01-06T08:00:00.000Z",
			"url": "https://example.aha.io/strategic_imperatives/DEMO-G-1", "_product": "DEMO"},
		{"id": "7000000000000000552", "reference_num": "OTHER-G-1", "name": "Other goal",
			"success_metric": obj{"name": "Not defined yet", "workflow_status": statusObj("goals", "Not started")},
			"product_id":     idOTHER, "progress": 0, "value": nil, "effort": nil, "description": obj{"body": ""},
			"created_at": "2026-01-05T08:00:00.000Z", "updated_at": "2026-01-06T08:00:00.000Z",
			"url": "https://example.aha.io/strategic_imperatives/OTHER-G-1", "_product": "OTHER"},
	}
	fx.requirements = []obj{{
		"id": "7000000000000000451", "reference_num": "DEMO-1-1", "name": "Column selection",
		"workflow_status":  statusObj("requirements", "Under consideration"),
		"feature":          obj{"id": "7000000000000000201", "reference_num": "DEMO-1", "name": "CSV export"},
		"assigned_to_user": nil, "original_estimate": nil, "work_done": nil, "description": obj{"body": ""},
		"created_at": "2026-02-01T09:00:00.000Z", "url": "https://example.aha.io/requirements/DEMO-1-1",
		"_product": "DEMO", "_feature": "DEMO-1",
	}}
	fx.epics = []obj{{
		"id": "7000000000000000301", "reference_num": "DEMO-E-1", "name": "Race day",
		"workflow_status":          statusObj("epics", "In design"),
		"release":                  obj{"id": "7000000000000000103", "reference_num": "DEMO-R-3", "name": "Summer release"},
		"initiative_reference_num": nil, "assigned_to_user": nil, "product_id": idDEMO, "start_date": nil,
		"due_date": nil, "progress": 0, "description": obj{"body": ""}, "created_at": "2026-01-10T08:00:00.000Z",
		"updated_at": "2026-01-11T08:00:00.000Z", "url": "https://example.aha.io/epics/DEMO-E-1", "_product": "DEMO",
	}}
	fx.ideas = []obj{}
	fx.comments = []obj{{
		"id": "7000000000000000701", "body": "Verified with the provider.", "user": userObj(idTest),
		"commentable": obj{"id": "7000000000000000203", "url": "https://example.aha.io/features/DEMO-3"},
		"created_at":  "2026-02-05T08:00:00.000Z", "url": "https://example.aha.io/comments/7000000000000000701",
		"_product": "DEMO", "_parent": "DEMO-3",
	}}
	def := func(n int, key, name, typ, fieldable string) obj {
		return obj{"id": fmt.Sprintf("70000000000000008%02d", n), "key": key, "name": name,
			"type": "CustomFieldDefinitions::" + typ, "custom_fieldable_type": fieldable}
	}
	fx.defs = []obj{
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
	f1, f2, f3 := fx.features[0], fx.features[1], fx.features[2]
	setEntry(fx, f1, "priority", "High")
	setEntry(fx, f1, "customer", "Acme")
	setEntry(fx, f1, "go_live", "2026-04-15")
	setEntry(fx, f1, "effort_days", "12.5")
	setEntry(fx, f1, "markets", []any{"EMEA", "APAC"})
	setEntry(fx, f1, "reviewers", []any{idSec})
	setEntry(fx, f1, "release_notes", "<p>Adds the export.</p><p>See the handbook.</p>")
	setEntry(fx, f1, "value_score", 31)
	setEntry(fx, f1, "business_case", obj{"worksheet": obj{"rows": 3}})
	setEntry(fx, f2, "priority", "Medium")
	setEntry(fx, f2, "effort_days", "8.0")
	setEntry(fx, f3, "priority", "High")
	setEntry(fx, f3, "customer", "Globex")
	setEntry(fx, f3, "go_live", "2026-07-01")
	setEntry(fx, f3, "effort_days", "3.0")
	setEntry(fx, f3, "markets", []any{"EMEA"})
	setEntry(fx, f3, "reviewers", []any{idTest, idSec})
	setEntry(fx, fx.releases[2], "budget_owner", "Finance")
	setEntry(fx, fx.products[0], "cost_center", "CC-100")
	return fx
}

func (fx *fixture) defByKey(key string) obj {
	for _, d := range fx.defs {
		if d["key"] == key {
			return d
		}
	}
	return nil
}

func lastPart(s string) string {
	if i := strings.LastIndex(s, "::"); i >= 0 {
		return s[i+2:]
	}
	return s
}

// setEntry sets the custom entry of key in record r, in the entry form of
// the spec (custom-fields, table of custom values in an API record).
func setEntry(fx *fixture, r obj, key string, value any) {
	d := fx.defByKey(key)
	name, typ := key, "string"
	if d != nil {
		name = d["name"].(string)
		typ = lastPart(d["type"].(string))
	}
	var entries []any
	if e, ok := r["_cf"].([]any); ok {
		entries = e
	}
	var entry obj
	switch typ {
	case "Records::UsersField", "UsersField":
		var ids, emails, full []any
		if list, ok := value.([]any); ok {
			for _, v := range list {
				s := fmt.Sprint(v)
				var u obj
				for _, cand := range fx.users {
					if cand["id"] == s || strings.EqualFold(fmt.Sprint(cand["email"]), s) {
						u = cand
					}
				}
				if u == nil {
					continue
				}
				ids = append(ids, u["id"])
				emails = append(emails, u["email"])
				full = append(full, obj{"id": u["id"], "name": u["name"], "email": u["email"]})
			}
		}
		entry = obj{"key": key, "name": name, "type": "array", "value": nonNil(ids), "email_value": nonNil(emails), "full_value": nonNil(full)}
	case "SelectMultipleEditable", "SelectMultipleConstant":
		entry = obj{"key": key, "name": name, "type": "array", "value": value}
	case "NumberField":
		entry = obj{"key": key, "name": name, "type": "number", "value": value}
	case "NoteField":
		entry = obj{"key": key, "name": name, "type": "note", "value": value}
	case "ScorecardField":
		entry = obj{"key": key, "name": name, "type": "scorecard", "value": value}
	case "DateField":
		entry = obj{"key": key, "name": name, "type": "date", "value": value}
	default:
		entry = obj{"key": key, "name": name, "type": "string", "value": value}
	}
	for i, e := range entries {
		if e.(obj)["key"] == key {
			entries[i] = entry
			r["_cf"] = entries
			return
		}
	}
	r["_cf"] = append(entries, entry)
}

func nonNil(l []any) []any {
	if l == nil {
		return []any{}
	}
	return l
}

func (fx *fixture) coll(typ string) *[]obj {
	switch typ {
	case "products":
		return &fx.products
	case "users":
		return &fx.users
	case "releases":
		return &fx.releases
	case "features":
		return &fx.features
	case "epics":
		return &fx.epics
	case "requirements":
		return &fx.requirements
	case "initiatives":
		return &fx.initiatives
	case "goals":
		return &fx.goals
	case "ideas":
		return &fx.ideas
	case "comments":
		return &fx.comments
	}
	return nil
}

var singularOf = map[string]string{
	"products": "product", "releases": "release", "features": "feature", "epics": "epic",
	"requirements": "requirement", "initiatives": "initiative", "goals": "goal", "ideas": "idea",
	"comments": "comment", "users": "user",
}

func (fx *fixture) find(typ, id string) obj {
	c := fx.coll(typ)
	if c == nil {
		return nil
	}
	for _, r := range *c {
		if fmt.Sprint(r["id"]) == id {
			return r
		}
		if typ == "products" && strings.EqualFold(fmt.Sprint(r["reference_prefix"]), id) {
			return r
		}
		if rn, ok := r["reference_num"].(string); ok && strings.EqualFold(rn, id) {
			return r
		}
	}
	return nil
}

func refOf(r obj) string {
	if rn, ok := r["reference_num"].(string); ok {
		return rn
	}
	if rp, ok := r["reference_prefix"].(string); ok {
		return rp
	}
	return fmt.Sprint(r["id"])
}

// public strips the hidden members and adds custom_fields when wanted.
func public(r obj, withCF bool) obj {
	out := obj{}
	for k, v := range r {
		if strings.HasPrefix(k, "_") {
			continue
		}
		out[k] = v
	}
	if withCF {
		if e, ok := r["_cf"].([]any); ok {
			out["custom_fields"] = e
		} else {
			out["custom_fields"] = []any{}
		}
	}
	return out
}

func project(r obj, fields string, omitCF bool) obj {
	if fields == "" {
		return public(r, false)
	}
	full := public(r, true)
	out := obj{"id": full["id"]}
	for _, f := range strings.Split(fields, ",") {
		f = strings.TrimSpace(f)
		if f == "custom_fields" && omitCF {
			continue
		}
		if v, ok := full[f]; ok {
			out[f] = v
		}
	}
	return out
}

func (fx *fixture) list(member string, recs []obj, q url.Values, omitCF bool) (int, any) {
	if s := q.Get("q"); s != "" {
		var kept []obj
		for _, r := range recs {
			if strings.Contains(strings.ToLower(fmt.Sprint(r["name"])), strings.ToLower(s)) {
				kept = append(kept, r)
			}
		}
		recs = kept
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	per, _ := strconv.Atoi(q.Get("per_page"))
	if per < 1 {
		per = 30
	}
	total := len(recs)
	pages := (total + per - 1) / per
	out := []any{}
	for i := (page - 1) * per; i < total && i < page*per; i++ {
		out = append(out, project(recs[i], q.Get("fields"), omitCF))
	}
	return 200, obj{member: out, "pagination": obj{"total_records": total, "total_pages": pages, "current_page": page}}
}

func (fx *fixture) filter(typ string, keep func(obj) bool) []obj {
	var out []obj
	for _, r := range *fx.coll(typ) {
		if keep(r) {
			out = append(out, r)
		}
	}
	return out
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func nested(r obj, a, b string) string {
	if m, ok := r[a].(obj); ok {
		return str(m[b])
	}
	return ""
}

func notFound() (int, any) { return 404, obj{"error": "Record not found"} }

var recordTypes = map[string]bool{"releases": true, "features": true, "epics": true, "requirements": true,
	"initiatives": true, "goals": true, "ideas": true, "comments": true, "products": true}

func (fx *fixture) route(method, path string, q url.Values, body []byte, omitCF bool) (int, any) {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	switch method {
	case "GET":
		return fx.get(segs, q, omitCF)
	case "POST":
		return fx.create(segs, body)
	case "PUT":
		if len(segs) == 2 && recordTypes[segs[0]] {
			return fx.update(segs[0], segs[1], body)
		}
	case "DELETE":
		return fx.remove(segs)
	}
	return notFound()
}

func (fx *fixture) get(segs []string, q url.Values, omitCF bool) (int, any) {
	switch {
	case len(segs) == 1 && segs[0] == "me":
		return 200, obj{"user": userObj(idTest)}
	case len(segs) == 1 && segs[0] == "users":
		return fx.list("users", fx.users, q, omitCF)
	case len(segs) == 1 && segs[0] == "products":
		return fx.list("products", fx.products, q, omitCF)
	case len(segs) == 1 && segs[0] == "custom_field_definitions":
		defs := []any{}
		for _, d := range fx.defs {
			defs = append(defs, d)
		}
		return 200, obj{"custom_field_definitions": defs}
	case len(segs) == 3 && segs[0] == "custom_field_definitions" && segs[2] == "options":
		opts, ok := fx.options[segs[1]]
		if !ok {
			if fx.findDef(segs[1]) == nil {
				return notFound()
			}
		}
		list := []any{}
		for i, o := range opts {
			list = append(list, obj{"id": fmt.Sprintf("%s%02d", segs[1], i), "text": o})
		}
		return 200, obj{"options": list}
	case len(segs) == 1 && (segs[0] == "features" || segs[0] == "epics" || segs[0] == "initiatives" || segs[0] == "goals" || segs[0] == "ideas"):
		return fx.list(segs[0], *fx.coll(segs[0]), q, omitCF)
	case len(segs) == 2 && recordTypes[segs[0]]:
		r := fx.find(segs[0], segs[1])
		if r == nil {
			return notFound()
		}
		return 200, obj{singularOf[segs[0]]: public(r, segs[0] != "comments")}
	case len(segs) == 3 && recordTypes[segs[0]]:
		parent := fx.find(segs[0], segs[1])
		if parent == nil {
			return notFound()
		}
		ref := refOf(parent)
		child := segs[2]
		if segs[0] == "products" && child == "workflows" {
			var wfs []any
			for _, id := range []string{"401", "402", "403", "404", "405"} {
				var sts []any
				for i, s := range workflowStatuses[id] {
					sts = append(sts, obj{"id": fmt.Sprintf("%s%02d", id, i+1), "name": s, "position": i + 1, "complete": s == "Shipped" || s == "Done" || s == "Achieved"})
				}
				n, _ := strconv.Atoi(id)
				st := map[string]string{"401": "Feature", "402": "Release", "403": "Initiative", "404": "StrategicImperative", "405": "Ideas::Idea"}[id]
				wfs = append(wfs, obj{"id": n, "name": "Workflow " + id, "statusable_type": st, "workflow_statuses": sts})
			}
			return 200, obj{"workflows": wfs}
		}
		if fx.coll(child) == nil {
			return notFound()
		}
		var keep func(obj) bool
		switch {
		case child == "comments":
			keep = func(r obj) bool { return strings.EqualFold(str(r["_parent"]), ref) }
		case segs[0] == "products":
			keep = func(r obj) bool { return r["_product"] == ref }
		case segs[0] == "releases" && child == "features":
			keep = func(r obj) bool { return strings.EqualFold(str(r["release_reference_num"]), ref) }
		case segs[0] == "releases" && child == "epics":
			keep = func(r obj) bool { return strings.EqualFold(nested(r, "release", "reference_num"), ref) }
		case segs[0] == "initiatives" && (child == "features" || child == "epics"):
			keep = func(r obj) bool { return strings.EqualFold(str(r["initiative_reference_num"]), ref) }
		case segs[0] == "epics" && child == "features":
			keep = func(r obj) bool { return strings.EqualFold(str(r["epic_reference_num"]), ref) }
		case segs[0] == "features" && child == "requirements":
			keep = func(r obj) bool { return strings.EqualFold(str(r["_feature"]), ref) }
		default:
			return notFound()
		}
		return fx.list(child, fx.filter(child, keep), q, omitCF)
	}
	return notFound()
}

func (fx *fixture) findDef(id string) obj {
	for _, d := range fx.defs {
		if d["id"] == id {
			return d
		}
	}
	return nil
}

func (fx *fixture) newID() string {
	fx.nextID++
	return strconv.FormatInt(fx.nextID, 10)
}

func nextRef(recs []obj, prefix string) string {
	max := 0
	for _, r := range recs {
		rn := str(r["reference_num"])
		if strings.HasPrefix(strings.ToUpper(rn), strings.ToUpper(prefix)) {
			if n, err := strconv.Atoi(rn[len(prefix):]); err == nil && n > max {
				max = n
			}
		}
	}
	return fmt.Sprintf("%s%d", prefix, max+1)
}

func (fx *fixture) productOf(r obj) obj {
	return fx.find("products", str(r["_product"]))
}

var infix = map[string]string{"releases": "R-", "epics": "E-", "initiatives": "S-", "goals": "G-", "ideas": "I-", "features": ""}

func (fx *fixture) create(segs []string, body []byte) (int, any) {
	if len(segs) != 3 {
		return notFound()
	}
	parent := fx.find(segs[0], segs[1])
	if parent == nil || !recordTypes[segs[0]] {
		return notFound()
	}
	child := segs[2]
	sing := singularOf[child]
	var wrapper map[string]json.RawMessage
	if json.Unmarshal(body, &wrapper) != nil || wrapper[sing] == nil {
		return 400, obj{"error": "Required parameter missing: " + sing}
	}
	var members obj
	if json.Unmarshal(wrapper[sing], &members) != nil {
		return 400, obj{"error": "Bad request"}
	}
	product := fx.productOf(parent)
	if segs[0] == "products" {
		product = parent
	}
	key := str(product["reference_prefix"])
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	rec := obj{"id": fx.newID(), "created_at": now, "updated_at": now, "_product": key, "product_id": product["id"], "description": obj{"body": ""}}
	switch {
	case child == "comments":
		rec = obj{"id": rec["id"], "body": "", "user": userObj(idTest), "created_at": now,
			"commentable": obj{"id": parent["id"], "url": "https://example.aha.io/" + segs[0] + "/" + refOf(parent)},
			"url":         "https://example.aha.io/comments/" + str(rec["id"]), "_product": key, "_parent": refOf(parent)}
	case child == "requirements":
		rec["reference_num"] = nextRef(fx.requirements, refOf(parent)+"-")
		rec["feature"] = obj{"id": parent["id"], "reference_num": refOf(parent), "name": parent["name"]}
		rec["_feature"] = refOf(parent)
		rec["workflow_status"] = statusObj(child, workflowStatuses["401"][0])
		delete(rec, "product_id")
	default:
		pre, ok := infix[child]
		if !ok {
			return notFound()
		}
		rec["reference_num"] = nextRef(*fx.coll(child), key+"-"+pre)
		rec["workflow_status"] = statusObj(child, workflowStatuses[workflowOfType[child]][0])
		if segs[0] == "releases" {
			if child == "epics" {
				rec["release"] = obj{"id": parent["id"], "reference_num": refOf(parent), "name": parent["name"]}
			} else {
				rec["release_reference_num"] = refOf(parent)
			}
		}
		if child == "features" {
			rec["tags"] = []any{}
		}
		if child == "releases" {
			rec["theme"] = obj{"body": ""}
			delete(rec, "description")
		}
	}
	if child == "goals" {
		sm, ok := members["success_metric"].(obj)
		if !ok {
			return 400, obj{"error": "Required parameter missing: success_metric"}
		}
		if strings.TrimSpace(str(sm["name"])) == "" {
			return 400, obj{"error": "Validation failed", "errors": obj{"message": "Success metric cannot be blank"}}
		}
		st := workflowStatuses["404"][0]
		if s := str(sm["workflow_status"]); s != "" {
			st = s
		}
		rec["success_metric"] = obj{"name": sm["name"], "workflow_status": statusObj("goals", st)}
		delete(members, "success_metric")
		delete(rec, "workflow_status")
	}
	if code, msg := fx.apply(child, rec, members); code != 0 {
		return code, msg
	}
	if child != "comments" {
		rec["url"] = "https://example.aha.io/" + child + "/" + refOf(rec)
	}
	c := fx.coll(child)
	*c = append(*c, rec)
	return 200, obj{sing: public(rec, child != "comments")}
}

// apply writes the members of a request body into rec.
func (fx *fixture) apply(typ string, rec obj, members obj) (int, any) {
	for k, v := range members {
		switch k {
		case "workflow_status":
			s := str(v)
			if typ == "goals" {
				if sm, ok := rec["success_metric"].(obj); ok {
					sm["workflow_status"] = statusObj("goals", s)
				}
				continue
			}
			rec["workflow_status"] = statusObj(typ, s)
		case "assigned_to_user":
			var u obj
			for _, cand := range fx.users {
				if cand["id"] == str(v) || strings.EqualFold(str(cand["email"]), str(v)) {
					u = cand
				}
			}
			if u == nil {
				return 400, obj{"error": "Unknown user"}
			}
			rec["assigned_to_user"] = u
		case "release":
			r := fx.find("releases", str(v))
			if r == nil {
				return notFound()
			}
			if typ == "epics" {
				rec["release"] = obj{"id": r["id"], "reference_num": r["reference_num"], "name": r["name"]}
			} else {
				rec["release_reference_num"] = r["reference_num"]
			}
		case "initiative":
			r := fx.find("initiatives", str(v))
			if r == nil {
				return notFound()
			}
			rec["initiative_reference_num"] = r["reference_num"]
		case "tags":
			list := []any{}
			for _, p := range strings.Split(str(v), ",") {
				if p = strings.TrimSpace(p); p != "" {
					list = append(list, p)
				}
			}
			rec["tags"] = list
		case "description", "theme":
			rec[k] = obj{"id": fx.newID(), "body": str(v)}
		case "success_metric":
			// Aha! ignores the success metric in an update.
		case "custom_fields":
			m, ok := v.(obj)
			if !ok {
				return 400, obj{"error": "custom_fields must be an object"}
			}
			keys := make([]string, 0, len(m))
			for key := range m {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				setEntry(fx, rec, key, m[key])
			}
		default:
			rec[k] = v
		}
	}
	return 0, nil
}

func (fx *fixture) update(typ, id string, body []byte) (int, any) {
	rec := fx.find(typ, id)
	if rec == nil {
		return notFound()
	}
	sing := singularOf[typ]
	var wrapper map[string]json.RawMessage
	if json.Unmarshal(body, &wrapper) != nil || wrapper[sing] == nil {
		return 400, obj{"error": "Required parameter missing: " + sing}
	}
	var members obj
	if json.Unmarshal(wrapper[sing], &members) != nil {
		return 400, obj{"error": "Bad request"}
	}
	if code, msg := fx.apply(typ, rec, members); code != 0 {
		return code, msg
	}
	rec["updated_at"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	return 200, obj{sing: public(rec, typ != "comments")}
}

func (fx *fixture) remove(segs []string) (int, any) {
	typ, id := "", ""
	switch {
	case len(segs) == 2 && recordTypes[segs[0]]:
		typ, id = segs[0], segs[1]
	case len(segs) == 4 && segs[0] == "products":
		if fx.find("products", segs[1]) == nil {
			return notFound()
		}
		typ, id = segs[2], segs[3]
	default:
		return notFound()
	}
	c := fx.coll(typ)
	if c == nil {
		return notFound()
	}
	for i, r := range *c {
		if fmt.Sprint(r["id"]) == id || strings.EqualFold(str(r["reference_num"]), id) {
			*c = append((*c)[:i], (*c)[i+1:]...)
			return 204, nil
		}
	}
	return notFound()
}

// readLines is a helper for tests that parse a stream line by line.
func readLines(r io.Reader, out chan<- string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for sc.Scan() {
		out <- sc.Text()
	}
	close(out)
}
