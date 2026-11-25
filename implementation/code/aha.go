// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Account is what one tool call runs with (TYPES: Account).
type Account struct {
	Subdomain  string
	Credential string
}

type failKind int

const (
	fNotFound failKind = iota
	fNotAuthorized
	fRateLimited
	fUnreachable
	fRejected
)

// apiFailure is an ApiFailure of the AhaAPI interface.
type apiFailure struct {
	kind   failKind
	status int
	text   string
	cause  string // no_answer | certificate | proxy | connection
	method string
	route  string
	write  bool
}

const requestTimeout = 60 * time.Second

// ahaClient sends the operations of the AhaAPI (aha-request).
type ahaClient struct {
	hc *http.Client
}

func newAhaClient() *ahaClient {
	tr := &http.Transport{
		// MECHANISM: the standard library reads HTTPS_PROXY, https_proxy,
		// NO_PROXY and no_proxy for an https address.
		Proxy:               http.ProxyFromEnvironment,
		DisableKeepAlives:   true,
		TLSHandshakeTimeout: requestTimeout,
		DialContext:         (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
		// The trust store of the platform; on Linux SSL_CERT_FILE and
		// SSL_CERT_DIR can name it. No setting turns the check off.
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &ahaClient{hc: &http.Client{
		Transport: tr,
		Timeout:   requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func isCertError(err error) bool {
	var a *tls.CertificateVerificationError
	var b x509.UnknownAuthorityError
	var c x509.HostnameError
	var d x509.CertificateInvalidError
	var e *x509.UnknownAuthorityError
	var g *x509.HostnameError
	var h *x509.CertificateInvalidError
	return errors.As(err, &a) || errors.As(err, &b) || errors.As(err, &c) || errors.As(err, &d) ||
		errors.As(err, &e) || errors.As(err, &g) || errors.As(err, &h)
}

func isTimeout(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded)
}

// do sends one request and reads the answer (aha-request, steps 1 to 10).
// The answer is the JSON value of the body, or nil for a DELETE.
func (c *ahaClient) do(ctx context.Context, acct Account, method, route string, params url.Values, body []byte) (any, *apiFailure) {
	u := "https://" + acct.Subdomain + ".aha.io/api/v1" + route
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	write := method != http.MethodGet
	fail := func(k failKind, status int, text, cause string) *apiFailure {
		return &apiFailure{kind: k, status: status, text: text, cause: cause, method: method, route: route, write: write}
	}
	for attempt := 0; ; attempt++ {
		var rd io.Reader
		if body != nil {
			rd = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, rd)
		if err != nil {
			return nil, fail(fUnreachable, 0, "", "connection")
		}
		req.Header.Set("Authorization", "Bearer "+acct.Credential)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "mcp-server-aha-aql/"+serverVersion+" (https://github.com/mge1512/aha-aql)")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		var connected atomic.Bool
		trace := &httptrace.ClientTrace{GotConn: func(httptrace.GotConnInfo) { connected.Store(true) }}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		proxyURL, _ := http.ProxyFromEnvironment(req)
		resp, err := c.hc.Do(req)
		if err != nil {
			return nil, fail(fUnreachable, 0, "", classify(err, proxyURL != nil, connected.Load()))
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if err != nil {
			cause := "connection"
			if isTimeout(err) {
				cause = "no_answer"
			}
			return nil, fail(fUnreachable, 0, "", cause)
		}
		st := resp.StatusCode
		switch {
		case st == http.StatusTooManyRequests:
			if attempt >= 2 {
				return nil, fail(fRateLimited, st, "", "")
			}
			wait := 5 * time.Second
			if h := resp.Header.Get("X-Ratelimit-Reset"); h != "" {
				if reset, err := strconv.ParseFloat(strings.TrimSpace(h), 64); err == nil {
					wait = time.Duration((reset - float64(time.Now().UnixNano())/1e9) * float64(time.Second))
					if wait < time.Second {
						wait = time.Second
					}
					if wait > 30*time.Second {
						wait = 30 * time.Second
					}
				}
			}
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, fail(fUnreachable, 0, "", "no_answer")
			}
			continue
		case st >= 300 && st <= 399:
			return nil, fail(fRejected, st, string(data), "")
		case st == 401 || st == 403:
			return nil, fail(fNotAuthorized, st, "", "")
		case st == 404:
			return nil, fail(fNotFound, st, string(data), "")
		case st < 200 || st > 299:
			return nil, fail(fRejected, st, string(data), "")
		}
		if method == http.MethodDelete {
			return nil, nil
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, fail(fRejected, st, "the answer is not JSON", "")
		}
		if dec.More() {
			return nil, fail(fRejected, st, "the answer is not JSON", "")
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fail(fRejected, st, "the answer has an unknown form", "")
		}
		return m, nil
	}
}

func classify(err error, proxied, connected bool) string {
	if isCertError(err) {
		return "certificate"
	}
	if !connected {
		if proxied {
			return "proxy"
		}
		if isTimeout(err) {
			return "no_answer"
		}
		return "connection"
	}
	if isTimeout(err) {
		return "no_answer"
	}
	return "connection"
}

func unknownForm(f *apiFailure, st int) *apiFailure {
	return &apiFailure{kind: fRejected, status: st, text: "the answer has an unknown form", method: f.method, route: f.route, write: f.write}
}

// listPage is a ListPage of the AhaAPI.
type listPage struct {
	records      []apiRec
	totalRecords int
	totalPages   int
}

func lastSegment(route string) string {
	if i := strings.LastIndex(route, "/"); i >= 0 {
		return route[i+1:]
	}
	return route
}

func intOf(v any) (int, bool) {
	switch x := v.(type) {
	case json.Number:
		i, err := strconv.Atoi(x.String())
		if err != nil {
			f, err2 := x.Float64()
			if err2 != nil {
				return 0, false
			}
			return int(f), true
		}
		return i, true
	case float64:
		return int(x), true
	}
	return 0, false
}

// apiList is ApiList(route, parameters).
func (c *ahaClient) apiList(ctx context.Context, acct Account, route string, params url.Values) (listPage, *apiFailure) {
	v, f := c.do(ctx, acct, http.MethodGet, route, params, nil)
	if f != nil {
		return listPage{}, f
	}
	m := v.(map[string]any)
	base := &apiFailure{method: http.MethodGet, route: route}
	list, ok := m[lastSegment(route)].([]any)
	if !ok {
		return listPage{}, unknownForm(base, 200)
	}
	var lp listPage
	for _, r := range list {
		rec, ok := r.(map[string]any)
		if !ok {
			return listPage{}, unknownForm(base, 200)
		}
		lp.records = append(lp.records, rec)
	}
	if p, present := m["pagination"]; present {
		pm, ok := p.(map[string]any)
		if !ok {
			return listPage{}, unknownForm(base, 200)
		}
		tr, ok1 := intOf(pm["total_records"])
		tp, ok2 := intOf(pm["total_pages"])
		if !ok1 || !ok2 {
			return listPage{}, unknownForm(base, 200)
		}
		lp.totalRecords, lp.totalPages = tr, tp
	} else {
		lp.totalRecords, lp.totalPages = len(lp.records), 1
	}
	return lp, nil
}

// apiGet is ApiGet(route): the record in the member of its singular.
func (c *ahaClient) apiGet(ctx context.Context, acct Account, route, member string) (apiRec, *apiFailure) {
	v, f := c.do(ctx, acct, http.MethodGet, route, nil, nil)
	if f != nil {
		return nil, f
	}
	m := v.(map[string]any)
	rec, ok := m[member].(map[string]any)
	if !ok {
		return nil, unknownForm(&apiFailure{method: http.MethodGet, route: route}, 200)
	}
	return rec, nil
}

// apiGetList is ApiGet on a route whose answer has a list in member.
func (c *ahaClient) apiGetList(ctx context.Context, acct Account, route, member string) ([]apiRec, *apiFailure) {
	v, f := c.do(ctx, acct, http.MethodGet, route, nil, nil)
	if f != nil {
		return nil, f
	}
	m := v.(map[string]any)
	list, ok := m[member].([]any)
	if !ok {
		return nil, unknownForm(&apiFailure{method: http.MethodGet, route: route}, 200)
	}
	var out []apiRec
	for _, r := range list {
		rec, ok := r.(map[string]any)
		if !ok {
			return nil, unknownForm(&apiFailure{method: http.MethodGet, route: route}, 200)
		}
		out = append(out, rec)
	}
	return out, nil
}

// apiWrite is ApiCreate (POST) or ApiUpdate (PUT).
func (c *ahaClient) apiWrite(ctx context.Context, acct Account, method, route, member string, body []byte) (apiRec, *apiFailure) {
	v, f := c.do(ctx, acct, method, route, nil, body)
	if f != nil {
		return nil, f
	}
	m := v.(map[string]any)
	rec, ok := m[member].(map[string]any)
	if !ok {
		return nil, unknownForm(&apiFailure{method: method, route: route, write: true}, 200)
	}
	return rec, nil
}

// apiDelete is ApiDelete(route).
func (c *ahaClient) apiDelete(ctx context.Context, acct Account, route string) *apiFailure {
	_, f := c.do(ctx, acct, http.MethodDelete, route, nil, nil)
	return f
}

// esc percent-encodes a value that replaces a route parameter.
func esc(s string) string { return url.PathEscape(s) }
