// generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

package main

// aha-request: sends one operation of the AhaAPI as an HTTPS request and
// reads the answer. Every request of the server to Aha! goes through here.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Clock is the clock of the waiting after status 429 and of the log line of
// execute. Tests of the package may replace it.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time        { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }

var clock Clock = realClock{}

const requestTimeout = 60 * time.Second

// ApiFailure kinds.
const (
	failNotFound      = "not_found"
	failNotAuthorized = "not_authorized"
	failRateLimited   = "rate_limited"
	failUnreachable   = "unreachable"
	failRejected      = "rejected"
)

type ApiFailure struct {
	Kind   string
	Status int
	Cause  string // no_answer, certificate, proxy, connection
	Text   string
}

type connError struct {
	cause string
	err   error
}

func (e *connError) Error() string { return e.cause + ": " + e.err.Error() }
func (e *connError) Unwrap() error { return e.err }

// proxyFor is the choice of the proxy for an address: a function of the
// environment and the address. The standard library reads HTTPS_PROXY,
// https_proxy, NO_PROXY and no_proxy.
var proxyFor = func(addr string) (*url.URL, error) {
	return http.ProxyFromEnvironment(&http.Request{URL: &url.URL{Scheme: "https", Host: addr}})
}

// dialTLS opens the connection to Aha!: directly, or through a CONNECT
// tunnel of the proxy of the environment, and always with TLS that checks
// the certificate against the trust store of the platform.
func dialTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, &connError{"connection", err}
	}
	proxy, err := proxyFor(addr)
	if err != nil {
		return nil, &connError{"proxy", err}
	}
	d := &net.Dialer{Timeout: requestTimeout}
	var conn net.Conn
	if proxy != nil {
		conn, err = dialProxy(ctx, d, proxy, addr)
		if err != nil {
			return nil, err
		}
	} else {
		conn, err = d.DialContext(ctx, network, addr)
		if err != nil {
			if ctx.Err() != nil {
				return nil, &connError{"no_answer", err}
			}
			return nil, &connError{"connection", err}
		}
	}
	tc := tls.Client(conn, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err := tc.HandshakeContext(ctx); err != nil {
		conn.Close()
		var cv *tls.CertificateVerificationError
		var ua x509.UnknownAuthorityError
		var he x509.HostnameError
		var ci x509.CertificateInvalidError
		switch {
		case errors.As(err, &cv), errors.As(err, &ua), errors.As(err, &he), errors.As(err, &ci):
			return nil, &connError{"certificate", err}
		case ctx.Err() != nil:
			return nil, &connError{"no_answer", err}
		}
		return nil, &connError{"connection", err}
	}
	return tc, nil
}

func dialProxy(ctx context.Context, d *net.Dialer, proxy *url.URL, addr string) (net.Conn, error) {
	phost := proxy.Host
	if proxy.Port() == "" {
		if proxy.Scheme == "https" {
			phost = net.JoinHostPort(proxy.Hostname(), "443")
		} else {
			phost = net.JoinHostPort(proxy.Hostname(), "80")
		}
	}
	conn, err := d.DialContext(ctx, "tcp", phost)
	if err != nil {
		return nil, &connError{"proxy", err}
	}
	if proxy.Scheme == "https" {
		tc := tls.Client(conn, &tls.Config{ServerName: proxy.Hostname(), MinVersion: tls.VersionTLS12})
		if err := tc.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, &connError{"proxy", err}
		}
		conn = tc
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	var req bytes.Buffer
	fmt.Fprintf(&req, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
	if proxy.User != nil {
		pw, _ := proxy.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(proxy.User.Username() + ":" + pw))
		fmt.Fprintf(&req, "Proxy-Authorization: Basic %s\r\n", auth)
	}
	req.WriteString("\r\n")
	if _, err := conn.Write(req.Bytes()); err != nil {
		conn.Close()
		return nil, &connError{"proxy", err}
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, &connError{"proxy", err}
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		conn.Close()
		return nil, &connError{"proxy", fmt.Errorf("status %d", resp.StatusCode)}
	}
	_ = conn.SetDeadline(time.Time{})
	if br.Buffered() > 0 {
		conn.Close()
		return nil, &connError{"proxy", errors.New("data after the answer to CONNECT")}
	}
	return conn, nil
}

var httpClient = &http.Client{
	Timeout: requestTimeout,
	Transport: &http.Transport{
		Proxy:               nil,
		DialTLSContext:      dialTLS,
		TLSHandshakeTimeout: requestTimeout,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     30 * time.Second,
	},
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// ApiAnswer is a JSON object as Aha! returns it, decoded with json.Number.
type ApiAnswer = map[string]any

// apiCall sends one request. expect is the member that the answer must
// have; list says that its value is a list. A delete needs no answer.
func apiCall(acct Account, method, route string, params url.Values, body []byte, expect string, list bool) (ApiAnswer, *ApiFailure) {
	addr := "https://" + acct.Subdomain + ".aha.io/api/v1" + route
	if len(params) > 0 {
		addr += "?" + params.Encode()
	}
	for attempt := 0; ; attempt++ {
		var rd io.Reader
		if body != nil {
			rd = bytes.NewReader(body)
		}
		req, err := http.NewRequest(method, addr, rd)
		if err != nil {
			return nil, &ApiFailure{Kind: failUnreachable, Cause: "connection"}
		}
		req.Header.Set("Authorization", "Bearer "+acct.Credential)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "mcp-server-aha-aql/"+serverVersion+" (https://github.com/mge1512/aha-aql)")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, &ApiFailure{Kind: failUnreachable, Cause: causeOf(err)}
		}
		data, rerr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if rerr != nil {
			return nil, &ApiFailure{Kind: failUnreachable, Cause: causeOf(rerr)}
		}
		st := resp.StatusCode
		switch {
		case st == 429:
			if attempt >= 2 {
				return nil, &ApiFailure{Kind: failRateLimited, Status: 429}
			}
			clock.Sleep(rateLimitWait(resp.Header.Get("X-Ratelimit-Reset")))
			continue
		case st >= 300 && st <= 399:
			return nil, &ApiFailure{Kind: failRejected, Status: st, Text: string(data)}
		case st == 401 || st == 403:
			return nil, &ApiFailure{Kind: failNotAuthorized, Status: st}
		case st == 404:
			return nil, &ApiFailure{Kind: failNotFound, Status: 404, Text: string(data)}
		case st < 200 || st > 299:
			return nil, &ApiFailure{Kind: failRejected, Status: st, Text: string(data)}
		}
		if method == http.MethodDelete {
			return ApiAnswer{}, nil
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, &ApiFailure{Kind: failRejected, Status: st, Text: "the answer is not JSON"}
		}
		obj, ok := v.(map[string]any)
		if !ok {
			return nil, &ApiFailure{Kind: failRejected, Status: st, Text: "the answer has an unknown form"}
		}
		m, has := obj[expect]
		if !has {
			return nil, &ApiFailure{Kind: failRejected, Status: st, Text: "the answer has an unknown form"}
		}
		if list {
			if _, ok := m.([]any); !ok {
				return nil, &ApiFailure{Kind: failRejected, Status: st, Text: "the answer has an unknown form"}
			}
			if p, has := obj["pagination"]; has {
				if _, ok := p.(map[string]any); !ok {
					return nil, &ApiFailure{Kind: failRejected, Status: st, Text: "the answer has an unknown form"}
				}
			}
		} else if _, ok := m.(map[string]any); !ok {
			return nil, &ApiFailure{Kind: failRejected, Status: st, Text: "the answer has an unknown form"}
		}
		return obj, nil
	}
}

func causeOf(err error) string {
	var ce *connError
	if errors.As(err, &ce) {
		return ce.cause
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "no_answer"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "no_answer"
	}
	return "connection"
}

func rateLimitWait(reset string) time.Duration {
	if reset == "" {
		return 5 * time.Second
	}
	r, err := strconv.ParseInt(strings.TrimSpace(reset), 10, 64)
	if err != nil {
		return 5 * time.Second
	}
	secs := r - clock.Now().Unix()
	if secs < 1 {
		secs = 1
	}
	if secs > 30 {
		secs = 30
	}
	return time.Duration(secs) * time.Second
}

// ListPage is one page of a list.
type ListPage struct {
	Records      []map[string]any
	TotalRecords int
	TotalPages   int
}

func lastPart(route string) string {
	return route[strings.LastIndex(route, "/")+1:]
}

func apiList(acct Account, route string, params url.Values) (*ListPage, *ApiFailure) {
	ans, f := apiCall(acct, http.MethodGet, route, params, nil, lastPart(route), true)
	if f != nil {
		return nil, f
	}
	lp := &ListPage{}
	for _, x := range ans[lastPart(route)].([]any) {
		if m, ok := x.(map[string]any); ok {
			lp.Records = append(lp.Records, m)
		}
	}
	if p, ok := ans["pagination"].(map[string]any); ok {
		lp.TotalRecords = toInt(p["total_records"])
		lp.TotalPages = toInt(p["total_pages"])
	} else {
		lp.TotalRecords = len(lp.Records)
		lp.TotalPages = 1
	}
	return lp, nil
}

func apiGet(acct Account, route, member string, list bool) (ApiAnswer, *ApiFailure) {
	return apiCall(acct, http.MethodGet, route, nil, nil, member, list)
}

func toInt(v any) int {
	switch x := v.(type) {
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			f, _ := x.Float64()
			return int(f)
		}
		return int(n)
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

// routeWith replaces {v}, {id} and {product} by percent-encoded values.
func routeWith(tmpl string, pairs ...string) string {
	out := tmpl
	for i := 0; i+1 < len(pairs); i += 2 {
		out = strings.ReplaceAll(out, "{"+pairs[i]+"}", url.PathEscape(pairs[i+1]))
	}
	return out
}
