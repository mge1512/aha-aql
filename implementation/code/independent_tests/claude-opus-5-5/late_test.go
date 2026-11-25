// generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30
// tests by: claude-opus-5-5

// This file was added after the implementation, when a coverage audit of
// the EXAMPLES found describe_takes_nothing_else without a test. See the
// Test Refinements table of TRANSLATION_REPORT.md.
package aqltests

import "testing"

func TestDescribeTakesNothingElse(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	s := e.stdio(nil)
	wantError(t, s.query(`DESCRIBE features WHERE product = 'DEMO'`),
		`DESCRIBE features lists the fields of features and takes nothing else. To read records: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`)
	e.wantNoCall()
}
