# aha-studio-aql

## META
Deployment:   enhance-existing
Language:     Go
Version:      0.1.0
Spec-Schema:  0.4.0
Author:       John Wang
Author:       Matthias G. Eckermann [?]
License:      MIT
Verification: none
Safety-Level: QM

This specification amends AQL (Aha Query Language), the query language of
aha-studio for the Aha! product management service. The base version is
aha-studio 0.12.0. The specification is the input for patches that are
offered to the upstream project. It does not describe aha-studio as a whole.

It specifies how a statement is read, which names a statement may use, how
a statement is run against the Aha! API, and the two MCP tools `query`
and `execute`. With these two tools a language model can work with Aha!
through AQL alone. It then does not need the one-tool-per-API-call set of
the base version.

Everything that aha-studio does and that this specification does not
mention stays as it is in the base version. The DELTA section lists the
differences from the base version, the parts that stay unchanged, and the
test evidence behind the rules.

## TYPES

```
Caller := agent | operator
// agent:    the statement arrives through the MCP tool query or the MCP
//           tool execute.
// operator: the statement arrives through the command line, the
//           interactive shell or the HTTP service.

AgentLimits := {
  default_limit: 50,     // rows returned when a read statement has no LIMIT
  max_limit:     200,    // most rows that one call returns
  max_pages:     15      // most pages of one list that one statement reads
}
// Constants. They apply to agent callers only.

PageSize := 200
// Records requested per page of a list. Applies to every caller.

MaxInValues := 20
// Most values of an IN list that are read with one request per value
// (see plan-fetch). Applies to every caller.

MaxListed := 30
// Most records or product keys that one message lists (see error-messages).

StatementKind := select | describe | insert | update | delete

EntityName := string
// Canonical plural name of a catalog entity, for example "features".

FieldName := string
// Canonical name of a field of an entity, for example "assigned_to".
// For operator callers also "custom.<key>" (see custom-field-references).

Label := string
// Name under which a value appears in a result row.

ProductKey      := string where matches("^[A-Za-z][A-Za-z0-9]*$")
NumericId       := string where matches("^[0-9]{6,}$")
ReferenceNumber := string where matches("^[A-Za-z][A-Za-z0-9]*-([A-Za-z]-)?[0-9]+(-[0-9]+)?$")
RecordId        := ReferenceNumber | NumericId | ProductKey
// What a statement writes as the id of a record. Products are identified
// by their key, comments and users by a NumericId, all other records by
// their ReferenceNumber. A NumericId is accepted wherever a
// ReferenceNumber is accepted.
//
// The form of a ReferenceNumber gives the kind of the record:
//   KEY-n      feature        DEMO-23
//   KEY-n-m    requirement    DEMO-23-1
//   KEY-R-n    release        DEMO-R-3
//   KEY-E-n    epic           DEMO-E-1
//   KEY-S-n    initiative     DEMO-S-1
//   KEY-G-n    goal           DEMO-G-1
//   KEY-I-n    idea           DEMO-I-1
// The letter is read without regard to case. Every other form has no kind.

Value    := Text | Number | Boolean | Null
Text     := string
Number   := integer or decimal number
Boolean  := true | false
Null     := no value
DateText := Text where matches("^[0-9]{4}-[0-9]{2}-[0-9]{2}$")

SelectItem    := Star | FieldItem | AggregateItem
Star          := all fields (see read-statement, step 1)
FieldItem     := { field: FieldName, label: Label }
AggregateItem := {
  function: count | sum | avg | min | max,
  field:    FieldName or none,       // none means COUNT(*)
  distinct: Boolean,
  label:    Label
}

Condition  := Comparison | NullTest | InList
            | AllOf(List<Condition>) | AnyOf(List<Condition>) | Negation(Condition)
Comparison := {
  field:    FieldName,
  operator: "=" | "!=" | "<" | "<=" | ">" | ">=" | "contains" | "like",
  value:    Text | Number | Boolean
}
NullTest   := { field: FieldName, negated: Boolean }   // IS NULL, IS NOT NULL
InList     := { field: FieldName, values: List<Value> }
// Conjunct: one member of the top-level AllOf of a WHERE condition. A WHERE
// condition that is not an AllOf has one conjunct, itself.

OrderKey := { key: FieldName | Label, descending: Boolean }

ReadStatement := {
  entity:   EntityName,
  items:    List<SelectItem>,
  distinct: Boolean,
  where:    Condition or none,
  group_by: List<FieldName>,
  order_by: List<OrderKey>,
  limit:    u32 or none
}
DescribeStatement := { entity: EntityName or none }
InsertStatement   := {
  entity: EntityName,
  parent: RecordId or none,
  fields: Map<FieldName, Value>          // in the order of the statement
}
UpdateStatement := {
  entity: EntityName,
  id:     RecordId,
  fields: Map<FieldName, Value>,
  guards: Map<FieldName, Value>          // WHERE id = '...' AND field = 'value'
}
DeleteStatement := {
  entity: EntityName,
  id:     RecordId,
  guards: Map<FieldName, Value>
}
PredicateWrite := {
  kind:   update | delete,
  entity: EntityName,
  fields: Map<FieldName, Value>,         // empty for delete
  where:  Condition
}
// PredicateWrite exists for operator callers only (see write-by-predicate).

Statement := ReadStatement | DescribeStatement | InsertStatement
           | UpdateStatement | DeleteStatement | PredicateWrite

Row := Map<Label, Value>
// The labels are in the order of the statement.

ReadResult := {
  entity:    EntityName,
  scope:     string,       // what was read, for example "release DEMO-R-3"
  rows:      List<Row>,
  count:     u32,          // number of rows
  total:     u32 or none,  // number of matching rows, when rows has fewer
  truncated: Boolean,      // rows were cut and the statement did not ask for it
  notes:     List<string>
}

WriteResult := {
  id:      RecordId or none,          // record that was created, changed or deleted;
                                      // none in the dry run of an insert
  values:  Map<FieldName, Value>,     // see insert-statement and update-statement
  message: string
}

AqlError := { kind: ErrorKind, message: string }

FieldDef := {
  name:           FieldName,
  type:           FieldType,
  reference_kind: RecordKind or none,   // for type reference
  source:         string,               // where the value is in an API record
  heavy:          Boolean,              // requested from a list only when used
  writable:       Boolean
}

EntityDef := {
  name:           EntityName,
  singular:       string,
  other_names:    List<string>,
  kind:           RecordKind or none,
  about:          string,
  fields:         List<FieldDef>,              // order of DESCRIBE and of messages
  default_fields: List<FieldName>,
  other_field_names: Map<string, FieldName>,
  read_routes:    List<{ field: FieldName, route: string }>,   // in order of use
  account_list:   string or none,              // route of the account-wide list
  name_search:    Boolean,                     // the account-wide list has the parameter q
  parent_filters: List<FieldName>,
  parent_kind:    RecordKind | any-record | none,   // parent of a new record
  parent_optional: Boolean,
  operations:     set of (create | update | delete),
  example:        string
}

Product := { key: ProductKey, id: NumericId, name: string }

ExampleKey := ProductKey
// The product key that tool descriptions, DESCRIBE results, examples and
// error messages show. Written {KEY} in this specification.

ApiRecord := Map<string, any>
// One record as the Aha! API returns it.

ListPage := {
  records:       List<ApiRecord>,
  total_records: u32,
  total_pages:   u32
}

ApiFailure := NotFound | RateLimited | Unreachable | Rejected
// NotFound:    the API answers with status 404.
// RateLimited: the API answers with status 429.
// Unreachable: no answer.
// Rejected:    every other status that is not a success; it has the status
//              and the text of the answer.
```

### TYPE: ErrorKind
Kind: enum

| Value | Description |
|---|---|
| syntax | The statement cannot be read, or it is not one of the accepted forms |
| bad_name | Unknown entity or field, or an entity or field that cannot be used this way |
| scope | The statement does not say which product or parent is meant, or it would read too much |
| bad_args | The tool call itself is wrong: parameter missing, empty or unknown |
| bad_product | A product that the account does not have |
| wrong_kind | An id of another kind of record, or a name where an id is needed |
| not_found | A record that does not exist |
| bad_value | A value that cannot be written to the field, or a required value that is missing |
| guard_failed | A condition next to the id is not true for the record; nothing was changed |
| rate_limited | Aha! answered that there were too many requests |
| network | Aha! cannot be reached |
| api_error | Aha! answered with another error |

### TYPE: FieldType
Kind: enum

| Value | Description |
|---|---|
| string | Short text |
| text | Long text. Read as plain text |
| number | Integer or decimal number |
| boolean | true or false |
| date | A day, written YYYY-MM-DD |
| datetime | A timestamp as the Aha! API writes it |
| status | The name of a workflow status |
| user | A user. Read as an e-mail address, or as a name where the entity says so |
| key | A product key |
| reference | The id of another record |

### TYPE: RecordKind
Kind: enum

| Value | Description |
|---|---|
| product | A product (workspace) |
| release | A release |
| feature | A feature |
| epic | An epic |
| requirement | A requirement of a feature |
| initiative | An initiative |
| goal | A goal |
| idea | An idea |
| comment | A comment on a record |
| user | A user of the account |

## INTERFACES

```
AhaAPI
  The Aha! REST API, version 1, of the configured account. Routes are
  written relative to its base address. A route parameter in braces is
  replaced by a RecordId or by the NumericId of a product.

  ApiList(route, parameters) -> ListPage | ApiFailure
      One page of a list. Parameters: page (1-based), per_page, fields
      (comma-separated names of the wanted members of each record),
      q (name search, on the routes that have it).
  ApiGet(route) -> ApiRecord | ApiFailure
      One record with all its members.
  ApiCreate(route, fields) -> ApiRecord | ApiFailure
      Creates one record under the parent of the route.
  ApiUpdate(route, fields) -> ApiRecord | ApiFailure
      Changes the given fields of one record.
  ApiDelete(route) -> none | ApiFailure
      Deletes one record.
  ApiSetCustomFields(kind, id, values) -> none | ApiFailure
      Sets custom field values of one record. The base version does this
      through the GraphQL endpoint of the same account.
```

The request bodies of ApiCreate and ApiUpdate are the record forms that the
Aha! API expects on the route. Where the base version already sends the
same request for one of its own tools, the request keeps its form. Where
the base version reaches the same result through the GraphQL endpoint
(create an idea, set custom field values), it may continue to do so.

A RateLimited answer is repeated as the client of the base version repeats
it. When it persists, the statement fails with rate_limited and returns no
partial result.

Test double: FakeAha. It serves a fixture account from memory and never
opens a network connection.

- ApiList returns pages of `per_page` records with correct `total_records` and `total_pages`.
- A record of a list has the members of the parameter `fields`, and `id`.
- `q` is a search for the text inside `name`, without regard to case.
- A route for a record that the fixture does not have is answered with NotFound.
- ApiCreate gives the new record the next free reference number.
- FakeAha records every call with its method, route, parameters and fields. A test can then check which requests a statement caused, and that a failed statement caused no write request.

The EXAMPLES use this fixture account. No default product is configured,
so the ExampleKey is PROD.

Products of the fixture account:

| key | id | name |
|---|---|---|
| DEMO | 7000000000000000001 | Demo Product |
| OTHER | 7000000000000000002 | Other Product |

Users: the current user is Test User, e-mail test.user@example.com, id
7000000000000000900. A second user is Second User, e-mail
second.user@example.com, id 7000000000000000901.

Workflow statuses of features in DEMO, in workflow order: Under
consideration, In design, Ready to develop, In development, Shipped, Will
not implement.

Releases of DEMO:

| id | name | release_date | parking_lot |
|---|---|---|---|
| DEMO-R-1 | What's next | (none) | true |
| DEMO-R-2 | Spring release | 2026-03-31 | false |
| DEMO-R-3 | Summer release | 2026-06-30 | false |
| DEMO-R-4 | Rider's cup release | 2026-12-15 | false |

Features of DEMO, in the order in which the API lists them:

| id | name | status | release | initiative | assigned_to | custom field priority |
|---|---|---|---|---|---|---|
| DEMO-1 | CSV export | In design | DEMO-R-2 | (none) | test.user@example.com | High |
| DEMO-2 | Offline maps | Under consideration | DEMO-R-2 | (none) | (none) | High |
| DEMO-3 | Live dashboard | In development | DEMO-R-3 | DEMO-S-1 | (none) | (none) |
| DEMO-4 | Live video feed | Under consideration | DEMO-R-3 | DEMO-S-1 | (none) | (none) |
| DEMO-5 | Rider's handbook | Shipped | DEMO-R-3 | (none) | (none) | (none) |
| DEMO-6 | Route planner | Under consideration | DEMO-R-1 | (none) | (none) | (none) |

Other records of the fixture account:

- The initiative DEMO-S-1 "Live coverage" with the status "In progress".
- The goal DEMO-G-1 "Grow the audience". Its API record has the status "On track" under its success metric and no workflow status of its own.
- The goal OTHER-G-1 "Other goal".
- The requirement DEMO-1-1 "Column selection" of the feature DEMO-1.
- One comment on DEMO-3 with the id 7000000000000000701, the body "Verified with the provider." and the user Test User.

OTHER has no releases and no features. No feature of the fixture account
has an epic.

## BEHAVIOR: query-tool
Constraint: required

The MCP tool `query`. It runs one read statement or one describe statement
for an agent caller and returns the result as a JSON object.

INPUTS:
```
query: string    // one AQL statement; the only parameter of the tool
```

OUTPUTS:
```
tool result: the result object as JSON text,
             or the message of an AqlError, with the tool result marked
             as an error
```

PRECONDITIONS:
- the tool is registered (see tool-registration)

STEPS:
1. If the call has a parameter other than `query` → bad_args, message A02.
2. If `query` is absent, is not text, or is empty after trimming → bad_args, message A01.
3. Run parse-statement with caller agent and the kinds select and describe allowed; on failure → the error of parse-statement.
4. For a describe statement: run describe-statement. Its result is the result object. Continue with step 7.
5. For a read statement: run read-statement with caller agent and without a default product; on failure → the error of read-statement.
6. Build the result object from the ReadResult. Its members, in this order:
   a. `entity`, `scope` and `count`;
   b. `total`, only when it has a value;
   c. `truncated`, only when it is true;
   d. `records`, the rows;
   e. `note`, only when there are notes: the notes joined by one space.
7. Return the result object as JSON text. Text is a JSON string, Number a JSON number, Boolean a JSON boolean and Null the JSON null.

POSTCONDITIONS:
- only ApiList and ApiGet were called
- `records` has at most max_limit rows
- each row has the labels of the select items, each with a value or null; a grouped result also has the fields of GROUP BY that are not selected (see build-rows)

ERRORS:
- bad_args: unknown parameter (A02); missing or empty statement (A01)
- every error of parse-statement and read-statement, passed on unchanged

## BEHAVIOR: execute-tool
Constraint: required

The MCP tool `execute`. It runs one write statement for an agent caller.
One call creates, changes or deletes at most one record.

INPUTS:
```
statement: string    // one AQL statement; the only parameter of the tool
```

OUTPUTS:
```
tool result: the result object as JSON text,
             or the message of an AqlError, with the tool result marked
             as an error
```

PRECONDITIONS:
- the tool is registered; it is not registered in read-only mode (see tool-registration)

STEPS:
1. If the call has a parameter other than `statement` → bad_args, message A02.
2. If `statement` is absent, is not text, or is empty after trimming → bad_args, message A03.
3. Run parse-statement with caller agent and the kinds insert, update and delete allowed; on failure → the error of parse-statement.
4. Run insert-statement, update-statement or delete-statement, according to the kind of the statement, with caller agent; on failure → the error of that behavior.
5. Build the result object from the WriteResult: `id`, then each entry of `values` in its order, then `message`.
6. Return the result object as JSON text, as in query-tool.

POSTCONDITIONS:
- at most one record of the account was created, changed or deleted
- when the result is an error that was found before the write request, no ApiCreate, ApiUpdate, ApiDelete or ApiSetCustomFields call was made

ERRORS:
- bad_args: unknown parameter (A02); missing or empty statement (A03)
- every error of parse-statement, insert-statement, update-statement and delete-statement, passed on unchanged

## BEHAVIOR: tool-registration
Constraint: required

Decides which AQL tools the MCP server offers and what their descriptions
say. The descriptions are what a language model learns the language from,
so their text is part of this specification.

INPUTS:
```
aql_group_selected: Boolean   // the AQL tool group is among the selected tool
                              // groups, or no selection is configured
read_only:          Boolean   // the server runs in read-only mode
```

OUTPUTS:
```
registered tools: query; execute unless read_only
```

PRECONDITIONS:
- the server has a selection of tool groups and a read-only mode (see PRECONDITIONS)

STEPS:
1. Run choose-example-key once, before any tool is registered.
2. If aql_group_selected: register `query` as a read tool with the description below. It has one parameter, `query`: text, required, with the description "One AQL statement".
3. If aql_group_selected and not read_only: register `execute` as a write tool with the description below. It has one parameter, `statement`: text, required, with the description "One AQL statement".
4. Never register a tool with the name `describe_aql`. The statement DESCRIBE replaces it.
5. In both descriptions, replace every {KEY} by the ExampleKey.
6. A call with a parameter that the tool does not declare reaches the tool and is answered with message A02. It is not ignored.

Description of `query`:

```
Read from Aha! with one AQL statement.
SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]
DESCRIBE entity
Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me.
id is the reference number of a record, for example '{KEY}-23'.
Filter by parent: product = '{KEY}', release = '{KEY}-R-3', feature = '{KEY}-23', initiative = '{KEY}-S-1'.
Other conditions: !=, <, >, CONTAINS 'text', IS NULL. Count with COUNT(*).
Text goes in single quotes. Write an apostrophe inside text twice: 'Year''s end'.
Examples:
SELECT id, name, status FROM features WHERE release = '{KEY}-R-3'
SELECT status, COUNT(*) FROM features WHERE product = '{KEY}' GROUP BY status
SELECT id, name, release_date FROM releases WHERE product = '{KEY}' AND parking_lot = false ORDER BY release_date
SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features'
SELECT email FROM me
```

Description of `execute`:

```
Change ONE record in Aha! with one AQL statement. Three forms:
INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)
UPDATE entity SET field = 'value', ... WHERE id = 'id'
DELETE FROM entity WHERE id = 'id'
parent is the record the new one belongs to: a release for features and epics, a product for releases, initiatives, goals and ideas, a feature for requirements, any record for comments.
Text and apostrophes as in query. DESCRIBE entity in query lists the field names.
Examples:
INSERT INTO features (parent, name) VALUES ('{KEY}-R-3', 'Export to CSV')
UPDATE features SET status = 'In design', assigned_to = 'me' WHERE id = '{KEY}-23'
INSERT INTO comments (parent, body) VALUES ('{KEY}-23', 'It''s ready.')
```

Each line of a description ends with a line break. The entity list in the
description of `query` is the list of catalog entities in catalog
order.

POSTCONDITIONS:
- in read-only mode the tool list has no tool that can change data in Aha! through AQL
- no description contains a product key of the account other than a configured default product

ERRORS:
- none

## BEHAVIOR: read-statement
Constraint: required

Runs one read statement against the Aha! API and returns the matching
records as rows.

INPUTS:
```
statement:       ReadStatement
caller:          Caller
default_product: ProductKey or none   // operator callers only
```

OUTPUTS:
```
result: ReadResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: its entity and every field in it are catalog names

STEPS:
1. Replace a Star item by fields.
   a. When the WHERE condition has a conjunct that compares the id field of the entity with `=`: all fields of the entity.
   b. Otherwise, for an agent caller: the default fields of the entity, followed by the other fields that the statement selects by name. Add note H01.
   c. Otherwise, for an operator caller: all fields of the entity that are not heavy.
2. Collect the used fields: the fields of the select items, of the WHERE condition, of GROUP BY and of ORDER BY.
3. Run plan-fetch with the statement, the used fields, the caller and the default product; on failure → the error of plan-fetch. plan-fetch returns the API records, the scope and the applied conjuncts. It also returns the total of a list that was not read to its end, and the mark that page 1 was read for a count only.
4. Run convert-record for every API record.
5. Keep each record for which evaluate-condition is true for the WHERE condition. A conjunct that the route has applied counts as true.
6. Run build-rows with the kept records, and with the total and the count-only mark of plan-fetch. It returns the rows, count, total and truncated.
7. If truncated: add note H02.
8. If there is no row and the statement has no aggregate item and no GROUP BY: add notes that explain the empty result, at most two. Go through the conjuncts that are a Comparison with a Text value, in the order of the statement. A `%` at the start or at the end of the value is ignored for these tests.
   a. The id field of the entity with the operator `=` gives H03.
   b. On products, the field `name` with a value that resolve-product finds gives H04.
   c. The field `name` with a value that has the form of a ReferenceNumber of the entity's kind gives H05.
   d. The field `name` with a value that has the form of a ReferenceNumber whose kind is a parent filter of the entity gives H06.
   e. Any other comparison of the field `name` with the operator `=` gives H07.
9. Return the ReadResult with entity, scope, rows, count, total, truncated and notes.

POSTCONDITIONS:
- every row satisfies the WHERE condition of the statement
- no matching record is absent from rows unless truncated is true or the statement has a LIMIT, and then total gives the number of matching rows
- every selected field is in every row; a field without a value is null
- only ApiList and ApiGet were called

ERRORS:
- every error of plan-fetch (scope, bad_product, wrong_kind, not_found, rate_limited, network, api_error), passed on unchanged

## BEHAVIOR: describe-statement
Constraint: required

Answers `DESCRIBE` and `DESCRIBE entity` from the catalog. It replaces
the tool `describe_aql` of the base version. It sends no request.

INPUTS:
```
statement: DescribeStatement
```

OUTPUTS:
```
result: JSON object
```

PRECONDITIONS:
- none

STEPS:
1. Without an entity: return an object with two members.
   a. `entities`: a list with one object per catalog entity, in catalog order. Each object has `entity`, the name, and `about`, the about text with {KEY} replaced.
   b. `note`: the text "DESCRIBE entity lists the fields of one entity, for example: DESCRIBE features".
2. With an entity: return an object with these members in this order.
   a. `entity`: the name.
   b. `about`: the about text with {KEY} replaced.
   c. `fields`: one object per field of the entity in catalog order, with `name`, `type`, and `writable` with the value true when the field is writable. A field that is not writable has no member `writable`. The text of `type` comes from the first table below.
   d. `filter_by_parent`: only when the entity has parent filters; a list with the text `<field> = '...'` for each of them.
   e. `status_values`: only when the entity has a field `status`; the text `SELECT name FROM statuses WHERE product = '{KEY}' AND entity = '<entity>'`.
   f. `change_with_execute`: the text "read-only" when the entity has no operation. Otherwise a list with one text per operation of the entity, in the order create, update, delete. The texts come from the second table below.
   g. `example`: the example statement of the entity with {KEY} replaced.

The text of `type`:

| Field type | Text |
|---|---|
| string, text, number, boolean | The name of the type |
| key | `product key` |
| user | `user (e-mail address)`; for a field that is read as a name, `user (name)` |
| status | `status name` |
| date | `date (YYYY-MM-DD)` |
| datetime | `timestamp` |
| reference | `id of a release`, with the record kind of the field, and with `an` before a vowel |

The texts of `change_with_execute`:

| Operation | Text |
|---|---|
| create | `INSERT INTO <entity> (parent, ...) VALUES (...): parent is <what>`; when a new record has no parent, `INSERT INTO <entity> (...) VALUES (...)` |
| update | `UPDATE <entity> SET field = 'value' WHERE id = '...'` |
| delete | `DELETE FROM <entity> WHERE id = '...'` |

`<what>` is `a product key` when the parent is a product, and `the id of
the commented record` for comments. For another record kind it is `the id
of a release`, with that kind and with `an` before a vowel. When the parent
is optional, ` (optional)` follows.

POSTCONDITIONS:
- the fields listed for an entity are exactly the fields that read-statement delivers for it
- no request was sent

ERRORS:
- none

## BEHAVIOR: insert-statement
Constraint: required

Creates one record.

INPUTS:
```
statement:       InsertStatement
caller:          Caller
dry_run:         Boolean              // operator callers only
default_release: RecordId or none     // operator callers only
```

OUTPUTS:
```
result: WriteResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: the entity allows create and every field in it is a writable field of the entity

STEPS:
1. If the statement gives no parent, the caller is an operator, the entity is features and default_release has a value: use it as the parent.
2. If there is no parent and the entity needs one → bad_value, message W18.
3. Check the parent against the parent kind of the entity.
   a. Parent kind product: run resolve-product without hints; on failure → the error of resolve-product.
   b. Parent kind is a record kind: the parent must be a ReferenceNumber of that kind or a NumericId. For a value of another form: if resolve-product finds it as a product → wrong_kind, message W24; otherwise → not_found, message W25, because no record of that kind can have such an id.
   c. Parent kind any-record (comments): the parent must be a ReferenceNumber of the kind feature, epic, idea, release, initiative, goal or requirement; otherwise → wrong_kind, message K08.
4. If the field `name` (for comments the field `body`) is absent or empty → bad_value, message W19.
5. Run check-write-values for the fields; on failure → the error of check-write-values. When a status is assigned and the parent is a record, check-write-values reads the parent to learn its product.
6. If dry_run: return a WriteResult without id, with the fields as values and the message "would create <singular> under <parent>". No write request is sent.
7. Call ApiCreate on the create route of the entity with the parent and the fields. NotFound → not_found, message W25. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Rejected → api_error, message W30.
8. Return the WriteResult. `id` is the id of the new record. `values` has `name` and `parent`; for a comment it has `parent`, `body` and `created_at`. `message` is "<Singular> <id> created" with the singular name of the entity starting with a capital letter; for a comment it is "Comment added to <parent>".

POSTCONDITIONS:
- exactly one record was created, with every field of the statement
- when an error is returned from steps 1 to 5, no ApiCreate call was made

ERRORS:
- bad_value: parent missing (W18); name or body missing (W19); errors of check-write-values
- wrong_kind: W24, K08
- bad_product: from resolve-product (P01)
- not_found: the parent does not exist, or the value cannot be the id of a parent (W25)
- rate_limited, network, api_error: X01, X02, W30

## BEHAVIOR: update-statement
Constraint: required

Changes one record. The record is found by its id alone. Further
conditions next to the id are guards: they are checked on that one record,
and when one of them is not true, nothing is changed.

INPUTS:
```
statement: UpdateStatement
caller:    Caller
dry_run:   Boolean       // operator callers only
```

OUTPUTS:
```
result: WriteResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: the entity allows update, every assigned field is a writable field and every guard field is a field of the entity

STEPS:
1. The id must be a RecordId that the entity accepts. For products: run resolve-product without hints; on failure → the error of resolve-product. For comments: a NumericId. For every other entity: a ReferenceNumber of the entity's kind or a NumericId. A ReferenceNumber of another kind → wrong_kind, message K02. Any other value → wrong_kind, message K04. A ReferenceNumber is used in capital letters from here on.
2. Call ApiGet on the id route of the entity; for products, take the record from the product list. NotFound → not_found, message W20. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Rejected → api_error, message X03.
   MECHANISM: the record is read with one request by its id. The statement never lists records to find the record to change.
3. Run convert-record for the record.
4. For each guard, in the order of the statement: run evaluate-condition with the operator "=" on the record. The first guard that is not true → guard_failed, message W29.
5. Run check-write-values for the assigned fields with the product of the record; on failure → the error of check-write-values.
6. If dry_run: return a WriteResult with the id, the assigned fields as values and the message "would update <singular> <id>". No write request is sent.
7. Call ApiUpdate on the update route with the assigned catalog fields. For an operator caller, assigned custom fields are then written with ApiSetCustomFields. RateLimited → rate_limited, message X01. Unreachable → network, message X02. NotFound, when the statement assigns exactly one reference → not_found, message W25 for that reference. Any other failure → api_error, message W30. When ApiUpdate succeeded and ApiSetCustomFields failed, the message also says which fields were written.
8. Return the WriteResult. `id` is the id of the record. `values` has `name`, or `parent` for a comment, followed by each assigned field with the value that the record has after the write. `message` is "<Singular> <id> status changed to '<status>'" when `status` is the only assigned field, and "<Singular> <id> updated" in every other case.

POSTCONDITIONS:
- exactly one record was changed, in every assigned field and in no other field
- when an error is returned from steps 1 to 5, no ApiUpdate and no ApiSetCustomFields call was made
- a value of empty text is sent as empty text; the field is not left out of the request

ERRORS:
- wrong_kind: K02, K04
- bad_product: an unknown product as the id of a product (P01)
- not_found: the record does not exist (W20); a referenced record does not exist (W25)
- guard_failed: W29
- bad_value, wrong_kind: errors of check-write-values
- rate_limited, network, api_error: X01, X02, X03, W30

## BEHAVIOR: delete-statement
Constraint: required

Deletes one record. Finding the record and the guards work as in
update-statement.

INPUTS:
```
statement: DeleteStatement
caller:    Caller
dry_run:   Boolean       // operator callers only
```

OUTPUTS:
```
result: WriteResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: the entity allows delete and every guard field is a field of the entity

STEPS:
1. Check the id as in update-statement, step 1 → wrong_kind, message K02 or K04.
2. Call ApiGet on the id route of the entity. NotFound → not_found, message W20. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Rejected → api_error, message X03.
3. Run convert-record for the record.
4. Check the guards as in update-statement, step 4 → guard_failed, message W29.
5. If dry_run: return a WriteResult with the id and the message "would delete <singular> <id>". No write request is sent.
6. Call ApiDelete on the delete route of the entity. Where the route contains the product, it is the product of the record that step 2 read. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message W30.
7. Return the WriteResult with the id, no values, and the message "<Singular> <id> deleted".

POSTCONDITIONS:
- exactly one record was deleted
- when an error is returned from steps 1 to 4, no ApiDelete call was made

ERRORS:
- wrong_kind: K02, K04
- not_found: W20
- guard_failed: W29
- rate_limited, network, api_error: X01, X02, X03, W30

## BEHAVIOR: operator-statements
Constraint: required

The command line (`query`, `exec`, saved queries), the interactive shell
and the HTTP service of the base version run statements for an operator
caller. They keep their commands, options and output formats. This behavior
says what changes for them.

INPUTS:
```
statement: string
options:   the options of the base version, among them
           the default product (--product), the dry run (--dry-run),
           the skipped confirmation (--force), the default release
           (--release), the execution mode (live, offline, prefer-cache)
           and the output format
```

OUTPUTS:
```
the output of the base version in the chosen format, and an exit status
```

PRECONDITIONS:
- none

STEPS:
1. Run parse-statement with caller operator; on failure → the command ends with the error of parse-statement.
2. Check the kind of the statement against the command, as the base version does. A write statement for a command that only reads, or a read statement for the command for write statements → syntax, with the message of the base version.
3. Read statement in live execution: run read-statement with caller operator and the default product of the options; on failure → the command ends with the error of read-statement. AgentLimits do not apply: without LIMIT all matching rows are returned. The command line and the shell write each note to the diagnostic stream.
4. Read statement in offline execution: answer from the local cache as the base version does. Statements that the base version answers from the cache return the same records as before.
5. Describe statement: run describe-statement and print its result.
6. InsertStatement, UpdateStatement or DeleteStatement: run the behavior of that statement with the dry-run option. insert-statement also gets the default release. On failure → the command ends with the error of that behavior.
7. PredicateWrite: run write-by-predicate where it is offered; where it is not offered → syntax, message W05.
8. A command that ends with an error writes the message to the diagnostic stream, and its exit status is not zero. The exit status is zero only when every requested change was made, or when a dry run found no error.

POSTCONDITIONS:
- a result is never cut without notice: without LIMIT every matching row is in the output
- an error found before a write request leaves the account unchanged

ERRORS:
- syntax: a statement of the wrong kind for the command (step 2); a PredicateWrite where it is not offered (W05)
- every error of parse-statement, read-statement, insert-statement, update-statement, delete-statement and write-by-predicate

## BEHAVIOR: write-by-predicate
Constraint: supported

An UPDATE or DELETE whose WHERE clause selects records by a condition
instead of by one id. It exists for operator callers at the command line
only. The MCP tool `execute` never accepts it. Each selected record is
changed through the same steps as in update-statement and
delete-statement.

INPUTS:
```
statement: PredicateWrite
dry_run:   Boolean
force:     Boolean        // do not ask for confirmation
default_product: ProductKey or none
```

OUTPUTS:
```
per record: WriteResult | AqlError
exit status
```

PRECONDITIONS:
- the caller is an operator at the command line
- the statement comes from parse-statement: it has a WHERE clause, and every assigned field is a writable field of the entity

STEPS:
1. Run check-write-values for the assigned fields with form_only true; on failure → the error of check-write-values. This check comes before any record is read, and it is also made in a dry run.
2. Run read-statement with a statement that selects all fields of the entity, with the WHERE condition of the statement and without LIMIT; on failure → the error of read-statement. The selected records are exactly the records that a read statement with the same WHERE clause returns.
3. Show the number of selected records and their ids.
4. If dry_run: stop. The exit status is zero.
5. Unless force: ask for confirmation. Without confirmation → stop without any change; the exit status is not zero.
6. For each selected record, run update-statement from step 5 on, or delete-statement from step 6 on, with the record that step 2 read. A failure for one record is reported with its id → the exit status is not zero; the remaining records are still processed.
7. Report the ids that were changed and the ids that failed.

POSTCONDITIONS:
- no record outside the selection of step 2 was changed
- the exit status is zero only when every selected record was changed

ERRORS:
- every error of check-write-values and read-statement, before any write request
- per record: the errors of update-statement and delete-statement

## BEHAVIOR/INTERNAL: tokenize
Constraint: required

Splits the text of a statement into tokens.

INPUTS:
```
text:    string
lenient: Boolean    // second reading, see parse-statement
```

OUTPUTS:
```
tokens: list of (kind, value, position) | AqlError
// kind: word | text | number | operator | punctuation
// position: 1-based position of the first character of the token
```

PRECONDITIONS:
- none

STEPS:
1. Skip white space, including line breaks. No white space is needed between an operator or a punctuation character and its neighbors.
2. A text starts with an apostrophe (') or with a double quote (") and ends at the next occurrence of the same character. Inside a text:
   a. the starting character written twice is one such character;
   b. a backslash followed by `n` is a line break, a backslash followed by `t` is a tab, and a backslash followed by any other character is that character;
   c. a line break is part of the text;
   d. when lenient is true and the text started with an apostrophe: an apostrophe that directly follows a letter or a digit and is directly followed by a letter is part of the text.
   If the input ends before the text ends → syntax, message S01.
3. An operator is one of `=`, `!=`, `<>`, `<`, `<=`, `>`, `>=`. `<>` is read as `!=`.
4. A punctuation character is one of `(`, `)`, `,`, `;`, `*`.
5. A date without quotes is a text. It has four digits, `-`, two digits, `-` and two digits, and it is not directly followed by a letter, a digit, `_` or `-`.
6. A number is an optional `-`, digits, and optionally `.` and digits, not directly followed by a letter, a digit, `_` or `.`.
7. A word is a letter or `_`, followed by letters, digits, `_` and `.`. Keywords are words. They are recognized without regard to case.
8. Any other character → syntax, message S02.

POSTCONDITIONS:
- with lenient false, an apostrophe ends a text unless it is doubled or follows a backslash

ERRORS:
- syntax: a text that is not closed (S01); an unexpected character (S02)

## BEHAVIOR/INTERNAL: parse-statement
Constraint: required

Reads the text of one statement and returns the statement with canonical
entity and field names. A statement that is correct as written is read
strictly. A statement that cannot be read strictly is read once more with
an apostrophe inside a word taken as part of the text.

INPUTS:
```
text:          string
caller:        Caller
allowed_kinds: set of StatementKind
```

OUTPUTS:
```
statement: Statement | AqlError
```

PRECONDITIONS:
- text is not empty

STEPS:
1. If select is allowed and the text, without surrounding white space, quotes and semicolons, is a ReferenceNumber that has a kind → syntax, message S05.
2. Strict reading: run tokenize with lenient false and parse the tokens with steps 4 to 9. If this succeeds, its result is the result.
3. Second reading. It is made when step 2 failed with a syntax error and the text has an apostrophe directly between a letter or digit and a letter. Run tokenize with lenient true and parse the tokens with steps 4 to 9. If this succeeds, its result is the result. If it fails → the error of step 2.
4. Take the kind from the first token.
   a. SELECT or FROM: select.
   b. The name of a function directly followed by `(`, or `*`: select. The statement is read as if SELECT stood before it.
   c. DESCRIBE: describe. INSERT: insert. UPDATE: update. DELETE: delete.
5. If the kind is not allowed. A write kind where only select and describe are allowed → syntax, message S07. Select or describe where only write kinds are allowed → syntax, message W01.
6. If the first token gives no kind.
   a. Where select is allowed: one word alone that resolve-names accepts as an entity → syntax, message S06. Anything else → syntax, message S04.
   b. Where only write kinds are allowed → syntax, message W02.
7. Parse the statement by the form of its kind (below). Every entity word and field word goes through resolve-names; on failure → the error of resolve-names.
8. After the statement, one `;` may follow. Any further token is an error.
   a. For a read statement → syntax, message S03. What was expected is "the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order".
   b. For a describe statement → syntax, message S14.
   c. For a write statement → syntax, message W03.
9. Constructs outside the forms below.
   a. For an agent caller they are refused. JOIN → syntax, message S08. HAVING → syntax, message S09. OFFSET, or a second number after LIMIT → syntax, message S10. SELECT inside parentheses → syntax, message S11. Anything else → syntax, message S03.
   b. For an operator caller, the constructs of the base version that the forms below do not have stay accepted and keep their behavior. These are JOIN, HAVING, SELECT inside parentheses, relative time values, the `release.` qualifier, SELECT written after WHERE, the entity `tags`, and custom field references (see custom-field-references).

Form of a read statement:

```
[SELECT [DISTINCT] item {, item}] FROM entity [[AS] alias]
    [WHERE condition]
    [GROUP BY field {, field}]
    [ORDER BY key [ASC | DESC] {, key [ASC | DESC]}]
    [LIMIT n]

item       = * | field [AS label] | function ( * | [DISTINCT] field ) [AS label]
function   = COUNT | SUM | AVG | MIN | MAX
key        = field | label of an aggregate item | function ( * | field )
condition  = condition OR condition | condition AND condition
           | NOT condition | ( condition ) | comparison
comparison = field operator value
           | field IS [NOT] NULL
           | field [NOT] IN ( value {, value} )
           | field [NOT] CONTAINS value
           | field [NOT] LIKE value
           | field [NOT] BETWEEN value AND value
value      = text | number | TRUE | FALSE | NULL
```

Rules for a read statement:

- Without SELECT, the item is `*`.
- NOT binds stronger than AND, and AND binds stronger than OR.
- `field = NULL` is read as `field IS NULL`, and `field != NULL` as `field IS NOT NULL`.
- `field BETWEEN a AND b` is read as `field >= a AND field <= b`.
- NOT before IN, CONTAINS, LIKE or BETWEEN negates that comparison.
- The label of a field item is the word as the statement writes it, without a qualifier, or the label after AS. The label of an aggregate item is the label after AS, or the function name in capital letters with its argument in parentheses as written: `COUNT(*)`, `COUNT(name)`, `COUNT(DISTINCT status)`.
- `COUNT(1)` and COUNT with the name of an entity as its argument are read as `COUNT(*)`.
- A qualifier before a field (`f.name`, `features.name`) is ignored, and `alias.*` is `*`.
- A word directly followed by `(` that is not one of the five functions → syntax, message S12.
- A word that is not a keyword, where a value is expected → syntax, message S13.
- LIMIT takes a whole number that is not negative.

Form of a describe statement:

```
DESCRIBE [entity]
```

Forms of a write statement:

```
INSERT INTO entity ( column {, column} ) VALUES ( value {, value} )
UPDATE entity SET field = value {, field = value} WHERE id = value {AND field = value}
DELETE FROM entity WHERE id = value {AND field = value}

value = text | number | TRUE | FALSE
```

Rules for a write statement:

- The entity goes through resolve-names. An unknown entity → bad_name, message W12. An entity without any operation → bad_name, message W13. An entity that does not have the operation of the statement → bad_name, message W14.
- INSERT has one row of values. A second row → syntax, message W07. A different number of columns and values → syntax, message W08.
- A column of INSERT is the parent when it is `parent`, the name of the parent kind of the entity, or another name of the parent field of the entity. The parent given twice with different values → syntax, message W09.
- A column or an assigned field that is the id field of the entity → bad_name, message W15. A column or an assigned field that is unknown or not writable → bad_name, message W16.
- An assignment is `field = value`. Anything between the value and the next `,` or WHERE → syntax, message W10.
- UPDATE and DELETE need a WHERE clause; without it → syntax, message W04.
- The first condition of the WHERE clause must compare the id field of the entity with `=`. The id field may have any spelling that resolve-names accepts. Any other first condition → syntax, message W05. The value must be a text or a number; otherwise → syntax, message W03.
- Each further condition must be joined with AND and have the form `field = value`. OR, NOT or parentheses → syntax, message W05. Another operator → syntax, message W06. A field that is not a field of the entity → bad_name, message W17. These conditions are the guards of the statement.
- NULL as a value → syntax, message W11. A word that is not a keyword, where a value is expected → syntax, message S13.
- Any other departure from the forms → syntax, message W03.
- For an operator caller at the command line, an UPDATE or DELETE whose WHERE clause does not have the form above is a PredicateWrite. Its WHERE clause is parsed as the condition of a read statement. Without a WHERE clause → syntax, message B01.

POSTCONDITIONS:
- the returned statement has canonical entity and field names only
- a text that can be read strictly is never read differently by the second reading
- no input is left after the statement

ERRORS:
- syntax: S03 to S14, W01 to W11, B01, and S01 and S02 from tokenize
- bad_name: W12 to W17, and N01 to N03 from resolve-names

## BEHAVIOR/INTERNAL: entity-catalog
Constraint: required

The catalog is a single table of entities, fields, types, API sources,
routes, parents and operations. Every other behavior of live execution
takes these facts from the catalog. The base version has them in several
places that disagree (see DELTA).

INPUTS:
```
none
```

OUTPUTS:
```
catalog: List<EntityDef>, in the order of the entries below
```

PRECONDITIONS:
- none

STEPS:
1. Take every entity name, field name, type, API source, route, parent, operation, about text and example from the entries below. resolve-names, describe-statement, plan-fetch, convert-record, check-write-values and error-messages use the catalog and no second list.
2. Use the order of the entries below wherever entities are listed, and the order of the field tables wherever fields are listed: in DESCRIBE results, in tool descriptions and in messages.
3. Replace {KEY} in about texts and examples by the ExampleKey.
4. Add an entity by adding an entry. No step of plan-fetch, convert-record or the write statements is written for one entity only, except where an entry below says so.

The entries give the API source of a field as the member of the API record
that has the value. `a.b` is member `b` of member `a`. Every field without
the note "heavy" is requested from a list; a heavy field is requested only
when the statement uses it (see plan-fetch).

### Common other names for fields

They apply to every entity except products, and only when the entity has
the field in the right column. The other field names of an entry take
precedence.

| Written | Means |
|---|---|
| reference_num, ref_num, ref, reference, reference_number, ref_number, refnum | id |
| product_id, product_key, key, project, project_id, workspace, product_name | product |
| workflow_status, state | status |
| title | name |
| created | created_at |
| updated | updated_at |
| assignee, assigned_to_user, assigned_user, owner, assigned | assigned_to |

### ENTITY: products

| Property | Value |
|---|---|
| Singular | product |
| Other names | project, projects, workspace, workspaces |
| Kind | product |
| About | Products (workspaces). The key of a product, for example {KEY}, starts the id of its records. |
| Default fields | key, name |
| Other field names | id, reference_prefix, prefix, reference_num, ref_num, ref, product, product_key, product_id mean key. parent_id means parent. title means name |
| Read | The product list: GET /products, all pages. Conditions are evaluated on the list |
| Parent filters | none |
| Create | POST /products. Parent: a product (the product line above it), optional |
| Update | PUT /products/{id} |
| Delete | not offered |
| Example | SELECT key, name FROM products |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| key | key | reference_prefix | yes | The id field of products. Conditions also accept the NumericId |
| name | string | name | yes | |
| product_line | boolean | product_line | no | false when the API record has no value |
| parent | key | parent_id | no | The key of the product with that NumericId |
| description | text | description | yes | |
| created_at | datetime | created_at | no | |
| url | string | url | no | |

### ENTITY: releases

| Property | Value |
|---|---|
| Singular | release |
| Other names | versions |
| Kind | release |
| About | Releases of a product. The parking lot is a release with parking_lot = true and no date. |
| Default fields | id, name, release_date, parking_lot |
| Other field names | parent, parent_id mean product. theme means description. date, ship_date mean release_date |
| Read routes, in order | id: GET /releases/{id}. product: GET /products/{product}/releases |
| Account-wide list | none |
| Parent filters | product |
| Create | POST /products/{product}/releases. Parent: a product |
| Update | PUT /releases/{id} |
| Delete | DELETE /products/{product}/releases/{id} |
| Example | SELECT id, name, release_date FROM releases WHERE product = '{KEY}' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| release_date | date | release_date | yes | |
| start_date | date | start_date | yes | |
| status | status | workflow_status.name | no | |
| parking_lot | boolean | parking_lot | yes | false when the API record has no value |
| released | boolean | released | no | false when the API record has no value |
| product | key | product_id | no | |
| progress | number | progress | no | |
| description | text | theme | yes | heavy |
| created_at | datetime | created_at | no | |
| url | string | url | no | |

### ENTITY: features

| Property | Value |
|---|---|
| Singular | feature |
| Other names | none |
| Kind | feature |
| About | Features. Each feature is in one release. |
| Default fields | id, name, status, release |
| Other field names | release_id, release_ref, release_reference_num, parent, parent_id mean release. initiative_id means initiative. epic_id means epic |
| Read routes, in order | id: GET /features/{id}. release: GET /releases/{release}/features. epic: GET /epics/{epic}/features. initiative: GET /initiatives/{initiative}/features. product: GET /products/{product}/features |
| Account-wide list | GET /features, with name search |
| Parent filters | product, release, initiative, epic |
| Create | POST /releases/{release}/features. Parent: a release |
| Update | PUT /features/{id} |
| Delete | DELETE /features/{id} |
| Example | SELECT id, name, status FROM features WHERE release = '{KEY}-R-3' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| status | status | workflow_status.name | yes | |
| release | reference: release | release_reference_num | yes | |
| initiative | reference: initiative | initiative_reference_num | yes | |
| epic | reference: epic | epic_reference_num | no | |
| assigned_to | user | assigned_to_user.email | yes | |
| product | key | product_id | no | |
| start_date | date | start_date | yes | |
| due_date | date | due_date | yes | |
| tags | string | tags | yes | The tags, joined by a comma and a space |
| score | number | score | no | |
| progress | number | progress | no | |
| description | text | description | yes | heavy |
| created_at | datetime | created_at | no | |
| updated_at | datetime | updated_at | no | |
| url | string | url | no | |

### ENTITY: epics

| Property | Value |
|---|---|
| Singular | epic |
| Other names | master_features |
| Kind | epic |
| About | Epics. An epic is in one release and groups features. |
| Default fields | id, name, status, release |
| Other field names | release_id, parent, parent_id mean release. initiative_id means initiative |
| Read routes, in order | id: GET /epics/{id}. release: GET /releases/{release}/epics. initiative: GET /initiatives/{initiative}/epics. product: GET /products/{product}/epics |
| Account-wide list | GET /epics, with name search |
| Parent filters | product, release, initiative |
| Create | POST /releases/{release}/epics. Parent: a release |
| Update | PUT /epics/{id} |
| Delete | DELETE /epics/{id} |
| Example | SELECT id, name, status FROM epics WHERE product = '{KEY}' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| status | status | workflow_status.name | yes | |
| release | reference: release | release_reference_num | no | |
| initiative | reference: initiative | initiative_reference_num | yes | |
| assigned_to | user | assigned_to_user.email | no | |
| product | key | product_id | no | |
| start_date | date | start_date | yes | |
| due_date | date | due_date | yes | |
| progress | number | progress | yes | |
| color | string | color | yes | |
| description | text | description | yes | heavy |
| created_at | datetime | created_at | no | |
| updated_at | datetime | updated_at | no | |
| url | string | url | no | |

### ENTITY: requirements

| Property | Value |
|---|---|
| Singular | requirement |
| Other names | none |
| Kind | requirement |
| About | Requirements of a feature. |
| Default fields | id, name, status, feature |
| Other field names | feature_id, parent, parent_id mean feature |
| Read routes, in order | id: GET /requirements/{id}. feature: GET /features/{feature}/requirements |
| Account-wide list | none |
| Parent filters | feature |
| Create | POST /features/{feature}/requirements. Parent: a feature |
| Update | PUT /requirements/{id} |
| Delete | DELETE /requirements/{id} |
| Example | SELECT id, name, status FROM requirements WHERE feature = '{KEY}-23' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| status | status | workflow_status.name | yes | |
| feature | reference: feature | feature.reference_num | no | When the API record has no value: the feature of the statement |
| assigned_to | user | assigned_to_user.email | yes | |
| original_estimate | number | original_estimate | yes | |
| work_done | number | work_done | yes | |
| description | text | description | yes | heavy |
| created_at | datetime | created_at | no | |
| url | string | url | no | |

### ENTITY: initiatives

| Property | Value |
|---|---|
| Singular | initiative |
| Other names | none |
| Kind | initiative |
| About | Initiatives of a product. Features can be linked to an initiative. |
| Default fields | id, name, status |
| Other field names | parent, parent_id mean product |
| Read routes, in order | id: GET /initiatives/{id}. product: GET /products/{product}/initiatives |
| Account-wide list | GET /initiatives, with name search |
| Parent filters | product |
| Create | POST /products/{product}/initiatives. Parent: a product |
| Update | PUT /initiatives/{id} |
| Delete | DELETE /products/{product}/initiatives/{id} |
| Example | SELECT id, name, status FROM initiatives WHERE product = '{KEY}' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| status | status | workflow_status.name | yes | |
| product | key | product_id | no | |
| start_date | date | start_date | yes | |
| end_date | date | end_date | yes | |
| progress | number | progress | no | |
| value | number | value | yes | |
| effort | number | effort | yes | |
| color | string | color | yes | |
| presented | boolean | presented | yes | |
| description | text | description | yes | heavy |
| created_at | datetime | created_at | no | |
| updated_at | datetime | updated_at | no | |
| url | string | url | no | |

### ENTITY: goals

| Property | Value |
|---|---|
| Singular | goal |
| Other names | strategic_imperatives |
| Kind | goal |
| About | Goals of a product. |
| Default fields | id, name |
| Other field names | parent, parent_id mean product |
| Read routes, in order | id: GET /goals/{id}. product: GET /products/{product}/goals |
| Account-wide list | GET /goals, with name search |
| Parent filters | product |
| Create | POST /products/{product}/goals. Parent: a product |
| Update | PUT /goals/{id} |
| Delete | DELETE /products/{product}/goals/{id} |
| Example | SELECT id, name FROM goals WHERE product = '{KEY}' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| status | status | success_metric.workflow_status.name | yes | When that has no value: workflow_status.name |
| product | key | product_id | no | |
| progress | number | progress | yes | |
| value | number | value | no | |
| effort | number | effort | no | |
| description | text | description | yes | heavy |
| created_at | datetime | created_at | no | |
| updated_at | datetime | updated_at | no | |
| url | string | url | no | |

### ENTITY: ideas

| Property | Value |
|---|---|
| Singular | idea |
| Other names | none |
| Kind | idea |
| About | Ideas submitted for a product. feature is the feature an idea was promoted to. |
| Default fields | id, name, status, votes |
| Other field names | parent, parent_id mean product. votes_count means votes. feature_id means feature |
| Read routes, in order | id: GET /ideas/{id}. product: GET /products/{product}/ideas |
| Account-wide list | GET /ideas, with name search |
| Parent filters | product |
| Create | POST /products/{product}/ideas. Parent: a product |
| Update | PUT /ideas/{id} |
| Delete | DELETE /ideas/{id} |
| Example | SELECT id, name, votes FROM ideas WHERE product = '{KEY}' ORDER BY votes DESC LIMIT 5 |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| status | status | workflow_status.name | yes | |
| votes | number | votes | no | |
| product | key | product_id | no | |
| feature | reference: feature | feature.reference_num | no | |
| visibility | string | visibility | yes | |
| description | text | description | yes | heavy |
| created_at | datetime | created_at | no | |
| updated_at | datetime | updated_at | no | |
| url | string | url | no | |

### ENTITY: comments

| Property | Value |
|---|---|
| Singular | comment |
| Other names | none |
| Kind | comment |
| About | Comments on a record. parent is the id of the commented record. |
| Default fields | id, body, user, parent, created_at |
| Other field names | content, text, comment, message, description mean body. author, created_by, author_name, user_name mean user. feature, feature_id, epic, epic_id, idea, idea_id, release, release_id, initiative, initiative_id, goal, goal_id, requirement, requirement_id, record, record_id, parent_id, commentable, product, product_id, product_key mean parent |
| Read routes, in order | id: GET /comments/{id}. parent: the comment route of the parent, chosen by the kind of the value: GET /features/{parent}/comments, /releases/{parent}/comments, /epics/{parent}/comments, /ideas/{parent}/comments, /initiatives/{parent}/comments, /goals/{parent}/comments, /requirements/{parent}/comments, and for a product GET /products/{product}/comments |
| Account-wide list | none |
| Parent filters | parent |
| Create | POST on the comment route of the parent. Parent: a feature, epic, idea, release, initiative, goal or requirement |
| Update | PUT /comments/{id} |
| Delete | DELETE /comments/{id} |
| Example | SELECT body, user, created_at FROM comments WHERE parent = '{KEY}-23' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | id | no | A NumericId |
| body | text | body | yes | |
| user | user | user.name | no | Read as the name of the user. Conditions also accept the e-mail address and the NumericId |
| parent | string | commentable.url | no | The last part of that address, which is the id of the commented record. When the API record has no value: the parent of the statement |
| created_at | datetime | created_at | no | |
| url | string | url | no | |

### ENTITY: users

| Property | Value |
|---|---|
| Singular | user |
| Other names | none |
| Kind | user |
| About | Users of the account. For the user you are working as, use: SELECT name, email FROM me |
| Default fields | id, name, email |
| Other field names | mail, e_mail mean email. user_id means id |
| Read | GET /users. Conditions are evaluated on the list. A condition on product is refused (message N03) |
| Parent filters | none |
| Create, update, delete | not offered |
| Example | SELECT name, email FROM users |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | id | no | A NumericId |
| name | string | name | no | |
| email | string | email | no | |
| created_at | datetime | created_at | no | |

### ENTITY: statuses

| Property | Value |
|---|---|
| Singular | status |
| Other names | workflow_statuses, workflows, workflow |
| Kind | none |
| About | The workflow statuses a record can have, per product and entity. |
| Default fields | name, entity |
| Other field names | status, id, title mean name. type, entity_type, kind mean entity |
| Read | GET /products/{product}/workflows. A conjunct product = value is required. One record per workflow status of each workflow, in workflow order |
| Parent filters | product, entity |
| Create, update, delete | not offered |
| Example | SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| name | string | name of the workflow status | no | |
| entity | string | record type of the workflow | no | Converted to an entity name, see below |
| position | number | position of the workflow status | no | |
| complete | boolean | complete flag of the workflow status | no | false when the API record has no value |
| product | key | none | no | The product of the statement |

The record type of a workflow gives the value of `entity`:

| Record type of the workflow | Value of `entity` |
|---|---|
| Feature | features |
| Release | releases |
| Idea | ideas |
| Initiative | initiatives |
| StrategicImperative | goals |
| Epic, MasterFeature | epics |
| Requirement | requirements |
| KeyResult | key_results |
| Page | pages |
| any other | the record type in lower case |

A condition on `entity` accepts every spelling that resolve-names accepts
for that entity. It also accepts the record type itself.

### ENTITY: me

| Property | Value |
|---|---|
| Singular | me |
| Other names | current_user, myself, whoami |
| Kind | none |
| About | The user you are working as. One record. |
| Default fields | name, email |
| Other field names | mail means email. user means name. user_id means id |
| Read | GET /me. One record |
| Parent filters | none |
| Create, update, delete | not offered |
| Example | SELECT name, email FROM me |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | id | no | A NumericId |
| name | string | name | no | |
| email | string | email | no | |

POSTCONDITIONS:
- every field in a table above is delivered by read-statement and listed by describe-statement
- every field marked writable can also be read

ERRORS:
- none

## BEHAVIOR/INTERNAL: resolve-names
Constraint: required

Turns an entity word or a field word of a statement into a catalog name.
It accepts the names that a language model writes without being told, and
it answers an unknown name with the valid names.

INPUTS:
```
word:    string                // as the statement writes it
entity:  EntityName or none    // none: the word is an entity word
caller:  Caller
writing: Boolean               // the word is part of a write statement
```

OUTPUTS:
```
name: EntityName | FieldName | none | AqlError
// none only when writing is true. The write rules of parse-statement then
// give the message (W12, W16 or W17).
```

PRECONDITIONS:
- none

STEPS:
1. Compare words without regard to case.
2. Entity word: the result is the first catalog entity, in catalog order, whose name, singular or other names have the word. For an operator caller, the entity `tags` of the base version is accepted as well and keeps its base-version behavior.
3. Entity word without a match: when writing is true, the result is none; otherwise → bad_name, message N01.
4. Field word: unless the word starts with `custom.`, remove a qualifier, which is everything up to and including the last `.`. Then take the first match in this order:
   a. a field of the entity;
   b. one of the other field names of the entity;
   c. one of the common other names, when the entity is not products and has the field that the name means;
   d. the entity's own name: the singular, or the singular followed by `_id`, means the id field of the entity, and the singular followed by `_name` means `name`. This applies only when the entity has that field.
5. Field word without a match, for an operator caller.
   a. A word that starts with `custom.` is a custom field reference. It is accepted in a read statement and as an assigned field of UPDATE (see custom-field-references).
   b. In a read statement, a field that the base version accepts for the entity and that the catalog does not have is accepted under its own name. Its value is the member of that name in the API record, or null. describe-statement does not list such a field.
6. Field word without a match, in every other case: when writing is true, the result is none. On users, a word that means `product` on other entities → bad_name, message N03. Any other word → bad_name, message N02.

The id field of an entity is `key` for products and `id` for every other
entity that has a field `id`. The entity statuses has no id field.

POSTCONDITIONS:
- for an agent caller the result is a name of the catalog
- a word that is a field of the entity is never read as another name

ERRORS:
- bad_name: unknown entity (N01); unknown field (N02); a product condition on users (N03)

## BEHAVIOR/INTERNAL: resolve-product
Constraint: required

Finds the product that a value means. A statement may write a product as
its key in any case, as its NumericId or as its name.

INPUTS:
```
value:  Text | Number
entity: EntityName     // entity of the statement, for the wording of P02 to P04
hints:  Boolean        // false in a write statement: every unknown value gives P01
```

OUTPUTS:
```
product: Product | AqlError
```

PRECONDITIONS:
- none

STEPS:
1. Take the product list. When it was not read before: call ApiList on /products for all pages, with the members that the fields of the entity products need. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message X03. The list may be kept between statements.
2. Compare the value with the key and the NumericId of every product, and then with the name of every product. The comparison is the equality of text of evaluate-condition. The first match is the result.
3. Without a match, when the product list was kept from an earlier statement: read it again as in step 1 and repeat step 2 once.
4. Without a match, when hints is true, take the first case that applies:
   a. the value is a ReferenceNumber whose kind is a parent filter of the entity → wrong_kind, message P02;
   b. the value is a ReferenceNumber of the kind of the entity → wrong_kind, message P03;
   c. the value is a ReferenceNumber with a kind, and the part before its first `-` is the key of a product → wrong_kind, message P04.
5. In every other case → bad_product, message P01.

POSTCONDITIONS:
- the key of the result is written as the account writes it, whatever the case of the value
- a product that was created after the list was read is found

ERRORS:
- bad_product: P01
- wrong_kind: P02, P03, P04
- rate_limited, network, api_error: X01, X02, X03

## BEHAVIOR/INTERNAL: plan-fetch
Constraint: required

Chooses the requests for a read statement and returns the API records. It
uses the narrowest route that the WHERE condition allows. A list costs one
request per page and never one request per record.

INPUTS:
```
statement:       ReadStatement
used_fields:     set of FieldName
caller:          Caller
default_product: ProductKey or none
```

OUTPUTS:
```
records:    List<ApiRecord>
scope:      string
total:      u32 or none       // total_records of the list, when the list was
                              // not read to its end
applied:    List<Condition>   // conjuncts that the chosen route has applied
count_only: Boolean           // page 1 was read for a count only (step 8 a)
or AqlError
```

PRECONDITIONS:
- the statement has catalog names only

STEPS:
1. Entities with a fixed source. For them the steps 2 to 7 do not apply.
   a. me: one ApiGet on /me. Scope: "current user".
   b. products: the product list, as resolve-product reads it. Scope: "all products".
   c. users: the list /users, read as step 8 says. Scope: "all users".
   d. statuses: the WHERE condition must have a conjunct `product = value`; otherwise → scope, message C01. Run resolve-product with hints for the value; on failure → the error of resolve-product. Call ApiGet on the workflow route of the product. Every workflow status of every workflow in the answer is one record. Scope: "product <KEY>". The conjunct is applied.
2. Route by an id or a parent. Go through the read routes of the entity in catalog order and leave out the route of the field `product`. Take the first route whose field a conjunct of the WHERE condition compares with `=`. When several conjuncts compare that field, take the first in the order of the statement. Read the route as step 3 says. The conjunct is applied.
3. Reading one route for one value.
   a. Field `id`: the value must be a ReferenceNumber of the entity's kind or a NumericId. For comments it must be a NumericId. Any other value → wrong_kind, with the message that step 4 gives. Call ApiGet on the id route. NotFound gives no record and no error. Scope: "<singular> <id>". The id is written as the record has it. When there is no record, it is the id of the statement in capital letters.
   b. Field `product`: run resolve-product with hints; on failure → the error of resolve-product. Read the list of the product route. Scope: "product <KEY>".
   c. Field `parent` of comments. When the value is a ReferenceNumber of the kind feature, release, epic, idea, initiative, goal or requirement: read the comment list of that record. Scope: "<kind> <VALUE>". When resolve-product finds the value: read the comment list of the product. Scope: "product <KEY>". Any other value → wrong_kind, message K08.
   d. Any other field: the value must be a ReferenceNumber of the kind of the field, or a NumericId. Any other value → wrong_kind, with the message that step 4 gives. Read the list of the route. Scope: "<field> <VALUE>".
   e. A list route that answers NotFound → not_found, message K07.
   f. A ReferenceNumber is sent in capital letters. A product is sent as its NumericId, never as the key that the statement writes.
4. The message for a value of the wrong kind, for the field `id` and for a parent field. Take the first case that applies:
   a. the value is a ReferenceNumber of another kind, the field is `id`, and the entity has a read route for a field of that kind → message K01;
   b. the value is a ReferenceNumber of another kind → message K02;
   c. resolve-product finds the value and the entity has the parent filter `product` → message K03;
   d. the field is `id` → message K04;
   e. the field is release, initiative or epic, and the WHERE condition has a conjunct `product = value` that resolve-product finds → message K05. The records of that kind in the product are read for it with one list request;
   f. in every other case → message K06.
5. Route by a list of values. This step applies when step 2 took no route. Take the first conjunct of the form `field IN (values)` that has at most MaxInValues values and whose field has a read route. The route of the field `product` counts here. Read every value as step 3 says, each list to its last page; the steps 8 a and 8 b do not apply. Put the records together and leave out a record that was read before. Scope: the scopes of the values, joined by a comma and a space. The conjunct is applied.
6. Route by product. This step applies when no route was taken so far and the entity has a read route for the field `product`. When the WHERE condition has a conjunct `product = value`: read it as step 3 b says, and the conjunct is applied. Otherwise, for an operator caller with a default product: read the route for the default product.
7. Account-wide list. This step applies when no route was taken so far.
   a. When the entity has no account-wide list → scope, message C01.
   b. When the entity has name search, look for a conjunct `name = text` or `name CONTAINS text` with a text that is not empty. The text of the first such conjunct is sent with the list request as the parameter `q`. This conjunct is not applied: read-statement still evaluates it on every record.
   c. Read the account-wide list. Scope: "all products".
8. Reading a list. Request it page by page with per_page = PageSize, from page 1 on, every page once.
   a. Count only. It applies when the select items are one `COUNT(*)` and nothing else, the statement has no GROUP BY, and every conjunct of the WHERE condition is applied. Request page 1 only. `total` is its total_records and count_only is true.
   b. Early end. It applies when every conjunct of the WHERE condition is applied and the statement has no ORDER BY, no GROUP BY, no DISTINCT and no aggregate item. Stop when enough records are read. Enough is the LIMIT of the statement; for an agent caller it is default_limit without a LIMIT, and never more than max_limit. For an operator caller without a LIMIT there is no early end. When the list was not read to its end, `total` is its total_records.
   c. Too many pages. For an agent caller: when page 1 reports more than max_pages pages, and neither a nor b ends the reading on page 1 → scope, message C02. No further page is requested.
   d. In every other case read the list to its last page.
   MECHANISM: every list request has the parameter `fields`. It has the API sources of all fields of the entity that are not heavy, and of the heavy fields among used_fields. For an operator caller it also has the base-version fields that the statement uses. One request per page delivers complete records. No request is sent per record of a list.
9. Lookups for the evaluation. When the WHERE condition compares a field of the type user with `me`, `myself` or `current_user`, read the current user with one ApiGet on /me. When it compares a field of the type key outside the applied conjuncts, take the product list as resolve-product does. Each lookup is read at most once per statement.
10. A request of the steps 1 to 9 that fails in another way than the steps say: RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message X03. No partial result is returned.

POSTCONDITIONS:
- every record of the result belongs to the scope
- every page of a list was requested at most once, with increasing page numbers
- the number of requests does not grow with the number of records in a page
- for an agent caller at most max_pages pages of one list were requested

ERRORS:
- scope: an entity that needs a product or a parent in the WHERE condition (C01); too many pages (C02)
- bad_product: P01, from resolve-product
- wrong_kind: P02 to P04 from resolve-product; K01 to K06 and K08
- not_found: a parent that does not exist (K07)
- rate_limited, network, api_error: X01, X02, X03

## BEHAVIOR/INTERNAL: convert-record
Constraint: required

Turns one API record into a record with the fields of the catalog.

INPUTS:
```
entity:        EntityName
api_record:    ApiRecord
known_product: ProductKey or none   // the route was a product route
known_parent:  RecordId or none     // the route was a parent route
```

OUTPUTS:
```
record:          Map<FieldName, Value>          // every field of the entity
other_spellings: Map<FieldName, List<Value>>    // accepted by conditions, never shown
```

PRECONDITIONS:
- the API record was read with the members that the catalog gives as sources

STEPS:
1. For every field of the entity, take the value from the API source of the catalog. A member that is absent, and a source `a.b` whose member `a` is absent, give null.
2. Convert by the type of the field.
   a. text: turn HTML into plain text. The end of a paragraph, of a list item, of a heading, of a block and of a table row, and a line break tag, become a line break. Every other tag is removed. Character references are replaced by their characters. Several spaces in a row become one space. Empty lines and surrounding white space are removed. Empty text gives null.
   b. date: the first 10 characters of the value.
   c. boolean: true or false. An absent value gives null, except where the catalog says false.
   d. key: for the field `product`, the known product; without it, the key of the product whose NumericId the source has. The product list is taken for this at most once per statement, and only when the statement uses the field.
   e. every other type: the value as the API has it.
3. Apply the notes of the catalog entry. The tags of a feature are joined by a comma and a space, and a feature without tags has null. A requirement without a feature in the API record takes the known parent, and so does a comment without a commented record. The status of a goal has two sources.
4. Collect the other spellings of a value. Conditions accept them; no result shows them.
   - `id`: the NumericId of the record. For products, `key`: the NumericId of the product.
   - `product`: the NumericId of the product.
   - `assigned_to`: the name and the NumericId of the user.
   - a field of the type reference: the NumericId and the name of the referenced record, where the API record has them.
   - `user` of comments: the e-mail address and the NumericId of the user.
   - `parent` of comments: the NumericId of the commented record.
   - `entity` of statuses: the singular of the entity and the record type of the workflow.
5. For an operator caller, add the base-version fields and the custom fields that the statement uses (see resolve-names and custom-field-references).

POSTCONDITIONS:
- the record has every field of the entity, each with a value or null
- a field of the type text has no HTML tags

ERRORS:
- none

## BEHAVIOR/INTERNAL: evaluate-condition
Constraint: required

Decides whether one record satisfies a condition. Every form of condition
that parse-statement accepts is evaluated. No form is skipped.

INPUTS:
```
record:    the output of convert-record
condition: Condition
entity:    EntityName
lookups:   the current user and the product list, where they were read
           (see plan-fetch, step 9)
```

OUTPUTS:
```
result: Boolean
```

PRECONDITIONS:
- every field of the condition is a field of the record
- the lookups that the condition needs were read; update-statement and delete-statement read them in the same way before they check their guards, and a failed request there is the error of plan-fetch, step 10

STEPS:
1. AllOf is true when every member is true. AnyOf is true when at least one member is true. Negation is true when its member is not true.
2. NullTest: a field has no value when it is null or empty text. IS NULL is true for a field without a value, and IS NOT NULL for a field with a value.
3. InList is true when the comparison `field = value` is true for at least one of its values.
4. Comparison. First replace the value of the statement in three cases.
   a. The field has the type user and the value is `me`, `myself` or `current_user` in any case. The value becomes the e-mail address of the current user. Where the field is read as a name, it becomes the name of the current user.
   b. The field has the type key and resolve-product finds the value. The value becomes the key of that product.
   c. The field is `entity` of statuses and resolve-names accepts the value as an entity word. The value becomes the name of that entity.
5. Text is compared in its comparison form. The comparison form has no surrounding white space and no difference of case. Typographic apostrophes and quotation marks are replaced by ' and ", and the no-break space is replaced by a space.
6. `=` is true when the value equals the value of the field or one of its other spellings:
   a. when one side is a Boolean: the truth values are equal, where the texts `true` and `false` and the numbers 1 and 0 are read as truth values;
   b. when the field has the type number: the numbers are equal;
   c. when the field has the type date or datetime and the value has 10 characters: the first 10 characters of the field equal the value;
   d. in every other case: the comparison forms are equal.
   A field without a value equals nothing.
7. `!=` is true when `=` is not true. It is true for a field without a value.
8. CONTAINS is true when the comparison form of the value is inside the comparison form of the field or of one of its other spellings. LIKE is true when the pattern covers the text from its first to its last character, where `%` stands for any text and `_` for one character. Both are false for a field without a value.
9. `<`, `<=`, `>` and `>=` are false for a field without a value. A field of the type number is compared as a number. A field of the type date or datetime is compared as text, and against a value of 10 characters only its first 10 characters are compared. Every other field is compared in its comparison form.

POSTCONDITIONS:
- the result depends only on the record, the condition and the lookups
- `=` on `name` is equality of the complete name, not a search

ERRORS:
- none

## BEHAVIOR/INTERNAL: build-rows
Constraint: required

Turns the records that satisfy the WHERE condition into the rows of the
result: grouping, aggregates, order, DISTINCT and limits.

INPUTS:
```
statement:  ReadStatement     // Star already replaced by fields
records:    the kept records, in the order in which they were read
caller:     Caller
list_total: u32 or none       // total of plan-fetch
count_only: Boolean           // count_only of plan-fetch
```

OUTPUTS:
```
rows:      List<Row>
count:     u32
total:     u32 or none
truncated: Boolean
```

PRECONDITIONS:
- when the statement has an aggregate item or GROUP BY and count_only is false, the records are all matching records: plan-fetch has read every list to its end

STEPS:
1. Count only: when count_only is true, the result is one row with the label of the item and the value list_total. count is 1, total has no value and truncated is false.
2. Grouped result: when the statement has an aggregate item or GROUP BY.
   a. The group fields are the fields of GROUP BY, followed by the selected fields that are not among them. `SELECT status, COUNT(*) FROM features` groups by status without GROUP BY.
   b. Records with equal values in all group fields are one group. Text values are equal when their comparison forms are equal (see evaluate-condition). The groups are in the order of their first record. Without group fields all records are one group, and there is one group also when there is no record.
   c. One row per group. First the group fields that the statement does not select, under their field names. Then the select items in the order of the statement: a field item has the value of the first record of the group, and an aggregate item has its result.
   d. COUNT(*) is the number of records of the group. COUNT(field) is the number of records with a value in the field. COUNT(DISTINCT field) is the number of different values. SUM and AVG use the values that are numbers. MIN and MAX use the order of step 4. Without a value the result is null, and 0 for COUNT. A result that is a whole number is written without decimals; every other result is rounded to two decimals.
3. Plain result: when the statement has no aggregate item and no GROUP BY. Sort the records as step 4 says. Make one row per record with the select items in the order of the statement, each under its label. With DISTINCT, leave out a row that equals an earlier row in all values.
4. ORDER BY. Sort by the first key. Rows that are equal in it are sorted by the next key, and so on. Rows that are equal in all keys keep their order. A key is a field, the label of an aggregate item, or an aggregate that the statement selects. Numbers come before text. Numbers are sorted by value and text by its comparison form. A row without a value in the key comes last, with ASC and with DESC. A grouped result is sorted after step 2.
5. Limits. The limit is the LIMIT of the statement. For an agent caller it is default_limit without a LIMIT, and never more than max_limit. For an operator caller without a LIMIT there is no limit. rows are the first rows up to the limit, and count is their number.
6. total has a value in two cases. When step 5 cut rows, it is the number of rows before the cut. When plan-fetch did not read the list to its end, it is list_total.
7. truncated is true when total has a value and the statement did not ask for the cut: it has no LIMIT, or for an agent caller a LIMIT above max_limit.

POSTCONDITIONS:
- an aggregate is computed over all matching records, also when step 5 cuts the rows of a grouped result
- every row has the labels of the statement in its order; in a grouped result the group fields that are not selected come first
- for an agent caller count is never above max_limit

ERRORS:
- none

## BEHAVIOR/INTERNAL: check-write-values
Constraint: required

Checks the values that a write statement assigns, before the write
request, and returns them in the form that is sent.

INPUTS:
```
entity:    EntityName
fields:    Map<FieldName, Value>
product:   Product or none     // product of the record that is changed
parent:    RecordId or none    // parent of a new record
form_only: Boolean             // check only what needs no record
```

OUTPUTS:
```
fields: Map<FieldName, Value> | AqlError
```

PRECONDITIONS:
- every field is a writable field of the entity, or for an operator caller a custom field reference

STEPS:
1. Go through the fields in the order of the statement and check each one by its type. The first value that fails ends the check. An assigned custom field is not checked: its value is sent as the statement has it.
2. string, text and key: the value must be Text; otherwise → bad_value, message W28. The text is sent as the statement has it. Empty text is a value.
3. number: a Number, or Text that reads as a number; otherwise → bad_value, message W26.
4. boolean: a Boolean, or the text `true` or `false` in any case; otherwise → bad_value, message W27.
5. date: Text of the form YYYY-MM-DD that is a day of the calendar. Of a timestamp that starts with such a date, the first 10 characters are taken. Any other value → bad_value, message W22.
6. user: `me`, `myself` or `current_user` in any case is replaced by the e-mail address of the current user, read with ApiGet on /me. An e-mail address and a NumericId are sent as they are. Any other value → bad_value, message W23.
7. reference: the value must be a ReferenceNumber of the kind of the field, or a NumericId. It is sent in capital letters. For a value of another form: when resolve-product finds it as a product → wrong_kind, message W24. The records of that kind in the product are read for the message with one list request. Any other value → not_found, message W25.
8. The field `tags`: the text is split at commas into tags. White space around a tag is removed and an empty tag is left out.
9. status, unless form_only is true.
   a. Take the workflow statuses of the entity in the product, in workflow order, as plan-fetch reads them for the entity statuses.
   b. For a new record the product is the product of the parent. When the parent is a record, read it with ApiGet on its id route to learn its product; NotFound → not_found, message W25.
   c. Compare the value with the names of the statuses. The comparison is the equality of text of evaluate-condition. The name is sent as the workflow writes it.
   d. Without a match, read the statuses again when they were kept from an earlier statement, and compare once more. Still without a match → bad_value, message W21.
10. A request of this behavior that fails in another way: RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message X03.

POSTCONDITIONS:
- no ApiCreate, ApiUpdate, ApiDelete or ApiSetCustomFields call was made
- a status is sent with the name that the workflow has

ERRORS:
- bad_value: W21, W22, W23, W26, W27, W28
- wrong_kind: a product where a record is needed (W24)
- not_found: W25
- rate_limited, network, api_error: X01, X02, X03

## BEHAVIOR/INTERNAL: custom-field-references
Constraint: required

Custom fields in the statements of an operator caller, written
`custom.<key>`. The base version accepts them. This behavior keeps them
and corrects how their values are read and compared. An agent caller
cannot use them: for an agent caller resolve-names answers `custom.<key>`
with message N02.

INPUTS:
```
reference: FieldName     // "custom.<key>"
record:    ApiRecord
```

OUTPUTS:
```
value: Value
```

PRECONDITIONS:
- the caller is an operator
- the entity is one for which the base version accepts custom field references

STEPS:
1. A statement can use a custom field reference in a select item, in the WHERE condition and in ORDER BY. It then reads the custom fields of the records together with the list, in the way the base version reads them. The list itself is read as step 8 of plan-fetch says: every page once, with its page number.
2. The value of the reference is the value of the custom field with that key in the API record. A value is never returned in an encoded form.
   a. Text is Text, a number is a Number and a truth value is a Boolean.
   b. A list of values is Text, with the values joined by a comma and a space.
   c. An absent field is Null.
3. evaluate-condition and build-rows treat the reference as a field of the type number when the value is a Number, and of the type string in every other case. `custom.priority = 'High'` is true for a record whose custom field priority has the value High.
4. The label of a selected reference is the reference as the statement writes it.
5. In UPDATE, an assigned custom field is written with ApiSetCustomFields after the catalog fields, as update-statement says.

POSTCONDITIONS:
- reading a list with custom fields ends after its last page
- a comparison with a custom field is true for the records whose value is equal

ERRORS:
- none

## BEHAVIOR/INTERNAL: error-messages
Constraint: required

The texts of all errors and notes. A language model learns from an error
what to write next. Every message therefore lists the valid names or
values, or shows a statement that parse-statement accepts.

INPUTS:
```
id:     the ID of a message in the tables below
values: the values for its placeholders
```

OUTPUTS:
```
message: string
```

PRECONDITIONS:
- choose-example-key has run

STEPS:
1. Take the text of the ID from the tables below and replace every placeholder. The ID itself is not part of the message.
2. Write a value of the statement, shown below as `'<value>'`, as a statement has to write it: in apostrophes, with every apostrophe of the value written twice. `<VALUE>` is the value in capital letters.
3. Replace `<KEY>` by the ExampleKey. `<PRODUCT>` is the key of the product that the message is about.
4. Take lists of entities and of fields from the catalog, in catalog order, joined by a comma and a space.
5. Show at most MaxListed entries of a list of records or of product keys. When there are more, end the list with `, and <n> more`.
6. The message of an error is the complete tool result of query-tool and of execute-tool for that error. A note is part of the result of a read statement.

The messages speak of the query tool and the execute tool. For an operator
caller these are the command for read statements and the command for write
statements; the texts stay the same.

A hint that is not empty starts with one space, so that it can follow the
period of the sentence before it.

Placeholders that several messages use:

| Placeholder | Text |
|---|---|
| `<example of X>` | The example statement of the entity X, with {KEY} replaced |
| `<entity list>` | The names of all catalog entities |
| `<field list>` | The names of all fields of the entity |
| `<writable list>` | The names of the writable fields of the entity |
| `<changeable list>` | The names of the entities that have an operation |
| `<read form>` | `SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]` |
| `<write forms>` | The three form lines of the description of `execute`, joined by a space, a vertical bar and a space |
| `<where>` | `the end of the statement`, or `position <n> (<text>)` with the position of the token that cannot be read and the next 24 characters of the statement from there, without surrounding white space |
| `<apostrophe hint>` | Empty, unless the statement has an apostrophe directly between a letter or digit and a letter. Then: ` The text contains an apostrophe (<word>). Inside text, write an apostrophe twice: <word doubled>`. `<word>` is the run of characters without a space around the first such apostrophe, without the characters ' " % ( ) , ; at its ends |
| `<example id>` | By kind: feature `<KEY>-23`, release `<KEY>-R-3`, initiative `<KEY>-S-1`, epic `<KEY>-E-1`, goal `<KEY>-G-1`, idea `<KEY>-I-1`, requirement `<KEY>-23-1` |
| `<records>` | One entry `<id> (<name>)` per record, joined by a comma and a space; `none` when there is no record |
| `<singular>`, `<kind>` | The singular of the entity, or the name of the record kind |

Messages of the tool calls:

| ID | Kind | Text |
|---|---|---|
| A01 | bad_args | `query parameter is required: one AQL statement. Example: <example of features>` |
| A02 | bad_args | `tool <tool> has one parameter: <parameter> (one AQL statement). It does not accept: <names>.<parameter hint>` |
| A03 | bad_args | `statement parameter is required: one INSERT, UPDATE or DELETE statement` |

In A02, `<names>` are the unknown parameter names of the call, joined by a
comma and a space. `<parameter hint>` is ` Name the product inside the
statement: ... WHERE product = '<KEY>'` when the tool is `query` and one of
the names is `product`. It is ` Pass the statement as <parameter>.` when one
of the names is `query`, `statement`, `aql` or `sql`. Otherwise it is empty.

Messages for a statement that cannot be read:

| ID | Kind | Text |
|---|---|---|
| S01 | syntax | `the text that starts with the <quote> at position <n> is not closed.<text hint>` |
| S02 | syntax | `unexpected character <c> at position <n>.<apostrophe hint>` |
| S03 | syntax | `cannot read the statement at <where>: expected <what>.<apostrophe hint> Form: <read form>. Example: <example of features>` |
| S04 | syntax | `a statement starts with SELECT or DESCRIBE.<apostrophe hint> Form: <read form>. Example: <example of features>` |
| S05 | syntax | `<id> is an id, not a statement. To read that record: SELECT * FROM <entity> WHERE id = '<id>'` |
| S06 | syntax | `<word> is an entity, not a statement. To list its records: <example of entity>. To see its fields: DESCRIBE <entity>` |
| S07 | syntax | `query only reads. Use the execute tool for INSERT, UPDATE and DELETE.` |
| S08 | syntax | `JOIN is not supported. Read one entity per statement and filter by its parent, for example: <example of features>` |
| S09 | syntax | `HAVING is not supported. Use WHERE, GROUP BY, ORDER BY and LIMIT. Example: SELECT status, COUNT(*) FROM features WHERE product = '<KEY>' GROUP BY status` |
| S10 | syntax | `OFFSET is not supported. Use WHERE to narrow the result and LIMIT to shorten it.` |
| S11 | syntax | `subqueries are not supported. Send one statement per call. Form: <read form>` |
| S12 | syntax | `unknown function <NAME>. Functions: COUNT, SUM, AVG, MIN, MAX. Example: SELECT COUNT(*) FROM features WHERE product = '<KEY>'` |
| S13 | syntax | `text goes in single quotes: write '<word>' instead of <word>` |
| S14 | syntax | `DESCRIBE <entity> lists the fields of <entity> and takes nothing else. To read records: <example of entity>` |

In S01, `<quote>` is `apostrophe` or `double quote`. `<text hint>` is the
apostrophe hint when that is not empty. Otherwise it is ` Text goes in
single quotes: 'like this'.` for a text that starts with an apostrophe, and
` Use single quotes for text: 'like this'.` for a text that starts with a
double quote.

In S03, `<what>` is one of these texts:

- `FROM and an entity`
- `an entity (<entity list>)`
- `a field, * or COUNT(*) after SELECT`
- `a field, * or COUNT(*)`
- `a field name`
- `a field name (<field list>)`
- `a field or * inside <FUNCTION>( )`
- `a name after AS`
- `BY after GROUP`
- `BY after ORDER`
- `a number after LIMIT`
- `a value in single quotes`
- `NULL after IS`
- `'(' after IN`
- `')'`
- `AND in BETWEEN`
- `a comparison after the field <field> (=, !=, <, >, CONTAINS, LIKE, IN, IS NULL)`
- `the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order`

Messages for names, scope, products and ids:

| ID | Kind | Text |
|---|---|---|
| N01 | bad_name | `unknown entity '<word>'. Entities: <entity list>. Example: <example of features>` |
| N02 | bad_name | `unknown field '<word>' on <entity>.<field hint> Fields of <entity>: <field list>. Example: <example of entity>` |
| N03 | bad_name | `users cannot be filtered by product. Fields of users: <field list>. Example: <example of users>` |
| C01 | scope | `<entity> are listed per <parents>: add <conditions> to WHERE. Example: <example of entity>` |
| C02 | scope | `this would read <total> records. <narrowing hint>` |
| P01 | bad_product | `unknown product '<value>'. Product keys: <keys>. To see their names: SELECT key, name FROM products` |
| P02 | wrong_kind | `'<value>' is a <kind>, not a product. Use: WHERE <kind> = '<VALUE>'` |
| P03 | wrong_kind | `'<value>' is the id of a <kind>, not a product. Use: WHERE id = '<VALUE>'` |
| P04 | wrong_kind | `'<value>' is a <kind>, not a product. Its product is <PRODUCT>: WHERE product = '<PRODUCT>'` |
| K01 | wrong_kind | `'<value>' is the id of a <kind>, not of a <singular>. For the <entity> of that <kind>: SELECT * FROM <entity> WHERE <kind> = '<VALUE>'` |
| K02 | wrong_kind | `'<value>' is the id of a <kind>, not of a <expected kind>. To read it: SELECT * FROM <entity of kind> WHERE id = '<VALUE>'` |
| K03 | wrong_kind | `'<value>' is a product, not a <expected kind>. Use: WHERE product = '<PRODUCT>'` |
| K04 | wrong_kind | `'<value>' is not an id. An id looks like '<example id>'. To search by name: SELECT * FROM <entity> WHERE name CONTAINS '<value>'` |
| K05 | wrong_kind | `<field> = needs the id of the <field>, not its name. The <entity of field> of <PRODUCT>: <records>` |
| K06 | wrong_kind | `<field> = needs the id of the <field>, for example '<example id>', not its name. To find the id: SELECT id, name FROM <entity of field> WHERE product = '<KEY>'` |
| K07 | not_found | `there is no <kind> with the id '<value>'. To find it: <example of the entity of that kind>` |
| K08 | wrong_kind | `parent = needs the id of the commented record, for example '<KEY>-23'. Got '<value>'.` |

In N02, `<field hint>` is ` Features cannot be filtered by goal.` for the
words `goal` and `goal_id` on features, and empty otherwise.

In C01, `<parents>` are the fields of the read routes of the entity other
than `id`, joined by ` or `, and `<conditions>` are the same fields, each
as `<field> = '...'`, joined by ` or `. For comments the text is `comments
are listed per record: add parent = '...' to WHERE. Example: <example of
comments>`. For statuses `<parents>` is `product`.

In C02, `<narrowing hint>` depends on the list that was read:

- for the account-wide list: `Narrow it down, for example with product = '<KEY>'`
- for another list: `Narrow it down, for example with <field> = '<example id>'`. `<field>` is the first parent filter of the entity, other than `product`, that the WHERE condition does not use
- when there is no such filter: `Use LIMIT without ORDER BY, GROUP BY and DISTINCT to read the first records, or COUNT(*) alone to count them.`

In P01, `<keys>` are the product keys of the account in alphabetical
order, joined by a comma and a space.

In K04, for comments the text is `'<value>' is not the id of a comment. To
find the id: SELECT id, body FROM comments WHERE parent = '<KEY>-23'`.

Messages of write statements:

| ID | Kind | Text |
|---|---|---|
| W01 | syntax | `execute only changes data. Use the query tool for SELECT and DESCRIBE.` |
| W02 | syntax | `a statement starts with INSERT, UPDATE or DELETE.<apostrophe hint> Forms: <write forms>` |
| W03 | syntax | `<problem><apostrophe hint> Forms: <write forms>` |
| W04 | syntax | `<KEYWORD> needs WHERE id = '...' with the id of the one record to change.<apostrophe hint> Forms: <write forms>` |
| W05 | syntax | `one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record.` |
| W06 | syntax | `after WHERE id = '...', further conditions must have the form AND field = 'value'` |
| W07 | syntax | `one INSERT creates one record. Send one statement per record.` |
| W08 | syntax | `<n> fields but <m> values` |
| W09 | syntax | `the parent is given twice with different values` |
| W10 | syntax | `a field can only be set to a value, not to an expression` |
| W11 | syntax | `NULL cannot be written. Leave the field out.` |
| W12 | bad_name | `unknown entity '<word>'. Entities that can be changed: <changeable list>` |
| W13 | bad_name | `<entity> cannot be changed. Entities that can be changed: <changeable list>` |
| W14 | bad_name | `<entity> cannot be <done>. Entities that can be <done>: <list>` |
| W15 | bad_name | `the id of a record cannot be set. Fields of <entity> that can be set: <writable list>` |
| W16 | bad_name | `<opening> Fields of <entity> that can be set: <writable list><parent note>` |
| W17 | bad_name | `unknown field '<word>' on <entity>. Fields of <entity>: <field list>` |
| W18 | bad_value | `parent is required: INSERT needs the column parent with the id of <parent text>. Form: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)` |
| W19 | bad_value | `<field> is required to create a <singular>` |
| W20 | not_found | `there is no <singular> with the id '<value>'` |
| W21 | bad_value | `workflow status '<value>' not found for <singular>; valid statuses: <names>` |
| W22 | bad_value | `<field>: invalid date '<value>', use YYYY-MM-DD` |
| W23 | bad_value | `<field>: there is no user '<value>'. Use an e-mail address, or 'me' for yourself` |
| W24 | wrong_kind | `'<value>' is a product, not a <kind>. Use the id of one of its <entity of kind>: <records>` |
| W25 | not_found | `there is no <kind> with the id '<value>'. To find the id, use the query tool: SELECT id, name FROM <entity of kind> WHERE product = '<PRODUCT>'` |
| W26 | bad_value | `<field> must be a number` |
| W27 | bad_value | `<field> must be true or false` |
| W28 | bad_value | `<field> must be text in single quotes` |
| W29 | guard_failed | `nothing was changed: <singular> <id> has <field> = '<current value>', not '<value>'` |
| W30 | api_error | `aha: API error (status <n>) on <method> <route>: <answer>` |
| B01 | syntax | `<KEYWORD> needs a WHERE clause. To change one record: WHERE id = '...'` |

In W03, `<problem>` is one of these texts:

- `expected INTO after INSERT.`
- `expected ( and the field names after the entity.`
- `expected a field name at <where>.`
- `expected ) after the field names at <where>.`
- `expected VALUES at <where>.`
- `expected ( after VALUES.`
- `expected ) after the values at <where>.`
- `expected SET after the entity.`
- `expected = after the field <field>.`
- `expected a value in single quotes at <where>.`
- `expected FROM after DELETE.`
- `WHERE id = needs the id in single quotes.`
- `unexpected text at <where>. One statement, one record.`

In W04 and B01, `<KEYWORD>` is `UPDATE` or `DELETE`. In W14, `<done>` is
`created` for INSERT, `changed` for UPDATE and `deleted` for DELETE, and
`<list>` are the entities that have that operation.

In W16, `<opening>` is `unknown field '<word>' on <entity>.` for a word
that resolve-names does not know, and `the field <field> of <entity> cannot
be set.` for a field that is not writable. `<parent note>` is `, and parent
in INSERT` when a new record of the entity has a parent, and empty
otherwise.

In W18, `<parent text>` is `a release`, `a product`, `a feature` or `the
record to comment on`. In W19, `<field>` is `name`, or `body` for a
comment. In W21, `<names>` are the names of the workflow statuses in
workflow order, joined by a comma and a space.

In W25, `<PRODUCT>` is the key of the product of the record that the
statement changes. Without such a record it is the part of the value
before its first `-`, when that part is the key of a product. In every
other case it is the ExampleKey.

In W29, for a field without a value the text is `nothing was changed:
<singular> <id> has no <field>, not '<value>'`.

Messages for failed requests:

| ID | Kind | Text |
|---|---|---|
| X01 | rate_limited | `aha: API error (status 429): too many requests` |
| X02 | network | `cannot reach Aha!: <reason>` |
| X03 | api_error | `aha: API error (status <n>) on <method> <route>: <answer>` |

In X03 and W30, `<answer>` is the first 300 characters of the text that
the API returned. W30 has the text of X03. It has its own ID because a
write request causes it.

Notes in the result of a read statement:

| ID | Text |
|---|---|
| H01 | `* shows <default fields>. Other fields of <entity>: <other fields>.` |
| H02 | `Showing <count> of <total> records. Narrow the result with WHERE, raise LIMIT (up to <max_limit>), or count with COUNT(*).` |
| H03 | `There is no <singular> with the id '<value>'.` |
| H04 | `'<value>' is the key of a product, not its name. Use: SELECT key, name FROM products WHERE key = '<PRODUCT>'` |
| H05 | `'<value>' looks like an id, not a name. Use: WHERE id = '<VALUE>'` |
| H06 | `'<value>' looks like the id of a <kind>, not a name. Use: WHERE <kind> = '<VALUE>'` |
| H07 | `No <singular> has exactly the name '<value>'. name = compares the whole name; to search inside names use: name CONTAINS '<value>'` |

POSTCONDITIONS:
- every complete statement that a message shows is accepted by parse-statement for an agent caller, after its placeholders are replaced
- no message tells the reader to use an option of the command line

ERRORS:
- none

## BEHAVIOR/INTERNAL: choose-example-key
Constraint: required

Chooses the product key that tool descriptions, DESCRIBE results, examples
and messages show. A language model uses every literal of an example as a
value. A key of the account that the user did not choose is therefore
never an example.

INPUTS:
```
default_product: the default product setting of the server, or none
caller:          Caller
```

OUTPUTS:
```
key: ExampleKey
```

PRECONDITIONS:
- none

STEPS:
1. Agent callers, once at the start of the server: when a default product is configured and resolve-product finds it, the ExampleKey is the key of that product.
2. In every other case the ExampleKey is PROD. While a product of the account has the ExampleKey as its key, in any case, add the letter X at its end.
3. When a default product is configured and resolve-product does not find it, the server writes a warning to its diagnostic output and continues with step 2.
4. When the product list cannot be read at the start, the ExampleKey is PROD, the server writes a warning to its diagnostic output, and the server starts.
5. Operator callers: the ExampleKey is the default product of the options when one is given, and PROD otherwise. The product list is not read for this.

POSTCONDITIONS:
- the ExampleKey is the configured default product, or a key that no product of the account had when the server started

ERRORS:
- none

## PRECONDITIONS

- The account, the credentials and the address of the Aha! API are configured as in the base version.
- The MCP server has a selection of tool groups with one group for the AQL tools, and a read-only mode in which no write tool is registered. The base version 0.12.0 has neither. Both are a prerequisite of tool-registration and are not specified in this document.
- The Aha! API reports total_records and total_pages for every list.
- The Aha! API accepts the parameter `fields` on the list routes of the catalog and a page size of PageSize. This was observed on one account on 2026-11-12 (see DELTA).

## POSTCONDITIONS

- After a read statement or a describe statement, no data in Aha! has changed.
- After a call of `execute` that returns a result, exactly one record was created, changed or deleted, and the result gives its id.
- After a write statement that returns an error from parsing, from name resolution, from the id check, from a guard or from a value check, no data in Aha! has changed.
- Every result of a read statement gives its scope, and it says so when rows were cut.
- Every error gives the valid names or values, or a statement that can be sent as it is.

## INVARIANTS

- [observable] every row of a read result satisfies the WHERE condition of its statement, for every form of condition that parse-statement accepts, OR and NOT included
- [observable] a read result has every matching record, unless truncated is true or the statement has a LIMIT, and then total gives the number of matching rows
- [observable] every field that describe-statement lists for an entity is delivered by read-statement with its value or with null, and an agent caller gets an error with the list of the valid fields for every other field word
- [observable] every read result gives its scope: the record, the parent, the product or the account-wide list that was read
- [observable] a result row has the labels of the select items in the order of the statement, a grouped result has the fields of GROUP BY that are not selected before them, and a field without a value is null
- [observable] one call of execute creates, changes or deletes at most one record
- [observable] a write statement is checked completely before its write request: an unknown column, an unknown field, a value of the wrong form, an unknown status and a guard that is not true are errors, and none of them changes data
- [observable] an accepted write statement sends every field that it assigns; no column, assignment or condition is ignored without an error
- [observable] the tool query sends only read requests, and in read-only mode no registered tool accepts INSERT, UPDATE or DELETE
- [observable] text that follows a complete statement is an error, for every statement kind and for every caller
- [observable] the error for an unknown entity or field lists the valid names, and every syntax error shows a form or a statement that parse-statement accepts
- [observable] every example statement in a tool description, in a DESCRIBE result, in an error message and in a note is accepted by parse-statement for an agent caller
- [observable] an example statement or example id in a tool description, a DESCRIBE result, an error message or a note has the ExampleKey, or the key of the product that the statement of the caller is about; the ExampleKey is the configured default product or a key that no product of the account has
- [observable] a statement that the strict reading accepts has the same result with and without the second reading
- [observable] in one reading of a list every page is requested once, with increasing page numbers, and the reading ends
- [observable] for an agent caller one statement requests at most max_pages pages of one list and returns at most max_limit rows
- [observable] an aggregate is computed over all matching records and never over a list that was cut
- [observable] offline execution from the local cache returns what it returns in the base version
- [implementation] the entity-catalog is the single source for name resolution, DESCRIBE, the parameter fields, record conversion, value checks and the lists in messages; live execution has no second list of entities or fields
- [implementation] no list is read with one request per record

## EXAMPLES

The examples use FakeAha with the fixture account of the INTERFACES
section, and every example starts from the unchanged fixture account. The
ExampleKey is PROD. A tool result is shown as compact JSON: white space
between the JSON tokens is not part of the expectation, the order of the
members is. A statement and a message are shown between backticks, which
are not part of them. "No write request" means that FakeAha recorded no
ApiCreate, ApiUpdate, ApiDelete and ApiSetCustomFields call.

### EXAMPLE: query_result_envelope (of: query-tool)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-3","count":3,"records":[{"id":"DEMO-3","name":"Live dashboard","status":"In development"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration"},{"id":"DEMO-5","name":"Rider's handbook","status":"Shipped"}]}`
  FakeAha recorded exactly one call: ApiList on /releases/DEMO-R-3/features, page 1

### EXAMPLE: query_rejects_product_parameter (of: query-tool)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT id FROM features` and with a
  second parameter product = `DEMO`
THEN:
  the tool result is an error with the message
  `tool query has one parameter: query (one AQL statement). It does not accept: product. Name the product inside the statement: ... WHERE product = 'PROD'`
  FakeAha recorded no call

### EXAMPLE: query_requires_a_statement (of: query-tool)
GIVEN:
  the fixture account
WHEN:
  query-tool is called without parameters
THEN:
  the tool result is an error with the message
  `query parameter is required: one AQL statement. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
WHEN:
  query-tool is called with a query that has three spaces and nothing else
THEN:
  the tool result is the same error

### EXAMPLE: query_refuses_write_statements (of: query-tool)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `UPDATE features SET name = 'X' WHERE id = 'DEMO-1'`
THEN:
  the tool result is an error with the message
  `query only reads. Use the execute tool for INSERT, UPDATE and DELETE.`
  FakeAha recorded no call

### EXAMPLE: execute_result_envelope (of: execute-tool)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'Export to PDF')`
THEN:
  the tool result is
  `{"id":"DEMO-7","name":"Export to PDF","parent":"DEMO-R-3","message":"Feature DEMO-7 created"}`
  FakeAha recorded exactly one call: ApiCreate on /releases/DEMO-R-3/features
  with the field name = `Export to PDF`

### EXAMPLE: execute_rejects_unknown_parameter (of: execute-tool)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called without the parameter statement and with a
  parameter query = `DELETE FROM features WHERE id = 'DEMO-6'`
THEN:
  the tool result is an error with the message
  `tool execute has one parameter: statement (one AQL statement). It does not accept: query. Pass the statement as statement.`
  FakeAha recorded no call

### EXAMPLE: execute_refuses_read_statements (of: execute-tool)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `SELECT id FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is an error with the message
  `execute only changes data. Use the query tool for SELECT and DESCRIBE.`
  FakeAha recorded no call

### EXAMPLE: tools_without_read_only_mode (of: tool-registration)
GIVEN:
  the AQL tool group is selected and read-only mode is off
WHEN:
  tool-registration runs and an MCP client lists the tools
THEN:
  the list has `query`, and its only parameter is `query`
  the list has `execute`, and its only parameter is `statement`
  the list has no tool `describe_aql`
  the description of `query` has the line
  `id is the reference number of a record, for example 'PROD-23'.`

### EXAMPLE: tools_in_read_only_mode (of: tool-registration)
GIVEN:
  the AQL tool group is selected and read-only mode is on
WHEN:
  tool-registration runs and an MCP client lists the tools
THEN:
  the list has `query`
  the list has no tool `execute` and no tool `describe_aql`

### EXAMPLE: count_per_status (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"In design","COUNT(*)":1},{"status":"Under consideration","COUNT(*)":3},{"status":"In development","COUNT(*)":1},{"status":"Shipped","COUNT(*)":1}]}`

### EXAMPLE: star_shows_default_fields (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT * FROM features WHERE product = 'DEMO' AND status = 'under consideration'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":3,"records":[{"id":"DEMO-2","name":"Offline maps","status":"Under consideration","release":"DEMO-R-2"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration","release":"DEMO-R-3"},{"id":"DEMO-6","name":"Route planner","status":"Under consideration","release":"DEMO-R-1"}],"note":"* shows id, name, status, release. Other fields of features: initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url."}`

### EXAMPLE: record_by_id_has_all_fields (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT * FROM features WHERE id = 'demo-3'`
  and read-statement runs it
THEN:
  `scope` is `feature DEMO-3` and `count` is 1
  the row has the 17 fields of features in catalog order
  id is `DEMO-3`, name is `Live dashboard`, status is `In development`,
  release is `DEMO-R-3`, initiative is `DEMO-S-1` and product is `DEMO`
  epic and assigned_to are null
  the result has no `note`
  FakeAha recorded one ApiGet call on /features/DEMO-3, one ApiList call on
  /products and no other call

### EXAMPLE: field_without_value_is_null (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id, assigned_to FROM features WHERE release = 'DEMO-R-2'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","assigned_to":"test.user@example.com"},{"id":"DEMO-2","assigned_to":null}]}`

### EXAMPLE: label_is_the_word_of_the_statement (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT ref_num, title FROM features WHERE release_id = 'DEMO-R-2'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"ref_num":"DEMO-1","title":"CSV export"},{"ref_num":"DEMO-2","title":"Offline maps"}]}`

### EXAMPLE: name_equality_and_search (of: read-statement)
GIVEN:
  the fixture account, in which two feature names start with "Live"
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name = 'live'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":0,"records":[],"note":"No feature has exactly the name 'live'. name = compares the whole name; to search inside names use: name CONTAINS 'live'"}`
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name CONTAINS 'live'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name = 'offline MAPS'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-2"}]}`

### EXAMPLE: unknown_id_gives_an_empty_result (of: read-statement)
GIVEN:
  the fixture account, which has no feature DEMO-99
WHEN:
  query-tool is called with query = `SELECT * FROM features WHERE id = 'DEMO-99'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-99","count":0,"records":[],"note":"There is no feature with the id 'DEMO-99'."}`

### EXAMPLE: default_limit_is_reported (of: read-statement)
GIVEN:
  the fixture account and, in addition, a product BULK with one release
  BULK-R-1 and 60 features BULK-1 to BULK-60 in it
WHEN:
  query-tool is called with query =
  `SELECT id, name FROM features WHERE product = 'BULK'`
  and read-statement runs it
THEN:
  `count` is 50, `total` is 60 and `truncated` is true
  `records` has the features BULK-1 to BULK-50
  `note` is
  `Showing 50 of 60 records. Narrow the result with WHERE, raise LIMIT (up to 200), or count with COUNT(*).`

### EXAMPLE: releases_need_a_product (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT id, name FROM releases`
  and read-statement gets the error of plan-fetch
THEN:
  the tool result is an error with the message
  `releases are listed per product: add product = '...' to WHERE. Example: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`

### EXAMPLE: describe_one_entity (of: describe-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `DESCRIBE features`
  and describe-statement answers it
THEN:
  `entity` is `features` and `about` is `Features. Each feature is in one release.`
  `fields` has 17 objects with the names id, name, status, release,
  initiative, epic, assigned_to, product, start_date, due_date, tags, score,
  progress, description, created_at, updated_at, url, in this order
  the objects of name, status, release, initiative, assigned_to, start_date,
  due_date, tags and description have `"writable":true`; no other object
  has a member `writable`
  the type of release is `id of a release`, of initiative `id of an initiative`,
  of assigned_to `user (e-mail address)`, of product `product key`, of
  start_date `date (YYYY-MM-DD)` and of created_at `timestamp`
  `filter_by_parent` is `["product = '...'","release = '...'","initiative = '...'","epic = '...'"]`
  `status_values` is
  `SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'`
  `change_with_execute` is
  `["INSERT INTO features (parent, ...) VALUES (...): parent is the id of a release","UPDATE features SET field = 'value' WHERE id = '...'","DELETE FROM features WHERE id = '...'"]`
  `example` is `SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
  FakeAha recorded no call

### EXAMPLE: describe_without_entity (of: describe-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `DESCRIBE`
  and describe-statement answers it
THEN:
  `entities` has 12 objects, one per catalog entity in catalog order
  the first object is
  `{"entity":"products","about":"Products (workspaces). The key of a product, for example PROD, starts the id of its records."}`
  `note` is
  `DESCRIBE entity lists the fields of one entity, for example: DESCRIBE features`
  every object has exactly the members `entity` and `about`
  FakeAha recorded no call

### EXAMPLE: insert_with_other_spellings (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `INSERT INTO features (release_id, title, assignee) VALUES ('DEMO-R-2', 'Dark mode', 'me')`
  and insert-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-7","name":"Dark mode","parent":"DEMO-R-2","message":"Feature DEMO-7 created"}`
  FakeAha recorded one ApiCreate call on /releases/DEMO-R-2/features with
  name = `Dark mode` and assigned_to = `test.user@example.com`

### EXAMPLE: insert_comment_with_apostrophe (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `INSERT INTO comments (parent, body) VALUES ('DEMO-3', 'It''s ready.')`
  and insert-statement runs it
THEN:
  FakeAha recorded one ApiCreate call on /features/DEMO-3/comments with
  body = `It's ready.`
  the tool result has, in this order, `id` with the NumericId of the new
  comment, `parent` with `DEMO-3`, `body` with `It's ready.`, `created_at`
  with a timestamp, and `message` with `Comment added to DEMO-3`

### EXAMPLE: insert_without_parent (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `INSERT INTO features (name) VALUES ('Dark mode')`
  and insert-statement runs it
THEN:
  the tool result is an error with the message
  `parent is required: INSERT needs the column parent with the id of a release. Form: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)`
  no write request

### EXAMPLE: insert_with_a_product_as_release (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `INSERT INTO features (parent, name) VALUES ('DEMO', 'Dark mode')`
  and insert-statement runs it
THEN:
  the tool result is an error with the message
  `'DEMO' is a product, not a release. Use the id of one of its releases: DEMO-R-1 (What's next), DEMO-R-2 (Spring release), DEMO-R-3 (Summer release), DEMO-R-4 (Rider's cup release)`
  no write request

### EXAMPLE: insert_under_an_unknown_release (of: insert-statement)
GIVEN:
  the fixture account, which has no release DEMO-R-99
WHEN:
  execute-tool is called with statement =
  `INSERT INTO features (parent, name) VALUES ('DEMO-R-99', 'Dark mode')`
  and insert-statement runs it
THEN:
  the tool result is an error with the message
  `there is no release with the id 'DEMO-R-99'. To find the id, use the query tool: SELECT id, name FROM releases WHERE product = 'DEMO'`
  the account has no new feature

### EXAMPLE: update_status_in_any_case (of: update-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET status = 'in design' WHERE id = 'demo-2'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-2","name":"Offline maps","status":"In design","message":"Feature DEMO-2 status changed to 'In design'"}`
  FakeAha recorded one ApiGet call on /features/DEMO-2 and one ApiUpdate
  call on /features/DEMO-2 with status = `In design`
  FakeAha recorded no ApiList call on a feature list

### EXAMPLE: update_assigned_to_me (of: update-statement)
GIVEN:
  the fixture account, in which the current user is test.user@example.com
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-2'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-2","name":"Offline maps","assigned_to":"test.user@example.com","message":"Feature DEMO-2 updated"}`

### EXAMPLE: update_with_a_true_guard (of: update-statement)
GIVEN:
  the fixture account, in which DEMO-3 has the status In development
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-3' AND status = 'in development'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-3","name":"Live dashboard","status":"Shipped","message":"Feature DEMO-3 status changed to 'Shipped'"}`

### EXAMPLE: update_with_a_false_guard (of: update-statement)
GIVEN:
  the fixture account, in which DEMO-2 has the name Offline maps
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-2' AND name = 'CSV export'`
  and update-statement runs it
THEN:
  the tool result is an error with the message
  `nothing was changed: feature DEMO-2 has name = 'Offline maps', not 'CSV export'`
  FakeAha recorded one ApiGet call on /features/DEMO-2
  no write request

### EXAMPLE: update_of_an_unknown_record (of: update-statement)
GIVEN:
  the fixture account, which has no feature DEMO-99
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET name = 'X' WHERE id = 'DEMO-99'`
  and update-statement runs it
THEN:
  the tool result is an error with the message
  `there is no feature with the id 'DEMO-99'`
  no write request

### EXAMPLE: update_sends_empty_text (of: update-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET description = '' WHERE id = 'DEMO-1'`
  and update-statement runs it
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-1 whose fields have
  description with empty text
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","description":null,"message":"Feature DEMO-1 updated"}`

### EXAMPLE: update_with_a_date_without_quotes (of: update-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET due_date = 2026-12-01 WHERE id = 'DEMO-1'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","due_date":"2026-12-01","message":"Feature DEMO-1 updated"}`

### EXAMPLE: delete_one_feature (of: delete-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement = `DELETE FROM features WHERE id = 'DEMO-6'`
  and delete-statement runs it
THEN:
  the tool result is `{"id":"DEMO-6","message":"Feature DEMO-6 deleted"}`
  FakeAha recorded one ApiGet call on /features/DEMO-6 and one ApiDelete
  call on /features/DEMO-6

### EXAMPLE: delete_a_release_through_its_product (of: delete-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement = `DELETE FROM releases WHERE id = 'DEMO-R-4'`
  and delete-statement runs it
THEN:
  the tool result is `{"id":"DEMO-R-4","message":"Release DEMO-R-4 deleted"}`
  FakeAha recorded one ApiDelete call on
  /products/7000000000000000001/releases/DEMO-R-4

### EXAMPLE: delete_with_a_false_guard (of: delete-statement)
GIVEN:
  the fixture account, in which DEMO-6 has the status Under consideration
WHEN:
  execute-tool is called with statement =
  `DELETE FROM features WHERE id = 'DEMO-6' AND status = 'Shipped'`
  and delete-statement runs it
THEN:
  the tool result is an error with the message
  `nothing was changed: feature DEMO-6 has status = 'Under consideration', not 'Shipped'`
  no write request

### EXAMPLE: delete_of_an_unknown_record (of: delete-statement)
GIVEN:
  the fixture account, which has no feature DEMO-99
WHEN:
  execute-tool is called with statement = `DELETE FROM features WHERE id = 'DEMO-99'`
  and delete-statement runs it
THEN:
  the tool result is an error with the message
  `there is no feature with the id 'DEMO-99'`
  no write request

### EXAMPLE: operator_default_product (of: operator-statements)
GIVEN:
  the fixture account
WHEN:
  operator-statements runs the command for read statements with the option
  --product DEMO and the statement `SELECT id, name FROM releases`
THEN:
  the output has four rows with the ids DEMO-R-1, DEMO-R-2, DEMO-R-3 and
  DEMO-R-4, in this order
  the exit status is zero

### EXAMPLE: operator_has_no_default_limit (of: operator-statements)
GIVEN:
  the fixture account and, in addition, a product BULK with one release
  BULK-R-1 and 60 features BULK-1 to BULK-60 in it
WHEN:
  operator-statements runs the command for read statements with the
  statement `SELECT id FROM features WHERE product = 'BULK'`
THEN:
  the output has 60 rows
  no note about a cut is written

### EXAMPLE: operator_update_without_where (of: operator-statements)
GIVEN:
  the fixture account
WHEN:
  operator-statements runs the command for write statements with the
  statement `UPDATE features SET name = 'X'`
THEN:
  the command ends with an error: the diagnostic stream has the message
  `UPDATE needs a WHERE clause. To change one record: WHERE id = '...'`
  the exit status is not zero
  FakeAha recorded no call

### EXAMPLE: operator_read_command_refuses_writes (of: operator-statements)
GIVEN:
  the fixture account
WHEN:
  operator-statements runs the command for read statements with the
  statement `DELETE FROM features WHERE id = 'DEMO-6'`
THEN:
  the command ends with an error and the exit status is not zero
  FakeAha recorded no call

### EXAMPLE: predicate_update_changes_the_selection_only (of: write-by-predicate)
GIVEN:
  the fixture account, in which DEMO-1 is the only feature with the status
  In design
WHEN:
  write-by-predicate runs through the command for write statements, with
  the option --force and the statement
  `UPDATE features SET status = 'Shipped' WHERE product = 'DEMO' AND status = 'In design'`
THEN:
  FakeAha recorded exactly one ApiUpdate call: on /features/DEMO-1 with
  status = `Shipped`
  the features DEMO-2 to DEMO-6 are unchanged
  the exit status is zero

### EXAMPLE: predicate_with_or_is_evaluated (of: write-by-predicate)
GIVEN:
  the fixture account
WHEN:
  write-by-predicate runs through the command for write statements, with
  the option --force and the statement
  `UPDATE features SET tags = 'review' WHERE id = 'DEMO-1' OR id = 'DEMO-2'`
THEN:
  FakeAha recorded exactly two ApiUpdate calls: on /features/DEMO-1 and on
  /features/DEMO-2
  the features DEMO-3 to DEMO-6 are unchanged

### EXAMPLE: predicate_dry_run_checks_the_values (of: write-by-predicate)
GIVEN:
  the fixture account
WHEN:
  write-by-predicate runs through the command for write statements, with
  the option --dry-run and the statement
  `UPDATE features SET due_date = 'soon' WHERE product = 'DEMO'`
THEN:
  the command ends with an error: the diagnostic stream has the message
  `due_date: invalid date 'soon', use YYYY-MM-DD`
  the exit status is not zero
  FakeAha recorded no call

### EXAMPLE: predicate_without_confirmation (of: write-by-predicate)
GIVEN:
  the fixture account
  the operator answers the confirmation with no
WHEN:
  write-by-predicate runs through the command for write statements, without
  the option --force and with the statement
  `DELETE FROM features WHERE product = 'DEMO' AND status = 'Shipped'`
THEN:
  the command shows one selected record, DEMO-5, and asks for confirmation
  the command ends with an error: nothing was changed and the exit status
  is not zero
  no write request

### EXAMPLE: doubled_apostrophe_in_text (of: tokenize)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name = 'Rider''s handbook'`
  and tokenize reads the doubled apostrophe as one apostrophe
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-5"}]}`

### EXAMPLE: date_without_quotes (of: tokenize)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND release_date > 2026-06-01`
  and tokenize reads the date as a text
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":2,"records":[{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`

### EXAMPLE: text_that_is_not_closed (of: tokenize)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE name = 'Offline maps`
  and tokenize reaches the end of the input inside the text
THEN:
  the tool result is an error with the message
  `the text that starts with the apostrophe at position 38 is not closed. Text goes in single quotes: 'like this'.`
  FakeAha recorded no call

### EXAMPLE: unexpected_character (of: tokenize)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT id FROM features WHERE name ~ 'x'`
  and tokenize reaches the character ~
THEN:
  the tool result is an error with the message
  `unexpected character ~ at position 36.`
  FakeAha recorded no call

### EXAMPLE: apostrophe_inside_a_word_is_read_as_text (of: parse-statement)
GIVEN:
  the fixture account, in which the release DEMO-R-4 has the name
  Rider's cup release
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND name = 'Rider's cup release'`
  and parse-statement reads it a second time after the strict reading failed
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-4"}]}`

### EXAMPLE: strict_reading_is_kept (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE release='DEMO-R-3'AND status='Shipped'`
  which parse-statement can read strictly, although an apostrophe stands
  between a digit and a letter
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-3","count":1,"records":[{"id":"DEMO-5"}]}`

### EXAMPLE: statements_without_select (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `FROM releases WHERE product = 'DEMO'`
  and parse-statement reads it as `SELECT * FROM releases WHERE product = 'DEMO'`
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","name":"What's next","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","name":"Spring release","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","name":"Summer release","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","name":"Rider's cup release","release_date":"2026-12-15","parking_lot":false}],"note":"* shows id, name, release_date, parking_lot. Other fields of releases: start_date, status, released, product, progress, description, created_at, url."}`
WHEN:
  query-tool is called with query = `COUNT(*) FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`

### EXAMPLE: text_after_the_statement (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' LIMIT 5 please`
  and parse-statement finds a token after the statement
THEN:
  the tool result is an error whose message starts with
  `cannot read the statement at position 56 (please): expected the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order.`
  FakeAha recorded no call

### EXAMPLE: join_is_refused_for_an_agent (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT f.name FROM features f JOIN releases r ON f.release = r.id`
  and parse-statement runs for an agent caller
THEN:
  the tool result is an error with the message
  `JOIN is not supported. Read one entity per statement and filter by its parent, for example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`

### EXAMPLE: id_or_entity_alone (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `DEMO-3` and parse-statement reads it
THEN:
  the tool result is an error with the message
  `DEMO-3 is an id, not a statement. To read that record: SELECT * FROM features WHERE id = 'DEMO-3'`
WHEN:
  query-tool is called with query = `features`
THEN:
  the tool result is an error with the message
  `features is an entity, not a statement. To list its records: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'. To see its fields: DESCRIBE features`

### EXAMPLE: describe_takes_nothing_else (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `DESCRIBE features WHERE product = 'DEMO'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `DESCRIBE features lists the fields of features and takes nothing else. To read records: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`

### EXAMPLE: apostrophe_hint_in_an_error (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT id FROM features WHERE name = 'it's`
  and parse-statement can read it neither strictly nor a second time
THEN:
  the tool result is an error whose message contains
  `The text contains an apostrophe (it's). Inside text, write an apostrophe twice: it''s`

### EXAMPLE: write_without_an_id_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET name = 'X' WHERE status = 'Shipped'`
  and parse-statement runs for an agent caller
THEN:
  the tool result is an error with the message
  `one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record.`
  FakeAha recorded no call
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET name = 'X' WHERE id = 'DEMO-1' OR id = 'DEMO-2'`
THEN:
  the tool result is the same error
  FakeAha recorded no call

### EXAMPLE: write_without_where_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement = `UPDATE features SET name = 'X'`
  and parse-statement runs for an agent caller
THEN:
  the tool result is an error with the message
  `UPDATE needs WHERE id = '...' with the id of the one record to change. Forms: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...) | UPDATE entity SET field = 'value', ... WHERE id = 'id' | DELETE FROM entity WHERE id = 'id'`
  FakeAha recorded no call
WHEN:
  execute-tool is called with statement = `DELETE FROM features`
THEN:
  the tool result is an error whose message starts with
  `DELETE needs WHERE id = '...' with the id of the one record to change.`
  FakeAha recorded no call

### EXAMPLE: insert_with_two_rows_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'A'), ('DEMO-R-3', 'B')`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `one INSERT creates one record. Send one statement per record.`
  no write request

### EXAMPLE: insert_with_an_unknown_column_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `INSERT INTO features (parent, name, bogus) VALUES ('DEMO-R-3', 'A', 'x')`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `unknown field 'bogus' on features. Fields of features that can be set: name, status, release, initiative, assigned_to, start_date, due_date, tags, description, and parent in INSERT`
  no write request

### EXAMPLE: assignment_of_a_word_or_an_expression (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET name = name WHERE id = 'DEMO-1'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `text goes in single quotes: write 'name' instead of name`
  FakeAha recorded no call
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET name = name + ' (old)' WHERE id = 'DEMO-1'`
THEN:
  the tool result is an error with the message
  `unexpected character + at position 33.`
  FakeAha recorded no call

### EXAMPLE: null_cannot_be_written (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET due_date = NULL WHERE id = 'DEMO-1'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `NULL cannot be written. Leave the field out.`
  FakeAha recorded no call

### EXAMPLE: write_to_a_read_only_entity (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE users SET name = 'X' WHERE id = '7000000000000000901'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `users cannot be changed. Entities that can be changed: products, releases, features, epics, requirements, initiatives, goals, ideas, comments`
  FakeAha recorded no call

### EXAMPLE: describe_and_read_have_the_same_fields (of: entity-catalog)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `DESCRIBE releases`, and the
  entity-catalog gives the fields
THEN:
  the names in `fields` are id, name, release_date, start_date, status,
  parking_lot, released, product, progress, description, created_at, url,
  in this order
WHEN:
  query-tool is called with query = `SELECT * FROM releases WHERE id = 'DEMO-R-2'`
THEN:
  the row has exactly these labels, in the same order

### EXAMPLE: goal_status_from_the_success_metric (of: entity-catalog)
GIVEN:
  the fixture account, in which the API record of DEMO-G-1 has the status
  On track under its success metric and no workflow status of its own
WHEN:
  query-tool is called with query =
  `SELECT id, status FROM goals WHERE product = 'DEMO'`
  and the entity-catalog gives the two sources of the status
THEN:
  the tool result is
  `{"entity":"goals","scope":"product DEMO","count":1,"records":[{"id":"DEMO-G-1","status":"On track"}]}`

### EXAMPLE: statuses_of_an_entity (of: entity-catalog)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'features'`
  and the entity-catalog gives the source of the entity statuses
THEN:
  the tool result is
  `{"entity":"statuses","scope":"product DEMO","count":6,"records":[{"name":"Under consideration"},{"name":"In design"},{"name":"Ready to develop"},{"name":"In development"},{"name":"Shipped"},{"name":"Will not implement"}]}`
WHEN:
  query-tool is called with query =
  `SELECT name FROM statuses WHERE product = 'demo' AND entity = 'feature'`
THEN:
  the tool result is the same

### EXAMPLE: current_user (of: entity-catalog)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT email FROM me`
  and the entity-catalog gives the source of the entity me
THEN:
  the tool result is
  `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`

### EXAMPLE: comments_of_a_record (of: entity-catalog)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT body, user FROM comments WHERE parent = 'DEMO-3'`
  and the entity-catalog gives the comment route of a feature
THEN:
  the tool result is
  `{"entity":"comments","scope":"feature DEMO-3","count":1,"records":[{"body":"Verified with the provider.","user":"Test User"}]}`
WHEN:
  query-tool is called with query =
  `SELECT body, user FROM comments WHERE feature = 'demo-3'`
THEN:
  the tool result is the same

### EXAMPLE: own_name_of_the_entity (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT feature_id, feature_name FROM features WHERE feature = 'DEMO-5'`
  and resolve-names reads feature and feature_id as id and feature_name as name
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-5","count":1,"records":[{"feature_id":"DEMO-5","feature_name":"Rider's handbook"}]}`
  FakeAha recorded exactly one call: ApiGet on /features/DEMO-5

### EXAMPLE: unknown_field_lists_the_fields (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id, severity FROM features WHERE product = 'DEMO'`
  and resolve-names does not find the word severity
THEN:
  the tool result is an error with the message
  `unknown field 'severity' on features. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
  FakeAha recorded no call

### EXAMPLE: unknown_entity_lists_the_entities (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT * FROM tickets`
  and resolve-names does not find the word tickets
THEN:
  the tool result is an error with the message
  `unknown entity 'tickets'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`

### EXAMPLE: users_cannot_be_filtered_by_product (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT name FROM users WHERE product = 'DEMO'`
  and resolve-names gets the word product on users
THEN:
  the tool result is an error with the message
  `users cannot be filtered by product. Fields of users: id, name, email, created_at. Example: SELECT name, email FROM users`

### EXAMPLE: custom_field_of_an_agent_caller (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id, custom.priority FROM features WHERE product = 'DEMO'`
  and resolve-names gets the word custom.priority for an agent caller
THEN:
  the tool result is an error whose message starts with
  `unknown field 'custom.priority' on features. Fields of features: id, name, status,`

### EXAMPLE: product_in_three_spellings (of: resolve-product)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT id FROM releases WHERE product = 'demo'`
  and resolve-product finds the product by its key in lower case
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1"},{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = '7000000000000000001'`
THEN:
  the tool result is the same
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = 'Demo Product'`
THEN:
  the tool result is the same

### EXAMPLE: products_are_not_mixed (of: resolve-product)
GIVEN:
  the fixture account, in which DEMO and OTHER have one goal each
WHEN:
  query-tool is called with query = `SELECT id, name FROM goals WHERE product = 'other'`
  and resolve-product finds the product OTHER
THEN:
  the tool result is
  `{"entity":"goals","scope":"product OTHER","count":1,"records":[{"id":"OTHER-G-1","name":"Other goal"}]}`

### EXAMPLE: unknown_product_lists_the_keys (of: resolve-product)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT id FROM releases WHERE product = 'DEMOX'`
  and resolve-product finds no product
THEN:
  the tool result is an error with the message
  `unknown product 'DEMOX'. Product keys: DEMO, OTHER. To see their names: SELECT key, name FROM products`

### EXAMPLE: release_id_where_a_product_is_needed (of: resolve-product)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT id FROM features WHERE product = 'DEMO-R-3'`
  and resolve-product gets the id of a release
THEN:
  the tool result is an error with the message
  `'DEMO-R-3' is a release, not a product. Use: WHERE release = 'DEMO-R-3'`

### EXAMPLE: in_list_reads_each_parent (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id, release FROM features WHERE release IN ('DEMO-R-1', 'DEMO-R-2')`
  and plan-fetch reads one list per value
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-1, release DEMO-R-2","count":3,"records":[{"id":"DEMO-6","release":"DEMO-R-1"},{"id":"DEMO-1","release":"DEMO-R-2"},{"id":"DEMO-2","release":"DEMO-R-2"}]}`
  FakeAha recorded exactly two calls: ApiList on /releases/DEMO-R-1/features
  and ApiList on /releases/DEMO-R-2/features

### EXAMPLE: count_needs_one_list_request (of: plan-fetch)
GIVEN:
  the fixture account and, in addition, a product BULK with one release
  BULK-R-1 and 60 features BULK-1 to BULK-60 in it
WHEN:
  query-tool is called with query =
  `SELECT COUNT(*) FROM features WHERE product = 'BULK'`
  and plan-fetch takes the count from page 1
THEN:
  the tool result is
  `{"entity":"features","scope":"product BULK","count":1,"records":[{"COUNT(*)":60}]}`
  FakeAha recorded exactly one ApiList call on the feature list of BULK

### EXAMPLE: every_page_is_read_once (of: plan-fetch)
GIVEN:
  the fixture account and, in addition, a product BIG with 450 features
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'BIG' ORDER BY id`
  and plan-fetch reads the list to its last page
THEN:
  FakeAha recorded exactly three ApiList calls on the feature list of BIG,
  with page 1, page 2 and page 3, in this order, each with per_page 200
  `count` is 50, `total` is 450 and `truncated` is true

### EXAMPLE: too_many_pages (of: plan-fetch)
GIVEN:
  the fixture account and, in addition, a product HUGE with 3100 features
WHEN:
  query-tool is called with query =
  `SELECT id, name FROM features WHERE product = 'HUGE' ORDER BY name`
  and plan-fetch learns from page 1 that the list has 16 pages
THEN:
  the tool result is an error with the message
  `this would read 3100 records. Narrow it down, for example with release = 'PROD-R-3'`
  FakeAha recorded exactly one ApiList call on the feature list of HUGE
WHEN:
  query-tool is called with query =
  `SELECT id, name FROM features WHERE product = 'HUGE'`
THEN:
  `count` is 50, `total` is 3100 and `truncated` is true
  FakeAha recorded exactly one more ApiList call on the feature list of HUGE

### EXAMPLE: release_by_name_lists_the_releases (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND release = 'Summer release'`
  and plan-fetch gets a name where the id of a release is needed
THEN:
  the tool result is an error with the message
  `release = needs the id of the release, not its name. The releases of DEMO: DEMO-R-1 (What's next), DEMO-R-2 (Spring release), DEMO-R-3 (Summer release), DEMO-R-4 (Rider's cup release)`

### EXAMPLE: parent_that_does_not_exist (of: plan-fetch)
GIVEN:
  the fixture account, which has no release DEMO-R-99
WHEN:
  query-tool is called with query = `SELECT id FROM features WHERE release = 'DEMO-R-99'`
  and plan-fetch gets NotFound for the list of the release
THEN:
  the tool result is an error with the message
  `there is no release with the id 'DEMO-R-99'. To find it: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`

### EXAMPLE: release_id_where_a_feature_id_is_needed (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT * FROM features WHERE id = 'DEMO-R-3'`
  and plan-fetch gets the id of a release for the field id of features
THEN:
  the tool result is an error with the message
  `'DEMO-R-3' is the id of a release, not of a feature. For the features of that release: SELECT * FROM features WHERE release = 'DEMO-R-3'`
  FakeAha recorded no ApiGet call

### EXAMPLE: comments_need_a_parent (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query = `SELECT body FROM comments`
  and plan-fetch finds no route
THEN:
  the tool result is an error with the message
  `comments are listed per record: add parent = '...' to WHERE. Example: SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'`

### EXAMPLE: parking_lot_and_missing_date (of: convert-record)
GIVEN:
  the fixture account, in which the API records of DEMO-R-2, DEMO-R-3 and
  DEMO-R-4 have no member parking_lot
WHEN:
  query-tool is called with query =
  `SELECT id, release_date, parking_lot FROM releases WHERE product = 'DEMO'`
  and convert-record converts the four API records
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","release_date":"2026-12-15","parking_lot":false}]}`

### EXAMPLE: description_is_plain_text (of: convert-record)
GIVEN:
  the fixture account, in which the API record of DEMO-1 has the description
  `<p>Export <b>all</b> columns.</p><p>Keep the order.</p>`
WHEN:
  query-tool is called with query = `SELECT description FROM features WHERE id = 'DEMO-1'`
  and convert-record converts the API record
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"description":"Export all columns.\nKeep the order."}]}`

### EXAMPLE: or_and_not_are_evaluated (of: evaluate-condition)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND (status = 'Shipped' OR status = 'In design')`
  and evaluate-condition evaluates the OR on every record
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-5"}]}`
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE release = 'DEMO-R-3' AND NOT status = 'Shipped'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-3","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`

### EXAMPLE: me_as_a_value (of: evaluate-condition)
GIVEN:
  the fixture account, in which the current user is test.user@example.com
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND assigned_to = 'me'`
  and evaluate-condition compares with the current user
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`

### EXAMPLE: fields_without_a_value_in_conditions (of: evaluate-condition)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND release_date IS NULL`
  and evaluate-condition tests the field
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-1"}]}`
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND parking_lot = false`
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":3,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE release = 'DEMO-R-2' AND assigned_to != 'test.user@example.com'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":1,"records":[{"id":"DEMO-2"}]}`

### EXAMPLE: grouping_without_group_by (of: build-rows)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT release, COUNT(*) FROM features WHERE product = 'DEMO'`
  and build-rows groups by the selected field
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":3,"records":[{"release":"DEMO-R-2","COUNT(*)":2},{"release":"DEMO-R-3","COUNT(*)":3},{"release":"DEMO-R-1","COUNT(*)":1}]}`

### EXAMPLE: rows_without_a_value_sort_last (of: build-rows)
GIVEN:
  the fixture account, in which DEMO-R-1 has no release date
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date`
  and build-rows sorts the rows
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"},{"id":"DEMO-R-1"}]}`
WHEN:
  query-tool is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date DESC`
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-4"},{"id":"DEMO-R-3"},{"id":"DEMO-R-2"},{"id":"DEMO-R-1"}]}`

### EXAMPLE: count_of_nothing_is_zero (of: build-rows)
GIVEN:
  the fixture account, in which no feature has the status Will not implement
WHEN:
  query-tool is called with query =
  `SELECT COUNT(*) FROM features WHERE product = 'DEMO' AND status = 'Will not implement'`
  and build-rows gets no record
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":0}]}`

### EXAMPLE: limit_of_the_statement_keeps_the_total (of: build-rows)
GIVEN:
  the fixture account
WHEN:
  query-tool is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' LIMIT 2`
  and build-rows cuts the rows
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"total":6,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`
  the result has no member `truncated` and no `note`

### EXAMPLE: tags_are_split_at_commas (of: check-write-values)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET tags = 'export, csv ,' WHERE id = 'DEMO-1'`
  and check-write-values prepares the value
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-1 with the two tags
  `export` and `csv`
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","tags":"export, csv","message":"Feature DEMO-1 updated"}`

### EXAMPLE: unknown_status_lists_the_statuses (of: check-write-values)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET status = 'Done' WHERE id = 'DEMO-2'`
  and check-write-values finds no workflow status with that name
THEN:
  the tool result is an error with the message
  `workflow status 'Done' not found for feature; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement`
  no write request

### EXAMPLE: date_that_is_not_a_date (of: check-write-values)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET due_date = 'next friday' WHERE id = 'DEMO-1'`
  and check-write-values checks the date
THEN:
  the tool result is an error with the message
  `due_date: invalid date 'next friday', use YYYY-MM-DD`
  no write request

### EXAMPLE: user_by_name (of: check-write-values)
GIVEN:
  the fixture account
WHEN:
  execute-tool is called with statement =
  `UPDATE features SET assigned_to = 'Second User' WHERE id = 'DEMO-2'`
  and check-write-values checks the user
THEN:
  the tool result is an error with the message
  `assigned_to: there is no user 'Second User'. Use an e-mail address, or 'me' for yourself`
  no write request

### EXAMPLE: custom_field_is_compared_as_text (of: custom-field-references)
GIVEN:
  the fixture account, in which DEMO-1 and DEMO-2 have the custom field
  priority with the value High
WHEN:
  operator-statements runs the command for read statements with the
  statement
  `SELECT id, custom.priority FROM features WHERE product = 'DEMO' AND custom.priority = 'High'`
  and custom-field-references gives the values
THEN:
  the output has two rows, DEMO-1 and DEMO-2
  the value of custom.priority in both rows is the text High, without
  quotation marks and not in an encoded form

### EXAMPLE: list_with_custom_fields_ends (of: custom-field-references)
GIVEN:
  the fixture account and, in addition, a product BIG with 450 features
WHEN:
  operator-statements runs the command for read statements with the
  statement `SELECT id, custom.priority FROM features WHERE product = 'BIG'`
  and custom-field-references reads the custom fields with the list
THEN:
  the output has 450 rows
  FakeAha recorded the pages 1, 2 and 3 of the feature list of BIG, each
  once

### EXAMPLE: statements_in_messages_can_be_sent (of: error-messages)
GIVEN:
  the ExampleKey PROD
WHEN:
  parse-statement gets, for an agent caller, each of these statements of
  the error-messages tables and of the catalog after the placeholders are
  replaced: the example statement of every catalog entity; the five
  example statements of the description of `query`; the three example
  statements of the description of `execute`;
  `SELECT status, COUNT(*) FROM features WHERE product = 'PROD' GROUP BY status`;
  `SELECT COUNT(*) FROM features WHERE product = 'PROD'`;
  `SELECT key, name FROM products`;
  `SELECT id, name FROM releases WHERE product = 'PROD'`;
  `SELECT * FROM features WHERE name CONTAINS 'x'`;
  `SELECT id, body FROM comments WHERE parent = 'PROD-23'`;
  `SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'`
THEN:
  parse-statement returns a statement for each of them

### EXAMPLE: example_key_is_the_default_product (of: choose-example-key)
GIVEN:
  the fixture account
  the default product of the server is configured as `demo`
WHEN:
  choose-example-key runs at the start of the server and an MCP client
  lists the tools
THEN:
  the description of `query` has the line
  `id is the reference number of a record, for example 'DEMO-23'.`
  no description contains `PROD` or `OTHER`

### EXAMPLE: example_key_without_a_default_product (of: choose-example-key)
GIVEN:
  the fixture account
  no default product is configured
WHEN:
  choose-example-key runs at the start of the server and an MCP client
  lists the tools
THEN:
  the description of `query` has the line
  `Filter by parent: product = 'PROD', release = 'PROD-R-3', feature = 'PROD-23', initiative = 'PROD-S-1'.`
  no description contains `DEMO` or `OTHER`

### EXAMPLE: example_key_is_not_a_product_of_the_account (of: choose-example-key)
GIVEN:
  the fixture account and, in addition, a product with the key PROD
  no default product is configured
WHEN:
  choose-example-key runs at the start of the server and an MCP client
  lists the tools
THEN:
  the description of `execute` has the line
  `INSERT INTO features (parent, name) VALUES ('PRODX-R-3', 'Export to CSV')`
  no description contains `'PROD-`

## DEPENDENCIES

No new dependency. The amendment uses what the base version uses: its
client for the Aha! REST API and for the GraphQL endpoint, and its MCP
server library.

## DEPLOYMENT

As in the base version. The MCP server runs on standard input and output
or on HTTP. The command line tool has commands for read statements, for
write statements, for the interactive shell and for the HTTP service. The
amendment adds no binary, no configuration file and no environment
variable.

Two existing settings change their effect. The default product of the MCP
server no longer limits the statements of the tool `query`; it only chooses
the ExampleKey (see choose-example-key). The option --product of the
command line limits read statements of every entity that has a product
route (see plan-fetch, step 6).

## DELTA

This section is not normative. The BEHAVIOR sections above describe the
target. This section lists what a series of patches changes in aha-studio
0.12.0 to reach the target, what stays as it is, and what the rules rest
on. The defects of the base version were observed against a local HTTP
mock of the Aha! API and in its source text.

### Read path: from the base version to the target

| # | Base version 0.12.0 | Target | Where |
|--:|---|---|---|
| 1 | A condition on status is lost. The feature list is requested without it, and the records of a list have no status, release or assignee | Conditions are evaluated on complete records | plan-fetch, convert-record, evaluate-condition |
| 2 | OR and NOT are not evaluated. Such a statement returns every record | Every form of condition is evaluated | evaluate-condition |
| 3 | `=` on name is sent as a search, and its results are not checked | `=` is equality and CONTAINS is the search | plan-fetch, evaluate-condition |
| 4 | A filter on a release costs one list request and one request per feature of the account | One request per page on the route of the parent | plan-fetch |
| 5 | The aggregate example of the help tool returns one group with status null | Aggregates over complete records, with groups in a fixed order | build-rows |
| 6 | A statement without LIMIT returns the first page as if it were complete | For an agent caller a default limit with count, total, truncated and a note; for an operator caller every row | read-statement, build-rows |
| 7 | The product comes from a tool parameter or an option, and an error tells the reader to use an option of the command line | The product is a condition of the statement, and every result gives its scope | query-tool, plan-fetch, error-messages |
| 8 | `id` is the internal number and the reference number is a second field | `id` is the reference number in conditions and in results; the internal number is accepted wherever an id is accepted | entity-catalog, convert-record |
| 9 | A doubled apostrophe is a parse error. A text that is not closed is reported as a missing FROM clause at line 0: the error of the tokenizer is ignored | Both are read or reported as what they are, with the position | tokenize, parse-statement |
| 10 | The error for an unknown entity lists four of the eleven entities, and the help tool lists four entities and no fields | Every list comes from the catalog; DESCRIBE lists every field | resolve-names, describe-statement, error-messages |
| 11 | Two examples of the help tool do not parse, and the example in the description of `query` returns records that do not satisfy it | Every example that the server shows is accepted by the parser and evaluated | error-messages, INVARIANTS |
| 12 | A custom field value is returned in an encoded form and never equals a text. Its list is requested again and again without a page number | Plain values, comparisons that match, and a list that ends | custom-field-references |

### Write path: from the base version to the target

| # | Base version 0.12.0 | Target | Where |
|--:|---|---|---|
| 1 | An UPDATE whose condition is lost (status, OR, NOT, equality on name) changes every record of the list | An agent caller changes one record, found by its id, with guards. An operator caller changes exactly the records that the same WHERE clause selects in a read statement | update-statement, write-by-predicate |
| 2 | Input after a complete statement is ignored: `SET name = name + ' (old)' WHERE ...` renames every record to the text name, and a second row of VALUES is ignored | Leftover input is an error for every statement kind | parse-statement |
| 3 | An UPDATE without WHERE changes every record | Refused for every caller | parse-statement |
| 4 | An unknown column of INSERT is ignored. An unknown field of UPDATE is found only after the records were listed, and not in a dry run | Every name is checked before the first request, also in a dry run | parse-statement, resolve-names |
| 5 | An empty value is skipped, and the call still reports one changed record | Empty text is sent. NULL is refused with a message | update-statement, parse-statement |
| 6 | The record of `WHERE id = ...` is found by listing every record of the account | One request by id | update-statement, delete-statement |
| 7 | DELETE is parsed and always fails | DELETE for eight entities | delete-statement |
| 8 | INSERT exists for features only, and the release comes from an option | INSERT for nine entities, with the parent as a column | insert-statement |
| 9 | The write path checks no names: an unknown field in WHERE matches nothing | Names are resolved as in a read statement | parse-statement, resolve-names |
| 10 | The MCP server has no tool for write statements | The tool `execute`, absent in read-only mode | execute-tool, tool-registration |

### New in the language

- DESCRIBE as a statement. It replaces the tool `describe_aql`.
- The entities statuses and me, and the value `me` for user fields.
- The other names of entities and fields, the entity's own name as a field, and labels that keep the word of the statement.
- The second reading for an apostrophe inside a word, and a date without quotes.
- Guards next to the id in UPDATE and DELETE.
- The ExampleKey in descriptions and messages.
- The entity-catalog as the single table behind live execution.

### What users of the base version will notice

- The tool `query` has one parameter. A call with `product` is answered with an error that shows where the product goes.
- The default product of the MCP server no longer limits the statements of `query`.
- The result of `query` is an object with entity, scope, count and records. Rows have exactly the selected fields.
- The tool `describe_aql` is gone.
- `id` is the reference number. A statement that compares `id` with an internal number still works.
- `=` on name finds only a name that is equal from its first to its last character.
- An agent caller gets at most 50 rows without LIMIT and 200 with it, and cannot use JOIN, HAVING, subqueries and OFFSET. The command line keeps them.
- At the command line, an UPDATE without WHERE is refused, and an UPDATE or DELETE with a condition changes fewer records than before wherever the base version lost the condition.

### Unchanged, and out of scope

- Offline execution from the local cache, and the commands that fill the cache. After this change `id` is the reference number in live execution and the internal number in offline execution. Whether offline execution follows is an open point.
- For an operator caller: JOIN, HAVING, subqueries, relative time values, the `release.` qualifier, the entity tags, saved queries and the output formats.
- Custom fields for an agent caller.
- The other tools of the MCP server. This specification removes none of them except `describe_aql`.
- The calls of the Aha! API that do not create, read, change or delete one record. The analysis counts 37 such calls among 344. A tool for them is a separate specification.
- Entities beyond the twelve of the catalog. The analysis counts 60 entities behind the 307 other calls. A further entity is one more entry in the catalog.
- Three defects of the other tools that came up during the tests:
  - the release list reports a released release as not released;
  - the feature tool returns no release, initiative, assignee or dates;
  - the MCP server of version 0.12.0 reports the version 0.8.0.
- The documentation of the base version, which has to follow this specification. Today it documents comments inside statements, which no version accepts, and uses `votes_count` for the field `votes`.

### What the rules rest on

- The read rules were tested as a prototype outside aha-studio, with read requests against one account on 2026-11-12. That account has about 1,200 features; the tested product has 35.
- The write rules were tested against a mock that accepts what Aha! is expected to accept. No write request was sent to Aha!.
- With the two tools of this specification, one small local model answered 48 of 50 test requests and 15 of 15 held-out requests at temperature 0. The 74 tools of the base version, in the same series, gave 48 of 50 and 14 of 15. The two tool definitions have 2.1 KB, the 74 have 29.3 KB.
- With a product key of the account in the examples that the requests were not about, 4 of 50 requests failed because the model read that product. This is the reason for choose-example-key.
- Every route of the catalog is in the description of the Aha! API that the analysis used. The test report states that the feature lists of a release, an initiative and a product accept the parameter `fields`, as do the product lists of releases, initiatives, goals and ideas, and that a page size of 200 is accepted.
- The writable fields of the catalog come from a draft table. They were not checked against the Aha! API. The draft also had start and end dates for goals; they are left out because the catalog cannot read them.
- Text fields are sent as the statement has them. Whether Aha! needs HTML for a line break in a description or a comment was not tested.

### Where this specification differs from the tested prototype

- It has two callers. The prototype had the agent caller only. The rules for the operator caller are not tested. They are the default product, the absence of limits, the kept constructs of the base version, the fields of the base version outside the catalog, and write-by-predicate.
- On the account-wide list, the search text of `name CONTAINS` is sent to the API and the condition is still evaluated on every record. The prototype relied on the search of the API.
- A list with too many pages is refused after page 1. The prototype read max_pages pages first. The hint of that message depends on the list that was read.
- `COUNT(*)` alone is answered from the record count of page 1 when the route applies every conjunct of the WHERE condition. The prototype read every page.
- The limits of an agent caller also cut the rows of a grouped result. The prototype cut them only by a LIMIT of the statement.
- A message lists at most MaxListed records or product keys.
- Comments accept the spellings of product as other names of parent. The prototype listed product as a filter of comments and did not accept the word.
- A product condition on users has its own message. In the prototype that message could not be reached.
- Some spellings that the prototype accepted are left out:
  - `notes` as another name of comments. Aha! has notes of its own.
  - `desc` as another name of description. DESC is a keyword.
  - SHOW and DESC as other spellings of DESCRIBE, names in backticks, `==` and ILIKE. The test report does not say that the model wrote them.
- The example statements of products and users have no name filter. The prototype's examples had search words of the test data.
- Four messages differ:
  - the failed guard has no status number;
  - the refusal of a WHERE without id also says how guards are written;
  - a field that cannot be set is not called unknown;
  - a parent that does not exist is reported with a statement that finds it, also for comments.
- A guard is checked on every field of the catalog. The mock of the tests knew three fields.
- NULL as a value of SET is refused, as in the prototype. The analysis expected that a field can be cleared with it. This is an open point.

### Proposals of the test report that are not applied

Both are untested, so the descriptions keep the tested text:

- form lines whose placeholder words cannot be read as names, for example `<entity>` and `<field>`;
- the filter line without `feature = '...'`, or with a note that it is for requirements and comments.
