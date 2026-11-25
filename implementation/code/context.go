// generated from spec: aha-aql.spec.md sha256:caab507ed82ebcb6a9f2c7ea6c55bfadfed1208642edbdd0229d95aee417348e

package main

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Product is a product of the account (TYPES: Product).
type Product struct {
	Key  string
	ID   string
	Name string
	rec  apiRec
}

// exampleKey holds the ExampleKey (choose-example-key). In stdio mode and
// manual mode there is one for the process, in http mode one per request.
type exampleKey struct {
	mu             sync.Mutex
	chosen         bool
	key            string
	defaultProduct string
	warn           func(string)
}

// stmtCtx is the state of one statement, or of one answer to the tool
// list. Nothing in it is kept after the statement.
type stmtCtx struct {
	ctx  context.Context
	acct Account
	c    *ahaClient
	ek   *exampleKey

	products     []*Product
	prodLoaded   bool
	prodHasCF    bool
	prodHasDesc  bool
	defsLoaded   bool
	defs         []cfDef
	defsStatus   int // status of NotAuthorized or NotFound for the definitions
	options      map[string][]string
	me           apiRec
	productRecs  map[string]apiRec
	workflowList map[string][]apiRec
	custom       map[string]*CustomField // custom fields resolved in the statement
}

func newStmt(ctx context.Context, acct Account, c *ahaClient, ek *exampleKey) *stmtCtx {
	return &stmtCtx{ctx: ctx, acct: acct, c: c, ek: ek, options: map[string][]string{},
		productRecs: map[string]apiRec{}, workflowList: map[string][]apiRec{}, custom: map[string]*CustomField{}}
}

// key gives the ExampleKey; it is chosen when it is first needed.
func (sc *stmtCtx) key() string {
	ek := sc.ek
	ek.mu.Lock()
	defer ek.mu.Unlock()
	if ek.chosen {
		return ek.key
	}
	ek.chosen = true
	prods, err := sc.productList(false, false)
	if err != nil {
		ek.key = "PROD"
		ek.warn("warning: the product list could not be read. Examples use the key PROD")
		return ek.key
	}
	if ek.defaultProduct != "" {
		if p := matchProduct(prods, ek.defaultProduct); p != nil {
			ek.key = p.Key
			return ek.key
		}
	}
	k := "PROD"
	for {
		taken := false
		for _, p := range prods {
			if strings.EqualFold(p.Key, k) {
				taken = true
			}
		}
		if !taken {
			break
		}
		k += "X"
	}
	ek.key = k
	if ek.defaultProduct != "" {
		ek.warn("warning: default-product names no product of the account. Examples use the key " + k)
	}
	return ek.key
}

// productList reads the product list of the statement, all pages, at most
// once (resolve-product, step 1).
func (sc *stmtCtx) productList(desc, cf bool) ([]*Product, *AqlError) {
	if sc.prodLoaded && (!desc || sc.prodHasDesc) && (!cf || sc.prodHasCF) {
		return sc.products, nil
	}
	pe := entityByName("products")
	used := map[string]bool{"description": desc}
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(pageSize))
	params.Set("fields", sourceMembers(pe, used, cf))
	var out []*Product
	for page := 1; ; page++ {
		params.Set("page", strconv.Itoa(page))
		lp, f := sc.c.apiList(sc.ctx, sc.acct, "/products", params)
		if f != nil {
			return nil, sc.failErr(f)
		}
		if cf && page == 1 && len(lp.records) > 0 && !anyHasMember(lp.records, "custom_fields") {
			return nil, sc.errX05(pe)
		}
		for _, r := range lp.records {
			out = append(out, &Product{Key: strOf(r["reference_prefix"]), ID: strOf(r["id"]), Name: strOf(r["name"]), rec: r})
		}
		if page >= lp.totalPages || page >= 1000 {
			break
		}
	}
	sc.products, sc.prodLoaded, sc.prodHasDesc, sc.prodHasCF = out, true, desc, cf
	return out, nil
}

func anyHasMember(recs []apiRec, m string) bool {
	for _, r := range recs {
		if _, ok := r[m]; ok {
			return true
		}
	}
	return false
}

func (sc *stmtCtx) errX05(e *EntityDef) *AqlError {
	return aqlErr("api_error", "Aha! returned this list of %s without custom fields. A record that is read by its id has them: add %s = '...' to WHERE.", e.Name, e.idField())
}

// matchProduct compares a value with the key and the NumericId of every
// product, then with the name of every product (resolve-product, step 2).
func matchProduct(prods []*Product, value string) *Product {
	for _, p := range prods {
		if textEq(p.Key, value) || textEq(p.ID, value) {
			return p
		}
	}
	for _, p := range prods {
		if textEq(p.Name, value) {
			return p
		}
	}
	return nil
}

func (sc *stmtCtx) productByID(id string) *Product {
	for _, p := range sc.products {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// findProductQuiet runs resolve-product without a message for an unknown value.
func (sc *stmtCtx) findProductQuiet(value string) (*Product, *AqlError) {
	prods, err := sc.productList(false, false)
	if err != nil {
		return nil, err
	}
	return matchProduct(prods, value), nil
}

// resolveProduct is resolve-product.
func (sc *stmtCtx) resolveProduct(value string, e *EntityDef, hints bool) (*Product, *AqlError) {
	prods, err := sc.productList(false, false)
	if err != nil {
		return nil, err
	}
	if p := matchProduct(prods, value); p != nil {
		return p, nil
	}
	if hints {
		up := strings.ToUpper(value)
		kind := refKind(value)
		if kind != "" && e != nil && e.hasParentFilter(kind) {
			return nil, aqlErr("wrong_kind", "%s is %s %s, not a product. Use: WHERE %s = '%s'", quote(value), article(kind), kind, kind, up)
		}
		if kind != "" && e != nil && kind == e.Kind {
			return nil, aqlErr("wrong_kind", "%s is the id of %s %s, not a product. Use: WHERE id = '%s'", quote(value), article(kind), kind, up)
		}
		if kind != "" {
			pre := up[:strings.Index(up, "-")]
			for _, p := range prods {
				if strings.EqualFold(p.Key, pre) {
					return nil, aqlErr("wrong_kind", "%s is %s %s, not a product. Its product is %s: WHERE product = '%s'", quote(value), article(kind), kind, p.Key, p.Key)
				}
			}
		}
	}
	var keys []string
	for _, p := range prods {
		keys = append(keys, p.Key)
	}
	return nil, errP01(value, keys)
}

// currentUser reads /me at most once per statement.
func (sc *stmtCtx) currentUser() (apiRec, *AqlError) {
	if sc.me != nil {
		return sc.me, nil
	}
	r, f := sc.c.apiGet(sc.ctx, sc.acct, "/me", "user")
	if f != nil {
		return nil, sc.failErr(f)
	}
	sc.me = r
	return r, nil
}

// productRecord reads /products/{id} at most once per statement.
func (sc *stmtCtx) productRecord(id string) (apiRec, *apiFailure) {
	if r, ok := sc.productRecs[id]; ok {
		return r, nil
	}
	r, f := sc.c.apiGet(sc.ctx, sc.acct, "/products/"+esc(id), "product")
	if f != nil {
		return nil, f
	}
	sc.productRecs[id] = r
	return r, nil
}

func (sc *stmtCtx) workflows(id string) ([]apiRec, *apiFailure) {
	if w, ok := sc.workflowList[id]; ok {
		return w, nil
	}
	w, f := sc.c.apiGetList(sc.ctx, sc.acct, "/products/"+esc(id)+"/workflows", "workflows")
	if f != nil {
		return nil, f
	}
	sc.workflowList[id] = w
	return w, nil
}

// statusGroup is the statuses of one entity in one product.
type statusGroup struct {
	entity string // value of the field entity
	prefix string // the part of the member name before _workflow
	names  []apiRec
}

// productStatuses gives the status records of a product, in the order of
// the entity statuses.
func (sc *stmtCtx) productStatuses(id string) ([]statusGroup, *AqlError) {
	prec, f := sc.productRecord(id)
	if f != nil {
		return nil, sc.failErr(f)
	}
	wfs, f := sc.workflows(id)
	if f != nil {
		return nil, sc.failErr(f)
	}
	find := func(member string) []apiRec {
		m, ok := prec[member].(map[string]any)
		if !ok {
			return nil
		}
		for _, w := range wfs {
			if sameDigits(w["id"], m["id"]) {
				var out []apiRec
				if list, ok := w["workflow_statuses"].([]any); ok {
					for _, s := range list {
						if sm, ok := s.(map[string]any); ok {
							out = append(out, sm)
						}
					}
				}
				if out == nil {
					out = []apiRec{}
				}
				return out
			}
		}
		return nil
	}
	var groups []statusGroup
	known := map[string]bool{}
	for _, wm := range workflowMembers {
		known[wm.member] = true
		if st := find(wm.member); st != nil {
			groups = append(groups, statusGroup{entity: wm.entity, prefix: strings.TrimSuffix(wm.member, "_workflow"), names: st})
		}
	}
	var others []string
	for k := range prec {
		if strings.HasSuffix(k, "_workflow") && !known[k] {
			others = append(others, k)
		}
	}
	sortStrings(others)
	for _, k := range others {
		if st := find(k); st != nil {
			p := strings.TrimSuffix(k, "_workflow")
			groups = append(groups, statusGroup{entity: p, prefix: p, names: st})
		}
	}
	return groups, nil
}
