# TRANSLATION_REPORT

| Field | Value |
|---|---|
| Spec-SHA256 | `9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262` |
| Spec-SHA256 (host) | `9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262` |
| Spec | `aha-aql.spec.md`, Version 0.13.0, Spec-Schema 0.4.0 |
| LLM-Name | `claude-opus-5-5` |
| Mode | `translator` |
| Deployment-Template | `mcp-server.template.md v0.3.24` |
| GO-FRAMEWORK | mcp-go (template default) |
| Date | 2026-11-24 |

**Included-Specs:** none. The META section of the spec declares no
`Includes:` directive. The merged spec text is the host spec file, so the
merged hash equals the host hash.

| Path | SHA256 |
|------|--------|
| (none) | |

## Translation Inputs (provenance)

- `Spec-SHA256 (merged):` `9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262`
- `Spec-SHA256 (host):` `9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262`
- `Decisions-Hints-SHA256:` none
- `Milestones-Hints-SHA256:` none
- `Template-SHA256:` `mcp-server.template.md` `22c55b355e28a6f2635413e72f2281faa3f7f1c79f79ea4eca76aba0b5b37b3e`
- `Prompt-SHA256:` `prompt.md` `702ec28de611ff65d76b2819cb53c675867fb58b1c090c6b17192cc73470d7d3`
- `Library-Hints-SHA256:` `mcp-server.go.mcp-go.hints.md` `3bcd5bb41d5c33057fe7bf9b21f0baf055755892691b057fdf0078b15ac20c82`
- `Style-Hints-SHA256:` none (no `/etc/pcd/hints/` and no `.pcd/hints/` directory exists)
- `Upgrade-Brief-SHA256:` none
- `Directive-SHA256:` none

Other files in the input directory that were read but are not translation
guidance: `ROLE.md` (`e7839aedd9096d8059dd7854cfe94a01bd492cfbce204a445d37f8306c8930cf`)
and `system-prompt.txt` (`65a41409b8abb5504b505bb3f12b316c66ae6706d8baa19b8924a595122b0e71`).

## Tests-First-Compliance

**yes.** The nine files in `independent_tests/claude-opus-5-5/` were written
before any implementation source file (`*.go` at the root, `go.mod`). The
structural guard of step 3 was checked on disk (`ls independent_tests/claude-opus-5-5/`
and `gofmt -e`) before `catalog.go`, the first implementation file, was
written. After the implementation, test files were changed only by `gofmt -w`
(formatting, before the first test run). No test was edited after a test run
(see Test Refinements).

## Continuity-Check

not applicable — no test-author input. The input directory has no
`independent_tests/<other-llm>/` directory and no `TEST_REPORT.md`. This is
a single-LLM run.

## Resume logic

The output directory `/tmp/pcd-output/` was empty at the start. No
deliverable was found; every file below was produced in this run.

## Target language, framework, module identity

- **Language:** Go (template default `LANGUAGE | Go | default`). No preset
  overrides it: no preset or style hints files exist.
- **GO-FRAMEWORK: mcp-go (template default).** `github.com/mark3labs/mcp-go`
  at the verified version of the hints file: `require github.com/mark3labs/mcp-go v0.46.0`.
  The rationale: the template default covers both required transports
  (stdio and Streamable HTTP), and the hints file gives a verified version.
  No preset selects go-sdk.
- **Module identity:** `aha-aql`. No authoritative source 1–3 was present;
  identity derived from spec title `aha-aql` via convention `Go module path =
  spec title (lower case, hyphens kept)`. To override, add a `Module:` field
  to spec META or a language-specific hints file. (The spec mentions
  `https://github.com/mge1512/aha-aql` only as text of the User-Agent header;
  that is not one of the four authoritative sources, so it was not used.)
- **Binary name:** `mcp-server-aha-aql` (spec DEPLOYMENT, template naming
  convention `mcp-server-{n}` with `{n}` = `aha-aql`).

## Delivery mode

Filesystem (delivery mode 1): all files were written directly into
`/tmp/pcd-output/`. No commit or push was possible or requested.

## Active MILESTONE

The spec has six MILESTONE sections (0.0.0 to 0.5.0). All have
`Status: pending`; none has `Status: active`. Therefore the full spec was
translated, as the prompt says. All 26 BEHAVIORs were implemented. No stub
was produced. No BEHAVIOR is "not yet scheduled".

The acceptance criteria of all milestones were run against the built binary
anyway (see the compile gate).

## Deliverables

| Template deliverable | File(s) | Spec hash embedded as |
|---|---|---|
| source | `main.go`, `go.mod`, and the split files `tools.go`, `config.go`, `catalog.go`, `lexer.go`, `parser.go`, `read.go`, `write.go`, `session.go`, `aha.go`, `jsonout.go` | `// generated from spec:` header; `specSHA256` constant in `main.go`; `--version` prints `spec:<hash>` |
| build | `Makefile` (targets build, test, install, clean, man; CGO_ENABLED=0) | `SPEC_SHA256` variable |
| docs | `README.md` | HTML comment |
| man | `mcp-server-aha-aql.1.md`, `mcp-server-aha-aql.1` (pandoc output) | header comment of the `.md`; text in SEE ALSO, which pandoc carries into the `.1` |
| license | `LICENSE` (SPDX MIT + URL, no license text) | trailing line |
| RPM | `aha-aql.spec` (template name `{n}.spec`) | `# pcd-spec-sha256:` comment |
| unit | `mcp-server-aha-aql.service` | comment line |
| DEB | `debian/control`, `debian/changelog`, `debian/rules`, `debian/copyright` | `X-PCD-Spec-SHA256:` in control; comment/`Comment:` field in the others |
| OCI | `Containerfile` | `LABEL pcd.spec.sha256=` |
| auxiliary (Phase 4) | `translation_report/translation-workflow.pikchr` | comment line |
| tests (Phase 1) | `independent_tests/claude-opus-5-5/*_test.go` (9 files) | `// generated from spec:` + `// tests by: claude-opus-5-5` |
| report | `TRANSLATION_REPORT.md` | `Spec-SHA256` field |

Files produced by the mandated tool steps, not written by hand: `go.sum`
(from `go mod tidy`, template compile gate step 2; the template
Containerfile copies it) and `vendor/` (from `go mod vendor`, which the
environment instructions of this run require; the Makefile, the RPM spec
and `debian/rules` build with `-mod=vendor`, so the OBS build needs no
network). Both are dependency metadata of the one direct dependency. No
`.gitignore`, CI file or other unrequested file was written. The build
output `./mcp-server-aha-aql` was removed with `make clean` after the
compile gate.

The spec has no DELIVERABLES section and no COMPONENT entries, so no
COMPONENT mapping applies.

## How the template sections were applied

- **TYPE-BINDINGS / GENERATED-FILE-BINDINGS:** the template has neither
  section. Spec TYPES were mapped by hand: Value → `nil | string | float64 |
  bool` (record values) and `Num` (number literal of a statement); Row →
  ordered `Obj`; AqlError → `*AqlError{Kind, Message}`; ApiFailure →
  `*ApiFailure{Kind, Status, Cause, Text}`; EntityDef/FieldDef → structs in
  `catalog.go`.
- **PUBLIC-API-SURFACE:** the template does not declare this constraint.
  The component is a binary; it exports no library API.
- **Constraint fields:** every BEHAVIOR of the spec and of the template has
  `Constraint: required`. No BEHAVIOR is `supported` or `forbidden`.
- **COMMAND-WORDS, CONFIG-ENV-VARS, TOOL-EXECUTION-ERRORS:** all three are
  declared in the spec DEPLOYMENT section, so they were implemented: the
  command words `query` and `execute` (manual mode), the environment table of
  load-configuration, and every AqlError as a tool result with `isError`.
- **Transports:** stdio uses `server.ServeStdio` (mcp-go handles each tool
  call in a worker goroutine, serialises writes, drains the workers on EOF,
  and handles SIGTERM/SIGINT). http uses `server.NewStreamableHTTPServer`
  in stateless mode behind an own `http.Server` on `/mcp`. A middleware
  implements request-account before the MCP handler; the listener is bound
  first, so a bind error exits 1 with a diagnostic; SIGTERM/SIGINT run
  `Shutdown` with a 30 second drain and exit 0. GET `/mcp` (SSE) is served
  by mcp-go.
- **Signal handling, JSON-RPC errors:** tool handlers are wrapped in a
  recover; a panic is logged to standard error with a UTC timestamp and
  answered as a JSON-RPC error.

## STEPS ordering per BEHAVIOR

The STEPS of every BEHAVIOR were implemented in the written order. Notes
where the code shape differs from a literal reading:

- **parse-statement:** step 7 (resolve-names) runs after the syntax pass of
  steps 4–9 for read statements, because the select items come before
  `FROM` and need the entity. Entity words are resolved at once (W12–W14,
  N01, S06). The second reading of step 3 is attempted only on a syntax
  error of the strict reading; the strict error is returned when it fails.
- **tokenize:** steps 1–8 are one scanner loop; the date-without-quotes
  test (step 5) runs before the number test (step 6).
- **plan-fetch:** steps 1 → 2 → 5 → 6 → 7 in the order of the spec; step 8
  (list reading, with 8 a count only, 8 b early end, 8 c too many pages,
  8 e custom fields, and the MECHANISM of the `fields` parameter) is one
  function; step 9 lookups and step 10 failure mapping run for every route.
- **read-statement:** steps 1–9 in order; step 6 a/b reads the star custom
  fields and, for products, the product record.
- **insert/update/delete-statement:** steps in order. MECHANISM of
  update-statement step 2 (one ApiGet by id, never a list) is followed.
- **aha-request:** steps 1–10 in order. MECHANISM step 3: the standard
  library (`http.ProxyFromEnvironment`) chooses the proxy; the CONNECT tunnel
  and TLS are done by a custom `DialTLSContext`, so each failure gets its
  cause (proxy, certificate, no answer, connection). MECHANISM clock: the
  waiting of step 7 and the time of the log line go through the replaceable
  `clock` variable in `aha.go`.
- **choose-example-key:** the key is chosen lazily. Every text that shows
  the ExampleKey carries a private-use marker; the marker is replaced only
  when a result or message is shown, so a call that shows no example causes
  no request for the key (EXAMPLE example_key_is_chosen_when_it_is_needed).

## INTERFACES test doubles

The spec declares one interface, AhaAPI, with the test double **FakeAha**.
FakeAha was produced in `independent_tests/claude-opus-5-5/fakeaha_test.go`.
The tests are black-box: the binary cannot be given a test double as a Go
value, so FakeAha takes the place of the HTTPS connection the way a sandbox
does it, through the declared environment variables. FakeAha is the HTTPS
proxy of `HTTPS_PROXY`; it opens the CONNECT tunnel, ends TLS with a
certificate for `*.aha.io` that a per-run test CA signs (`SSL_CERT_FILE`,
with `SSL_CERT_DIR` set to an empty directory), and answers from memory with
the fixture account of the INTERFACES section. It records every request
(method, host, path, parameters, headers, body, the proxy that was used),
and it supports every control that the spec lists: status/header/body
overrides N times, no answer, refusal of the tunnel, custom definitions
added later, lists without `custom_fields`, and additional products (BULK,
BIG, HUGE, PROD). It never opens a network connection. The production
implementation is not used by the tests.

## Specification ambiguities

1. **entity_without_custom_fields, goals field list.** The EXAMPLE text lists
   the fields of goals as `id, name, status, product, progress, value, effort,
   description, created_at, updated_at, url`, without `success_metric`. The
   rule for `<field list>` is "the names of all fields of the entity", and
   the catalog entry of goals has `success_metric` as its third field
   (describe_shows_a_field_that_is_set_in_insert_only confirms it). The
   implementation follows the rule and the catalog; the test
   `TestEntityWithoutCustomFields` asserts the message with `success_metric`.
   The spec author should correct the EXAMPLE text.
2. **Lists kept between statements.** The spec allows ("may") keeping the
   product list, the custom field definitions, the options and the
   workflows between statements in stdio mode. record_by_id_has_all_fields
   counts an ApiList on `/products` for its statement although the tool list
   was answered before. The conservative reading: nothing is kept between
   statements, in any mode. Each statement reads each list at most once.
   The "read again when kept" steps (resolve-product 3, custom-fields 6 c,
   check-write-values 9 d and 10 c) therefore never apply.
3. **Annotations of `query`.** The spec gives readOnlyHint true and
   openWorldHint true. mcp-go defaults destructiveHint to true; the
   implementation sets destructiveHint false for `query`, because the tool
   only reads. idempotentHint keeps the mcp-go default (false).
4. **Aggregates other than COUNT with `*`.** The grammar allows
   `function ( * | ... )` for all five functions, but only `COUNT(*)` has a
   meaning. `SUM(*)` and the like get S03 "a field or * inside SUM( )".
5. **ORDER BY a label.** The grammar names fields, labels of aggregate
   items and aggregates. The implementation also accepts the `AS` label of
   a field item (it sorts by that field). This accepts more statements and
   never changes the result of a statement that the strict grammar accepts.
6. **W25 for requirements.** W25 shows `SELECT id, name FROM <entity of
   kind> WHERE product = '<PRODUCT>'`. For the kind requirement this
   statement is not accepted (requirements have no field product), which
   conflicts with the INVARIANT that every message statement is accepted.
   The text was implemented as written.
7. **K01 for comments.** Step 4 a of plan-fetch speaks of "a read route for
   a field of that kind". For comments the route `parent` accepts every
   commentable kind; the implementation counts it, so `SELECT * FROM
   comments WHERE id = 'DEMO-3'` gets K01 with `WHERE feature = 'DEMO-3'`
   (an accepted spelling of parent).
8. **Warnings.** The spec says only "write a warning". Every warning is one
   line on standard error that starts with `warning: ` and has no credential.
9. **Statement in S05.** `<id>` is shown as the statement writes it.
10. **Time-dependent EXAMPLEs.** execute_writes_a_log_line,
    execute_logs_a_refused_statement and too_many_requests_is_sent_again
    assume a test clock. A black-box test cannot replace the clock of the
    binary; the tests check the form of the time (`YYYY-MM-DDThh:mm:ssZ`)
    and real waiting times (the header `X-Ratelimit-Reset` is set two
    seconds in the future).
11. **Proxy address.** request_through_the_proxy_of_the_environment names
    `http://10.200.0.1:3128`. In the black-box test FakeAha is the proxy, so
    its own loopback address takes that place.
12. **host_of_aha_without_the_proxy.** With `NO_PROXY=.aha.io` the binary
    connects to `example.aha.io` directly. The test checks that FakeAha
    received nothing and that the result is a network error
    (`cannot reach Aha!: `). On a machine with network access this makes one
    DNS lookup and one TLS handshake attempt to the real host; the handshake
    fails because the trust store has only the test CA, so no credential
    is sent.

## Rules that could not be implemented exactly as written

- **Containerfile.** The template text has `COPY go.mod go.sum ./` and
  `RUN go mod download`. Because the dependencies are vendored, the builder
  copies `vendor/` and builds with `-mod=vendor` instead of downloading.
  The final stage `FROM scratch` has no trust store, but aha-request needs
  one for TLS to Aha!; the CA bundle of the builder
  (`/var/lib/ca-certificates/ca-bundle.pem`) is copied to
  `/etc/ssl/ca-bundle.pem`. `ENTRYPOINT` is the http mode as required; a
  `CMD ["listen=0.0.0.0:8080"]` makes the exposed port reachable, which
  causes the non-loopback warning of load-configuration on purpose.
- **Go regexp limit.** The Credential pattern `^[!-~]{1,4096}$` exceeds the
  repeat limit (1000) of Go's `regexp`. It is implemented as an equivalent
  byte check (`credPattern` in `session.go`).
- **Environment variables read by the standard library.** The server code
  reads only AHA_SUBDOMAIN, AHA_API_KEY, AHA_DEFAULT_PRODUCT and
  AHA_READ_ONLY (`getenv` in `main.go` refuses any other name). The proxy
  and trust store variables are read by the Go standard library, as the
  MECHANISM of aha-request requires; the standard library also consults
  `HTTP_PROXY` and `REQUEST_METHOD` inside `ProxyFromEnvironment`, but they
  have no effect on an https address. Go runtime variables (for example
  GODEBUG) are outside the control of the program.
- **Manual mode G10 for an empty value.** `query=` (empty) is not G10; the
  tool receives an empty statement and answers A01 (exit code 1), as the
  STEPS of manual-mode say (G10 only for an absent argument).

## Compile gate (Phase 6)

| Step | Command | Result |
|---|---|---|
| 1 Framework selection | mcp-go v0.46.0 (template default, hints-verified) | done |
| 2 Dependency resolution | `go mod tidy`, then `go mod vendor`, `go mod verify` | pass ("all modules verified"); go.mod has only the one direct dependency |
| 3 Compilation | `go build ./...` and `CGO_ENABLED=0 go build -mod=vendor` (`make build`) | pass, no warnings |
| 3a Vet / format | `go vet . ./independent_tests/...`, `gofmt -l` (outside vendor) | pass, no output |
| 4 Translator test run | `go test -count=1 ./independent_tests/claude-opus-5-5/...` (also via `make test`) | **pass: 187 passed, 0 failed, 0 skipped**, about 61 s (the three 60-second EXAMPLEs run in parallel) |
| 5 Test-author test run | — | not applicable (single-LLM run) |
| 6 Man page | `make man` (pandoc 2.18) | pass |

Toolchain used: go1.27.4 linux/amd64; `go.mod` declares `go 1.24`.

Milestone acceptance criteria, run with `./mcp-server-aha-aql` built by
`make build` (all 20 lines of the milestones 0.0.0, 0.1.0 and 0.5.0; the
`make test` lines of 0.1.0 to 0.5.0 are the test run above):

| Milestone | Criterion | Result |
|---|---|---|
| 0.0.0 | `test -x ./mcp-server-aha-aql` | pass |
| 0.0.0 | `--version` shows a 64-digit hash | pass |
| 0.1.0 | `bogus` → G12 text, exit 2 | pass (2 lines) |
| 0.1.0 | `env -i ... stdio` → G05; `subdomain=example` → G06 | pass (2 lines) |
| 0.1.0 | `bogus=1` → G02; `AHA_READ_ONLY=maybe` → G04; key file 0644 → G08 | pass (3 lines) |
| 0.1.0–0.4.0 | `make test` | pass |
| 0.5.0 | DELETE in query → S07, exit 1; same from stdin | pass (3 lines) |
| 0.5.0 | SELECT in execute → W01 | pass |
| 0.5.0 | execute with read-only → G11, exit 2 | pass (2 lines) |
| 0.5.0 | `HTTPS_PROXY=http://127.0.0.1:1` → X02 proxy reason, no stdout | pass (2 lines) |
| 0.5.0 | stdio tools/list with and without AHA_READ_ONLY | pass (3 lines) |

## Tool list

| Tool | Parameter | Description (first line) | Annotations |
|---|---|---|---|
| `query` | `query` (string, required, "One AQL statement") | Read from Aha! with one AQL statement. | readOnlyHint true, destructiveHint false, openWorldHint true |
| `execute` (not registered with read_only) | `statement` (string, required, "One AQL statement") | Change ONE record in Aha! with one AQL statement. Three forms: | readOnlyHint false, destructiveHint true, idempotentHint false, openWorldHint true |

The full descriptions are the texts of tool-registration; `{KEY}` is
replaced per Account by the ExampleKey in a tool filter of mcp-go. No MCP
resource and no MCP prompt is registered.

## Test results — translator suite

All tests are in `independent_tests/claude-opus-5-5/`. All are black-box:
they build the binary into a temporary directory (`go build ../..` in
`TestMain`), run it with `exec.Command` and a clean environment (HOME and
every file in a per-test temporary directory), and talk MCP over standard
input/output or HTTP. **187 passed, 0 failed, 0 skipped.**

Passed: TestAccountFromTheEnvironment, TestAggregatesAndDistinct,
TestAnswersInTheJSONFormOfAha, TestAnswerWithAnUnknownForm,
TestApostropheHintInAnError, TestApostropheInsideAWordIsReadAsText,
TestArgumentsTakePrecedence, TestAssignmentOfAWordOrAnExpression,
TestBareKeyOfACustomField, TestBodiesForAGoal, TestBodiesOfTheWriteRequestText,
TestBodyOfACreateRequest, TestBodyOfAnUpdateRequest, TestBodyWithCustomFields,
TestCatalogNameWinsOverACustomKey, TestCommentsNeedAParent,
TestCommentsOfARecord, TestConditionsOnCustomFields, TestCountNeedsOneListRequest,
TestCountOfNothingIsZero, TestCountPerStatus, TestCredentialIsNotAnOption,
TestCredentialThatAhaDoesNotAccept, TestCurrentUser, TestCustomFieldOfAProduct,
TestCustomFieldOfATypeThatIsNotSupported, TestCustomFieldsAreReadByName,
TestCustomFieldsAreRequestedOnlyWhenUsed, TestCustomFieldsInGroupByAndOrderBy,
TestCustomFieldsPerRecordType, TestCustomFieldsThatTheKeyCannotRead,
TestCustomFieldThatCannotBeSet, TestCustomFieldThatWasAddedLater,
TestCustomValuesAreCheckedByKind, TestCustomValuesByKind, TestDateThatIsNotADate,
TestDateWithoutQuotes, TestDefaultLimitIsReported, TestDefinitionsAreNotKeptInHTTPMode,
TestDeleteAReleaseThroughItsProduct, TestDeleteOfAnUnknownRecord,
TestDeleteOneFeature, TestDeleteWithACustomGuard, TestDeleteWithAFalseGuard,
TestDescribeAndReadHaveTheSameFields, TestDescribeOneEntity,
TestDescribeShowsAFieldThatIsSetInInsertOnly, TestDescribeTakesNothingElse,
TestDescribeWhenAhaCannotBeReached, TestDescribeWithoutEntity,
TestDescriptionIsPlainText, TestDoubledApostropheInText, TestDryRunShowsAListAsText,
TestEntityWithoutCustomFields, TestEveryPageIsReadOnce,
TestExampleKeyIsChosenWhenItIsNeeded, TestExampleKeyIsNotAProductOfTheAccount,
TestExampleKeyIsTheDefaultProduct, TestExampleKeyWhenTheProductListCannotBeRead,
TestExampleKeyWithoutADefaultProduct, TestExecuteLogsARefusedStatement,
TestExecuteLogsTheErrorKind, TestExecuteRefusesReadStatements,
TestExecuteRejectsUnknownParameter, TestExecuteResultEnvelope,
TestExecuteWritesALogLine, TestFieldWithoutValueIsNull,
TestFieldsWithoutAValueInConditions, TestGoalStatusFromTheSuccessMetric,
TestGracefulShutdownHTTP, TestGroupingWithoutGroupBy, TestHostOfAhaWithoutTheProxy,
TestHTTPBindError, TestHTTPModeUsesNoCredentialOfTheProcess,
TestHTTPRequestBringsItsAccount, TestHTTPRequestForAnotherAccount,
TestHTTPRequestFromAWebPage, TestHTTPRequestsShareNothing,
TestHTTPRequestWithoutACredential, TestHTTPRequestWithoutAnAccount,
TestHTTPToolListAndToolError, TestIDOrEntityAlone, TestInListReadsEachParent,
TestInsertCommentWithApostrophe, TestInsertGoalWithoutASuccessMetric,
TestInsertUnderAnUnknownRelease, TestInsertWithACustomFieldIsRefused,
TestInsertWithAnUnknownColumnIsRefused, TestInsertWithAProductAsRelease,
TestInsertWithOtherSpellings, TestInsertWithoutParent, TestInsertWithTwoRowsIsRefused,
TestJoinIsRefused, TestKeyFilePermissionAcceptance, TestKeyFileThatOtherUsersCanRead,
TestLabelIsTheWordOfTheStatement, TestLabelOfACustomField,
TestLimitOfTheStatementKeepsTheTotal, TestListenWarning,
TestListWithoutCustomFieldsIsAnError, TestManualDryRunSendsNoWriteRequest,
TestManualErrorGoesToStandardError, TestManualExecuteIsRefusedInReadOnly,
TestManualModeAcceptance, TestManualModeWritesNoLogLine,
TestManualQueryPrintsTheToolResult, TestManualStatementArgumentIsMissing,
TestManualStatementFromStandardInput, TestMeAsAValue, TestNameEqualityAndSearch,
TestNotesForEmptyResults, TestNullCannotBeWritten, TestOptionThatIsNotInTheList,
TestOrAndNotAreEvaluated, TestOwnNameOfTheEntity, TestParentThatDoesNotExist,
TestParkingLotAndMissingDate, TestProductInThreeSpellings,
TestProductsAreNotMixed, TestProductsCannotBeChanged,
TestProtocolErrorsAreJSONRPCErrors, TestProxyDoesNotOpenTheTunnel,
TestQueryRefusesWriteStatements, TestQueryRejectsProductParameter,
TestQueryRequiresAStatement, TestQueryResultEnvelope, TestReadOnlyVariableOfTheWrongForm,
TestReadSyntaxMessages, TestRecordByIDHasAllFields, TestRedirectIsNotFollowed,
TestReleaseByNameListsTheReleases, TestReleaseIDWhereAFeatureIDIsNeeded,
TestReleaseIDWhereAProductIsNeeded, TestReleaseOfAnEpic, TestReleasesNeedAProduct,
TestRequestAddressAndHeaders, TestRequestThroughTheProxyOfTheEnvironment,
TestRowsWithoutAValueSortLast, TestSettingsFromTheEnvironment,
TestStarShowsDefaultFields, TestStarShowsTheCustomValuesOfAProduct,
TestStarWithAnIDShowsCustomValues, TestStarWithoutAnIDHasCatalogFieldsOnly,
TestStartMessages, TestStartWithoutAnAccount, TestStatementsInMessagesCanBeSent,
TestStatementsWithoutSelect, TestStatusesOfAnEntity, TestStatusOfAnEpic,
TestStdioEndsOnEOF, TestStrictReadingIsKept, TestTagsAreSplitAtCommas,
TestTextAfterTheStatement, TestTextThatIsNotClosed, TestTooManyPages,
TestTooManyRequestsIsSentAgain, TestTooManyRequestsThreeTimes,
TestToolsInReadOnlyMode, TestToolsWithoutReadOnlyMode, TestUnexpectedCharacter,
TestUnknownCustomFieldListsTheCustomFields, TestUnknownEntityListsTheEntities,
TestUnknownFieldListsTheFields, TestUnknownFirstWord, TestUnknownIDGivesAnEmptyResult,
TestUnknownProductListsTheKeys, TestUnknownStatusListsTheStatuses,
TestUpdateAssignedToMe, TestUpdateOfAFieldThatIsSetInInsertOnly,
TestUpdateOfAnUnknownRecord, TestUpdateSendsEmptyText, TestUpdateSetsACustomField,
TestUpdateStatusInAnyCase, TestUpdateWithACustomGuard, TestUpdateWithADateWithoutQuotes,
TestUpdateWithAFalseGuard, TestUpdateWithATrueGuard, TestUserByName,
TestUsersCannotBeFilteredByProduct, TestValuesOfTheWrongForm, TestVersionFlag,
TestWriteNameMessages, TestWriteRequestWithoutAnAnswer, TestWriteSyntaxMessages,
TestWriteToAReadOnlyEntity, TestWriteWithoutAnIDIsRefused,
TestWriteWithoutWhereIsRefused.

Expected-long tests (annotated in the test comments, timeout 150 s per
call, suite timeout 15 min in the Makefile): TestDescribeWhenAhaCannotBeReached,
TestWriteRequestWithoutAnAnswer and TestExampleKeyWhenTheProductListCannotBeRead
(60 s each, in parallel); TestTooManyRequestsThreeTimes (10 s).

A mutation check (two deliberate defects in a scratch copy, removed
afterwards) made two tests fail, which shows that the assertions detect
changed output.

## Test results — test-author suite

Not applicable: no test-author suite is present in the input.

## Test Refinements

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| (all 187 tests) | first run: failed (every test starts the binary, and the binary panicked at start) | code fixed | The binary panicked at start: the Credential pattern `^[!-~]{1,4096}$` exceeds the repeat limit of Go's regexp. Replaced by an equivalent byte check (TYPES Credential). No test was changed. |
| (all 187 tests) | passed after the fix | none | — |

No test was edited after a test run.

## Per-EXAMPLE confidence

Confidence is **Medium** for every EXAMPLE that a named test verifies:
Tests-First-Compliance is yes and the named test passes without a live
external service, but no test-author suite exists, which caps the rating at
Medium by definition. Every EXAMPLE of the spec has a test whose name is the
EXAMPLE name in CamelCase (`query_result_envelope` → `TestQueryResultEnvelope`);
all tests run against FakeAha, never against Aha!.

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| query_result_envelope, query_rejects_product_parameter, query_requires_a_statement, query_refuses_write_statements | Medium | Test<Name> (stdio MCP, FakeAha call log) | — |
| execute_result_envelope, execute_rejects_unknown_parameter, execute_refuses_read_statements | Medium | Test<Name> | — |
| execute_writes_a_log_line, execute_logs_a_refused_statement | Medium | Test<Name>, TestExecuteLogsTheErrorKind | the literal time 2026-11-13T09:30:00Z (clock not replaceable from outside; the form of the time is checked) |
| tools_without_read_only_mode, tools_in_read_only_mode | Medium | Test<Name> (exact description texts, annotations, schema) | — |
| manual_query_prints_the_tool_result, manual_statement_from_standard_input, manual_dry_run_sends_no_write_request, manual_error_goes_to_standard_error, manual_statement_argument_is_missing, manual_execute_is_refused_in_read_only | Medium | Test<Name> (exec of the command words) | — |
| account_from_the_environment, arguments_take_precedence, settings_from_the_environment, read_only_variable_of_the_wrong_form, start_without_an_account, key_file_that_other_users_can_read, credential_is_not_an_option, values_of_the_wrong_form, http_mode_uses_no_credential_of_the_process, unknown_first_word | Medium | Test<Name>, TestStartMessages | key file path /run/user/1000/aha-key and /home/user/aha-key replaced by sandbox paths |
| http_request_brings_its_account, http_request_without_a_credential, http_request_for_another_account, http_request_from_a_web_page, http_requests_share_nothing | Medium | Test<Name> (HTTP POST /mcp) | — |
| count_per_status, star_shows_default_fields, record_by_id_has_all_fields, field_without_value_is_null, label_is_the_word_of_the_statement, name_equality_and_search, unknown_id_gives_an_empty_result, default_limit_is_reported, releases_need_a_product, star_with_an_id_shows_custom_values, star_without_an_id_has_catalog_fields_only, star_shows_the_custom_values_of_a_product, custom_fields_in_group_by_and_order_by | Medium | Test<Name> | — |
| describe_one_entity, describe_without_entity, describe_shows_a_field_that_is_set_in_insert_only | Medium | Test<Name> | — |
| describe_when_aha_cannot_be_reached | Medium | TestDescribeWhenAhaCannotBeReached (real 60 s wait) | — |
| insert_with_other_spellings, insert_comment_with_apostrophe, insert_without_parent, insert_with_a_product_as_release, insert_under_an_unknown_release, insert_goal_without_a_success_metric | Medium | Test<Name> | — |
| update_status_in_any_case, update_assigned_to_me, update_with_a_true_guard, update_with_a_false_guard, update_of_an_unknown_record, update_sends_empty_text, update_with_a_date_without_quotes, update_sets_a_custom_field, update_with_a_custom_guard, dry_run_shows_a_list_as_text | Medium | Test<Name> | — |
| delete_one_feature, delete_a_release_through_its_product, delete_with_a_false_guard, delete_of_an_unknown_record, delete_with_a_custom_guard | Medium | Test<Name> | — |
| doubled_apostrophe_in_text, date_without_quotes, text_that_is_not_closed, unexpected_character | Medium | Test<Name> | — |
| apostrophe_inside_a_word_is_read_as_text, strict_reading_is_kept, statements_without_select, text_after_the_statement, join_is_refused, id_or_entity_alone, describe_takes_nothing_else, apostrophe_hint_in_an_error | Medium | Test<Name>, TestReadSyntaxMessages | — |
| write_without_an_id_is_refused, write_without_where_is_refused, insert_with_two_rows_is_refused, insert_with_an_unknown_column_is_refused, update_of_a_field_that_is_set_in_insert_only, assignment_of_a_word_or_an_expression, null_cannot_be_written, write_to_a_read_only_entity, products_cannot_be_changed, label_of_a_custom_field, insert_with_a_custom_field_is_refused, custom_field_that_cannot_be_set | Medium | Test<Name>, TestWriteSyntaxMessages, TestWriteNameMessages | — |
| describe_and_read_have_the_same_fields, goal_status_from_the_success_metric, release_of_an_epic, statuses_of_an_entity, current_user, comments_of_a_record, custom_fields_per_record_type | Medium | Test<Name> | — |
| custom_fields_are_read_by_name, unknown_custom_field_lists_the_custom_fields, custom_field_of_a_type_that_is_not_supported, custom_fields_that_the_key_cannot_read, custom_field_that_was_added_later, definitions_are_not_kept_in_http_mode | Medium | Test<Name> | — |
| entity_without_custom_fields | Medium | TestEntityWithoutCustomFields | the literal goals field list of the EXAMPLE (ambiguity 1: the test asserts the list with success_metric) |
| own_name_of_the_entity, unknown_field_lists_the_fields, unknown_entity_lists_the_entities, users_cannot_be_filtered_by_product, bare_key_of_a_custom_field, catalog_name_wins_over_a_custom_key | Medium | Test<Name> | — |
| product_in_three_spellings, products_are_not_mixed, unknown_product_lists_the_keys, release_id_where_a_product_is_needed | Medium | Test<Name> | — |
| in_list_reads_each_parent, count_needs_one_list_request, every_page_is_read_once, too_many_pages, release_by_name_lists_the_releases, parent_that_does_not_exist, release_id_where_a_feature_id_is_needed, comments_need_a_parent, answers_in_the_json_form_of_aha, answer_with_an_unknown_form, custom_fields_are_requested_only_when_used, list_without_custom_fields_is_an_error, custom_field_of_a_product | Medium | Test<Name> | — |
| parking_lot_and_missing_date, description_is_plain_text, custom_values_by_kind | Medium | Test<Name> | — |
| or_and_not_are_evaluated, me_as_a_value, fields_without_a_value_in_conditions, conditions_on_custom_fields | Medium | Test<Name> | — |
| grouping_without_group_by, rows_without_a_value_sort_last, count_of_nothing_is_zero, limit_of_the_statement_keeps_the_total | Medium | Test<Name>, TestAggregatesAndDistinct | — |
| tags_are_split_at_commas, unknown_status_lists_the_statuses, status_of_an_epic, date_that_is_not_a_date, user_by_name, custom_values_are_checked_by_kind, option_that_is_not_in_the_list | Medium | Test<Name> | — |
| body_of_an_update_request, body_of_a_create_request, body_with_custom_fields, bodies_for_a_goal | Medium | Test<Name>, TestBodiesOfTheWriteRequestText | — |
| request_address_and_headers, redirect_is_not_followed, credential_that_aha_does_not_accept | Medium | Test<Name> | — |
| request_through_the_proxy_of_the_environment | Medium | TestRequestThroughTheProxyOfTheEnvironment | the literal proxy address http://10.200.0.1:3128 (FakeAha's loopback address takes its place) |
| host_of_aha_without_the_proxy | Medium | TestHostOfAhaWithoutTheProxy | the direct connection itself is not observed; only that FakeAha (the proxy) got nothing |
| proxy_does_not_open_the_tunnel | Medium | TestProxyDoesNotOpenTheTunnel, TestManualModeAcceptance | — |
| too_many_requests_is_sent_again | Medium | TestTooManyRequestsIsSentAgain | "the clock was asked to wait 2 seconds, once": checked as a real wait of about one to two seconds |
| too_many_requests_three_times | Medium | TestTooManyRequestsThreeTimes | "wait 5 seconds, twice": checked as a real wait of at least 9 seconds |
| write_request_without_an_answer | Medium | TestWriteRequestWithoutAnAnswer (real 60 s wait) | — |
| statements_in_messages_can_be_sent | Medium | TestStatementsInMessagesCanBeSent | acceptance is checked by the absence of every parse and name message, not by a parser call |
| example_key_is_the_default_product, example_key_without_a_default_product, example_key_is_not_a_product_of_the_account, example_key_is_chosen_when_it_is_needed, example_key_when_the_product_list_cannot_be_read | Medium | Test<Name> | — |
| Template: minimal_tool_server_stdio | Medium | TestStdioEndsOnEOF, TestQueryResultEnvelope | — |
| Template: minimal_tool_server_http | Medium | TestHTTPToolListAndToolError, TestHTTPRequestBringsItsAccount | — |
| Template: unknown_transport_argument, unknown_first_word_with_command_words | Medium | TestUnknownFirstWord | — |
| Template: tool_error_handling | Medium | TestProtocolErrorsAreJSONRPCErrors | an internal error of a tool handler cannot be caused from outside; the recover path and its timestamped log entry are verified by code review only |
| Template: declared_tool_execution_error | Medium | every expectError assertion (isError true, one text, no JSON-RPC error) | — |
| Template: graceful_shutdown_http, http_bind_error | Medium | TestGracefulShutdownHTTP, TestHTTPBindError | draining of an in-flight request during SIGTERM is not exercised |
| Template: setting_from_declared_environment_variable | Medium | TestArgumentsTakePrecedence, TestSettingsFromTheEnvironment | — |
| Template: framework_mcp_go_default | Medium | go.mod content, this report | — |
| Template: framework_go_sdk_override | Low | not applicable (no preset override) | — |

INVARIANTS marked `[implementation]` (certificate check on every connection,
no data kept in http mode, one direct dependency, the catalog as the single
source, no request per record of a list) are verified by code review and
indirectly by tests (TestHostOfAhaWithoutTheProxy relies on the certificate
check; TestDefinitionsAreNotKeptInHTTPMode and TestHTTPRequestsShareNothing
on the per-request state; `go.mod` has one direct dependency).

## Documentation note

README.md and the man page were checked with the claudism-pass scanner
(variant: US). The time hits it reported are `127.0.0.1:8080` addresses and
a UTC log example; one sentence over 25 words was split into a list, and
"must not" was rewritten as "must be readable by its owner only".
