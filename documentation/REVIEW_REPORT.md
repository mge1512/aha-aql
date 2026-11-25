# Review Report

<!-- variant: us -->

| Field | Value |
|---|---|
| Component | aha-aql (binary `mcp-server-aha-aql`) |
| Spec version | 0.13.0 |
| Spec-SHA256 | `9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262` |
| Code under review | github.com/mge1512/aha-aql, commit `bd457b25d39352c80830b4dcbce8fe7e47062809`, directory `code/` |
| Paths | As in the commits `bd457b2` and `d2b5174`. Commit `638583a` of 2026-11-25 moved the files without a change of their content: the code is in `implementation/code/`, the specification is `specification/aha-aql.spec.md`, the inputs of the run are in `implementation/spec/`, the template and the hints are also in `pcd-artefacts/`, and the changelog is in `documentation/` |
| Translator model | claude-opus-5-5 (one run, mode translator, tests by the translator) |
| Reviewer model | claude-fable-5-2 |
| Review date | 2026-11-24 |
| Review procedure | pcd-translate 0.3.1, `modes/review-procedure.md` (reviewer.md 0.1.0, SHA256 `b290f7e2c1f7fdab50363240c1a85354172923760184036c39f5cb099c27185e`) |
| Updated | 2026-11-24, after the answers of the maintainer: the section "Answers of the maintainer", the column State, F-028, F-029, the new F-030 and the recommendation. 2026-11-25: F-029 and F-030 after the statement on the vendor tarball, the package build in F-030, and the row Paths. Later that day: the decisions on the name and the label, and pull request 6 of the PCD repository |

## Summary

The code is a faithful translation of specification 0.13.0. Its 187 tests
pass in this session, and so do the 20 offline acceptance commands of the
milestones. I sent 583 further statements through the test double. They
gave what the specification asks for, with one small exception (F-018).
The 30 findings are therefore almost all about the specification. Six
findings have the weight "high" (F-001 to F-006). In each of them the code
follows a rule of the specification and returns a wrong or misleading
result without an error.

Recommendation: EDIT-AND-REGENERATE. The last section proposes an order.

State on 2026-11-24, after the answers of the maintainer: the units 1 to
8 of that order are written. They are the versions 0.14.0 to 0.20.0 of
the specification, with one diff, one changelog entry and one pull
request text each. Unit 9 waits for two checks on the test account. No
version after 0.13.0 is translated yet.

Findings by class:

| Class | Count | Findings |
|---|---|---|
| SPEC-INCOMPLETENESS | 15 | F-001, F-002, F-003, F-005 to F-010, F-013, F-014, F-021, F-022, F-025, F-026 |
| SPEC-AMBIGUITY | 10 | F-004, F-011, F-012, F-015, F-016, F-017, F-019, F-020, F-023, F-024 |
| TRANSLATION-ERROR | 2 | F-018, F-030 |
| HINTS-GAP | 2 | F-027, F-029 |
| TEMPLATE-GAP | 1 | F-028 |
| REPORT-GAP, LICENSE-DEVIATION, UNCLASSIFIED | 0 | |

Provenance of the review:

```
Currency:              Spec-SHA256 of TRANSLATION_REPORT.md equals sha256sum of spec/aha-aql.spec.md
Template-SHA256:       mcp-server.template.md 22c55b355e28a6f2635413e72f2281faa3f7f1c79f79ea4eca76aba0b5b37b3e (as the report says)
Library-Hints-SHA256:  mcp-server.go.mcp-go.hints.md 3bcd5bb41d5c33057fe7bf9b21f0baf055755892691b057fdf0078b15ac20c82 (as the report says)
Prompt-SHA256:         prompt.md 702ec28de611ff65d76b2819cb53c675867fb58b1c090c6b17192cc73470d7d3 (as the report says)
spec: (pcd-slice):     none
Bundles:               none. The repository has no spec.d. The specification file was read.
Skill-SHA256:          pcd-translate/SKILL.md 582ba0ac76f667cfebd8af4454824ba74b419de15805b9534695b502022accf6
Skill-Manifest-SHA256: f5962b5b5a480b8ffba8ba9815700be4a6f65ec9c31413a2bf96d9efb5d43b32
Toolchain:             none. The guard reports "pcd-lint missing" and "pcd-slice missing" (exit 1).
Guard-Override:        none
```

## What was checked

| Check | Result |
|---|---|
| Build from the sources of the commit, Go 1.24.7, modules from their repositories | The binary reports `mcp-server-aha-aql 0.13.0 spec:9f8c797a...`. The content of the modules matches `go.sum` |
| Tests of the repository | 187 passed, 0 failed, 0 skipped, 121.6 seconds |
| Acceptance commands of the milestones 0.0.0, 0.1.0 and 0.5.0 that need no network | 20 of 20 pass |
| Catalog against `DESCRIBE` of the binary | 116 fields of 12 entities: same names, same order, same column Writable |
| API sources of the catalog against the parameter `fields` of the list requests | Same members in the same order for the 10 entities that have a list |
| Further statements through the test double (`aha-aql.review-probes.txt`) | 583 statements, 409 for `query` and 174 for `execute`. 379 results and 204 errors. The findings below quote them. Part 9 of the file has 34 more statements, sent after the answers of the maintainer for the cases of unit 5 |
| Spec hash and license in the artefacts | Every tracked file under `code/` except `go.sum` has the hash. `LICENSE` has the SPDX identifier MIT and a link, and no license text. `debian/copyright` agrees with the license files of the five modules |
| Package build with rpmbuild 4.18.2, downloads switched off (2026-11-25) | The RPM file as it is fails in `%build`. With a second source for the vendor tarball the package builds, and the tests pass in `%check` (F-030) |

Weights used below:

- high: the code follows the specification and returns a wrong or
  misleading result without an error.
- medium: an error or a message that misleads and is visible, or a rule
  that allows two servers which the examples cannot both satisfy.
- low: wording, small rules that the translator had to choose, packaging.

## Findings at a glance

The column State has the version of the specification that has the
edit, or it says what is still open.

| ID | Weight | Subject | Place in the specification | State |
|---|---|---|---|---|
| F-001 | high | The note H03 says that a record does not exist when it exists | read-statement, step 8 | 0.14.0 |
| F-002 | high | Text with numbers is ordered character by character: DEMO-10 before DEMO-2 | build-rows, step 4 | 0.15.0 |
| F-003 | high | A value of another form for a date, a timestamp, a number or a boolean gives a wrong result | evaluate-condition, steps 6 and 9 | 0.16.0 |
| F-004 | high | NULL in an IN list has two readings | evaluate-condition, step 3 | 0.16.0 |
| F-005 | high | `=` and IN on a field with several values compare the joined text of all values | evaluate-condition | 0.17.0 |
| F-006 | high | A write result is not compared with the values that were sent | update-statement, step 8 | open: unit 9 |
| F-007 | medium | A field that a write statement assigns twice | parse-statement | 0.18.0 |
| F-008 | medium | A result object with the same member twice | update-statement, step 8; parse-statement | 0.18.0 |
| F-009 | medium | Empty text in a condition and in a guard | evaluate-condition, parse-statement | 0.16.0 |
| F-010 | medium | A date, a user, a number or a reference cannot be emptied | check-write-values | open: unit 9 |
| F-011 | medium | W25 for a requirement shows a statement that the server refuses | error-messages | 0.18.1 |
| F-012 | medium | Lists "may be kept", and the examples count requests | resolve-product, custom-fields, check-write-values | 0.19.0 |
| F-013 | medium | Examples that a black-box test cannot check | INTERFACES, aha-request, EXAMPLES | 0.20.0 |
| F-014 | medium | A write that names a record that does not exist: no rule of FakeAha, no example | update-statement, step 7 | open: unit 9 |
| F-015 | low | Wording of messages: a and an, H03 for products, S05, H07 | error-messages | 0.18.1 |
| F-016 | low | An example lists the fields of goals without `success_metric` | EXAMPLES | 0.18.2 |
| F-017 | low | Grammar: `*` in SUM, the label of COUNT(1), the form of a label | parse-statement | 0.15.0 |
| F-018 | low | ORDER BY accepts keys that the grammar does not have | parse-statement | 0.15.0 |
| F-019 | low | Grouped results: a key outside the group fields, records without a value | build-rows | 0.15.0 |
| F-020 | low | IN list as a route: repeated values, requests before the check | plan-fetch, step 5 | 0.19.0 |
| F-021 | low | statuses: a value of `entity` without a match gets no note | read-statement, step 8 | 0.14.0 |
| F-022 | low | Two annotations of `query` are not stated | tool-registration | 0.20.0 |
| F-023 | low | Form and text of warnings | load-configuration, choose-example-key | 0.20.0 |
| F-024 | low | The INVARIANT on environment variables cannot be met as written | INVARIANTS | 0.20.0 |
| F-025 | low | `description` of products is requested with every product list | entity products | 0.19.0 |
| F-026 | low | META has no `Module:` | META | 0.20.0 |
| F-027 | low | The pattern of Credential and the repeat limit of the Go package regexp | hints | open: hints |
| F-028 | low | Container image: trust store and listen address | template | 0.20.0; open: template |
| F-029 | low | No target prepares `vendor/`, and no rule keeps it out of the source tarball | hints | open: pull request 6 of the PCD repository |
| F-030 | low | The RPM file expects `vendor/` inside the source tarball | artefact | open: next translation |

## Findings

Line numbers are those of `spec/aha-aql.spec.md` 0.13.0. "Follows the
step" means that the code does what the specification says, so the fix is
in the specification. All statements ran on the fixture account.

### F-001 SPEC-INCOMPLETENESS: the note H03 says that a record does not exist when it exists

- Weight: high
- Location (spec): read-statement, steps 8 and 8 a (lines 998 and 999); message H03 (line 2774)
- Location (artefact): `read.go`, function `emptyNotes`; follows the step
- Evidence: DEMO-1 exists and has the status "In design".
  `SELECT * FROM features WHERE id = 'DEMO-1' AND status = 'Shipped'`
  returns no record and the note `There is no feature with the id 'DEMO-1'.`
  `SELECT id FROM features WHERE id = 'DEMO-1' ORDER BY name LIMIT 0`
  returns `"count":0,"total":1` and the same note.
  `WHERE id = 'DEMO-1' AND id = 'DEMO-2'` gets the note for both ids.
- Proposed edit: step 8 applies "when no record was kept in step 5"; today
  it applies "when there is no row". Step 8 a becomes: "The id field of the
  entity with the operator `=`. When plan-fetch read the id route for this
  value and found no record: H03. When it found the record: the new note
  H08, `The <singular> <ID> exists. It does not satisfy the other conditions
  of the statement.`" One EXAMPLE for each case.
- Rationale: a language model that reads "there is no feature" can create
  the feature a second time.

### F-002 SPEC-INCOMPLETENESS: text with numbers is ordered character by character

- Weight: high
- Location (spec): build-rows, step 4 (line 2289); evaluate-condition, step 9 (line 2242)
- Location (artefact): `read.go`, functions `orderCmp` and `compare`; follow the steps
- Evidence: with the features DEMO-1 to DEMO-11, `ORDER BY id` gives
  DEMO-1, DEMO-10, DEMO-11, DEMO-2 and so on. `ORDER BY id DESC LIMIT 1`
  gives DEMO-9, and so does `SELECT MAX(id)`. `id > 'DEMO-9'` and
  `id BETWEEN 'DEMO-2' AND 'DEMO-10'` give no record.
- Proposed edit: build-rows, step 4: "Text is sorted by its comparison
  form. A run of digits inside text is compared as a number: `DEMO-9`
  comes before `DEMO-10`." evaluate-condition, step 9: "Every other field
  is compared in the order of build-rows, step 4." MIN and MAX already use
  that order. One EXAMPLE with eleven features.
- Rationale: "the newest feature" and "the features after DEMO-9" are
  ordinary questions. Today the answer is wrong and looks valid.

### F-003 SPEC-INCOMPLETENESS: a value of another form for a date, a timestamp, a number or a boolean gives a wrong result

- Weight: high
- Location (spec): evaluate-condition, steps 6 and 9 (lines 2234 to 2242), ERRORS "none"
- Location (artefact): `read.go`, functions `compare` and `valueEq`; follow the steps
- Evidence: every feature of the fixture has the updated_at
  `2026-02-02T08:00:00.000Z`. No statement below gets an error or a note.
  - `updated_at = '2026-02-02T08:00:00Z'` gives no record. With `.000Z` it gives all six.
  - `updated_at >= '2026-02-02T09:00:00+01:00'`, the same point in time, gives no record.
  - `updated_at > '2026-02-02 23:00'` gives all six.
  - `created_at > '01.02.2026'` gives all six, and `created_at > 'yesterday'` gives none.
  - `release_date >= '2026-06-30T00:00:00Z'` leaves out the release of 2026-06-30.
  - `score > 'abc'` gives none. `released > 0` gives all four releases, and all four have released false.
- Proposed edit, in three parts:
  1. parse-statement, for a read statement and for the guards of a write
     statement: "The value of a comparison with `=`, `!=`, `<`, `<=`, `>`,
     `>=`, IN or BETWEEN must fit the type of the field. It is checked
     before the records are requested."
     - number: a Number, or Text that reads as a number.
     - boolean: TRUE, FALSE, 1, 0, 'true' or 'false', and only with `=`
       and `!=`.
     - date: a DateText. Of a timestamp the first 10 characters are
       taken, as check-write-values, step 5, does.
     - datetime: a DateText or a timestamp. A timestamp is a date, then
       `T` or a space, then hh:mm. Seconds, a fraction, and `Z` or an
       offset such as `+01:00` can follow. Without `Z` and without an
       offset it is UTC.
     - Any other value is an error of the kind bad_value, with a new
       message: `<field> is a <type text>. Write the value as <form>. Got
       '<value>'.`
  2. evaluate-condition, steps 6 and 9: "A datetime field and a timestamp
     are compared as points in time. A datetime field and a DateText are
     compared by the day in UTC." CONTAINS and LIKE keep comparing the
     text of the field.
  3. INTERFACES, "Values in an API record": the form in which Aha! writes
     a timestamp. The specification does not say it, and the checks of
     2026-11-24 did not record it. EXAMPLES for an offset, for a value
     without a fraction and for the new message.
- Rationale: a model writes a timestamp in the form it knows. The result
  has to be the same with and without digits after the seconds.

### F-004 SPEC-AMBIGUITY: NULL in an IN list

- Weight: high
- Location (spec): parse-statement, grammar of `value` and the rule on `field = NULL` (lines 1304 to 1322); evaluate-condition, steps 3 and 6 (lines 2228 and 2234); plan-fetch, step 5 (line 2118)
- Location (artefact): `read.go`, functions `planFetch` and `eval`: NULL in a list is skipped. The translation report does not name this decision.
- Evidence:
  `SELECT id, initiative FROM features WHERE product = 'DEMO' AND initiative IN ('DEMO-S-1', NULL)`
  returns DEMO-3 and DEMO-4 with the scope `initiative DEMO-S-1`. The four
  features without an initiative are not in the result.
  `SELECT id FROM features WHERE release IN (NULL)` returns
  `{"entity":"features","scope":"","count":0,"records":[]}` and sends no
  request.
- The two readings: step 3 says that a list is true when `field = value`
  is true for one value, and parse-statement reads `field = NULL` as
  `field IS NULL`. So NULL in a list means "no value". Step 6 says that a
  field without a value equals nothing. So NULL in a list matches nothing.
- Proposed edit: evaluate-condition, step 3: "NULL in the list stands for
  a field without a value. The list is then also true for such a field."
  plan-fetch, step 5: "A list with NULL is not taken as a route."
  plan-fetch, POSTCONDITIONS: "the scope is never empty". The alternative
  with less text: refuse NULL in a list with a syntax message that shows
  `(<field> IN (...) OR <field> IS NULL)`.
- Rationale: this dialect reads `= NULL` as IS NULL so that a model does
  not fail on it. A list should not behave in another way.

### F-005 SPEC-INCOMPLETENESS: `=` and IN on a field with several values compare the joined text of all values

- Weight: high
- Location (spec): evaluate-condition, steps 5 to 8; convert-record, step 3 (tags); custom-fields, table of custom values (tags, fixed_tags, users)
- Location (artefact): `read.go`, function `equals`; follows the steps
- Evidence: an update gave DEMO-1 the tags `urgent, customer`. After
  that, `tags = 'urgent'` and `tags IN ('urgent')` give no record.
  `tags = 'customer, urgent'` gives none, and `tags = 'urgent, customer'`
  gives DEMO-1.
  `custom.markets = 'EMEA'` finds DEMO-3 (EMEA) and not DEMO-1 (EMEA, APAC).
  `custom.reviewers = 'me'` gives no record, and the current user is one of
  the two reviewers of DEMO-3. The EXAMPLE conditions_on_custom_fields
  shows the form that works, CONTAINS.
- Proposed edit: evaluate-condition, new step: "A field with several
  values is `tags` of features, and a custom field of the kind tags,
  fixed_tags or users. For such a field `=` is true when one value of the
  field equals the value of the statement. A value with commas is split as
  check-write-values, step 8, says. `=` is then true when the field has
  each of these values, in any order. `!=` is the negation. IN is true
  when `=` is true for one value of the list. A user of a users field is
  compared by e-mail address, name and NumericId, and `me` stands for the
  current user. CONTAINS and LIKE compare the joined text, as today." The
  smaller alternative: keep the rule and add a note to an empty result,
  after the pattern of H07: `<field> = compares the whole list. For one
  value use: <field> CONTAINS '<value>'`.
- Rationale: a model that looks for the features with the tag urgent is
  likely to write `tags = 'urgent'`, and the answer "none" looks valid.

### F-006 SPEC-INCOMPLETENESS: a write result is not compared with the values that were sent

- Weight: high as a risk. It was not observed for a field that the catalog marks as writable.
- Location (spec): update-statement, step 8 (line 1157); execute, step 5
- Location (artefact): `write.go`, function `update`; follows the step
- Evidence: on 2026-11-24 Aha! answered an update of the success metric
  of a goal with the status 200 and kept the old text. This is check 24
  of the notes and the reason for version 0.13.0. Other members can
  behave in the same way. One candidate is a custom field that the layout
  of the product does not have (check 18, not observed). The result would
  then be `"custom.x":null` with the message `Feature ... updated`. For a
  status the message is built from the answer: `status changed to
  '<status after the write>'`. When Aha! ignores the member, the message
  names the old status as the new one.
- Proposed edit: update-statement, new step after step 7: "Compare each
  assigned field with its value in the answer, with the `=` of
  evaluate-condition. A field of the type text and a custom field of the
  kind note are not compared, because Aha! stores them as HTML. For each
  field that differs, the WriteResult has a note: `Aha! did not store
  <field>. Its value is '<value>'.` The message is then `<Singular> <id>
  updated`." execute, step 5: "`note` follows `message`, only when there
  are notes." INVARIANTS: "no result says that a value was changed when
  the answer of Aha! shows another value". INTERFACES: "A test can tell
  FakeAha to ignore one member of an update."
- Rationale: version 0.13.0 closed this gap for one field. The comparison
  closes it for every field, also for those whose behavior depends on how
  an account is set up.

### F-007 SPEC-INCOMPLETENESS: a field that a write statement assigns twice

- Weight: medium
- Location (spec): parse-statement, rules for a write statement (lines 1353 to 1356); INVARIANT "an accepted write statement sends every field that it assigns" (line 2857)
- Location (artefact): `parser.go`; `write.go`, function `writeBody`
- Evidence:
  `UPDATE features SET status = 'Shipped', status = 'In design' WHERE id = 'DEMO-2'`
  sends `{"feature":{"workflow_status":"Shipped","workflow_status":"In design"}}`.
  Two names of one field do the same:
  `SET release = 'DEMO-R-3', parent = 'DEMO-R-1'` sends `release` twice,
  and `INSERT INTO comments (parent, body, text) VALUES ('DEMO-3', 'one', 'two')`
  sends `{"comment":{"body":"one","body":"two"}}`.
- Proposed edit: parse-statement: "A statement names each field once,
  whatever the spelling. A column of INSERT or an assigned field of UPDATE
  that names a field again is an error: syntax, new message `the field
  <field> is given twice`. The parent of INSERT keeps the rule of W09."
  An EXAMPLE with two of the statements above.
- Rationale: JSON does not define which of two equal members counts. One
  assignment is lost without an error.

### F-008 SPEC-INCOMPLETENESS: a result object with the same member twice

- Weight: medium
- Location (spec): update-statement, step 8 (line 1157); parse-statement, rule on labels (line 1325); build-rows, step 2 c
- Location (artefact): `write.go`, function `update`; `read.go`, function `buildRows`; follow the steps
- Evidence:
  `UPDATE goals SET name = 'Grow', progress = 40 WHERE id = 'DEMO-G-1'`
  returns
  `{"id":"DEMO-G-1","name":"Grow","name":"Grow","progress":40,"message":"Goal DEMO-G-1 updated"}`.
  Every update that assigns `name` does this.
  `SELECT COUNT(*) AS n, COUNT(*) AS n ...` returns `{"n":6,"n":6}`.
  `SELECT id, name AS id ...` returns `{"id":"DEMO-1","id":"CSV export"}`.
- Proposed edit: update-statement, step 8: "`values` has `name`, or
  `parent` for a comment, and then each assigned field that is not yet in
  it." parse-statement, rules for a read statement: "A select item that
  repeats an earlier item with the same label is left out. Two different
  items with one label are an error. So is a label that equals a group
  field which the row shows. Both are syntax errors with a new message:
  `two select items have the label '<label>'. Give one of them another
  label with AS`."
  INVARIANTS: "no JSON object of a result and no request body has two
  members with the same name".
- Rationale: a JSON reader keeps one of the two members, and which one
  depends on the reader.

### F-009 SPEC-INCOMPLETENESS: empty text in a condition and in a guard

- Weight: medium
- Location (spec): evaluate-condition, steps 2, 6 and 7; parse-statement, rule on `field = NULL` (line 1322) and rules for guards; plan-fetch, step 4 e
- Location (artefact): `read.go`, function `compare`; `parser.go`; follow the steps
- Evidence:
  `SELECT id FROM features WHERE product = 'DEMO' AND initiative = ''`
  gets the error
  `initiative = needs the id of the initiative, not its name. The initiatives of DEMO: DEMO-S-1 (Live coverage)`.
  `initiative != ''` is true for all six features, also for the four
  without an initiative. A guard cannot test for "no value":
  `... WHERE id = 'DEMO-2' AND description = ''` gets
  `nothing was changed: feature DEMO-2 has no description, not ''`, and
  `AND description IS NULL` is refused with W06.
- Proposed edit: parse-statement, rules for a read statement: "`field =
  ''` is read as `field IS NULL`, and `field != ''` as `field IS NOT
  NULL`. evaluate-condition, step 2, already takes empty text as no
  value." Rules for a write statement: "A guard `field = ''` is true when
  the field has no value."
- Rationale: "assign it to me when nobody has it" is an ordinary guard.
  Today no statement can say it.

### F-010 SPEC-INCOMPLETENESS: a date, a user, a number or a reference cannot be emptied

- Weight: medium
- Location (spec): check-write-values, steps 3, 5, 6, 7 and 10 b (lines 2329 to 2333); update-statement, POSTCONDITIONS (line 1163); custom-fields, "Empty text is a value" (line 1955); message W11
- Location (artefact): `write.go`, function `checkByType`; follows the steps
- Evidence: `UPDATE features SET due_date = '' WHERE id = 'DEMO-2'` gets
  `due_date: invalid date '', use YYYY-MM-DD`. `SET assigned_to = ''` gets
  `assigned_to: there is no user ''. Use an e-mail address, or 'me' for yourself`.
  `SET release = ''` gets `there is no release with the id ''. To find the id, ...`.
  `SET custom.go_live = ''` and `SET custom.effort_days = ''` are refused
  in the same way. Against the test double, empty text does empty the
  description, the tags and the custom kinds text, note, choice,
  fixed_choice, tags and users. NULL cannot be written (W11). So a due
  date, an assignee, an initiative and a custom date or number cannot be
  removed, and no message says so.
- Proposed edit, in two parts. Now: one new message for empty text in
  these fields, `<field> cannot be emptied with this server. Empty text is
  accepted for text fields and tags`. Later, after a check on the test
  account: a table in check-write-values. It says for each type whether
  empty text empties the field, and what is sent. The check is a new part
  of `aha-checks.sh`. It finds out which request empties `due_date`,
  `assigned_to_user`, `initiative` and a custom date or number: empty text
  or the JSON null.
- Rationale: removing a value is an ordinary change. Section 6 of the
  notes covers it for custom text and custom lists only (check 16).

### F-011 SPEC-AMBIGUITY: W25 for a requirement shows a statement that the server refuses

- Weight: medium
- Location (spec): error-messages, W25 (lines 2680 and 2731); INVARIANT on example statements (line 2862). Point 6 of the translation report.
- Location (artefact): `write.go`, function `errW25`; follows the text
- Evidence: `INSERT INTO comments (parent, body) VALUES ('DEMO-1-9', 'c')`
  gets
  `there is no requirement with the id 'DEMO-1-9'. To find the id, use the query tool: SELECT id, name FROM requirements WHERE product = 'DEMO'`.
  That statement gets `unknown field 'product' on requirements. ...`.
- Proposed edit: error-messages: "In W25, for the kind requirement the
  statement is `SELECT id, name FROM requirements WHERE feature =
  '<FEATURE>'`. `<FEATURE>` is the value without its last hyphen and
  number when the value has the form of a requirement id. In every other
  case it is `<KEY>-23`." Add the statement to the EXAMPLE
  statements_in_messages_can_be_sent.
- Rationale: the message and the INVARIANT cannot both be true.

### F-012 SPEC-AMBIGUITY: lists "may be kept", and the examples count requests

- Weight: medium
- Location (spec): resolve-product, steps 1 and 3 (lines 2052 and 2054); custom-fields, steps 2 c and 6 c (lines 1883 and 1890) and the paragraph on options; check-write-values, steps 9 a, 9 d and 10 c (lines 2336 and 2339); INTERFACES (lines 430 and 431); EXAMPLE record_by_id_has_all_fields (line 3374). Point 2 of the translation report.
- Location (artefact): `session.go`, type `run`: nothing is kept after a statement
- Evidence: INTERFACES says that the tool list was answered before every
  example, so the product list was read for the ExampleKey. The example
  then counts one ApiList call on /products for its own statement. A
  server that keeps the list, which the steps allow, fails the example.
  The translator kept nothing, and the "read again" steps are never used.
  The price: of the 451 statements of this review that sent a request,
  271 also read the product list.
- Proposed edit: resolve-product, step 1, and custom-fields, step 2 c,
  for both modes: "The list is read at most once per statement and is not
  kept after it." Delete the "read again" steps and sentences, and adjust
  the POSTCONDITIONS that speak of a second reading.
  choose-example-key: "Between statements the process keeps the
  ExampleKey and nothing else." If the extra request costs too much
  later, a cache gets a unit of its own, with a lifetime and examples.
- Rationale: a "may" allows two servers with different request counts,
  and the examples count requests.

### F-013 SPEC-INCOMPLETENESS: examples that a black-box test cannot check

- Weight: medium
- Location (spec): INTERFACES, "Test double: FakeAha" (line 403); aha-request, MECHANISM (line 2462); execute, MECHANISM (line 642); EXAMPLES execute_writes_a_log_line, execute_logs_a_refused_statement, too_many_requests_is_sent_again, too_many_requests_three_times, request_through_the_proxy_of_the_environment, host_of_aha_without_the_proxy, proxy_does_not_open_the_tunnel. Points 10 to 12 of the translation report.
- Location (artefact): `independent_tests/claude-opus-5-5/fakeaha_test.go` and `harness_test.go`
- Evidence: the translation prompt asks for black-box tests that start
  the binary. The specification describes FakeAha as a replacement of the
  HTTPS connection that calls the proxy function of the server, and a
  clock that tests replace. Both need code of the test inside the server
  process. The translator made FakeAha an HTTPS proxy with a certificate
  authority of the test (HTTPS_PROXY, SSL_CERT_FILE), the way the
  specification describes for a sandbox. The consequences:
  - The two log examples are checked for the form of the time only.
  - The tests of the two examples for the status 429 wait about 2 and 10 seconds, and three other tests wait 60 seconds each.
  - The address `http://10.200.0.1:3128` of three examples is replaced by the address of FakeAha.
  - For host_of_aha_without_the_proxy the binary tries a direct
    connection to `example.aha.io`. On a machine with a route to the
    internet this is a DNS request and a TLS handshake with a host of
    Aha!. No credential is sent, because the certificate check fails
    first.
- Proposed edit: INTERFACES: "FakeAha is an HTTPS server outside the
  server process. A test gives the server the address of FakeAha in
  HTTPS_PROXY and the certificate authority of the test in SSL_CERT_FILE,
  as a sandbox does. The server has no hook for tests." Remove the two
  MECHANISM sentences on a replaceable connection and clock. Write the
  THEN lines as facts that a test sees from outside:
  - "the time of the log line is between the start and the end of the call"
  - "the second request arrives at least 1 second after the first"
  - "the proxy of the test" in place of the literal address

  Remove host_of_aha_without_the_proxy from EXAMPLES, and keep its rule
  in STEPS and MECHANISM. No test can check it without a connection
  attempt to a host of Aha!.
- Rationale: an example is the acceptance evidence of a behavior. Seven
  examples can be checked today only in a changed form.

### F-014 SPEC-INCOMPLETENESS: a write that names a record that does not exist

- Weight: medium
- Location (spec): update-statement, step 7 (line 1156), the exit W25 "when the statement assigns exactly one reference"; insert-statement, step 6; INTERFACES, rules of FakeAha
- Location (artefact): `fakeaha_test.go`: a reference in a body is stored without a check
- Evidence: `UPDATE features SET release = 'DEMO-R-99' WHERE id = 'DEMO-2'`
  returns `"release":"DEMO-R-99"` and `Feature DEMO-2 updated`.
  `INSERT INTO features (parent, name, initiative) VALUES ('DEMO-R-3', 'd8', 'DEMO-S-99')`
  creates the feature. No test reaches the exit W25 of update-statement.
  What Aha! answers to such a request was not observed.
- Proposed edit: first a check on the test account. It sends two updates
  that name a record which does not exist, one in `release` and one in
  `initiative`. It prints the status and the stored value of each answer.
  Then one of two edits. When Aha! answers with the status 404: the rule
  "ApiCreate and ApiUpdate with a reference to a record that the fixture
  does not have are answered with the status 404" for FakeAha, and an
  EXAMPLE for the exit. When Aha! answers with success: the removal of the
  exit, because F-006 then reports the value. Another error status needs
  no edit, because W30 reports it.
- Rationale: a STEP with an error exit needs an example that reaches it.

### F-015 SPEC-AMBIGUITY: wording of messages

- Weight: low
- Location (spec): error-messages, K01 to K03, P02 to P04, W19, W24, H03 (line 2774), H07 (line 2778), S05 (line 2561). Point 9 of the translation report covers S05.
- Location (artefact): `read.go`, function `article`: the code has an article rule and uses it for DESCRIBE only, where the specification writes `id of an initiative`
- Evidence: `'DEMO' is a product, not a epic.` (K03).
  `'DEMO-S-1' is the id of a initiative, not of a epic.` (K01).
  `'DEMO-E-1' is a epic, not a product.` (P04).
  `'DEMO' is a product, not a initiative.` (W24).
  `name is required to create a idea` (W19).
  `SELECT * FROM products WHERE key = 'NOPE'` gets the note
  `There is no product with the id 'NOPE'.`
  The text `demo-3` alone gets `demo-3 is an id, not a statement. ... WHERE id = 'demo-3'`.
  `SELECT * FROM me WHERE name = 'x'` gets `No me has exactly the name 'x'.`
- Proposed edit: error-messages, new step: "Before `<kind>`, `<expected
  kind>` and `<singular>` the article is `an` when the word starts with a,
  e, i, o or u." H03: `There is no <singular> with the <id field>
  '<value>'.` S05: `<ID>`, the id in capital letters. H07 is not given for
  the entity me.
- Rationale: eight message texts write `a` before a kind, and three kinds
  need `an`: epic, idea and initiative.

### F-016 SPEC-AMBIGUITY: an example lists the fields of goals without `success_metric`

- Weight: low
- Location (spec): EXAMPLE entity_without_custom_fields (line 4430); entity goals. Point 1 of the translation report.
- Location (artefact): the code follows the catalog, and `TestEntityWithoutCustomFields` asserts the list with `success_metric`
- Evidence: the message of the example has `Fields of goals: id, name,
  status, product, ...`. The catalog has `success_metric` as the third
  field since version 0.11.0.
- Proposed edit: put `success_metric` after `name` in the text of the
  example. This is my omission in the change to 0.11.0.
- Rationale: an example and the rule that it shows have to agree.

### F-017 SPEC-AMBIGUITY: three forms that the grammar leaves open

- Weight: low
- Location (spec): parse-statement, `item` (line 1304) and the label rules (lines 1325 and 1326). Point 4 of the translation report covers the first form.
- Location (artefact): `parser.go`, function `parseItem`
- Evidence: the grammar allows `SUM(*)`. The code answers `expected a
  field or * inside SUM( )`, which names `*` as valid. `SELECT COUNT(1)
  ...` returns the label `COUNT(*)`, and the rule says "with its argument
  in parentheses as written". `SELECT id AS 'my id' ...` is accepted, and
  the grammar does not say whether a label is a word or a text.
- Proposed edit: `item = * | field [AS label] | COUNT ( * | [DISTINCT]
  field ) [AS label] | (SUM | AVG | MIN | MAX) ( [DISTINCT] field ) [AS
  label]`. "`*` in another function than COUNT: syntax, message S03 with
  the text `a field inside <FUNCTION>( )`." "The label of `COUNT(1)`, and
  of COUNT with the name of an entity, is `COUNT(*)`." "`label = word |
  text`."
- Rationale: each form has two defensible answers today.

### F-018 TRANSLATION-ERROR: ORDER BY accepts keys that the grammar does not have

- Weight: low
- Location (spec): parse-statement, `key` (line 1306); build-rows, step 4 (line 2289)
- Location (artefact): `parser.go`; `read.go`, function `sortRows`
- Evidence: `SELECT name AS n ... ORDER BY n` is accepted. The report
  names this (point 5). It does not name two other cases.
  `SELECT status, COUNT(*) ... GROUP BY status ORDER BY COUNT(name)` sorts
  by an aggregate that the statement does not select.
  `SELECT id FROM features WHERE product = 'DEMO' ORDER BY COUNT(*)` is
  accepted in a result without groups and has no effect.
- Proposed edit: make the first two cases valid. They are useful, and
  they change no result of a statement that the grammar accepts today.
  `key = field | label of a select item | function ( * | [DISTINCT] field
  )`, and "an aggregate key does not have to be selected". "An aggregate
  key in a statement without an aggregate item and without GROUP BY:
  syntax, message S03."
- Rationale: the specification should say what the code may accept, so
  that the next translation does the same.

### F-019 SPEC-AMBIGUITY: grouped results

- Weight: low
- Location (spec): build-rows, steps 2 a, 2 b and 4 (lines 2283 to 2289)
- Location (artefact): `read.go`, functions `buildRows` and `groupKey`
- Evidence:
  `SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status ORDER BY name`
  is accepted and sorts the groups by the name of the first record of each
  group. Records without a value in a group field form one group, for
  example `{"tags":null,"COUNT(*)":5}`. Step 2 b does not say so, and
  evaluate-condition says that a field without a value equals nothing.
- Proposed edit: step 4: "In a grouped result a field key must be a group
  field. Any other field is an error: syntax, message S03 with the text `a
  field of GROUP BY, a selected field or an aggregate after ORDER BY`."
  Step 2 b: "Records without a value in a group field are one group for
  that field."
- Rationale: both rules are what a reader expects, and neither is written.

### F-020 SPEC-AMBIGUITY: an IN list as a route

- Weight: low
- Location (spec): plan-fetch, step 5 (line 2118)
- Location (artefact): `read.go`, function `planFetch`
- Evidence: `id IN ('DEMO-1', 'DEMO-1', 'DEMO-1')` sends three requests
  for DEMO-1 and has the scope `feature DEMO-1, feature DEMO-1, feature
  DEMO-1`. `id IN ('DEMO-1', 'DEMO-R-2')` sends the request for DEMO-1 and
  then answers with the error K01 for the second value.
- Proposed edit: "Check the form of every value of the list before the
  first request. A value that equals an earlier value of the list, without
  regard to case, is read once, and the scope names it once."
- Rationale: a statement that fails should cost no request where the
  server can know that before it sends one.

### F-021 SPEC-INCOMPLETENESS: statuses with a value of `entity` that matches nothing

- Weight: low
- Location (spec): entity statuses (line 1815); read-statement, step 8
- Location (artefact): `read.go`, function `emptyNotes`; follows the step
- Evidence: `SELECT * FROM statuses WHERE product = 'DEMO' AND entity = 'comments'`
  returns an empty result without a note about the cause, and so does
  `entity = 'nonsense'`.
- Proposed edit: read-statement, step 8, new case: "On statuses, a
  conjunct `entity = value` that no record satisfies gives a new note:
  `<PRODUCT> has statuses for: <entities>.`"
- Rationale: every other empty result of the catalog explains itself.

### F-022 SPEC-INCOMPLETENESS: two annotations of `query` are not stated

- Weight: low
- Location (spec): tool-registration, step 1 (line 675). Point 3 of the translation report.
- Location (artefact): `tools.go`, function `newMCPServer`
- Evidence: the step gives readOnlyHint and openWorldHint. The translator
  set destructiveHint to false, because the library would otherwise send
  its default, and left idempotentHint at its default false.
- Proposed edit: "readOnlyHint true, destructiveHint false, idempotentHint
  true, openWorldHint true". The schema of MCP gives the two added hints
  a meaning only for a tool whose readOnlyHint is false. Its defaults for
  them are true and false (see Sources). So the edit changes what a host
  shows and nothing else.
- Rationale: the next translation should not have to choose.

### F-023 SPEC-AMBIGUITY: form and text of warnings

- Weight: low
- Location (spec): load-configuration, steps 6 d and 7 (lines 823 and 824); choose-example-key, steps 4 and 5 (lines 2813 and 2814). Point 8 of the translation report.
- Location (artefact): `session.go`, function `warn`; `config.go`
- Evidence: the steps say "write a warning" and what it says in general.
  The code writes one line that starts with `warning: `, with four texts
  of its own, for example
  `warning: the product list could not be read; the examples use the key PROD`.
- Proposed edit: a table of warnings beside the start messages: the form
  `warning: <text>`, one line on standard error, and the four texts.
- Rationale: an operator who filters the log needs fixed texts, and a
  test can then assert them.

### F-024 SPEC-AMBIGUITY: the INVARIANT on environment variables cannot be met as written

- Weight: low
- Location (spec): INVARIANTS (line 2872); load-configuration (line 846); aha-request, MECHANISM of step 3
- Location (artefact): `aha.go`: the proxy function of the standard library; `main.go`, function `getenv`
- Evidence: the INVARIANT says that the server reads ten named variables
  "and no other". The MECHANISM of aha-request names the proxy function
  of the standard library. In Go that function also looks at HTTP_PROXY
  and REQUEST_METHOD, and the Go runtime reads variables of its own. None
  of them changes the way to an https address. The server code reads the
  four AHA_ variables only. The translation report names this.
- Proposed edit: "the settings of the server come from AHA_SUBDOMAIN,
  AHA_API_KEY, AHA_DEFAULT_PRODUCT and AHA_READ_ONLY and from no other
  variable. The standard library of the language reads the variables for
  the proxy and for the trust store. The server code reads no other
  variable." The same wording in load-configuration.
- Rationale: no implementation can meet the INVARIANT as it is written,
  so no test can check it.

### F-025 SPEC-INCOMPLETENESS: `description` of products is requested with every product list

- Weight: low
- Location (spec): entity products, field description (line 1460); resolve-product, step 1 (line 2052); plan-fetch, step 1 b
- Location (artefact): `session.go`, function `productList`; follows the catalog
- Evidence: `description` of products is the only description of the
  catalog without the note heavy. The product list is therefore requested
  with
  `fields=reference_prefix,name,product_line,parent_id,description,created_at,url`
  for every statement that names a product.
- Proposed edit: mark the field as heavy. resolve-product, step 1: "with
  the members of the fields of products that are not heavy". plan-fetch,
  step 1 b: "When used_fields has `description`, the list is read for this
  statement with it."
- Rationale: the same rule as for the other seven descriptions, and a
  smaller answer for the request that most statements send.

### F-026 SPEC-INCOMPLETENESS: META has no `Module:`

- Weight: low
- Location (spec): META (line 8)
- Location (artefact): `go.mod`: `module aha-aql`
- Evidence: the template takes the module identity from the META field
  `Module:` first. Without it the translator derived `aha-aql` from the
  title, and the report says so.
- Proposed edit: `Module: github.com/mge1512/aha-aql`, after the pattern
  of pcd-lint and pcd-slice. A session build of pcd-lint 0.4.0 accepts the
  field in strict mode. The packaged tool at the pull-request gate decides.
- Rationale: the module path should name the place of the source.

### F-027 HINTS-GAP: the pattern of Credential and the repeat limit of the Go package regexp

- Weight: low
- Location (spec): TYPES, Credential (line 62)
- Location (hints): `mcp-server.go.mcp-go.hints.md`
- Evidence: the pattern `^[!-~]{1,4096}$` has a repeat count above the
  limit of 1000 of that package. The report says under Test Refinements
  that in the first test run every test failed, because the binary stopped
  at its start. The translator replaced the pattern by a check of the
  length and of the characters.
- Proposed edit: hints for Go: "The package regexp refuses a repeat count
  above 1000. Check such a `matches` constraint as a length and a set of
  characters." Optional in the specification: `Credential := string of 1
  to 4096 characters, each from "!" to "~"`.
- Rationale: the next translator should know this before the first test run.

### F-028 TEMPLATE-GAP: container image

- Weight: low
- Location (template): `mcp-server.template.md` 0.3.24, deliverable OCI ("Final stage FROM scratch", "Expose port 8080"); http-transport (default listen `127.0.0.1:8080`)
- Location (artefact): `Containerfile`
- Evidence: an image from scratch has no trust store, and aha-request
  needs one. The translator copied the CA bundle of the builder image to
  `/etc/ssl/ca-bundle.pem`. The default listen address is a loopback
  address, which nothing outside the container reaches. The translator set
  `CMD ["listen=0.0.0.0:8080"]`, so the server starts with the warning of
  load-configuration, step 6 d. The report explains both decisions. The
  builder image has the tag `latest`, as the template says.
- Proposed edit: template, row OCI: a rule for the trust store of a server
  that calls an HTTPS service, and the listen address of the image. A
  fixed tag or a digest for the builder image is a question for the same
  row. Specification, DEPLOYMENT, new paragraph "Container image".
- Decision of the maintainer (2026-11-24): the image keeps the loopback
  address. Version 0.20.0 has the paragraph in this form: the image has
  the certificate authorities of its build as its trust store, and it
  listens on `127.0.0.1:8080`. A TLS endpoint in the same pod reaches it
  there. An operator who publishes the port sets `listen=0.0.0.0:8080` and
  gets the warning G13. Proposed text for the template, row OCI: "The image
  keeps the default listen address of the http transport. A server that
  calls an HTTPS service gets the CA bundle of the builder stage at a path
  that the standard library reads."
- Rationale: both decisions touch the security of the deployment, and the
  translator should not have to make them.

### F-029 HINTS-GAP: no target prepares `vendor/`, and no rule keeps it out of the source tarball

- Weight: low
- Location (hints): `mcp-server.go.mcp-go.hints.md`, the sections on the Makefile and on the RPM spec
- Location (template): `mcp-server.template.md` 0.3.24, deliverables table, rows `build`, `RPM`, `DEB` and `OCI`
- Location (artefact): `Makefile`, `Containerfile`, `debian/rules`; the repository has no `.gitignore`
- Facts from the maintainer. 2026-11-24: `vendor/` is never a part of
  the deliverable, and the Makefile has to prepare it. The RPM file
  `aha-aql.spec` was removed from the first commit by accident, and
  commit `d2b5174` has it again (see F-030). 2026-11-25: the vendor
  directory is packaged in a separate tar file with the name
  `%{name}-%{version}-vendor.tar.gz`, and the place for the rule is the
  hints file of the PCD repository.
- Evidence in the run: `Makefile` builds with `GOFLAGS ?= -mod=vendor`.
  Of its targets, `vendor-tarball` alone runs `go mod vendor`, and it
  writes `mcp-server-aha-aql-0.13.0-vendor.tar.xz`. `build` and `test`
  do not depend on it, so `make build` fails in a fresh clone.
  `Containerfile` has `COPY vendor/ vendor/`, and `debian/rules` exports
  `GOFLAGS=-mod=vendor`. The run had the instruction "Use 'go mod
  vendor' for dependencies" in its system prompt. The Makefile of the
  hints has the same gap: its `build` uses `-mod=vendor` and depends on
  no target that makes `vendor/`. The hints list `vendor/` for
  `.gitignore`. The template does not name that file as a deliverable,
  and the run wrote none.
- Evidence in the PCD repository (github.com/mge1512/pcd, master at
  `89315cc`): the cli-tool template has the rule in a form without a
  language. The vendored tree is not part of the source tarball, and a
  second tarball `<n>-X.Y.Z-vendor.tar.<extension>` is `Source1`. This
  is ruling T-03 of the maintainer of 2026-06-10. The mcp-server
  template has no such paragraph. Ruling T-06 of the same list says
  that Go-specific advice is in hints files and not in the templates.
  The hints file writes the tarball with `.tar.xz`. Of the tools of the
  repository, pcd-slice writes `-vendor.tar.gz`, and mcp-server-pcd has
  `-vendor.tar.xz` in its specification and in its generated files.
- Proposed edit: pull request 6 of github.com/mge1512/pcd, opened on
  2026-11-25. The file `mcp-server.go.mcp-go.hints.diff` has its change
  of the hints file. It adds a section with the rule and two wrong
  patterns. The Makefile of the hints gets a target `vendor`, which
  runs `go mod vendor` when `vendor/` is missing or older than `go.mod`
  or `go.sum`, and `build` depends on it. A paragraph says that `%build`
  and `%check` call the `go` commands themselves. `.gitignore` gets
  `*-vendor.tar.xz`. The same diff applies to the copy in
  `pcd-artefacts/` of the repository of this server:
  `patch -p2 -d pcd-artefacts < mcp-server.go.mcp-go.hints.diff`.
- Decisions of the maintainer, later on 2026-11-25: the tarball keeps
  the ending `.tar.xz`, as the hints and the first implementation have
  it. The label of the hints file stays the library version, and the
  library version is raised with the change. The pull request therefore
  moves the hints from mcp-go v0.46.0 to v1.6.0.
- What v1.6.0 changes for this server: it needs Go 1.25.5 or later. A
  tool filter runs for every `tools/call` as well as for `tools/list`.
  The code of the run fills the example key in a tool filter, so one of
  its 187 tests fails with v1.6.0: an http request reads the product
  list once more. With a hook after `tools/list` in place of the
  filter, all 187 tests pass. The hints name both facts.
- Checked: the Makefile rule, on a fresh copy of the code of the run.
  `make build` runs `go mod vendor` once and builds. A second run does
  nothing for `vendor`. After `touch go.mod` the rule runs again. F-030
  has the package build with the two tarballs.
- Correction: the first form of this finding proposed text with
  `go mod vendor` for a row of the template. Ruling T-06 excludes that.
  For the template, the paragraph of the cli-tool template on the two
  tarballs fits as it is. That is a template change of its own.
- Open points for the maintainer:
  - The specification of mcp-server-pcd pins mcp-go v0.46.0 and points
    to the hints file. It needs a unit of its own.
  - The Debian package and the image need `vendor/` in the build tree
    too. The rule names the tarball for the RPM package. For the image,
    the Containerfile of the template does not use `vendor/`: its
    builder stage runs `go mod download`.
  - The checksum of mcp-go v1.6.0 was not compared with the public
    checksum database, which this session could not reach.
- Rationale: three packaging paths rest on a directory that no rule
  creates. A build without network is the goal, and one rule in the
  hints gives it to every translation of an MCP server in Go.

### F-030 TRANSLATION-ERROR: the RPM file expects `vendor/` inside the source tarball

- Weight: low
- Location (artefact): `aha-aql.spec` of commit `d2b5174`, lines 6, 7, 15 and 34; `Makefile`, target `vendor-tarball`
- Location (hints): `mcp-server.go.mcp-go.hints.md`, the section on the RPM spec
- Evidence: the RPM file has one source, `Source0`, and `%autosetup`
  without an option. Its comment says: "The source tarball contains
  vendor/ (go mod vendor), so the build needs no network access." The
  Makefile of the same run says that its vendor tarball is "for OBS builds
  that keep vendor/ out of the source tarball". The RPM sketch of the
  hints has `Source1` for that tarball and `%autosetup -a 1`. A source
  tarball that is made from the repository has no `vendor/`.
- Evidence from a package build (2026-11-25): rpmbuild 4.18.2, with
  `GOPROXY=off` and an empty module cache. With the file as it is and a
  source tarball from the files of the repository, `%build` stops with
  "go: inconsistent vendoring". With two more lines,
  `Source1: %{name}-%{version}-vendor.tar.gz` and `%autosetup -a 1`,
  the steps `%prep`, `%build`, `%install` and `%check` pass, and
  rpmbuild writes the package. The vendor tarball came from
  `make vendor-tarball`, with the Makefile rule of F-029. Limits of this
  build: the rpm of Ubuntu 24.04 and not the rpm of a SUSE system, the
  option `--nodeps`, and `_unitdir` set by hand. The macros for systemd
  were not installed, so the scriptlets of this package are not usable.
  It was no build on a build service.
  The run used the ending `.tar.gz` of the first statement. The decided
  name ends in `.tar.xz`.
- Proposed action: none on the file. The next translation writes it from
  the hints rule of F-029, with `Source1: %{name}-%{version}-vendor.tar.xz`
  and `%autosetup -a 1`. Everything else in the file is as the
  template asks: the spec hash, `pandoc` in `BuildRequires` and in
  `%build`, the man page and the unit in `%files`, and the unit as a file
  that the package installs.
- Rationale: two files of one run describe two different source layouts.
  The RPM file does not cover the layout that the repository has.

## Answers of the maintainer (2026-11-24)

Two facts, before the questions:

- The RPM file was removed from the first commit by accident. It is no
  finding against the run (F-030 reviews the file).
- `vendor/` is never a part of the deliverable. The Makefile has to
  prepare it (F-029).
- Added on 2026-11-25: the vendor directory is packaged in a separate
  tar file with the name `%{name}-%{version}-vendor.tar.xz` (first statement: `.tar.gz`). The place
  for the rule is the hints file of the PCD repository (F-029).

Eight questions and the answers:

| Question | Answer | Where it is |
|---|---|---|
| Which units now | The units 1 to 8, in the proposed order. Unit 9 after two new checks | Versions 0.14.0 to 0.20.0; checks 25 and 26 of `aha-checks.sh` |
| F-005: `=` on a field with several values | A new rule: `=` and IN are true when the field has the value among its values. A value with commas means that the field has each of them. CONTAINS stays a search in the text | 0.17.0 |
| F-004: NULL in an IN list | It stands for a field without a value, as `= NULL` already means IS NULL. Such a list is no route | 0.16.0 |
| F-029: `vendor/` | `build` and `test` depend on a target that runs `go mod vendor` when `vendor/` is missing or older than `go.mod` and `go.sum`. A package build extracts the vendor tarball first. `vendor/` is in `.gitignore` | Open: hints, `mcp-server.go.mcp-go.hints.diff` |
| F-012: kept lists | Nothing is kept between statements | 0.19.0 |
| F-006: a write result that differs from what was sent | A note in the result | Open: unit 9 |
| F-013: examples that a test cannot check | FakeAha is an HTTPS server outside the process. The examples on time say what a test sees from outside. The example for a direct connection is removed, and its rule stays | 0.20.0 |
| F-028: listen address of the container image | Loopback by default. An operator who publishes the port sets the address and gets the warning | 0.20.0 |

Where a version differs from the proposed edit of its finding:

- F-001: the note H08 also shows the statement that reads the record.
- F-005: the fixture account has tags on two features, because the
  examples need them. plan-fetch reads the current user also for `me` as
  one part of a value with commas.
- F-008: there are two messages, S15 for two items with one label and S16
  for a label that is the name of another field of the row. A field that
  GROUP BY names twice is shown once. I found that case only after the
  report: `GROUP BY status, status` returned `"status"` twice.
- F-013: the certificate check has an example, which a test can reach
  with the new form of FakeAha. The example for a listen address uses
  192.0.2.1, an address for documentation, so that no test opens a port
  to a network.
- F-015: the singular of the entity me is `user` in every message. The
  report proposed to give no note H07 for that entity.
- F-016: it is a version of its own, 0.18.2, because it changes an
  example of another behavior than unit 6.
- F-023: the warning for the listen address has no address in its text. A
  postcondition of load-configuration forbids the value of an option in a
  warning.
- F-024: there are two lines in INVARIANTS: one observable line for the
  settings and one implementation line for the variables that the code
  reads.

## Review limitations

- The reviewer wrote the specification, in the same session. A cold read
  was not possible, and I may have missed defects in text that I wrote.
- The guard of the skill did not find the packaged tools, and the
  repository has no bundles. I read the specification file itself and cite
  it by behavior and line.
- Nothing was run against Aha!. Every result above comes from FakeAha,
  which the translator wrote from the same specification. FakeAha and the
  code agree with the specification also where the specification is wrong.
  What Aha! does is known from the checks of 2026-11-24 only.
- Read in full: the translation report, `main.go`, `config.go`, `aha.go`,
  `session.go` and `tools.go`. Read in parts: `read.go`, `write.go`,
  `parser.go`, `lexer.go`, `catalog.go`, `jsonout.go` and the tests. Their
  behavior was checked from outside.
- My 583 statements used the stdio mode. The http mode and the manual mode
  were checked by the tests of the repository and by the acceptance
  commands only.
- The statements went through the test harness of the repository, with one
  test file of my own in a working copy. That file is not part of the
  repository and is not delivered.
- The modules were compared with `go.sum`. The public checksum database
  was not reachable from this session.
- The package build of F-030 ran outside a build service, with the rpm
  of another distribution and without the build dependencies. Downloads
  of modules were switched off, so it shows that the two tarballs are
  enough for the build. It does not show that the package builds on a
  SUSE system.
- The versions 0.14.0 to 0.20.0 are not translated. The lint and the
  slice check accept them, with builds of this session and not with the
  packaged tools. No program has run their rules, and no test has run
  their examples.
- This is no security review and no license certification. Both are modes
  of their own.

## Three read-only statements for the test product

They show F-001, F-002 and F-003 on data of Aha!. The command `query`
sends GET requests only. The tests show that, and so do my statements
against the test double. Set `k` to the path of the key file, and replace
DEMO-1 by the id of a feature that exists.

```
k="$HOME/.config/aha/api-key"
mcp-server-aha-aql query read-only=true api-key-file="$k" \
  query="SELECT id FROM features WHERE id = 'DEMO-1' AND name = 'no such name'"
mcp-server-aha-aql query read-only=true api-key-file="$k" \
  query="SELECT id FROM features WHERE product = 'DEMO' ORDER BY id DESC LIMIT 3"
mcp-server-aha-aql query read-only=true api-key-file="$k" \
  query="SELECT COUNT(*) FROM features WHERE product = 'DEMO' AND created_at > '01.01.2100'"
```

Expected today: the first result has a note that the feature does not
exist. The second shows ids that are not the three highest numbers when
the product has more than nine features. The third counts every feature
of the product, where the correct answer is an error. The account name
comes from AHA_SUBDOMAIN.

## Recommendation

EDIT-AND-REGENERATE.

No finding is unclassified after the answers of the maintainer, so this
is no case for human triage.

A proposed order. Each line is one unit for the change mode of pcd-spec,
and its version step follows the rule of that skill.

| Order | Unit | Findings | State |
|---|---|---|---|
| 1 | Notes for an empty result | F-001, F-021 | 0.14.0 |
| 2 | Order, keys and grammar of a read statement | F-002, F-017, F-018, F-019 | 0.15.0 |
| 3 | Values in conditions | F-003, F-004, F-009 | 0.16.0 |
| 4 | Fields with several values | F-005 | 0.17.0 |
| 5 | Repeated fields and repeated members | F-007, F-008 | 0.18.0 |
| 6 | Texts of messages and one example | F-011, F-015, F-016 | 0.18.1 and 0.18.2 |
| 7 | Requests: kept lists, product list, IN lists | F-012, F-020, F-025 | 0.19.0 |
| 8 | Tests, INVARIANT, annotations, warnings, Module, container paragraph | F-013, F-022, F-023, F-024, F-026, the specification part of F-028 | 0.20.0 |
| 9 | Writes, after two new checks on the test account | F-006, F-010, F-014 | Open |
| Outside the specification | Hints, template, next translation | F-027, F-028, F-029, F-030 | Open |

Unit 9 needs two facts about Aha!: which request removes a value
(F-010), and what Aha! answers to an update that names a record which
does not exist (F-014). The checks 25 and 26 of `aha-checks.sh` ask for
them on the test product. The short run creates two releases, one
initiative and one feature, and it deletes them at the end:

```
AHA_CHECK_SHORT=1 ./aha-checks.sh show     # prints the requests and sends nothing
AHA_CHECK_SHORT=1 ./aha-checks.sh write
```

The lines of the checks 25 and 26 are what unit 9 needs. The output has
no API key. It has names, ids and e-mail addresses of the account.

Then one translation after the last unit, as a full regeneration. Several
units add an INVARIANT or change messages in more than one behavior, and
the code is new.

A second test suite by another model raises the confidence of the
examples above Medium. It would not have found F-001 to F-005, because
tests follow the specification. Those five came from statements that a
user would send.

## Sources

- [Schema reference of the Model Context Protocol, revision 2025-11-25, interface ToolAnnotations](https://modelcontextprotocol.io/specification/2025-11-25/schema): the defaults of the four hints and the condition on readOnlyHint, used in F-022
