# TRANSLATION_REPORT.md — mcp-server-aha-aql

| Field | Value |
|---|---|
| Spec-SHA256 | `b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30` |
| Spec-SHA256 (host) | `b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30` |
| Spec | `aha-aql.spec.md`, Version 0.20.1, Spec-Schema 0.4.0 |
| Deployment-Template | `mcp-server.template.md` v0.3.24 |
| LLM-Name | `claude-opus-5-5` |
| Mode | `translator` |
| Run | single-LLM (no test-author output in the input directory) |
| Date | 2026-11-25 |

## Included-Specs

The host spec declares Spec-Schema 0.4.0 and no `Includes:` directive. The
merged spec text is therefore byte-identical to the host file, and the merged
hash equals the host hash.

| Path | SHA256 |
|------|--------|
| (none) | |

## Translation Inputs (provenance)

- `Spec-SHA256 (merged):` `b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30`
- `Spec-SHA256 (host):` `aha-aql.spec.md` `b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30`
- `Decisions-Hints-SHA256:` none
- `Milestones-Hints-SHA256:` none
- `Template-SHA256:` `mcp-server.template.md` `22c55b355e28a6f2635413e72f2281faa3f7f1c79f79ea4eca76aba0b5b37b3e`
- `Prompt-SHA256:` `prompt.md` `702ec28de611ff65d76b2819cb53c675867fb58b1c090c6b17192cc73470d7d3`
- `Library-Hints-SHA256:` `mcp-server.go.mcp-go.hints.md` `f873012218200cddc8a96b42e2db95cffc8213195f590f28b83bec924d8ea0a3`
- `Style-Hints-SHA256:` none (no `/etc/pcd/hints/` and no `.pcd/hints/` exist)
- `Role-SHA256:` `ROLE.md` `e7839aedd9096d8059dd7854cfe94a01bd492cfbce204a445d37f8306c8930cf`
- `System-Prompt-SHA256:` `system-prompt.txt` `65a41409b8abb5504b505bb3f12b316c66ae6706d8baa19b8924a595122b0e71`
- `Upgrade-Brief-SHA256:` none
- `Directive-SHA256:` none

## Tests-First-Compliance

**no**, with one documented exception.

- The 10 test files with 208 tests (`fakeaha_test.go`, `harness_test.go`,
  `read_test.go`, `describe_test.go`, `write_test.go`, `config_test.go`,
  `manual_test.go`, `tools_test.go`, `http_test.go`, `aharequest_test.go`)
  were written before any implementation source file. The structural guard
  ran before `go.mod` and before the first `.go` file of the implementation:
  10 test files were present.
- A coverage audit after the compile gate found that the EXAMPLE
  `describe_takes_nothing_else` had no test. I added
  `TestDescribeTakesNothingElse` in a new file, `late_test.go`, after the
  implementation existed. No existing test was changed for it.
- Strictly, the prompt requires every file in the test directory to come
  before the implementation. So the answer is `no`. The confidence table
  below is Medium for every EXAMPLE. That level also follows from the
  single-LLM run, because no test-author suite exists.

## Continuity-Check

not applicable — no test-author input

## Active MILESTONE

The spec has six MILESTONE sections (0.0.0 to 0.5.0), and all of them have
`Status: pending`. None is active, so the full spec was translated, as the
prompt requires. No stubs were produced. All BEHAVIORs are implemented. No
BEHAVIOR is "not yet scheduled".

The acceptance criteria of all milestones were run anyway, against a binary
built in a scratch directory (removed afterwards). Results:

| Milestone | Criterion | Result |
|---|---|---|
| 0.0.0 | `test -x ./mcp-server-aha-aql` | pass |
| 0.0.0 | `--version` has a 64-character hex hash | pass |
| 0.1.0 | `make test` | pass |
| 0.1.0 | `bogus` gives the G12 line and exit code 2 (2 criteria) | pass |
| 0.1.0 | `env -i ... stdio` gives G05, G06, G02, G04 for AHA_READ_ONLY (4 criteria) | pass |
| 0.1.0 | key file with mode 644 gives G08 | pass |
| 0.2.0, 0.3.0, 0.4.0 | `make test` | pass |
| 0.5.0 | `make test` | pass |
| 0.5.0 | manual `query` refuses DELETE (message and exit 1), also with `query=-` (3 criteria) | pass |
| 0.5.0 | manual `execute` refuses SELECT | pass |
| 0.5.0 | manual `execute read-only=true` gives G11 and exit 2 (2 criteria) | pass |
| 0.5.0 | proxy `http://127.0.0.1:1` gives the X02 proxy message, nothing on stdout (2 criteria) | pass |
| 0.5.0 | stdio tool list has `execute`; with AHA_READ_ONLY=true it has `query` and no `execute` (3 criteria) | pass |

## Target language and framework

- Language: **Go** (template default, no preset override).
- GO-FRAMEWORK: mcp-go (template default). Module
  `github.com/mark3labs/mcp-go v1.6.0`, the verified version of the library
  hints file. The hints file requires Go 1.25.5 or later, and the `go` line
  of `go.mod` is `go 1.25.5`.
- Rationale: mcp-go covers both required transports (stdio and Streamable
  HTTP). It has `server.Hooks.AddAfterListTools`, and the server needs it to
  put the ExampleKey into the tool descriptions per request. The hints file
  recommends this hook. go-sdk was not considered further, because no preset
  asks for it.

## Module identity

Resolved module identity: `github.com/mge1512/aha-aql`.
Source: (1) spec META field `Module: github.com/mge1512/aha-aql`. No hints
file and no prior manifest named another identity. The template has no
`MODULE-IDENTITY` constraint row, but the spec META field is authoritative.

## Delivery mode

Filesystem. All files are written directly to `/tmp/pcd-output/`. The
output directory is not a git repository, so nothing was committed.

## Deliverables

| Template deliverable | File(s) |
|---|---|
| source | `main.go`, `go.mod`, plus `tools.go`, `config.go`, `catalog.go`, `aha.go`, `context.go`, `custom.go`, `lexer.go`, `parser.go`, `read.go`, `describe.go`, `write.go`, `messages.go`, `util.go`. The template allows tool handlers in further files for a larger server. The server has about 5,000 lines of Go. |
| (dependency lock) | `go.sum`, written by `go mod tidy` (compile gate step 2). The template's Containerfile runs `COPY go.mod go.sum`. |
| build | `Makefile` (targets `build`, `test`, `install`, `clean`, `man`, plus `vendor` and `vendor-tarball` from the hints file) |
| docs | `README.md` |
| man | `mcp-server-aha-aql.1.md`, `mcp-server-aha-aql.1` (built with pandoc 2.18) |
| license | `LICENSE` (SPDX `MIT` and URLs, no license text) |
| RPM | `aha-aql.spec` (`{n}.spec` with n = `aha-aql`; package Name `mcp-server-aha-aql`) |
| unit | `mcp-server-aha-aql.service` |
| DEB | `debian/control`, `debian/changelog`, `debian/rules`, `debian/copyright` |
| OCI | `Containerfile` |
| Phase 4 | `translation_report/translation-workflow.pikchr` |
| report | `TRANSLATION_REPORT.md` |
| tests | `independent_tests/claude-opus-5-5/*_test.go` (11 files) |

`vendor/` was made during the compile gate and then removed. The hints file
says that `vendor/` is never part of the source. `make vendor` makes it
again, and `make vendor-tarball` makes the tarball `Source1` of the RPM.

### Spec hash embedding

| Artefact | Form |
|---|---|
| every `.go` file (implementation and tests) | `// generated from spec: aha-aql.spec.md sha256:<hash>`; tests also `// tests by: claude-opus-5-5` |
| `go.mod` | the same comment on the first line |
| binary | `mcp-server-aha-aql --version` writes `mcp-server-aha-aql 0.20.1 spec:<hash>` |
| `Makefile` | `SPEC_SHA256 := <hash>` |
| `aha-aql.spec` | `# pcd-spec-sha256: <hash>` |
| `debian/control` | `X-PCD-Spec-SHA256: <hash>` |
| `debian/rules`, `debian/copyright`, `debian/changelog` | comment, `Comment:` field, changelog entry |
| `Containerfile` | `LABEL pcd.spec.sha256="<hash>"` |
| `mcp-server-aha-aql.service`, `README.md`, `mcp-server-aha-aql.1.md`, `LICENSE`, pikchr | comment, or the section SPECIFICATION in the man page (it is in `mcp-server-aha-aql.1` too) |

`go.sum` has no hash. It is a file of the Go tool, and comments are not
allowed in it.

## Transport implementation notes

- **stdio**: `server.NewStdioServer(...).Listen(ctx, os.Stdin, os.Stdout)`.
  The `ctx` comes from `signal.NotifyContext` for SIGTERM and SIGINT. mcp-go
  handles each tool call in a worker goroutine and serialises the writes.
  On EOF, mcp-go cancels the request context and then waits for the
  workers. For that reason the requests to Aha! use
  `context.WithoutCancel`, so that a call that has started is answered
  (drain). The exit code is 0 on EOF and on a signal. The mcp-go error
  logger writes to standard error through the same mutex as the log lines.
- **http**: `net.Listen` first. A bind error gives
  `error: cannot listen on <addr>: <reason>` and the exit code 1. Then
  `server.NewStreamableHTTPServer` with endpoint `/mcp` runs behind the
  middleware of request-account in an `http.ServeMux`. POST /mcp handles
  JSON-RPC, and GET /mcp gives the SSE stream (mcp-go). On SIGTERM or SIGINT:
  `http.Server.Shutdown` with a 30 second drain timeout, then exit 0. The
  server writes one start line to standard error:
  `mcp-server-aha-aql 0.20.1: listening on http://<addr>/mcp`.
- **request-account**: the middleware checks every request to `/mcp`
  before mcp-go sees it: Origin → 403, missing or malformed `Aha-Api-Key` →
  400, malformed `Aha-Subdomain` → 400, another account → 403, no account →
  400. The body is a JSON-RPC error with `id` null and code -32600. The
  middleware puts the Account and a new ExampleKey state into the request
  context. Nothing of a request is kept after the answer.
- **Command words** `query` and `execute` (COMMAND-WORDS): manual mode, see
  `runManual` in `main.go`.
- **TOOL-EXECUTION-ERRORS**: every AqlError is
  `mcp.NewToolResultError(message)`. Unknown tools and malformed requests
  stay JSON-RPC errors (mcp-go).

## Tool list

| Tool | Parameter | Annotations | Description |
|---|---|---|---|
| `query` | `query` (string, required, "One AQL statement") | readOnlyHint true, destructiveHint false, idempotentHint true, openWorldHint true | the text of tool-registration, with `{KEY}` replaced per tool list |
| `execute` | `statement` (string, required, "One AQL statement") | readOnlyHint false, destructiveHint true, idempotentHint false, openWorldHint true | the text of tool-registration; not registered when read_only is true |

No resource and no prompt is registered.

## How STEPS ordering was applied

- **load-configuration** (`config.go`): steps 1 to 11 in order: first word
  (G12), every argument `key=value` (G01), G09 in http mode, unknown and
  repeated keys (G02, G03), variables for absent options, value checks in
  the order 6 a to 6 d (G04, warning G13), http mode (G14), G05, key file
  (G07 no such file / not a regular file / cannot be read, then G08, then
  content), AHA_API_KEY (G06, G07).
- **manual-mode** (`main.go`): load-configuration, then the statement
  argument (G10), standard input for `-`, then G11, then the tool call, then
  stdout or `error: ` on stderr.
- **query / execute** (`tools.go`): A02, A01/A03, parse-statement with the
  allowed kinds, then describe-, read-, insert-, update- or
  delete-statement, then the result object. execute writes the log line
  when the call ends, with or without an error.
- **tokenize** (`lexer.go`) and **parse-statement** (`parser.go`): step 1
  (S05) before the strict reading; the second reading only after a syntax
  error and only when the text has an apostrophe between a letter or digit
  and a letter; then kind detection (steps 4 to 6), the form, the end check
  (step 8), and the refused constructs (step 9).
- **read-statement** (`read.go`, `runRead`): star replacement (1), used
  fields (2), plan-fetch (3), convert-record (4), filter with applied
  conjuncts counted as true (5), star custom fields (6 a to c), build-rows,
  H02 (7), notes for an empty result (8).
- **plan-fetch** (`read.go`, `planFetch`): fixed sources (1), id and parent
  routes in catalog order without the product route (2, 3), wrong-kind
  messages (4), IN routes with all values checked before the first request
  (5), product route (6), account-wide list (7), list reading with count
  only, early end, too many pages and the custom-fields check (8), lookups
  (9), failure mapping (10).
  MECHANISM: every list request has `fields` with the top-level API
  sources. It names `custom_fields` only when a custom field is used.
  Requests are made per page and never per record.
- **update-statement / delete-statement** (`write.go`, `runChange`): id
  check (K02/K04), one ApiGet by id (MECHANISM: never a list), convert,
  guards in order (W29), check-write-values, dry run, write request, result.
- **insert-statement** (`write.go`, `runInsert`): W18, parent check (P01,
  W24, W25, K08), W19, check-write-values, dry run, ApiCreate, result.
- **aha-request** (`aha.go`): address, headers, proxy through
  `http.ProxyFromEnvironment` (MECHANISM), TLS with the platform trust store,
  failures before an answer (Unreachable, not sent again), no redirect,
  429 with waiting from `X-Ratelimit-Reset` (1 to 30 s, default 5 s, at most
  two repetitions), 401/403, 404, other status, JSON and form checks.
- **choose-example-key** (`context.go`, `stmtCtx.key`): chosen when it is
  first needed. It is kept for the process in stdio and manual mode, and
  chosen per request in http mode. G15 and G16 are written as the steps say.

## INTERFACES test doubles

`FakeAha` (`independent_tests/claude-opus-5-5/fakeaha_test.go`) is the
declared test double. It is an HTTP proxy on 127.0.0.1 that answers
CONNECT for `example.aha.io:443` and ends TLS inside the tunnel. It uses a
leaf certificate that a per-test certificate authority signs. The test gives
the server `HTTPS_PROXY` and `SSL_CERT_FILE`, as a sandbox does. FakeAha
answers from memory with the fixture account of INTERFACES: products,
releases, features, epic, requirement, initiative, goals, comment, users,
workflows, the 13 custom field definitions with options and custom values.
It also has every control that the INTERFACES section lists: per-route
status, header, body or no answer, refusal of CONNECT, leaving out
`custom_fields` in lists, extra definitions, extra products (BULK, BIG,
HUGE, PROD). It records calls and CONNECT requests with their time. FakeAha
opens no network connection of its own. The implementation has no hook for
tests: no option, no variable and no code path only for tests.

The tests are black-box. They start the binary that `TestMain` builds
(`go build -o <tempdir>/mcp-server-aha-aql .` from the project root, two
directories up). They talk MCP over stdio and HTTP, run the command words,
and look at stdout, stderr, the exit codes and the requests at FakeAha. The
tests are hermetic. HOME, the CA file and every key file are in
`t.TempDir()`. The helper `writeSandboxFile` refuses paths outside the
sandbox. The environment of the server is built from scratch, and nothing
is inherited. Illustrative paths of the EXAMPLES (`/run/user/1000/aha-key`,
`/home/user/aha-key`) are rewritten to sandbox paths. The illustrative port
8080 is replaced by a free port.

## TYPE-BINDINGS, GENERATED-FILE-BINDINGS, Constraint fields

- The template has no `## TYPE-BINDINGS` section and no
  `## GENERATED-FILE-BINDINGS` section. Logical types map to Go types as
  follows: Value → `nil | string | float64 | bool`, TextList → `[]string`,
  Row and result objects → an ordered JSON object type (`omap`). JSON text
  is written without HTML escaping and without white space outside strings.
- Every BEHAVIOR of the spec has `Constraint: required`. All of them are
  implemented. The template BEHAVIORs stdio-transport and http-transport
  are required and implemented. No BEHAVIOR is `supported` or `forbidden`.
- The spec has no DELIVERABLES section with COMPONENT entries. The mapping
  above comes from the template.
- PUBLIC-API-SURFACE: the template does not declare this constraint. The
  binary is a `package main` command and exports no library API.

## Compile gate (Phase 6)

| Step | Command | Result |
|---|---|---|
| 1 Framework | mcp-go v1.6.0 (template default) | done |
| 2 Dependencies | `go mod tidy` | pass. Indirect dependencies were resolved by the tool (google/jsonschema-go, google/uuid, santhosh-tekuri/jsonschema/v6, spf13/cast, yosida95/uritemplate/v3, golang.org/x/text) |
| 3 Compilation | `go build ./...`, `CGO_ENABLED=0`, Go 1.27.4 | pass after one fix (see Test Refinements) |
| — | `go vet ./...`, `gofmt -l` | pass, no output |
| — | `go mod vendor`, `make build`, `make test` | pass (vendor/ removed afterwards) |
| 4 Translator tests | `go test ./independent_tests/claude-opus-5-5/...` | **209 passed, 0 failed, 0 skipped** (61 s) |
| 5 Test-author tests | — | not applicable (single-LLM run) |
| — | `pandoc mcp-server-aha-aql.1.md -s -t man -o mcp-server-aha-aql.1` | pass |

Not executed: `rpmbuild`/OBS, `dpkg-buildpackage` and `podman build` of the
Containerfile. The build host has none of these tools. The commands in
these files are the same as in the Makefile, and `make build`, `make test`
and `make man` passed.

## Test results — translator suite

All tests are in `independent_tests/claude-opus-5-5/`. Every test passed.
No test was skipped. No test needs a live Aha! account. The tests on
timeouts wait for the real 60 seconds of aha-request
(`TestDescribeWhenAhaCannotBeReached`, `TestWriteRequestWithoutAnAnswer`,
`TestExampleKeyWhenTheProductListCannotBeRead`). The tests on 429 wait
2 to 10 seconds. These tests run in parallel and have timeouts of 150
seconds.

Passed (209): TestAccountFromTheEnvironment, TestAFieldThatIsGivenTwice,
TestAnswersInTheJSONFormOfAha, TestAnswerWithAnUnknownForm,
TestApostropheHintInAnError, TestApostropheInsideAWordIsReadAsText,
TestArgumentsOfTheWrongForm, TestArgumentsTakePrecedence,
TestARowHasEachLabelOnce, TestArticlesAndIDsInMessages,
TestAssignmentOfAWordOrAnExpression, TestBareKeyOfACustomField,
TestBodiesForAGoal, TestBodyOfACreateRequest, TestBodyOfAnUpdateRequest,
TestBodyWithCustomFields, TestCatalogNameWinsOverACustomKey,
TestCertificateThatTheTrustStoreDoesNotHave, TestCommentsNeedAParent,
TestCommentsOfARecord, TestConditionsOnCustomFields,
TestCountNeedsOneListRequest, TestCountOfNothingIsZero, TestCountPerStatus,
TestCredentialIsNotAnOption, TestCredentialThatAhaDoesNotAccept,
TestCurrentUser, TestCustomFieldOfAProduct,
TestCustomFieldOfATypeThatIsNotSupported, TestCustomFieldsAreReadByName,
TestCustomFieldsAreRequestedOnlyWhenUsed,
TestCustomFieldsInGroupByAndOrderBy, TestCustomFieldsPerRecordType,
TestCustomFieldsThatTheKeyCannotRead, TestCustomFieldThatCannotBeSet,
TestCustomFieldThatWasAddedLater, TestCustomValuesAreCheckedByKind,
TestCustomValuesByKind, TestDateThatIsNotADate, TestDateWithoutQuotes,
TestDefaultLimitIsReported, TestDefaultProductThatTheAccountDoesNotHave,
TestDefinitionsAreNotKeptInHTTPMode, TestDeleteAReleaseThroughItsProduct,
TestDeleteOfAnUnknownRecord, TestDeleteOneFeature,
TestDeleteWithACustomGuard, TestDeleteWithAFalseGuard,
TestDescribeAndReadHaveTheSameFields, TestDescribeOneEntity,
TestDescribeShowsAFieldThatIsSetInInsertOnly,
TestDescribeTakesNothingElse, TestDescribeWhenAhaCannotBeReached,
TestDescribeWithoutEntity, TestDescriptionIsPlainText,
TestDescriptionOfAProductIsRequestedWhenItIsUsed,
TestDoubledApostropheInText, TestDryRunShowsAListAsText,
TestEmptyTextAndNullStandForNoValue, TestEntityWithoutCustomFields,
TestEveryAssignedFieldIsSent, TestEveryPageIsReadOnce,
TestExampleKeyIsChosenWhenItIsNeeded, TestExampleKeyIsKeptForTheProcess,
TestExampleKeyIsNotAProductOfTheAccount, TestExampleKeyIsTheDefaultProduct,
TestExampleKeyWhenTheProductListCannotBeRead,
TestExampleKeyWithoutADefaultProduct, TestExecuteLogsARefusedStatement,
TestExecuteRefusesReadStatements, TestExecuteRejectsUnknownParameter,
TestExecuteRequiresAStatement, TestExecuteResultEnvelope,
TestExecuteWritesALogLine, TestFieldsWithoutAValueInConditions,
TestFieldsWithSeveralValues, TestFieldWithoutValueIsNull,
TestGoalStatusFromTheSuccessMetric, TestGracefulShutdownHTTP,
TestGracefulShutdownStdio, TestGroupingWithoutGroupBy, TestHTTPBindError,
TestHTTPExecuteWritesALogLine, TestHTTPModeUsesNoCredentialOfTheProcess,
TestHTTPRequestBringsItsAccount, TestHTTPRequestForAnotherAccount,
TestHTTPRequestFromAWebPage, TestHTTPRequestsShareNothing,
TestHTTPRequestWithoutACredential, TestHTTPRequestWithoutAnAccount,
TestHTTPToolListAndExampleKeyPerRequest, TestIDOrEntityAlone,
TestIDsAreSortedByTheirNumbers, TestInListIsCheckedBeforeTheFirstRequest,
TestInListReadsEachParent, TestInsertCommentWithApostrophe,
TestInsertGoalWithoutASuccessMetric, TestInsertUnderAnUnknownRelease,
TestInsertWithACustomFieldIsRefused, TestInsertWithAnUnknownColumnIsRefused,
TestInsertWithAProductAsRelease, TestInsertWithOtherSpellings,
TestInsertWithoutParent, TestInsertWithTwoRowsIsRefused, TestJoinIsRefused,
TestKeyFileThatCannotBeUsed, TestKeyFileThatOtherUsersCanRead,
TestKeysAndArgumentsThatAreRefused, TestLabelIsTheWordOfTheStatement,
TestLabelOfACustomField, TestLabelsOfSelectItems,
TestLimitOfTheStatementKeepsTheTotal, TestListenAddressThatIsNotLoopback,
TestListWithoutCustomFieldsIsAnError, TestManualDryRunOfInsertAndDelete,
TestManualDryRunSendsNoWriteRequest, TestManualErrorGoesToStandardError,
TestManualExecuteIsRefusedInReadOnly, TestManualModeWritesNoLogLine,
TestManualProxyThatCannotBeReached, TestManualQueryPrintsTheToolResult,
TestManualQueryRefusesWrites, TestManualStatementArgumentIsMissing,
TestManualStatementFromStandardInput, TestMeAsAValue,
TestNameEqualityAndSearch, TestNullCannotBeWritten,
TestOptionThatIsNotInTheList, TestOrAndNotAreEvaluated,
TestOrderByALabelAndByAnAggregate, TestOtherStatusIsAnAPIError,
TestOwnNameOfTheEntity, TestParentThatDoesNotExist,
TestParkingLotAndMissingDate, TestProductInThreeSpellings,
TestProductsAreNotMixed, TestProductsCannotBeChanged,
TestProxyDoesNotOpenTheTunnel, TestQueryRefusesWriteStatements,
TestQueryRejectsProductParameter, TestQueryRequiresAStatement,
TestQueryResultEnvelope, TestReadOnlyVariableOfTheWrongForm,
TestRecordByIDHasAllFields, TestRecordsWithoutAValueAreOneGroup,
TestRecordThatExistsAndDoesNotMatch, TestRedirectIsNotFollowed,
TestReleaseByNameListsTheReleases, TestReleaseIDWhereAFeatureIDIsNeeded,
TestReleaseIDWhereAProductIsNeeded, TestReleaseOfAnEpic,
TestReleasesNeedAProduct, TestRequestAddressAndHeaders,
TestRequestThroughTheProxyOfTheEnvironment, TestRowsWithoutAValueSortLast,
TestSettingsFromTheEnvironment, TestStarShowsDefaultFields,
TestStarShowsTheCustomValuesOfAProduct, TestStartWithoutAnAccount,
TestStarWithAnIDShowsCustomValues, TestStarWithoutAnIDHasCatalogFieldsOnly,
TestStatementsInMessagesCanBeSent, TestStatementsWithoutSelect,
TestStatusesOfAnEntity, TestStatusesOfAnEntityThatHasNone,
TestStatusOfAnEpic, TestStrictReadingIsKept, TestTagsAreSplitAtCommas,
TestTextAfterTheStatement, TestTextThatIsNotClosed,
TestTheProductListIsNotKept, TestTimestampsArePointsInTime,
TestToolsInReadOnlyMode, TestToolsWithoutReadOnlyMode, TestTooManyPages,
TestTooManyRequestsIsSentAgain, TestTooManyRequestsThreeTimes,
TestUnexpectedCharacter, TestUnknownCustomFieldListsTheCustomFields,
TestUnknownEntityListsTheEntities, TestUnknownFieldListsTheFields,
TestUnknownFirstWord, TestUnknownIDGivesAnEmptyResult,
TestUnknownProductListsTheKeys, TestUnknownStatusListsTheStatuses,
TestUnknownToolIsAProtocolError, TestUpdateAssignedToMe,
TestUpdateOfAFieldThatIsSetInInsertOnly, TestUpdateOfAnUnknownRecord,
TestUpdateOfTheNameShowsItOnce, TestUpdateSendsEmptyText,
TestUpdateSetsACustomField, TestUpdateStatusInAnyCase,
TestUpdateWithACustomGuard, TestUpdateWithADateWithoutQuotes,
TestUpdateWithAFalseGuard, TestUpdateWithAGuardForNoValue,
TestUpdateWithATrueGuard, TestUserByName,
TestUsersCannotBeFilteredByProduct, TestValuesOfTheWrongForm,
TestValuesThatDoNotFitTheType, TestVersionFlag,
TestWriteRequestHasAContentType, TestWriteRequestThatAhaRejects,
TestWriteRequestWithoutAnAnswer, TestWriteToAReadOnlyEntity,
TestWriteWithoutAnIDIsRefused, TestWriteWithoutWhereIsRefused.

Failed: none. Skipped: none.

## Test results — test-author suite

Not applicable. No `independent_tests/<other-llm>/` and no `TEST_REPORT.md`
were in the input directory.

## Test Refinements

| Test | Result before | Action | Rationale |
|---|---|---|---|
| all 208 tests of the first run | failed (the binary panicked at start) | code fixed | `regexp.MustCompile("^[!-~]{1,4096}$")` panics in Go, because the regexp package allows a repeat count of at most 1000. The check of the TYPE `Credential` is now a function that tests the length (1 to 4096) and the byte range `!` to `~`. No test was changed. |
| TestDescribeTakesNothingElse | did not exist | test added (after implementation) | The coverage audit found the EXAMPLE `describe_takes_nothing_else` without a test. A new test in `late_test.go` asserts the exact S14 message of the EXAMPLE and that FakeAha recorded no call. It passed on its first run. No existing test was edited. |
| all other tests | passed | none | — |

Before the first test run, `gofmt -w` changed only the white space of
`fakeaha_test.go`. That is not a refinement, because no test had run yet.

## Specification ambiguities

1. **Dry run of INSERT.** Step 5 says "a WriteResult without id, with the
   checked fields as values". The parent is not a field. The result is
   `{<checked fields>..., "message":"would create <singular> under <parent>"}`.
   The parent is shown as the product key, or as the id in capital letters.
   Tested by `TestManualDryRunOfInsertAndDelete`. No EXAMPLE gives this case.
2. **One CONNECT per request.** The EXAMPLE
   `request_through_the_proxy_of_the_environment` counts one CONNECT for one
   statement after the tool list. To make this deterministic, the HTTP
   client does not keep connections open (`DisableKeepAlives`). Each request
   opens its own tunnel and TLS session. This also fits "the server keeps
   nothing".
3. **ExampleKey and the product list of the statement.** When a message of
   a statement needs the ExampleKey, choose-example-key uses the product
   list of that statement. If the statement has not read it, the ExampleKey
   reads it and keeps it for the rest of the statement. So the list is read
   at most once per statement (resolve-product step 1). It is not kept after
   the statement.
4. **AHA_READ_ONLY, read-only and dry-run** accept exactly `true` and
   `false`, in lower case (G04 otherwise). The spec writes "`true` or
   `false`" and does not mention case.
5. **A02 `<names>`.** A JSON object has no defined order. The unknown
   parameter names are sorted alphabetically.
6. **N02 `<word>`** is the word without a qualifier (as N04 defines it), for
   example `severity` for `f.severity`.
7. **`WHERE id = NULL` in a write statement**: W11 ("NULL as a value") takes
   precedence over the W03 text "WHERE id = needs the id in single quotes.".
   A bare word gives S13.
8. **JOIN variants**: `JOIN`, `INNER`, `LEFT`, `RIGHT`, `FULL`, `CROSS`,
   `OUTER` and `NATURAL` are keywords and give S08. So they cannot be an
   alias.
9. **COUNT with an entity name** means a canonical (plural) entity name, for
   example `COUNT(features)`. `COUNT(release)` counts the field release of
   features, because `release` is a field word there.
10. **Templates `{n}.spec`**: n = `aha-aql` (binary `mcp-server-{n}`), so the
    RPM spec file is `aha-aql.spec`, with `Name: mcp-server-aha-aql`.
11. **http start line.** The server writes one informational start line to
    standard error. The deployment template allows "startup messages" on
    stderr. It is no warning and has no value of an option except the listen
    address.
12. **MCP sessions in http mode** use the mcp-go default (session ids are
    made and not checked). Each HTTP request still brings its own Account,
    and no data of Aha! is kept in a session.
13. **Users in the custom field `users`**: when an entry has no
    `email_value`, the e-mail addresses come from `full_value`, else from
    the texts of `value`, as custom-fields says. Each user's name and
    NumericId are taken from `full_value` when it is there.
14. **Date of a timestamp in a write**: `2026-02-02T08:00:00Z` is accepted
    for a date field, and its first 10 characters are sent. A text that
    starts with a date but is no valid timestamp gives W22.

## Rules that could not be implemented exactly as written

- **check-write-values `form_only`.** No caller of the spec passes true, so
  the parameter is not implemented. Every check always runs completely.
- **Template EXAMPLE tool_error_handling** ("stderr contains error log
  entry with timestamp"). Internal errors (panics) are caught by mcp-go
  `WithRecovery` and answered as JSON-RPC errors. mcp-go logs nothing for
  them. No test can cause an internal error through the black-box interface.
- **Permission bits of the key file on Windows.** They are not checked
  there (`permissionBits` is false on Windows), as the DEPLOYMENT section
  says ("applies on Linux and macOS").
- **Containerfile trust store.** The image copies
  `/var/lib/ca-certificates/ca-bundle.pem` of the BCI builder to
  `/etc/ssl/ca-bundle.pem`, a path that Go reads on Linux. The path in the
  builder was not verified with an actual image build.

## Per-EXAMPLE confidence

Confidence is **Medium** for every EXAMPLE. A named translator test passes
without a live service. But this is a single-LLM run, so no test-author
suite exists, and Tests-First-Compliance is `no` (see above). No EXAMPLE has
Low confidence.

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| query_result_envelope | Medium | TestQueryResultEnvelope | — |
| query_rejects_product_parameter | Medium | TestQueryRejectsProductParameter | — |
| query_requires_a_statement | Medium | TestQueryRequiresAStatement | — |
| query_refuses_write_statements | Medium | TestQueryRefusesWriteStatements | — |
| execute_result_envelope | Medium | TestExecuteResultEnvelope | — |
| execute_rejects_unknown_parameter | Medium | TestExecuteRejectsUnknownParameter | — |
| execute_refuses_read_statements | Medium | TestExecuteRefusesReadStatements | — |
| execute_writes_a_log_line | Medium | TestExecuteWritesALogLine, TestHTTPExecuteWritesALogLine | — |
| execute_logs_a_refused_statement | Medium | TestExecuteLogsARefusedStatement | — |
| tools_without_read_only_mode | Medium | TestToolsWithoutReadOnlyMode | — |
| tools_in_read_only_mode | Medium | TestToolsInReadOnlyMode | — |
| manual_query_prints_the_tool_result | Medium | TestManualQueryPrintsTheToolResult | — |
| manual_statement_from_standard_input | Medium | TestManualStatementFromStandardInput | — |
| manual_dry_run_sends_no_write_request | Medium | TestManualDryRunSendsNoWriteRequest | — |
| manual_error_goes_to_standard_error | Medium | TestManualErrorGoesToStandardError | — |
| manual_statement_argument_is_missing | Medium | TestManualStatementArgumentIsMissing | — |
| manual_execute_is_refused_in_read_only | Medium | TestManualExecuteIsRefusedInReadOnly | — |
| account_from_the_environment | Medium | TestAccountFromTheEnvironment | — |
| arguments_take_precedence | Medium | TestArgumentsTakePrecedence | key file in the sandbox instead of /run/user/1000 |
| settings_from_the_environment | Medium | TestSettingsFromTheEnvironment | — |
| read_only_variable_of_the_wrong_form | Medium | TestReadOnlyVariableOfTheWrongForm | — |
| start_without_an_account | Medium | TestStartWithoutAnAccount | — |
| key_file_that_other_users_can_read | Medium | TestKeyFileThatOtherUsersCanRead | key file in the sandbox instead of /home/user |
| credential_is_not_an_option | Medium | TestCredentialIsNotAnOption | — |
| values_of_the_wrong_form | Medium | TestValuesOfTheWrongForm | — |
| http_mode_uses_no_credential_of_the_process | Medium | TestHTTPModeUsesNoCredentialOfTheProcess | free port instead of 8080 |
| listen_address_that_is_not_loopback | Medium | TestListenAddressThatIsNotLoopback | free port instead of 8080 for the loopback case |
| unknown_first_word | Medium | TestUnknownFirstWord | — |
| http_request_brings_its_account | Medium | TestHTTPRequestBringsItsAccount | — |
| http_request_without_a_credential | Medium | TestHTTPRequestWithoutACredential | — |
| http_request_for_another_account | Medium | TestHTTPRequestForAnotherAccount | — |
| http_request_from_a_web_page | Medium | TestHTTPRequestFromAWebPage | — |
| http_requests_share_nothing | Medium | TestHTTPRequestsShareNothing | — |
| count_per_status | Medium | TestCountPerStatus | — |
| star_shows_default_fields | Medium | TestStarShowsDefaultFields | — |
| record_by_id_has_all_fields | Medium | TestRecordByIDHasAllFields | — |
| field_without_value_is_null | Medium | TestFieldWithoutValueIsNull | — |
| label_is_the_word_of_the_statement | Medium | TestLabelIsTheWordOfTheStatement | — |
| name_equality_and_search | Medium | TestNameEqualityAndSearch | — |
| unknown_id_gives_an_empty_result | Medium | TestUnknownIDGivesAnEmptyResult | — |
| record_that_exists_and_does_not_match | Medium | TestRecordThatExistsAndDoesNotMatch | — |
| statuses_of_an_entity_that_has_none | Medium | TestStatusesOfAnEntityThatHasNone | — |
| default_limit_is_reported | Medium | TestDefaultLimitIsReported | — |
| releases_need_a_product | Medium | TestReleasesNeedAProduct | — |
| star_with_an_id_shows_custom_values | Medium | TestStarWithAnIDShowsCustomValues | — |
| star_without_an_id_has_catalog_fields_only | Medium | TestStarWithoutAnIDHasCatalogFieldsOnly | — |
| star_shows_the_custom_values_of_a_product | Medium | TestStarShowsTheCustomValuesOfAProduct | — |
| custom_fields_in_group_by_and_order_by | Medium | TestCustomFieldsInGroupByAndOrderBy | — |
| describe_one_entity | Medium | TestDescribeOneEntity | — |
| describe_when_aha_cannot_be_reached | Medium | TestDescribeWhenAhaCannotBeReached | — |
| describe_without_entity | Medium | TestDescribeWithoutEntity | — |
| describe_shows_a_field_that_is_set_in_insert_only | Medium | TestDescribeShowsAFieldThatIsSetInInsertOnly | — |
| insert_with_other_spellings | Medium | TestInsertWithOtherSpellings | — |
| insert_comment_with_apostrophe | Medium | TestInsertCommentWithApostrophe | — |
| insert_without_parent | Medium | TestInsertWithoutParent | — |
| insert_with_a_product_as_release | Medium | TestInsertWithAProductAsRelease | — |
| insert_under_an_unknown_release | Medium | TestInsertUnderAnUnknownRelease | — |
| insert_goal_without_a_success_metric | Medium | TestInsertGoalWithoutASuccessMetric | — |
| update_status_in_any_case | Medium | TestUpdateStatusInAnyCase | — |
| update_assigned_to_me | Medium | TestUpdateAssignedToMe | — |
| update_with_a_true_guard | Medium | TestUpdateWithATrueGuard | — |
| update_with_a_false_guard | Medium | TestUpdateWithAFalseGuard | — |
| update_with_a_guard_for_no_value | Medium | TestUpdateWithAGuardForNoValue | — |
| update_of_an_unknown_record | Medium | TestUpdateOfAnUnknownRecord | — |
| update_sends_empty_text | Medium | TestUpdateSendsEmptyText | — |
| update_with_a_date_without_quotes | Medium | TestUpdateWithADateWithoutQuotes | — |
| update_of_the_name_shows_it_once | Medium | TestUpdateOfTheNameShowsItOnce | — |
| update_sets_a_custom_field | Medium | TestUpdateSetsACustomField | — |
| update_with_a_custom_guard | Medium | TestUpdateWithACustomGuard | — |
| dry_run_shows_a_list_as_text | Medium | TestDryRunShowsAListAsText | — |
| delete_one_feature | Medium | TestDeleteOneFeature | — |
| delete_a_release_through_its_product | Medium | TestDeleteAReleaseThroughItsProduct | — |
| delete_with_a_false_guard | Medium | TestDeleteWithAFalseGuard | — |
| delete_of_an_unknown_record | Medium | TestDeleteOfAnUnknownRecord | — |
| delete_with_a_custom_guard | Medium | TestDeleteWithACustomGuard | — |
| doubled_apostrophe_in_text | Medium | TestDoubledApostropheInText | — |
| date_without_quotes | Medium | TestDateWithoutQuotes | — |
| text_that_is_not_closed | Medium | TestTextThatIsNotClosed | — |
| unexpected_character | Medium | TestUnexpectedCharacter | — |
| apostrophe_inside_a_word_is_read_as_text | Medium | TestApostropheInsideAWordIsReadAsText | — |
| strict_reading_is_kept | Medium | TestStrictReadingIsKept | — |
| statements_without_select | Medium | TestStatementsWithoutSelect | — |
| text_after_the_statement | Medium | TestTextAfterTheStatement | — |
| join_is_refused | Medium | TestJoinIsRefused | — |
| id_or_entity_alone | Medium | TestIDOrEntityAlone | — |
| describe_takes_nothing_else | Medium | TestDescribeTakesNothingElse | test written after the implementation (see Test Refinements) |
| apostrophe_hint_in_an_error | Medium | TestApostropheHintInAnError | — |
| write_without_an_id_is_refused | Medium | TestWriteWithoutAnIDIsRefused | — |
| write_without_where_is_refused | Medium | TestWriteWithoutWhereIsRefused | — |
| insert_with_two_rows_is_refused | Medium | TestInsertWithTwoRowsIsRefused | — |
| insert_with_an_unknown_column_is_refused | Medium | TestInsertWithAnUnknownColumnIsRefused | — |
| update_of_a_field_that_is_set_in_insert_only | Medium | TestUpdateOfAFieldThatIsSetInInsertOnly | — |
| assignment_of_a_word_or_an_expression | Medium | TestAssignmentOfAWordOrAnExpression | — |
| null_cannot_be_written | Medium | TestNullCannotBeWritten | — |
| write_to_a_read_only_entity | Medium | TestWriteToAReadOnlyEntity | — |
| products_cannot_be_changed | Medium | TestProductsCannotBeChanged | — |
| label_of_a_custom_field | Medium | TestLabelOfACustomField | — |
| labels_of_select_items | Medium | TestLabelsOfSelectItems | — |
| keys_and_arguments_that_are_refused | Medium | TestKeysAndArgumentsThatAreRefused | — |
| a_row_has_each_label_once | Medium | TestARowHasEachLabelOnce | — |
| a_field_that_is_given_twice | Medium | TestAFieldThatIsGivenTwice | — |
| values_that_do_not_fit_the_type | Medium | TestValuesThatDoNotFitTheType | — |
| insert_with_a_custom_field_is_refused | Medium | TestInsertWithACustomFieldIsRefused | — |
| custom_field_that_cannot_be_set | Medium | TestCustomFieldThatCannotBeSet | — |
| describe_and_read_have_the_same_fields | Medium | TestDescribeAndReadHaveTheSameFields | — |
| goal_status_from_the_success_metric | Medium | TestGoalStatusFromTheSuccessMetric | — |
| release_of_an_epic | Medium | TestReleaseOfAnEpic | — |
| statuses_of_an_entity | Medium | TestStatusesOfAnEntity | — |
| current_user | Medium | TestCurrentUser | — |
| comments_of_a_record | Medium | TestCommentsOfARecord | — |
| custom_fields_per_record_type | Medium | TestCustomFieldsPerRecordType | — |
| custom_fields_are_read_by_name | Medium | TestCustomFieldsAreReadByName | — |
| unknown_custom_field_lists_the_custom_fields | Medium | TestUnknownCustomFieldListsTheCustomFields | — |
| custom_field_of_a_type_that_is_not_supported | Medium | TestCustomFieldOfATypeThatIsNotSupported | — |
| entity_without_custom_fields | Medium | TestEntityWithoutCustomFields | — |
| custom_fields_that_the_key_cannot_read | Medium | TestCustomFieldsThatTheKeyCannotRead | — |
| custom_field_that_was_added_later | Medium | TestCustomFieldThatWasAddedLater | — |
| definitions_are_not_kept_in_http_mode | Medium | TestDefinitionsAreNotKeptInHTTPMode | — |
| own_name_of_the_entity | Medium | TestOwnNameOfTheEntity | — |
| unknown_field_lists_the_fields | Medium | TestUnknownFieldListsTheFields | — |
| unknown_entity_lists_the_entities | Medium | TestUnknownEntityListsTheEntities | — |
| users_cannot_be_filtered_by_product | Medium | TestUsersCannotBeFilteredByProduct | — |
| bare_key_of_a_custom_field | Medium | TestBareKeyOfACustomField | — |
| catalog_name_wins_over_a_custom_key | Medium | TestCatalogNameWinsOverACustomKey | — |
| product_in_three_spellings | Medium | TestProductInThreeSpellings | — |
| the_product_list_is_not_kept | Medium | TestTheProductListIsNotKept | — |
| products_are_not_mixed | Medium | TestProductsAreNotMixed | — |
| unknown_product_lists_the_keys | Medium | TestUnknownProductListsTheKeys | — |
| release_id_where_a_product_is_needed | Medium | TestReleaseIDWhereAProductIsNeeded | — |
| in_list_reads_each_parent | Medium | TestInListReadsEachParent | — |
| count_needs_one_list_request | Medium | TestCountNeedsOneListRequest | — |
| every_page_is_read_once | Medium | TestEveryPageIsReadOnce | — |
| too_many_pages | Medium | TestTooManyPages | — |
| release_by_name_lists_the_releases | Medium | TestReleaseByNameListsTheReleases | — |
| parent_that_does_not_exist | Medium | TestParentThatDoesNotExist | — |
| release_id_where_a_feature_id_is_needed | Medium | TestReleaseIDWhereAFeatureIDIsNeeded | — |
| comments_need_a_parent | Medium | TestCommentsNeedAParent | — |
| answers_in_the_json_form_of_aha | Medium | TestAnswersInTheJSONFormOfAha | — |
| answer_with_an_unknown_form | Medium | TestAnswerWithAnUnknownForm | — |
| custom_fields_are_requested_only_when_used | Medium | TestCustomFieldsAreRequestedOnlyWhenUsed | — |
| list_without_custom_fields_is_an_error | Medium | TestListWithoutCustomFieldsIsAnError | — |
| in_list_is_checked_before_the_first_request | Medium | TestInListIsCheckedBeforeTheFirstRequest | — |
| description_of_a_product_is_requested_when_it_is_used | Medium | TestDescriptionOfAProductIsRequestedWhenItIsUsed | — |
| custom_field_of_a_product | Medium | TestCustomFieldOfAProduct | — |
| parking_lot_and_missing_date | Medium | TestParkingLotAndMissingDate | — |
| description_is_plain_text | Medium | TestDescriptionIsPlainText | — |
| custom_values_by_kind | Medium | TestCustomValuesByKind | — |
| or_and_not_are_evaluated | Medium | TestOrAndNotAreEvaluated | — |
| me_as_a_value | Medium | TestMeAsAValue | — |
| fields_without_a_value_in_conditions | Medium | TestFieldsWithoutAValueInConditions | — |
| conditions_on_custom_fields | Medium | TestConditionsOnCustomFields | — |
| fields_with_several_values | Medium | TestFieldsWithSeveralValues | — |
| timestamps_are_points_in_time | Medium | TestTimestampsArePointsInTime | — |
| empty_text_and_null_stand_for_no_value | Medium | TestEmptyTextAndNullStandForNoValue | — |
| grouping_without_group_by | Medium | TestGroupingWithoutGroupBy | — |
| rows_without_a_value_sort_last | Medium | TestRowsWithoutAValueSortLast | — |
| count_of_nothing_is_zero | Medium | TestCountOfNothingIsZero | — |
| limit_of_the_statement_keeps_the_total | Medium | TestLimitOfTheStatementKeepsTheTotal | — |
| ids_are_sorted_by_their_numbers | Medium | TestIDsAreSortedByTheirNumbers | — |
| order_by_a_label_and_by_an_aggregate | Medium | TestOrderByALabelAndByAnAggregate | — |
| records_without_a_value_are_one_group | Medium | TestRecordsWithoutAValueAreOneGroup | — |
| tags_are_split_at_commas | Medium | TestTagsAreSplitAtCommas | — |
| unknown_status_lists_the_statuses | Medium | TestUnknownStatusListsTheStatuses | — |
| status_of_an_epic | Medium | TestStatusOfAnEpic | — |
| date_that_is_not_a_date | Medium | TestDateThatIsNotADate | — |
| user_by_name | Medium | TestUserByName | — |
| custom_values_are_checked_by_kind | Medium | TestCustomValuesAreCheckedByKind | — |
| option_that_is_not_in_the_list | Medium | TestOptionThatIsNotInTheList | — |
| body_of_an_update_request | Medium | TestBodyOfAnUpdateRequest | — |
| body_of_a_create_request | Medium | TestBodyOfACreateRequest | — |
| body_with_custom_fields | Medium | TestBodyWithCustomFields | — |
| bodies_for_a_goal | Medium | TestBodiesForAGoal | — |
| request_address_and_headers | Medium | TestRequestAddressAndHeaders, TestWriteRequestHasAContentType | — |
| request_through_the_proxy_of_the_environment | Medium | TestRequestThroughTheProxyOfTheEnvironment | — |
| certificate_that_the_trust_store_does_not_have | Medium | TestCertificateThatTheTrustStoreDoesNotHave | — |
| proxy_does_not_open_the_tunnel | Medium | TestProxyDoesNotOpenTheTunnel, TestManualProxyThatCannotBeReached | — |
| too_many_requests_is_sent_again | Medium | TestTooManyRequestsIsSentAgain | — |
| too_many_requests_three_times | Medium | TestTooManyRequestsThreeTimes | — |
| redirect_is_not_followed | Medium | TestRedirectIsNotFollowed | — |
| credential_that_aha_does_not_accept | Medium | TestCredentialThatAhaDoesNotAccept | — |
| write_request_without_an_answer | Medium | TestWriteRequestWithoutAnAnswer, TestWriteRequestThatAhaRejects | — |
| statements_in_messages_can_be_sent | Medium | TestStatementsInMessagesCanBeSent | through the black-box interface, "accepted by parse-statement" means "no syntax or name error from the tool" |
| articles_and_ids_in_messages | Medium | TestArticlesAndIDsInMessages | — |
| example_key_is_the_default_product | Medium | TestExampleKeyIsTheDefaultProduct | — |
| example_key_without_a_default_product | Medium | TestExampleKeyWithoutADefaultProduct | — |
| example_key_is_not_a_product_of_the_account | Medium | TestExampleKeyIsNotAProductOfTheAccount | — |
| default_product_that_the_account_does_not_have | Medium | TestDefaultProductThatTheAccountDoesNotHave | — |
| example_key_is_chosen_when_it_is_needed | Medium | TestExampleKeyIsChosenWhenItIsNeeded, TestExampleKeyIsKeptForTheProcess | — |
| example_key_when_the_product_list_cannot_be_read | Medium | TestExampleKeyWhenTheProductListCannotBeRead | — |

Template EXAMPLES (mcp-server.template.md):

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| minimal_tool_server_stdio | Medium | TestQueryResultEnvelope, TestUnknownToolIsAProtocolError (exit 0 on EOF) | — |
| minimal_tool_server_http | Medium | TestHTTPRequestBringsItsAccount, TestHTTPRequestsShareNothing | — |
| unknown_transport_argument / unknown_first_word_with_command_words | Medium | TestUnknownFirstWord | — |
| tool_error_handling | Low | reasoning only (mcp-go WithRecovery) | the stderr log entry for an internal error is not produced; see "Rules that could not be implemented exactly" |
| declared_tool_execution_error | Medium | every wantError assertion (isError true, one text, no JSON-RPC error) | — |
| graceful_shutdown_http | Medium | TestGracefulShutdownHTTP, TestGracefulShutdownStdio | draining of a request in flight during SIGTERM is not tested |
| http_bind_error | Medium | TestHTTPBindError, TestListenAddressThatIsNotLoopback | — |
| setting_from_declared_environment_variable | Medium | TestSettingsFromTheEnvironment, TestArgumentsTakePrecedence | — |
| framework_mcp_go_default | Medium | `go.mod` requires `github.com/mark3labs/mcp-go v1.6.0`; this report: "GO-FRAMEWORK: mcp-go (template default)" | — |
| framework_go_sdk_override | — | not applicable (no preset override) | — |

### Unverified claims

- The forms of the real Aha! API (PRECONDITIONS of the spec) are verified
  only against FakeAha, which follows the INTERFACES section. No request
  went to a live Aha! account.
- The RPM, DEB and OCI builds were not executed (no rpmbuild,
  dpkg-buildpackage or podman on the build host).
- Windows and macOS were not tested.
- No test drains a request that is in flight while SIGTERM arrives.
