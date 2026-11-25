- A row has each label once. A select item that has the label and the content of an earlier item is left out. The content of an item is its field, or its function with its argument: `SELECT id, id` is read as `SELECT id`, and `COUNT(1)` after `COUNT(*)` is left out. Labels are compared as they are written: `id` and `ID` are two labels. A field that GROUP BY names a second time, in any spelling, is left out in the same way.
# aha-aql

## META
Deployment:   mcp-server
Version:      0.20.0
Spec-Schema:  0.4.0
Author:       Matthias G. Eckermann <mge@mailbox.org>
License:      MIT
Verification: none
Safety-Level: QM
Module:       github.com/mge1512/aha-aql

This specification describes mcp-server-aha-aql, a small MCP server for the
Aha! product management service. A language model works with Aha! through
one query language, AQL, and two tools: `query` reads, and `execute`
changes one record. The same two tools can be called from the command line
(see manual-mode).

AQL (Aha Query Language) comes from the open source project aha-studio.
This server implements a dialect of AQL and shares no code with aha-studio.
The dialect differs from the original in four points:

- `id` is the reference number of a record.
- `=` on a name is equality.
- A statement reads one entity.
- A write statement changes one record, and it finds that record by its id.

The server calls the Aha! REST API directly. Its settings come from
options and from environment variables. The credential comes from a key
file or from an environment variable, and in http mode from each request
(see load-configuration and request-account). The server does not obtain a
credential itself: this version has no OAuth flow.

## TYPES

```
Mode := stdio | http | manual
// stdio:  MCP over standard input and output. The process has one Account.
// http:   MCP over HTTP. Every request brings its own Account.
// manual: one tool call from the command line (see manual-mode).

Limits := {
  default_limit: 50,     // rows returned when a read statement has no LIMIT
  max_limit:     200,    // most rows that one call returns
  max_pages:     15      // most pages of one list that one statement reads
}
// Constants.

PageSize := 200
// Records requested per page of a list.

MaxInValues := 20
// Most values of an IN list that are read with one request per value
// (see plan-fetch).

MaxListed := 30
// Most records, product keys, custom fields or options that one message
// lists (see error-messages).

Subdomain := string where matches("^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$")
// The account name in https://<subdomain>.aha.io, in lower case.

Credential := string where matches("^[!-~]{1,4096}$")
// A bearer value that Aha! accepts: an API key or an OAuth2 access token.
// It has 1 to 4096 printable ASCII characters and no space. The server
// does not interpret it. It only sends it to Aha!. In a sandbox it can be a
// placeholder that the sandbox replaces on the way to Aha!.

Account := { subdomain: Subdomain, credential: Credential }
// What one tool call runs with (see request-account).

Settings := {
  subdomain:       Subdomain or none,
  api_key_file:    string or none,   // path of a file that has the Credential
  default_product: string or none,   // key, numeric id or name of a product
  read_only:       Boolean,          // default false
  listen:          string,           // http mode; default of the deployment template
  dry_run:         Boolean           // manual mode, tool execute; default false
}
// subdomain, default_product and read_only come from an option or from an
// environment variable. The option takes precedence (see load-configuration).

StatementKind := select | describe | insert | update | delete

EntityName := string
// Canonical plural name of a catalog entity, for example "features".

FieldName := string
// Canonical name of a field of an entity, for example "assigned_to". The
// name of a custom field is "custom." followed by its key in lower case,
// for example "custom.priority" (see custom-fields).

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
TextList := List<Text>
// The checked value of a field that has several values: the field tags, and
// a custom field for tags or for users (see check-write-values).

SelectItem    := Star | FieldItem | AggregateItem
Star          := all fields (see read-statement, step 1)
FieldItem     := { field: FieldName, label: Label }
AggregateItem := {
  function: count | sum | avg | min | max,
  field:    FieldName or none,       // none means COUNT(*)
  distinct: Boolean,
  label:    Label
}

Condition  := Comparison | NullTest | InList | AllOf | AnyOf | Negation
Comparison := {
  field:    FieldName,
  operator: "=" | "!=" | "<" | "<=" | ">" | ">=" | "contains" | "like",
  value:    Text | Number | Boolean
}
NullTest   := { field: FieldName, negated: Boolean }   // IS NULL, IS NOT NULL
InList     := { field: FieldName, values: List<Value> }
AllOf      := { members: List<Condition> }   // AND
AnyOf      := { members: List<Condition> }   // OR
Negation   := { member: Condition }          // NOT
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
Statement := ReadStatement | DescribeStatement | InsertStatement
           | UpdateStatement | DeleteStatement

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
  writable:       Boolean,
  request_member: string or none        // member of a request body, when it is
                                        // not the name of the field
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
  operations:     set of (create | update | delete),
  custom_record_types: List<string>,           // record types of Aha! whose custom
                                               // fields the entity has; empty: none
  example:        string
}

CustomField := {
  name:       FieldName,    // "custom." and the key in lower case
  key:        string,       // the key of the definition, as Aha! writes it
  label:      string,       // the name of the definition, as a person sees it in Aha!
  definition: string,       // the id of the definition; its options are read with it
  kind:       CustomKind
}
// One custom field of an entity that the server can use. Its source is a
// custom field definition of the account (see custom-fields).

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

ApiFailure    := NotFound | NotAuthorized | RateLimited | Unreachable | Rejected
NotFound      := { }
// The API answers with status 404. Aha! also answers so when the
// credential has no access to the record.
NotAuthorized := { status: 401 | 403 }
// The API answers with status 401 or 403.
RateLimited   := { }
// The API answers with status 429 after the repetitions of aha-request.
Unreachable   := { cause: no_answer | certificate | proxy | connection }
// No answer in time, a certificate that cannot be verified, a proxy that
// does not open the tunnel, or no connection. Message X02 has one reason
// text for each cause.
Rejected      := { status: u32, text: string }
// Every other status that is not a success, with the text of the answer.

ErrorKind := syntax | bad_name | scope | bad_args | bad_product | wrong_kind
           | not_found | bad_value | guard_failed | not_authorized
           | rate_limited | network | api_error
// The kind of an AqlError.
// syntax:         The statement cannot be read, or it is not one of the
//                 accepted forms
// bad_name:       Unknown entity or field, or an entity or field that
//                 cannot be used this way
// scope:          The statement does not say which product or parent is
//                 meant, or it would read too much
// bad_args:       The tool call itself is wrong: parameter missing, empty
//                 or unknown
// bad_product:    A product that the account does not have
// wrong_kind:     An id of another kind of record, or a name where an id
//                 is needed
// not_found:      A record that does not exist
// bad_value:      A value that cannot be written to the field or compared
//                 with it, or a required value that is missing
// guard_failed:   A condition next to the id is not true for the record;
//                 nothing was changed
// not_authorized: Aha! did not accept the credential, or the credential
//                 has no access to the record
// rate_limited:   Aha! answered that there were too many requests
// network:        Aha! cannot be reached
// api_error:      Aha! answered with another error

FieldType := string | text | number | boolean | date | datetime | status
           | user | key | reference
// The type of a catalog field.
// string:    Short text
// text:      Long text. Read as plain text
// number:    Integer or decimal number
// boolean:   true or false
// date:      A day, written YYYY-MM-DD
// datetime:  A timestamp as the Aha! API writes it
// status:    The name of a workflow status
// user:      A user. Read as an e-mail address, or as a name where the
//            entity says so
// key:       A product key
// reference: The id of another record

CustomKind := text | url | note | number | date | choice | fixed_choice
            | tags | fixed_tags | users | scorecard
// The kind of a custom field (see custom-fields).
// text:         One line of text
// url:          The address of a web page
// note:         Long text. Read as plain text
// number:       Integer or decimal number
// date:         A day, written YYYY-MM-DD
// choice:       One value of a list that users can extend
// fixed_choice: One option of a list that only an administrator changes
// tags:         Several values of a list that users can extend
// fixed_tags:   Several options of a list that only an administrator
//               changes
// users:        Several users of the account
// scorecard:    A score that Aha! computes. It can be read, and it cannot
//               be set

RecordKind := product | release | feature | epic | requirement | initiative
            | goal | idea | comment | user
// The kind of a record in Aha!.
// product:     A product (workspace)
// release:     A release
// feature:     A feature
// epic:        An epic
// requirement: A requirement of a feature
// initiative:  An initiative
// goal:        A goal
// idea:        An idea
// comment:     A comment on a record
// user:        A user of the account
```

## INTERFACES

```
AhaAPI
  The Aha! REST API, version 1, of one account. Its base address is
  https://<subdomain>.aha.io/api/v1, and routes are written relative to it.
  A route parameter in braces is replaced by a RecordId, by the NumericId
  of a product or by the id of a custom field definition. Every operation
  runs with the Account of the tool call and is sent as aha-request says.

  ApiList(route, parameters) -> ListPage | ApiFailure
      GET. One page of a list. Parameters: page (1-based), per_page, fields
      (comma-separated names of the wanted members of each record),
      q (name search, on the routes that have it). An answer without page
      information is the complete list: total_pages is 1, and total_records
      is the number of its records.
  ApiGet(route) -> ApiRecord | ApiFailure
      GET. One record with all its members.
  ApiCreate(route, body) -> ApiRecord | ApiFailure
      POST. Creates one record under the parent of the route.
  ApiUpdate(route, body) -> ApiRecord | ApiFailure
      PUT. Changes the given fields of one record.
  ApiDelete(route) -> none | ApiFailure
      DELETE. Deletes one record.
```

The server uses no other operation of Aha!, and it sends requests for no
other host. The only other host that it connects to is an HTTPS proxy that
the environment gives (see aha-request).
The bodies of ApiCreate and ApiUpdate are built as write-request says.

The answers of Aha! as JSON. An answer with a status from 200 to 299 has a
JSON object as its body. ApiDelete needs no body. The table says where an
operation finds its result in that object:

| Operation | Form of the answer | Result |
|---|---|---|
| ApiGet, ApiCreate and ApiUpdate for one record | One member, with an object as its value. The name of the member is the singular of the record: `feature`, `release`, `epic`, `requirement`, `initiative`, `goal`, `idea`, `comment` or `product`. For /me it is `user` | That object is the ApiRecord |
| ApiList | The records are the list in the member that has the name of the last part of the route: `features` for /releases/{release}/features, `custom_field_definitions` for /custom_field_definitions. The member `pagination` is an object with the numbers `total_records` and `total_pages` | The ListPage. Without the member `pagination` the list is complete, as ApiList says |
| ApiGet on /products/{product} | The record of the product, in the member `product`. For each record kind it has a member whose name ends in `_workflow`, for example `feature_workflow` and `epic_workflow`. The value of such a member is an object with the member `id`, the id of a workflow | The workflow of each record kind, as the entity statuses says |
| ApiGet on /products/{product}/workflows | The member `workflows` is a list. A workflow has the member `id` and the member `workflow_statuses`, a list of objects with `name`, `position` and `complete` | The statuses of each workflow, as the entity statuses says |
| ApiGet on /custom_field_definitions/{definition}/options | The member `options` is a list of objects with the member `text` | The options, as custom-fields says |

An answer with such a status that does not have this form is a failure:
Rejected, with the status and the text `the answer has an unknown form`.

Values in an API record:

- A long text is a JSON object, and its member `body` has the text as HTML. Aha! returns `description` in this form, and `theme` of a release. convert-record takes the text from `body`. A text source whose value is a JSON string is taken as it is, for example `body` of a comment.
- `tags` of a feature is a list of JSON strings.
- A member that names a status, a user or another record is an object. The catalog writes its source with a dot: `workflow_status.name`, `assigned_to_user.email`, `feature.reference_num`.
- A feature names its release, its initiative and its epic in members of their own, for example `release_reference_num`. An epic has `initiative_reference_num` and no `release_reference_num`. It names its release in the object `release`, whose member `reference_num` has the id.
- The success metric of a goal is the object `success_metric`. Its member `name` has the text of the metric, and its member `workflow_status` has the status of the goal. A goal has no member `workflow_status` of its own.
- A timestamp is a JSON string in an ISO 8601 form, for example `2026-02-02T08:00:00.000Z`. evaluate-condition reads it as a point in time. The result of a comparison does not depend on the number of digits after the seconds, and it does not depend on how the zone is written.

The parameter `fields` of ApiList names top-level members only. For the
API source `workflow_status.name` it names `workflow_status`, and each
name is there once.

Test double: FakeAha. FakeAha is an HTTPS server outside the server
process, and it is the HTTPS proxy of the test. A test gives the server
the address of FakeAha in HTTPS_PROXY and the certificate authority of
the test in SSL_CERT_FILE, as a sandbox does (see aha-request). The
server then asks FakeAha with CONNECT for a tunnel to `example.aha.io`,
port 443. Inside the tunnel FakeAha ends the TLS connection with a
certificate for that host name, which the certificate authority of the
test has signed. FakeAha gets every request with its method, address,
headers and body, and answers it from memory with a fixture account. It
opens no network connection of its own.

The server has no hook for tests: no option, no variable and no part of
its code is there for a test only. A test starts the server as a user
starts it, and sees what a user sees: the tool results, the two output
streams, the exit code, and the requests that arrive at FakeAha.

- FakeAha serves the subdomain `example` and accepts the credential `fixture-key`. A request with another credential is answered with the status 403. A CONNECT request for another host is a failure of the test.
- Every answer of FakeAha has the JSON form of the section above. A record is in the member with its singular name, and a list is in the member with the name of the last part of the route. `description` and `theme` are objects with the member `body`.
- ApiList returns pages of `per_page` records with correct `total_records` and `total_pages`.
- A record of a list has the members of the parameter `fields`, and `id`.
- `q` is a search for the text inside `name`, without regard to case.
- A route for a record that the fixture does not have is answered with the status 404.
- ApiCreate gives the new record the next free reference number.
- ApiCreate on a goal route needs the member `success_metric` with a `name` that is not empty. Without that member the answer has the status 400 and the body `{"error":"Required parameter missing: success_metric"}`. With an empty name it has the status 400 and the body `{"error":"Validation failed","errors":{"message":"Success metric cannot be blank"}}`. The new goal has the status of `success_metric.workflow_status`, and without that member the first status of the workflow of goals.
- ApiUpdate on a goal takes the status from the member `workflow_status` of the goal. It ignores the member `success_metric`, as Aha! does: the answer has the status 200, and the text of the success metric is the old one.
- ApiList on /custom_field_definitions returns the custom field definitions of the fixture account as one list without page information, in the order of the table below. A test can tell FakeAha to add a definition.
- ApiGet on /custom_field_definitions/{id}/options returns the options of that definition in their order.
- A record that is read by its id has the member `custom_fields`. A product that is read with ApiGet on /products/{id}, with its NumericId, has that member too. A record of a list has it only when the parameter `fields` names `custom_fields`. The member is a list with one entry per custom field that has a value in the record. It is an empty list for a record without such a value. The entries have the form that custom-fields describes.
- ApiUpdate with the member `custom_fields` sets these custom values, and the answer has them as entries. For a users field FakeAha accepts the id or the e-mail address of a user of the fixture account.
- A test can tell FakeAha to leave the member `custom_fields` out of every record of a list.
- A test can tell FakeAha to answer a route with a given status and header a given number of times, or to give no answer. The value of such a header can be a time: the UTC Unix time of the answer plus a number of seconds. A test can also give FakeAha the body of an answer.
- FakeAha records every CONNECT request with its host, its port and its headers. A test can tell FakeAha to refuse CONNECT: that is a proxy that does not open the tunnel.
- FakeAha records every call with its method, address, headers, parameters and body, and with the time when it arrived. A test can then check which requests a statement caused, which credential they had, and that a failed statement caused no write request.

The EXAMPLES use this fixture account. Unless an example says otherwise:

- The server runs in stdio mode. Its Account has the subdomain `example` and the credential `fixture-key`.
- The environment of the server has HTTPS_PROXY with the address of FakeAha and SSL_CERT_FILE with the certificate authority of the test. It has no NO_PROXY and no other proxy variable.
- The setting default_product has no value, and read_only is false.
- The tool list was answered before the call of the example. The ExampleKey is therefore chosen, and it is PROD.
- The calls that an example counts are the calls that its statement caused.

Products of the fixture account:

| key | id | name |
|---|---|---|
| DEMO | 7000000000000000001 | Demo Product |
| OTHER | 7000000000000000002 | Other Product |

Users: the current user is Test User, e-mail test.user@example.com, id
7000000000000000900. A second user is Second User, e-mail
second.user@example.com, id 7000000000000000901.

Workflows of DEMO. The list of workflows of the product has these five
workflows, in this order. The record of the product names each of them in
the members of the middle column:

| Workflow id | Members of the product record | Statuses, in workflow order |
|---|---|---|
| 401 | feature_workflow, epic_workflow, requirement_workflow | Under consideration, In design, Ready to develop, In development, Shipped, Will not implement |
| 402 | release_workflow | Planned, In progress, Shipped |
| 403 | initiative_workflow | Not started, In progress, Done |
| 404 | strategic_imperative_workflow | Not started, On track, At risk, Achieved |
| 405 | idea_workflow | Needs review, Planned, Shipped |

The list of workflows writes these ids as JSON numbers, and the record of
the product writes them as JSON strings. The workflow 401 has the record
type Feature. No workflow of the fixture account has the record type Epic
or Requirement. OTHER names the same five workflows.

Releases of DEMO:

| id | name | release_date | parking_lot |
|---|---|---|---|
| DEMO-R-1 | What's next | (none) | true |
| DEMO-R-2 | Spring release | 2026-03-31 | false |
| DEMO-R-3 | Summer release | 2026-06-30 | false |
| DEMO-R-4 | Rider's cup release | 2026-12-15 | false |

Features of DEMO, in the order in which the API lists them:

| id | name | status | release | initiative | assigned_to |
|---|---|---|---|---|---|
| DEMO-1 | CSV export | In design | DEMO-R-2 | (none) | test.user@example.com |
| DEMO-2 | Offline maps | Under consideration | DEMO-R-2 | (none) | (none) |
| DEMO-3 | Live dashboard | In development | DEMO-R-3 | DEMO-S-1 | (none) |
| DEMO-4 | Live video feed | Under consideration | DEMO-R-3 | DEMO-S-1 | (none) |
| DEMO-5 | Rider's handbook | Shipped | DEMO-R-3 | (none) | (none) |
| DEMO-6 | Route planner | Under consideration | DEMO-R-1 | (none) | (none) |

Every feature of DEMO has the created_at `2026-02-01T08:00:00.000Z`.
DEMO-1 has the updated_at `2026-02-02T08:00:00.000Z`, and the other five
features have the updated_at `2026-02-03T23:30:00.000Z`.

DEMO-3 has the tags live and dashboard, in this order, and DEMO-4 has
the tag live. No other feature of the fixture account has tags.

Other records of the fixture account:

- The initiative DEMO-S-1 "Live coverage" with the status "In progress".
- The goal DEMO-G-1 "Grow the audience" with the success metric "Monthly viewers". Its API record has the status "On track" in `success_metric.workflow_status` and no member `workflow_status` of its own.
- The goal OTHER-G-1 "Other goal" with the success metric "Not defined yet".
- The requirement DEMO-1-1 "Column selection" of the feature DEMO-1.
- The epic DEMO-E-1 "Race day" with the status "In design", in the release DEMO-R-3. Its API record names the release in the member `release`, an object with `id`, `reference_num` and `name`. It has no member `release_reference_num` and no member `color`.
- One comment on DEMO-3 with the id 7000000000000000701, the body "Verified with the provider." and the user Test User.

OTHER has no releases and no features. No feature of the fixture account
has an epic.

Custom field definitions of the fixture account, in the order in which the
API lists them:

| id | key | name | type | custom_fieldable_type |
|---|---|---|---|---|
| 7000000000000000801 | priority | Priority | CustomFieldDefinitions::SelectConstant | Feature |
| 7000000000000000802 | customer | Customer | CustomFieldDefinitions::SelectEditable | Feature |
| 7000000000000000803 | go_live | Go-live date | CustomFieldDefinitions::DateField | Feature |
| 7000000000000000804 | effort_days | Effort in days | CustomFieldDefinitions::NumberField | Feature |
| 7000000000000000805 | markets | Markets | CustomFieldDefinitions::SelectMultipleEditable | Feature |
| 7000000000000000806 | platforms | Platforms | CustomFieldDefinitions::SelectMultipleConstant | Feature |
| 7000000000000000807 | reviewers | Reviewers | CustomFieldDefinitions::Records::UsersField | Feature |
| 7000000000000000808 | release_notes | Release notes | CustomFieldDefinitions::NoteField | Feature |
| 7000000000000000809 | value_score | Value score | CustomFieldDefinitions::ScorecardField | Feature |
| 7000000000000000810 | business_case | Business case | CustomFieldDefinitions::EquationField | Feature |
| 7000000000000000811 | budget_owner | Budget owner | CustomFieldDefinitions::TextField | Release |
| 7000000000000000812 | cost_center | Cost center | CustomFieldDefinitions::TextField | Project |
| 7000000000000000813 | segment | Segment | CustomFieldDefinitions::SelectConstant | Ideas::Idea |

Options: priority has High, Medium and Low, in this order. platforms has
Web, iOS and Android. customer has Acme and Globex. segment has Enterprise
and Midmarket.

Custom values of the features of DEMO:

| id | priority | customer | go_live | effort_days | markets | reviewers |
|---|---|---|---|---|---|---|
| DEMO-1 | High | Acme | 2026-04-15 | 12.5 | EMEA, APAC | Second User |
| DEMO-2 | Medium | (none) | (none) | 8 | (none) | (none) |
| DEMO-3 | High | Globex | 2026-07-01 | 3 | EMEA | Test User, Second User |

The API records have these values in the entry forms of custom-fields. A
number is text there: "12.5", "8.0" and "3.0".

Other custom values of the fixture account:

- DEMO-1 has the release_notes `<p>Adds the export.</p><p>See the handbook.</p>`, the value_score 31 and a worksheet in business_case.
- The release DEMO-R-3 has the budget_owner "Finance".
- The product DEMO has the cost_center "CC-100".

No other record has a custom value. No feature has a value in platforms.

## BEHAVIOR: query
Constraint: required

The MCP tool `query`. It runs one read statement or one describe statement
and returns the result as a JSON object.

A statement that fails is a normal answer of the tool. The call succeeds
as a JSON-RPC call. Its result is marked as an error (`isError` is true)
and has the message as its only text. A language model reads tool results,
and the messages are written for it. JSON-RPC error responses are for
protocol errors and for internal errors of the server, as the deployment
template says.

INPUTS:
```
query:   string    // one AQL statement; the only parameter of the tool
account: Account   // from request-account; not a parameter of the tool
```

OUTPUTS:
```
tool result: the result object as JSON text,
             or the message of an AqlError, with the tool result marked
             as an error
```

PRECONDITIONS:
- the tool is registered (see tool-registration)
- the Account of the call is known: from request-account for an MCP call, and from load-configuration in manual mode

STEPS:
1. If the call has a parameter other than `query` → bad_args, message A02.
2. If `query` is absent, is not text, or is empty after trimming → bad_args, message A01.
3. Run parse-statement with the kinds select and describe allowed; on failure → the error of parse-statement.
4. For a describe statement: run describe-statement; on failure → the error of describe-statement. Its result is the result object. Continue with step 7.
5. For a read statement: run read-statement; on failure → the error of read-statement.
6. Build the result object from the ReadResult. Its members, in this order:
   a. `entity`, `scope` and `count`;
   b. `total`, only when it has a value;
   c. `truncated`, only when it is true;
   d. `records`, the rows;
   e. `note`, only when there are notes: the notes joined by one space.
7. Return the result object as JSON text. Text is a JSON string, Number a JSON number, Boolean a JSON boolean and Null the JSON null. The JSON text has no line break and no white space outside its strings.

POSTCONDITIONS:
- only ApiList and ApiGet were called
- `records` has at most max_limit rows
- each row has the labels of the select items, each with a value or null; a grouped result also has the fields of GROUP BY that are not selected, and a record that `*` selects by its id field also has its custom fields that have a value (see build-rows)

ERRORS:
- bad_args: unknown parameter (A02); missing or empty statement (A01)
- every error of parse-statement, describe-statement and read-statement, passed on unchanged

## BEHAVIOR: execute
Constraint: required

The MCP tool `execute`. It runs one write statement. One call creates,
changes or deletes at most one record. A statement that fails is answered
as the tool `query` answers it.

INPUTS:
```
statement: string    // one AQL statement; the only parameter of the tool
account:   Account   // from request-account; not a parameter of the tool
dry_run:   Boolean   // false in every MCP call; see manual-mode
```

OUTPUTS:
```
tool result: the result object as JSON text,
             or the message of an AqlError, with the tool result marked
             as an error
log line:    one line on the diagnostic output, in stdio and http mode
```

PRECONDITIONS:
- for an MCP call the tool is registered; it is not registered when the setting read_only is true (see tool-registration)
- the Account of the call is known, as for the tool `query`

STEPS:
1. If the call has a parameter other than `statement` → bad_args, message A02.
2. If `statement` is absent, is not text, or is empty after trimming → bad_args, message A03.
3. Run parse-statement with the kinds insert, update and delete allowed; on failure → the error of parse-statement.
4. Run insert-statement, update-statement or delete-statement, according to the kind of the statement, with dry_run; on failure → the error of that behavior.
5. Build the result object from the WriteResult: `id`, when it has a value, then each entry of `values` in its order, then `message`.
6. Return the result object as JSON text, as the tool `query` does.

Log line. In stdio mode and in http mode, every call of `execute` writes
one line to the diagnostic output (standard error) when it ends, with or
without an error:

```
<time> execute account=<subdomain> op=<op> entity=<entity> id=<id> result=<result>
```

| Part | Value |
|---|---|
| `<time>` | The time when the call ended, in UTC, as YYYY-MM-DDThh:mm:ssZ |
| `<subdomain>` | The subdomain of the Account |
| `<op>` | `insert`, `update` or `delete`. `none` when parse-statement did not accept the statement |
| `<entity>` | The entity name of the statement. `none` when parse-statement did not accept the statement |
| `<id>` | The id of the record that was created, or the id that the statement gives, in capital letters, when it has the form of a RecordId. `none` in every other case |
| `<result>` | `ok`, or the ErrorKind of the error |

The line has no value of a field, no text of the statement and no
credential.

POSTCONDITIONS:
- at most one record of the account was created, changed or deleted
- when the result is an error that was found before the write request, no ApiCreate, ApiUpdate or ApiDelete call was made
- in stdio mode and in http mode exactly one log line was written for the call

ERRORS:
- bad_args: unknown parameter (A02); missing or empty statement (A03)
- every error of parse-statement, insert-statement, update-statement and delete-statement, passed on unchanged

## BEHAVIOR: tool-registration
Constraint: required

Decides which tools the MCP server offers and what their descriptions say.
The server has two tools, `query` and `execute`. No other BEHAVIOR of this
specification is an MCP tool. The descriptions are what a language model
learns the language from, so their text is part of this specification.

INPUTS:
```
read_only: Boolean   // the setting read_only
```

OUTPUTS:
```
registered tools: query; execute unless read_only
```

PRECONDITIONS:
- load-configuration has succeeded

STEPS:
1. Register `query` with the description below. It has one parameter, `query`: text, required, with the description "One AQL statement". Its annotations say that it only reads: readOnlyHint true, destructiveHint false, idempotentHint true, openWorldHint true.
2. Unless read_only is true: register `execute` with the description below. It has one parameter, `statement`: text, required, with the description "One AQL statement". Its annotations say that it changes data and can delete: readOnlyHint false, destructiveHint true, idempotentHint false, openWorldHint true.
3. Register no other tool, no MCP resource and no MCP prompt.
4. When the tool list is answered: replace every {KEY} in both descriptions by the ExampleKey of the Account that asks (see choose-example-key).
5. A call with a parameter that the tool does not declare reaches the tool and is answered with message A02. It is not ignored, and it is not answered with a JSON-RPC error.

Description of `query`:

```
Read from Aha! with one AQL statement.
SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]
DESCRIBE entity
Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me.
id is the reference number of a record, for example '{KEY}-23'.
Filter by parent: product = '{KEY}', release = '{KEY}-R-3', feature = '{KEY}-23', initiative = '{KEY}-S-1'.
Other conditions: !=, <, >, CONTAINS 'text', IS NULL. Count with COUNT(*).
Custom fields are written custom.key. DESCRIBE entity lists them.
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
- the tool list is `query` and `execute`, or `query` alone when read_only is true
- no description contains a product key of the account other than the product of the setting default_product

ERRORS:
- none

## BEHAVIOR: manual-mode
Constraint: required

Runs one tool call from the command line and ends. A person can see with
it what a statement returns, without a language model and without an MCP
host. The MCP server does not offer manual mode as a tool. Manual mode
runs the same `query` and `execute` as the MCP tools, with the same
limits and the same messages.

INPUTS:
```
tool:      query | execute   // the first word of the command line
arguments: List<string>      // key=value, see load-configuration
standard input               // read only when the statement is "-"
```

OUTPUTS:
```
standard output: the tool result as JSON text and a line break, or nothing
standard error:  nothing, or one line that starts with "error: "
exit code:       0 | 1 | 2
```

PRECONDITIONS:
- the first word of the command line is `query` or `execute`

STEPS:
1. Run load-configuration; on failure → exit code 2, with the message of load-configuration on standard error.
2. Take the statement. For `query` it is the value of the argument `query`, and for `execute` the value of the argument `statement`. If that argument is absent → exit code 2, message G10.
3. If the value is `-`: read standard input to its end as UTF-8 text. That text is the statement.
4. If the tool is `execute` and the setting read_only is true → exit code 2, message G11. No request is sent.
5. Call the tool with the statement as its only parameter and with the Account of the process. For `execute`, dry_run is the setting dry_run.
6. If the tool returns an AqlError → write `error: `, the message and a line break to standard error; exit code 1. Nothing is written to standard output.
7. Write the tool result and a line break to standard output; exit code 0.

Dry run. With `dry-run=true` the tool `execute` checks the statement
completely: the form, the names, the id, the guards and the values. It
sends read requests for these checks and no write request. The result says
what the statement would do (see insert-statement, update-statement and
delete-statement).

POSTCONDITIONS:
- standard output has exactly the text that the MCP tool returns for the same statement and the same Account, and one line break
- no request was sent before load-configuration succeeded
- with dry_run true no ApiCreate, ApiUpdate or ApiDelete call was made
- no log line of `execute` was written

ERRORS:
- exit code 2: the start errors G01 to G08 and G12 of load-configuration; statement argument missing (G10); `execute` with read_only (G11)
- exit code 1: every AqlError of the tool, with its message

## BEHAVIOR/INTERNAL: load-configuration
Constraint: required

Reads the first word of the command line and the settings of the process,
from its arguments and from its environment. For stdio mode and manual
mode it also returns the Account of the process.

Every setting of the server has an option. A host or a sandbox usually
gives some settings to a process through its environment, and these
settings also have an environment variable. An option takes precedence
over its variable. The credential comes from a key file or from a
variable.

The command line has no option that takes the credential itself. The
arguments of a process can be seen by other users of the same system.

INPUTS:
```
word:        string or none    // the first word of the command line
arguments:   List<string>      // every argument after the first word
environment: the variables AHA_SUBDOMAIN, AHA_API_KEY, AHA_DEFAULT_PRODUCT
             and AHA_READ_ONLY
```

OUTPUTS:
```
mode:     Mode
settings: Settings
account:  Account or none     // none in http mode
or a start error: one line on standard error, and exit code 2
```

PRECONDITIONS:
- none

STEPS:
1. Take the mode from the first word. `stdio` and `http` are the modes of these names, and `query` and `execute` are manual mode. If the word is another one, or if the command line has no first word → G12. A flag that the deployment template defines, for example `--version`, is not a first word.
2. Read each argument as `key=value`, split at its first `=`. If an argument has no `=` or an empty key → G01.
3. In http mode: if the key `api-key-file` is given → G09.
4. If a key is not an option of the word (see the first table below) → G02. If a key is given twice → G03.
5. Take from the environment each setting that has a variable (see the first table below) and whose option is absent. A variable that is absent or empty gives no value.
6. Check the values. In G04 the name of the variable stands for the key when the value comes from a variable.
   a. `read-only` and `dry-run`: `true` or `false`; otherwise → G04.
   b. `subdomain`: put the value into lower case. If it does not match Subdomain → G04.
   c. `default-product` and `api-key-file`: if the option is given with an empty value → G04.
   d. `listen`: the value goes to the http transport of the deployment template. If its host is not a loopback address: write the warning G13 to the diagnostic output. The server starts.
7. In http mode: the account is none, and the variable AHA_API_KEY is not read. If that variable is not empty: write the warning G14 to the diagnostic output. Continue with step 11.
8. In stdio mode and manual mode: if there is no subdomain → G05.
9. Credential from a file. When the option `api-key-file` is given, the variable AHA_API_KEY is not read.
   a. If the path is not a regular file that the process can read → G07.
   b. On a platform that has permission bits: if a permission bit for the group or for other users is set → G08.
   c. Read at most 4097 bytes. The credential is that text without surrounding white space. If it does not match Credential → G07.
10. Credential from the environment. Without the option `api-key-file`, the credential is the value of AHA_API_KEY without surrounding white space. If the variable is absent or empty → G06. If the value does not match Credential → G07.
11. Return the mode, the settings and the account.

Options:

| Key | Value | Accepted for | Variable | Default |
|---|---|---|---|---|
| subdomain | The account name: the part of the address before `.aha.io` | stdio, http, query, execute | AHA_SUBDOMAIN | none |
| api-key-file | The path of a file that has the credential | stdio, query, execute | none. Without the option, the credential itself comes from AHA_API_KEY | none |
| default-product | The key, the numeric id or the name of a product. It is used as the ExampleKey | stdio, http, query, execute | AHA_DEFAULT_PRODUCT | none |
| read-only | true or false. With true the tool `execute` is not offered | stdio, http, query, execute | AHA_READ_ONLY | false |
| listen | host:port for the http transport | http | none | As the deployment template says |
| dry-run | true or false. See manual-mode | execute | none | false |
| query | One read statement or describe statement, or `-` | query | none | Required |
| statement | One write statement, or `-` | execute | none | Required |

Environment variables. The code of the server reads the first four
variables of this table and no other variable. They give settings. The
standard library of the language reads the other variables of the table
for the connection to Aha!. A variable that the standard library or the
runtime of the language reads for another purpose is not a setting of
the server:

| Variable | Use | Read in |
|---|---|---|
| AHA_SUBDOMAIN | The setting subdomain, when the option is absent | load-configuration |
| AHA_API_KEY | The credential in stdio mode and manual mode, when the option `api-key-file` is absent | load-configuration |
| AHA_DEFAULT_PRODUCT | The setting default_product, when the option is absent | load-configuration |
| AHA_READ_ONLY | The setting read_only, when the option is absent | load-configuration |
| HTTPS_PROXY, https_proxy | The proxy for the connection to Aha! | aha-request |
| NO_PROXY, no_proxy | The hosts that are reached without the proxy | aha-request |
| SSL_CERT_FILE, SSL_CERT_DIR | The trust store, on Linux | aha-request |

Start messages. Each one is a single line on standard error:

| ID | Text |
|---|---|
| G01 | `error: argument <n> is not key=value` |
| G02 | `error: unknown option '<key>' for <word>. Options: <keys>` |
| G03 | `error: option '<key>' is given twice` |
| G04 | `error: <key> must be <what>` |
| G05 | `error: no Aha! account. Use subdomain=NAME or set AHA_SUBDOMAIN` |
| G06 | `error: no credential. Use api-key-file=PATH or set AHA_API_KEY` |
| G07 | `error: no usable credential in <source>: <reason>` |
| G08 | `error: <path> can be read by other users. Run: chmod 600 <path>` |
| G09 | `error: http mode has no option api-key-file. Each request brings its credential in the header Aha-Api-Key` |
| G10 | `error: <word> needs <key>=STATEMENT, or <key>=- to read the statement from standard input` |
| G11 | `error: execute is not available, because read-only is true (option read-only or variable AHA_READ_ONLY)` |
| G12 | `error: unknown transport '<word>'. Valid: stdio, http, query, execute` |

Warnings. Each one is a single line on standard error. After a warning
the server goes on:

| ID | Text | Written by |
|---|---|---|
| G13 | `warning: the listen address is not a loopback address. Every request has a credential: put a TLS endpoint in front of the server` | load-configuration, step 6 d |
| G14 | `warning: AHA_API_KEY is not read in http mode. Each request brings its credential in the header Aha-Api-Key` | load-configuration, step 7 |
| G15 | `warning: default-product names no product of the account. Examples use the key <KEY>` | choose-example-key, step 4 |
| G16 | `warning: the product list could not be read. Examples use the key PROD` | choose-example-key, step 5 |

In G15, `<KEY>` is the ExampleKey that step 3 of choose-example-key
gives. No warning has the value of an option or of a variable.

In G12, `<word>` is the first word as it was given, and it is empty when
the command line has none. The text has the form of the message of the
deployment template for an unknown transport, with the two words of manual
mode added.

In G01, `<n>` is the position of the argument after the first word,
counted from 1. In G02, `<keys>` are the options of the word in the order
of the table, joined by a comma and a space. In G10, `<key>` is `query`
for the word `query` and `statement` for the word `execute`.

In G04, `<what>` depends on the key:

| Key | `<what>` |
|---|---|
| read-only, dry-run, AHA_READ_ONLY | `true or false` |
| subdomain, AHA_SUBDOMAIN | `the account name before .aha.io, for example mycompany` |
| default-product | `a product key, id or name` |
| api-key-file | `a file path` |

In G07, `<source>` is the path of the file, or `AHA_API_KEY`. `<reason>`
is one of `no such file`, `not a regular file`, `cannot be read`, `empty`
and `not a single value of at most 4096 printable characters`.

POSTCONDITIONS:
- no request was sent
- an option took precedence over its variable, and the key file over AHA_API_KEY
- no start message and no warning has the credential, and none has the value of an argument or of a variable other than the path of the key file
- the settings come from the arguments, from the key file and from the four AHA_ variables of the second table, and from no other variable and no other file

ERRORS:
- start error with exit code 2: G01 to G09, and G12

## BEHAVIOR/INTERNAL: request-account
Constraint: required

Gives the Account that one MCP request runs with. In stdio mode it is the
Account of the process. In http mode every request brings its own Account
in two headers, and the server has none.

This version does not use the authorization of the MCP specification,
which is optional. The credential of Aha! is data that a request needs,
and it has its own header. The header Authorization and the status 401
are not used.

INPUTS:
```
mode:     stdio | http
settings: Settings
account:  Account or none    // the Account of the process, in stdio mode
request:  in http mode, the HTTP request with its headers Aha-Api-Key,
          Aha-Subdomain and Origin
```

OUTPUTS:
```
account: Account
or, in http mode, an HTTP answer with the status 400 or 403;
the request is then not handled
```

PRECONDITIONS:
- load-configuration has succeeded

STEPS:
1. In stdio mode: return the Account of the process.
2. In http mode: make the checks of the steps 3 to 7 for every HTTP request to the MCP endpoint, before the request is handled. They also apply to a request that calls no tool.
3. If the request has an `Origin` header → status 403, error text `requests from a web page are not accepted`.
4. Read the header `Aha-Api-Key`. If the request does not have it, or its value does not match Credential → status 400, error text `no credential. Send the header Aha-Api-Key with an Aha! API key`.
5. If the request has the header `Aha-Subdomain`: put its value into lower case. If it does not match Subdomain → status 400, error text `Aha-Subdomain must be the account name before .aha.io`.
6. If the settings have a subdomain and the header gives another one → status 403, error text `this server works with one Aha! account only`.
7. If neither the settings nor the header give a subdomain → status 400, error text `no Aha! account. Send the header Aha-Subdomain`.
8. Return the Account with that subdomain and the value of the header `Aha-Api-Key`.

A refused request gets the status and a body. The body has the form that
the deployment template requires for HTTP errors: a JSON-RPC error
response. Its id is null, its code is -32600 and its message is the error
text.

The Account of a request is used for that request only. When the request
is answered, the server keeps neither the credential nor anything that it
read from Aha! with it. In stdio mode the server keeps nothing that it
read from Aha! after a statement either. The one value that it keeps is
the ExampleKey (see choose-example-key).

POSTCONDITIONS:
- in http mode no request reaches a tool, the tool list or any other MCP method without a credential
- no answer of this behavior has a credential, and none says which account the server works with

ERRORS:
- status 400: no credential or a credential of the wrong form; no account or an account name of the wrong form
- status 403: a request with an Origin header; an account other than the account of the server

## BEHAVIOR/INTERNAL: read-statement
Constraint: required

Runs one read statement against the Aha! API and returns the matching
records as rows.

INPUTS:
```
statement: ReadStatement
```

OUTPUTS:
```
result: ReadResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: its entity is a catalog name, and every field in it is a catalog name or the name of a custom field of the entity

STEPS:
1. Replace a Star item by fields. They are catalog fields. Custom fields come to a Star item only in step 6 a, for one record that the statement selects by its id field.
   a. When the WHERE condition has a conjunct that compares the id field of the entity with `=`: all fields of the entity.
   b. Otherwise: the default fields of the entity, followed by the other fields that the statement selects by name. Add note H01. Here a custom field is in the result only when the statement names it.
   c. After the replacement, a field item that has the label and the field of an earlier item is left out, as parse-statement says for two equal items: `SELECT *, name` shows `name` once.
2. Collect the used fields: the fields of the select items, of the WHERE condition, of GROUP BY and of ORDER BY. Custom fields are among them.
3. Run plan-fetch with the statement and the used fields; on failure → the error of plan-fetch. plan-fetch returns the API records, the scope and the applied conjuncts. It also returns the total of a list that was not read to its end, and the mark that page 1 was read for a count only.
4. Run convert-record for every API record, with the custom fields among the used fields.
5. Keep each record for which evaluate-condition is true for the WHERE condition. A conjunct that the route has applied counts as true.
6. Run build-rows with the kept records, the star custom fields, and the total and the count-only mark of plan-fetch. It returns the rows, count, total and truncated.
   a. Star custom fields. There are some only when all of this is true: step 1 a applied, the statement has no aggregate item and no GROUP BY, and one record is kept. Then run custom-fields for the entity without a key; on failure → the error of custom-fields. The star custom fields are the custom fields of its result that the statement does not select by name and whose name is not a label of the statement. Their values are taken as step 4 of convert-record says, from the API record that step b names.
   b. The API record for these values. For a record that plan-fetch read by its id route, it is the API record of the kept record. A product comes from the product list, which has no custom fields. When the list of star custom fields is not empty, read the kept product with one ApiGet on /products/{product}, with its NumericId. Its answer is the API record. NotFound → no star custom field has a value. Any other failure → the error that step 10 of plan-fetch gives for it.
   c. In every other case there are no star custom fields, and a Star item causes no request for custom field definitions.
7. If truncated: add note H02.
8. If no record was kept in step 5 and the statement has no aggregate item and no GROUP BY: add notes that explain the empty result, at most two. Go through the conjuncts that are a Comparison with a Text value, in the order of the statement. A `%` at the start or at the end of the value is ignored for these tests.
   a. The id field of the entity with the operator `=`. This case applies when plan-fetch read the id route for the value of this conjunct. On products and on users it applies to every such conjunct, because their list is complete. If no record with that id was read, the conjunct gives H03. If the record was read, another condition is not true for it, and the conjunct gives H08.
   b. On products, the field `name` with a value that resolve-product finds gives H04.
   c. The field `name` with a value that has the form of a ReferenceNumber of the entity's kind gives H05.
   d. The field `name` with a value that has the form of a ReferenceNumber whose kind is a parent filter of the entity gives H06.
   e. Any other comparison of the field `name` with the operator `=` gives H07.
   f. On statuses, the field `entity` with the operator `=`, when the product has no status for the entity of the value, gives H09.
9. Return the ReadResult with entity, scope, rows, count, total, truncated and notes.

POSTCONDITIONS:
- every row satisfies the WHERE condition of the statement
- no matching record is absent from rows unless truncated is true or the statement has a LIMIT, and then total gives the number of matching rows
- every selected field is in every row; a field without a value is null
- a star custom field is in the row only when it has a value in the record
- only ApiList and ApiGet were called

ERRORS:
- every error of plan-fetch (scope, bad_product, wrong_kind, not_found, not_authorized, rate_limited, network, api_error), passed on unchanged; a list that comes without the custom fields of the statement is one of them (X05)
- rate_limited, network, api_error: X01, X02, X03, from custom-fields for the star custom fields
- not_authorized, rate_limited, network, api_error: X04, X01, X02, X03, from the reading of a product for its star custom fields

## BEHAVIOR/INTERNAL: describe-statement
Constraint: required

Answers `DESCRIBE` and `DESCRIBE entity` from the catalog. For one entity
it adds the custom fields that the account defines (see custom-fields).

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
   c. `fields`: one object per field of the entity in catalog order, with `name`, `type`, and `writable` with the value true when the field is writable. A field that is not writable has no member `writable`. For a field that is writable in INSERT only, the value of `writable` is the text `INSERT only`. The text of `type` comes from the first table below.
   d. `custom_fields`: run custom-fields for the entity without a key; on failure → the error of custom-fields. An empty list gives no member. Otherwise the member is a list with one object per custom field, in the order of custom-fields. An object has `name`, `label`, `type`, and `writable` with the value true when the kind can be set and the entity has the operation update. The text of `type` comes from the table of kinds in custom-fields.
   e. `custom_fields_note`: only when the result has the member `custom_fields`; the text `* shows custom fields only with <id field> = '...', and only those that have a value. To select one by name: SELECT <id field>, <name> FROM <entity> WHERE <id field> = '<example id>'`. `<name>` is the first name of `custom_fields`, and `<example id>` is the example id of error-messages for the kind of the entity.
   f. `filter_by_parent`: only when the entity has parent filters; a list with the text `<field> = '...'` for each of them.
   g. `status_values`: only when the entity has a field `status`; the text `SELECT name FROM statuses WHERE product = '{KEY}' AND entity = '<entity>'`.
   h. `change_with_execute`: the text "read-only" when the entity has no operation. Otherwise a list with one text per operation of the entity, in the order create, update, delete. The texts come from the second table below.
   i. `example`: the example statement of the entity with {KEY} replaced.

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
| create | `INSERT INTO <entity> (parent, ...) VALUES (...): parent is <what>` |
| update | `UPDATE <entity> SET field = 'value' WHERE id = '...'` |
| delete | `DELETE FROM <entity> WHERE id = '...'` |

`<what>` is `a product key` when the parent is a product, and `the id of
the commented record` for comments. For another record kind it is `the id
of a release`, with that kind and with `an` before a vowel.

POSTCONDITIONS:
- the names in `fields` are exactly the catalog fields that read-statement delivers for the entity
- every name in `custom_fields` is accepted as a field word in a statement on the entity
- the requests of this behavior are the reading of the product list for the ExampleKey (see choose-example-key) and the reading of the custom field definitions (see custom-fields), and no other request

ERRORS:
- rate_limited, network, api_error: X01, X02, X03, from custom-fields

## BEHAVIOR/INTERNAL: insert-statement
Constraint: required

Creates one record.

INPUTS:
```
statement: InsertStatement
dry_run:   Boolean       // true only in manual mode with dry-run=true
```

OUTPUTS:
```
result: WriteResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: the entity allows create and every field in it is a writable catalog field of the entity; parse-statement refuses a custom field in INSERT

STEPS:
1. If the statement gives no parent → bad_value, message W18.
2. Check the parent against the parent kind of the entity.
   a. Parent kind product: run resolve-product without hints; on failure → the error of resolve-product.
   b. Parent kind is a record kind: the parent must be a ReferenceNumber of that kind or a NumericId. For a value of another form: if resolve-product finds it as a product → wrong_kind, message W24; otherwise → not_found, message W25, because no record of that kind can have such an id.
   c. Parent kind any-record (comments): the parent must be a ReferenceNumber of the kind feature, epic, idea, release, initiative, goal or requirement; otherwise → wrong_kind, message K08.
3. If the field `name` (for comments the field `body`) is absent or empty → bad_value, message W19. For goals the field `success_metric` is required in the same way, because Aha! creates no goal without it.
4. Run check-write-values for the fields; on failure → the error of check-write-values. When a status is assigned and the parent is a record, check-write-values reads the parent to learn its product.
5. If dry_run: return a WriteResult without id, with the checked fields as values, and the message "would create <singular> under <parent>". No write request is sent.
6. Run write-request for the entity and the checked fields. Call ApiCreate on the create route of the entity with the parent and that body. NotFound → not_found, message W25. NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Rejected → api_error, message W30.
7. Return the WriteResult. `id` is the id of the new record. `values` has `name` and `parent`; for a comment it has `parent`, `body` and `created_at`. `message` is "<Singular> <id> created" with the singular name of the entity starting with a capital letter; for a comment it is "Comment added to <parent>".

POSTCONDITIONS:
- exactly one record was created, with every field of the statement
- when an error is returned from steps 1 to 4, no ApiCreate call was made
- with dry_run true no ApiCreate call was made

ERRORS:
- bad_value: parent missing (W18); name, body or the success metric of a goal missing (W19); errors of check-write-values
- wrong_kind: W24, K08
- bad_product: from resolve-product (P01)
- not_found: the parent does not exist, or the value cannot be the id of a parent (W25)
- not_authorized, rate_limited, network, api_error: X04, X01, X02, W30

## BEHAVIOR/INTERNAL: update-statement
Constraint: required

Changes one record. The record is found by its id alone. Further
conditions next to the id are guards: they are checked on that one record,
and when one of them is not true, nothing is changed.

INPUTS:
```
statement: UpdateStatement
dry_run:   Boolean       // true only in manual mode with dry-run=true
```

OUTPUTS:
```
result: WriteResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: the entity allows update, every assigned field is a field that an UPDATE can set or a custom field that can be set, and every guard field is a field or a custom field of the entity

STEPS:
1. The id must be a RecordId that the entity accepts. For comments: a NumericId. For every other entity: a ReferenceNumber of the entity's kind or a NumericId. A ReferenceNumber of another kind → wrong_kind, message K02. Any other value → wrong_kind, message K04. A ReferenceNumber is used in capital letters from here on.
2. Call ApiGet on the id route of the entity. NotFound → not_found, message W20. NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Rejected → api_error, message X03.
   MECHANISM: the record is read with one request by its id. The statement never lists records to find the record to change.
3. Run convert-record for the record, with the custom fields that the statement uses.
4. For each guard, in the order of the statement: run evaluate-condition for it on the record. A guard is a comparison with the operator "=", or the test for no value that parse-statement reads from `field = ''`. The first guard that is not true → guard_failed, message W29.
5. Run check-write-values for the assigned fields with the product of the record; on failure → the error of check-write-values.
6. If dry_run: return a WriteResult with the id, the assigned fields with their checked values, and the message "would update <singular> <id>". A checked value that is a TextList is shown as text, joined by a comma and a space. No write request is sent.
7. Run write-request for the entity and the checked fields. Call ApiUpdate on the update route with that body. NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. NotFound, when the statement assigns exactly one reference → not_found, message W25 for that reference. Any other failure → api_error, message W30.
8. Return the WriteResult. `id` is the id of the record. `values` has `name`, or `parent` for a comment, and then each assigned field that is not yet in it. Each one has the value that the record has after the write, so a statement that assigns `name` shows `name` once. A custom field is in `values` under its name, `custom.` and the key, with the value that convert-record gives for the answer of the write request. `message` is "<Singular> <id> status changed to '<status>'" when `status` is the only assigned field, and "<Singular> <id> updated" in every other case.

POSTCONDITIONS:
- exactly one record was changed, in every assigned field and in no other field
- when an error is returned from steps 1 to 5, no ApiUpdate call was made
- with dry_run true no ApiUpdate call was made
- a value of empty text is sent as empty text; the field is not left out of the request

ERRORS:
- wrong_kind: K02, K04
- not_found: the record does not exist (W20); a referenced record does not exist (W25)
- guard_failed: W29
- bad_value, wrong_kind: errors of check-write-values
- not_authorized, rate_limited, network, api_error: X04, X01, X02, X03, W30

## BEHAVIOR/INTERNAL: delete-statement
Constraint: required

Deletes one record. Finding the record and the guards work as in
update-statement.

INPUTS:
```
statement: DeleteStatement
dry_run:   Boolean       // true only in manual mode with dry-run=true
```

OUTPUTS:
```
result: WriteResult | AqlError
```

PRECONDITIONS:
- the statement comes from parse-statement: the entity allows delete and every guard field is a field or a custom field of the entity

STEPS:
1. Check the id as in update-statement, step 1 → wrong_kind, message K02 or K04.
2. Call ApiGet on the id route of the entity. NotFound → not_found, message W20. NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Rejected → api_error, message X03.
3. Run convert-record for the record, with the custom fields that the guards use.
4. Check the guards as in update-statement, step 4 → guard_failed, message W29.
5. If dry_run: return a WriteResult with the id and the message "would delete <singular> <id>". No write request is sent.
6. Call ApiDelete on the delete route of the entity. Where the route contains the product, it is the product of the record that step 2 read. NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message W30.
7. Return the WriteResult with the id, no values, and the message "<Singular> <id> deleted".

POSTCONDITIONS:
- exactly one record was deleted
- when an error is returned from steps 1 to 4, no ApiDelete call was made
- with dry_run true no ApiDelete call was made

ERRORS:
- wrong_kind: K02, K04
- not_found: W20
- guard_failed: W29
- not_authorized, rate_limited, network, api_error: X04, X01, X02, X03, W30

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
9. Constructs outside the forms below are refused. JOIN → syntax, message S08. HAVING → syntax, message S09. OFFSET, or a second number after LIMIT → syntax, message S10. SELECT inside parentheses → syntax, message S11. Anything else → syntax, message S03.

Form of a read statement:

```
[SELECT [DISTINCT] item {, item}] FROM entity [[AS] alias]
    [WHERE condition]
    [GROUP BY field {, field}]
    [ORDER BY key [ASC | DESC] {, key [ASC | DESC]}]
    [LIMIT n]

item       = * | field [AS label] | function ( * | [DISTINCT] field ) [AS label]
function   = COUNT | SUM | AVG | MIN | MAX
label      = word | text
key        = field | label | function ( * | [DISTINCT] field )
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
- `field = ''` is read as `field IS NULL`, and `field != ''` as `field IS NOT NULL`. No field has empty text as its value: evaluate-condition takes empty text as no value.
- In an IN list, NULL and empty text stand for a field without a value (see evaluate-condition, step 3).
- The value of a comparison with `=`, `!=`, `<`, `<=`, `>`, `>=`, IN or BETWEEN must fit the type of its field, as the table of values in a condition says (below). A custom field has the field type of its kind (see custom-fields). A value that does not fit → bad_value, message V01. `<`, `<=`, `>`, `>=` or BETWEEN on a field of the type boolean → bad_value, message V02. The check needs no record, and it is made before the first request for records. CONTAINS and LIKE take any text.
- `field BETWEEN a AND b` is read as `field >= a AND field <= b`.
- NOT before IN, CONTAINS, LIKE or BETWEEN negates that comparison.
- The label of a field item is the word as the statement writes it, without a qualifier, or the label after AS. The label of an aggregate item is the label after AS, or the function name in capital letters with its argument in parentheses as written: `COUNT(*)`, `COUNT(name)`, `COUNT(DISTINCT status)`.
- `COUNT(1)` and COUNT with the name of an entity as its argument are read as `COUNT(*)`. Their label is `COUNT(*)`.
- `*` is the argument of COUNT only, and it does not follow DISTINCT. `*` in SUM, AVG, MIN or MAX → syntax, message S03. What was expected is `a field inside <FUNCTION>( )`. `*` after DISTINCT → syntax, message S03. What was expected is `a field after DISTINCT`.
- A label after AS is a word that is not a keyword, or a text that is not empty. It is used as the statement writes it.
- Two select items with the same label and different content → syntax, message S15.
- A label after AS must not be the name of another field that the row can show. There are two cases, and both → syntax, message S16. First: the label is the name of a field of GROUP BY that the statement does not select; build-rows shows that field under its name. Second: the statement has a Star item, and the label is the name of a field of the entity that is not the field of its own item.
- A key of ORDER BY is a field, a label of a select item or an aggregate. A label stands for its select item. A word that is a label of the statement and also a field word is the label. An aggregate key does not have to be a select item.
- In a statement with an aggregate item or GROUP BY, a key that is a field must be a field of GROUP BY or a selected field. Any other field → syntax, message S03. What was expected is `a field of GROUP BY, a selected field or an aggregate after ORDER BY`.
- In a statement without an aggregate item and without GROUP BY, an aggregate key → syntax, message S03. What was expected is `a field name`.
- A qualifier before a field (`f.name`, `features.name`) is ignored, and `alias.*` is `*`.
- The part `custom.` of a custom field word is not a qualifier. It belongs to the word and to its label: the label of `f.custom.priority` is `custom.priority`.
- A word directly followed by `(` that is not one of the five functions → syntax, message S12.
- A word that is not a keyword, where a value is expected → syntax, message S13.
- LIMIT takes a whole number that is not negative.

Values in a condition:

| Type of the field | Accepted value | Compared as |
|---|---|---|
| number | A Number, or Text that reads as a number | A number |
| boolean | TRUE or FALSE, the number 1 or 0, or the text `true` or `false` in any case | A truth value |
| date | A DateText that is a day of the calendar, or a timestamp | A day: the first 10 characters of the value |
| datetime | A DateText that is a day of the calendar, or a timestamp | A day or a point in time, as evaluate-condition says |
| every other type | Text, a Number or a Boolean | Text, as evaluate-condition says |

A timestamp is text that starts with a DateText. `T` or one space follows,
and then the time as hh:mm. The time can go on with :ss and with a fraction
of the second. `Z`, or an offset of the form +hh:mm or -hh:mm, can end the
text. Without `Z` and without an offset the timestamp is in UTC. NULL fits
every type.

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
- A write statement sets each field once, whatever the spelling: `title` is `name`, and in an UPDATE of a feature `parent` is `release`. A column of INSERT or an assigned field of UPDATE that names the field of an earlier column or assignment → syntax, message W33. A custom field counts in the same way. The parent of INSERT has the rule of W09 instead: the parent given twice with one value is one parent. A guard may name a field that the statement assigns.
- A column or an assigned field that is the id field of the entity → bad_name, message W15. A column or an assigned field that is unknown or not writable → bad_name, message W16. An assigned field of UPDATE that is writable in INSERT only → bad_name, message W16.
- An assigned field of UPDATE can be a custom field of the entity. A column or an assigned field that is a custom field of a kind that cannot be set → bad_name, message W16. Every other custom field as a column of INSERT → bad_name, message W31: a custom field is set with UPDATE only.
- An assignment is `field = value`. Anything between the value and the next `,` or WHERE → syntax, message W10.
- UPDATE and DELETE need a WHERE clause; without it → syntax, message W04.
- The first condition of the WHERE clause must compare the id field of the entity with `=`. The id field may have any spelling that resolve-names accepts. Any other first condition → syntax, message W05. The value must be a text or a number; otherwise → syntax, message W03.
- Each further condition must be joined with AND and have the form `field = value`. OR, NOT or parentheses → syntax, message W05. Another operator → syntax, message W06. A field that is not a field of the entity → bad_name, message W17. A custom field of the entity is a field of the entity here. These conditions are the guards of the statement. The value of a guard must fit the type of its field, as the table of values in a condition says; otherwise → bad_value, message V01. A guard `field = ''` is read as `field IS NULL`: it is true when the field has no value.
- NULL as a value → syntax, message W11. A word that is not a keyword, where a value is expected → syntax, message S13.
- Any other departure from the forms → syntax, message W03.

POSTCONDITIONS:
- the returned statement has canonical entity and field names only
- a text that can be read strictly is never read differently by the second reading
- no input is left after the statement

ERRORS:
- syntax: S03 to S16, W01 to W11 and W33, and S01 and S02 from tokenize
- bad_name: W12 to W17 and W31, and N01 to N07 from resolve-names
- bad_value: a value in a condition or in a guard that does not fit the type of its field (V01), and an ordering operator on a boolean field (V02)
- rate_limited, network, api_error: X01, X02, X03, when resolve-names needs the custom field definitions and they cannot be read

## BEHAVIOR/INTERNAL: entity-catalog
Constraint: required

The catalog is a single table of entities, fields, types, API sources,
routes, parents and operations. Every other behavior takes these facts
from the catalog. Custom fields are not in the catalog, because every
account defines its own. An entry only names the record types of Aha!
whose custom fields the entity has (see custom-fields).

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
1. Take every entity name, field name, type, API source, request member, route, parent, operation, record type for custom fields, about text and example from the entries below. resolve-names, describe-statement, custom-fields, plan-fetch, convert-record, check-write-values, write-request and error-messages use the catalog and no second list.
2. Use the order of the entries below wherever entities are listed, and the order of the field tables wherever fields are listed: in DESCRIBE results, in tool descriptions and in messages.
3. Replace {KEY} in about texts and examples by the ExampleKey.
4. Add an entity by adding an entry. No step of plan-fetch, convert-record or the write statements is written for one entity only, except where an entry below says so.

The entries give the API source of a field as the member of the API record
that has the value. `a.b` is member `b` of member `a`. Every field without
the note "heavy" is requested from a list; a heavy field is requested only
when the statement uses it (see plan-fetch).

The column Writable has `yes`, `no` or `INSERT only`. A field with `INSERT
only` can be set when the record is created. An UPDATE cannot set it.

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

### Request members of written fields

A written field is sent under its own name, as write-request says. The
exceptions are:

| Field | Request member |
|---|---|
| status | workflow_status |
| assigned_to | assigned_to_user |
| description of releases | theme |
| success_metric of goals | success_metric. Its value is an object with the member `name` |
| status of goals, in INSERT | workflow_status inside the object of success_metric |

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
| Custom fields | Record type Project. They can be read and cannot be set, because products are read only. For a Star item the product is read with GET /products/{product} (see read-statement) |
| Create, update, delete | not offered |
| Example | SELECT key, name FROM products |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| key | key | reference_prefix | no | The id field of products. Conditions also accept the NumericId |
| name | string | name | no | |
| product_line | boolean | product_line | no | false when the API record has no value |
| parent | key | parent_id | no | The key of the product with that NumericId |
| description | text | description | no | heavy |
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
| Custom fields | Record type Release |
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
| Custom fields | Record type Feature |
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
| Custom fields | Record types Epic, MasterFeature |
| Create | POST /releases/{release}/epics. Parent: a release |
| Update | PUT /epics/{id} |
| Delete | DELETE /epics/{id} |
| Example | SELECT id, name, status FROM epics WHERE product = '{KEY}' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| status | status | workflow_status.name | yes | |
| release | reference: release | release.reference_num | no | The API record of an epic has its release as an object |
| initiative | reference: initiative | initiative_reference_num | yes | |
| assigned_to | user | assigned_to_user.email | no | |
| product | key | product_id | no | |
| start_date | date | start_date | yes | |
| due_date | date | due_date | yes | |
| progress | number | progress | yes | |
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
| Custom fields | Record type Requirement |
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
| Custom fields | Record type Initiative |
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
| About | Goals of a product. A new goal needs a success metric. |
| Default fields | id, name |
| Other field names | parent, parent_id mean product |
| Read routes, in order | id: GET /goals/{id}. product: GET /products/{product}/goals |
| Account-wide list | GET /goals, with name search |
| Parent filters | product |
| Custom fields | Record types StrategicImperative, Goal |
| Create | POST /products/{product}/goals. Parent: a product |
| Update | PUT /goals/{id} |
| Delete | DELETE /products/{product}/goals/{id} |
| Example | SELECT id, name FROM goals WHERE product = '{KEY}' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| id | string | reference_num | no | |
| name | string | name | yes | |
| success_metric | string | success_metric.name | INSERT only | The text of the success metric. Required in INSERT. Aha! refuses an empty text, and it does not change the text in an update |
| status | status | success_metric.workflow_status.name | yes | Aha! keeps the status of a goal with its success metric |
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
| Custom fields | Record type Idea |
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
| Custom fields | none |
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
| Custom fields | none |
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
| Read | GET /products/{product} and GET /products/{product}/workflows. A conjunct product = value is required. One record per entity and workflow status, as the text below says |
| Parent filters | product, entity |
| Custom fields | none |
| Create, update, delete | not offered |
| Example | SELECT name FROM statuses WHERE product = '{KEY}' AND entity = 'features' |

| Field | Type | API source | Writable | Notes |
|---|---|---|---|---|
| name | string | name of the workflow status | no | |
| entity | string | the member of the product that names the workflow | no | Converted to an entity name, see below |
| position | number | position of the workflow status | no | |
| complete | boolean | complete flag of the workflow status | no | false when the API record has no value |
| product | key | none | no | The product of the statement |

The record of a product names the workflow of each record kind in a
member whose name ends in `_workflow`. The value of that member is an
object, and its member `id` is the id of a workflow in the list of
workflows of the product. The statuses of that workflow are the statuses
of the kind. Aha! writes such an id as a JSON number or as a JSON string.
Two ids are the same when their digits are the same.

| Member of the product | Value of `entity` |
|---|---|
| release_workflow | releases |
| feature_workflow | features |
| epic_workflow | epics |
| requirement_workflow | requirements |
| initiative_workflow | initiatives |
| strategic_imperative_workflow | goals |
| idea_workflow | ideas |
| key_result_workflow | key_results |
| page_workflow | pages |
| any other member whose name ends in `_workflow` | the part of the name before `_workflow` |

The records are in the order of this table. The members of the last row
follow in the order of their names. Within one entity the records are in
the order of the workflow. Several kinds can name one workflow. Its
statuses are then delivered once for each of these entities. A member
that is not an object gives no record, and so does a member whose
workflow is not in the list of workflows.

The record type of a workflow, its member `statusable_type`, does not say
which entities use it. Features, epics and requirements can share one
workflow, and its record type is then Feature.

A condition on `entity` accepts every spelling that resolve-names accepts
for that entity. It also accepts the part of the member name before
`_workflow`, for example `strategic_imperative` for goals.

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
| Custom fields | none |
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

## BEHAVIOR/INTERNAL: custom-fields
Constraint: required

Gives the custom fields of an entity. A custom field is a field that an
Aha! account defines for one type of record. The catalog does not have
these fields, so the server reads their definitions from the account. A
statement writes a custom field as `custom.` followed by its key, for
example `custom.priority`. The tables of this section say how a custom
field is read and how it is set. describe-statement, read-statement,
convert-record, evaluate-condition, check-write-values and write-request
use them.

INPUTS:
```
entity:   EntityName
key:      string or none    // none: the caller wants all custom fields of the entity
explicit: Boolean           // the statement wrote the word with `custom.`
```

OUTPUTS:
```
without a key: List<CustomField>     // sorted by name
with a key:    CustomField | none | AqlError
// none only when explicit is false. The caller then gives its own message
// for an unknown word.
```

PRECONDITIONS:
- the entity is a catalog entity

STEPS:
1. Take the record types that the catalog entry of the entity names for custom fields. If the entry names none, the entity has no custom fields, and no request is sent. Without a key the result is an empty list. With a key: when explicit is true → bad_name, message N06; otherwise the result is none.
2. Take the custom field definitions of the account. When the statement has not read them yet, call ApiList on /custom_field_definitions with per_page = PageSize, page by page to the last page.
   a. NotAuthorized or NotFound: the credential cannot read the definitions. The server then works as if the account had no custom fields. Without a key the result is an empty list. With a key: when explicit is true → bad_name, message N07; otherwise the result is none.
   b. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Rejected → api_error, message X03.
   c. The definitions, or the answer of step 2 a, are read at most once per statement. They are not kept after the statement, in stdio mode and in http mode.
3. Choose the definitions of the entity. The record type of a definition is its member `custom_fieldable_type`, and its type name is its member `type`. Of both, only the part after the last `::` counts, and it is compared without regard to case: `Ideas::Idea` is `Idea`, and `CustomFieldDefinitions::Records::UsersField` is `UsersField`. A definition belongs to the entity when its record type is one of the record types of step 1. When two definitions of the entity have the same key, the first one in the order of the API counts.
4. Give each definition of the entity its kind from the table of kinds below. A definition whose type name is not in that table is not supported: the server does not read it and does not set it.
5. Without a key: the result has one CustomField for each supported definition of the entity. The list is sorted by name, character by character.
6. With a key: compare the key with the keys of the definitions of the entity, without regard to case.
   a. A supported definition has the key: the result is its CustomField.
   b. A definition that is not supported has the key → bad_name, message N05.
   c. No definition has the key. When explicit is false, the result is none. When the entity has a supported definition → bad_name, message N04. In every other case → bad_name, message N06.

Kinds. The type names are the names of Aha! for its types of custom
fields. The field type says how evaluate-condition and build-rows treat a
value of the field.

| Type name | Kind | Field type | Text of `type` in DESCRIBE | Can be set |
|---|---|---|---|---|
| TextField | text | string | `string` | yes |
| UrlField | url | string | `string (web address)` | yes |
| NoteField | note | text | `text` | yes |
| NumberField | number | number | `number` | yes |
| DateField | date | date | `date (YYYY-MM-DD)` | yes |
| SelectEditable | choice | string | `string` | yes |
| SelectConstant | fixed_choice | string | `one option of a list` | yes |
| SelectMultipleEditable | tags | string | `tags, separated by commas` | yes |
| SelectMultipleConstant | fixed_tags | string | `options of a list, separated by commas` | yes |
| UsersField | users | user | `users (e-mail addresses), separated by commas` | yes |
| ScorecardField | scorecard | number | `number` | no |

Aha! has more types of custom fields. Examples are worksheets
(EquationField), attachments (AttachmentField) and checkboxes
(CheckboxField). Others are relations to other records or to custom tables
(LinkMany, FeaturesField and more). These types are not in the table, so
step 4 applies to them.

Custom values in an API record. The member `custom_fields` of an API
record is a list of entries. An entry has the members `key`, `name`,
`type` and `value`. The entry of a custom field is the first entry whose
`key` equals the key of the field. convert-record takes the value of the
field from that entry:

| Kind | `value` of the entry | Value of the field |
|---|---|---|
| text, url, choice, fixed_choice | A text | The text |
| note | A text with HTML | Plain text, made as convert-record makes it for the field type text |
| number | A number, or a number as text such as "300.0" | The number. A number without a fraction is a whole number: "300.0" gives 300 |
| scorecard | A number | The number |
| date | A text that starts with a date | The first 10 characters |
| tags, fixed_tags | A list of texts | The texts, joined by a comma and a space |
| users | A list of user ids. The entry also has `email_value`, the list of the e-mail addresses of these users, and `full_value`, a list with one object per user that has `id`, `name` and `email` | The e-mail addresses of `email_value`, joined by a comma and a space |

A record without the entry has null in the field. So has an entry whose
`value` is null, empty text or an empty list. A users entry without
`email_value` gives the member `email` of the objects of `full_value`,
and without both it gives the texts of `value`. A `value` of another form
than the table shows is delivered as text: a text as it is, and every
other value as its JSON text.

Values to set. check-write-values checks the value of a statement by this
table, and write-request sends it:

| Kind | Accepted value | Value in the request |
|---|---|---|
| text, url, choice | Text | A JSON string |
| note | Text | A JSON string. The text is not converted to HTML |
| number | A Number, or Text that reads as a number | A JSON number |
| date | As a field of the type date | A JSON string of the form YYYY-MM-DD |
| fixed_choice | Text that is one of the options, or empty text | A JSON string with the text of the option |
| tags | Text. It is split at commas into tags, as for the field `tags` | A JSON list of strings |
| fixed_tags | As tags. Each tag is one of the options | A JSON list of strings with the texts of the options |
| users | Text. It is split at commas into parts. A part is `me`, an e-mail address or a NumericId | A JSON list of strings. `me` is sent as the NumericId of the current user, and every other part as it is |
| scorecard | none | none |

Empty text is a value. For text, url, note, choice and fixed_choice it is
sent as an empty JSON string. For tags, fixed_tags and users it is sent as
an empty JSON list.

Options. A definition of the kind fixed_choice or fixed_tags has options.
They are read with ApiGet on /custom_field_definitions/{definition}/options.
The member `options` of the answer is a list, and the member `text` of
each of its objects is the text of one option. The order of the list is
the order of the options. The options of a definition are read at most
once per statement and are not kept after it.

POSTCONDITIONS:
- every CustomField of a result comes from a definition of the account whose record type the catalog names for the entity
- a word that the catalog resolves causes no request of this behavior
- one statement reads the definitions at most once, and the server does not keep them after the statement

ERRORS:
- bad_name: an unknown custom field (N04); a custom field of a type that is not supported (N05); an entity without custom fields (N06); definitions that the credential cannot read (N07)
- rate_limited, network, api_error: X01, X02, X03

## BEHAVIOR/INTERNAL: resolve-names
Constraint: required

Turns an entity word or a field word of a statement into a catalog name.
A field word can also become the name of a custom field of the entity. The
behavior accepts the names that a language model writes without being
told, and it answers an unknown name with the valid names.

INPUTS:
```
word:    string                // as the statement writes it
entity:  EntityName or none    // none: the word is an entity word
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
2. Entity word: the result is the first catalog entity, in catalog order, whose name, singular or other names have the word.
3. Entity word without a match: when writing is true, the result is none; otherwise → bad_name, message N01.
4. Field word. The name is the part of the word after its last `.`. When the part before the name is `custom`, or ends with `.custom`, the word is a custom field word: continue with step 6. Every other part before the name is a qualifier and is ignored. Take the first match in this order:
   a. a field of the entity;
   b. one of the other field names of the entity;
   c. one of the common other names, when the entity is not products and has the field that the name means;
   d. the entity's own name: the singular, or the singular followed by `_id`, means the id field of the entity, and the singular followed by `_name` means `name`. This applies only when the entity has that field;
   e. a custom field of the entity that has the name as its key. Run custom-fields for the entity with the name as the key and with explicit false; on failure → the error of custom-fields. When it returns a CustomField, the result is the name of that custom field.
5. Field word without a match: when writing is true, the result is none. On users, a word that means `product` on other entities → bad_name, message N03. Any other word → bad_name, message N02.
6. Custom field word. Run custom-fields for the entity with the name as the key and with explicit true; on failure → the error of custom-fields. The result is the name of the CustomField: `custom.` and the key in lower case.

The id field of an entity is `key` for products and `id` for every other
entity that has a field `id`. The entity statuses has no id field.

A catalog name takes precedence over the key of a custom field. A custom
field whose key is also a field name, or another name of a field, can be
written only with `custom.` before it.

POSTCONDITIONS:
- the result is a name of the catalog, the name of a custom field of the entity, or none
- a word that is a field of the entity is never read as another name
- a word that the steps 4 a to 4 d resolve causes no request

ERRORS:
- bad_name: unknown entity (N01); unknown field (N02); a product condition on users (N03); N04 to N07 from custom-fields
- rate_limited, network, api_error: X01, X02, X03, from custom-fields

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
1. Take the product list. When the statement has not read it yet: call ApiList on /products for all pages. The parameter `fields` names the API sources of the fields of products that are not heavy. NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message X03. The list is read at most once for one statement, and at most once for the answer to the tool list. It is not kept after that, in stdio mode and in http mode.
2. Compare the value with the key and the NumericId of every product, and then with the name of every product. The comparison is the equality of text of evaluate-condition. The first match is the result.
3. Without a match, when hints is true, take the first case that applies:
   a. the value is a ReferenceNumber whose kind is a parent filter of the entity → wrong_kind, message P02;
   b. the value is a ReferenceNumber of the kind of the entity → wrong_kind, message P03;
   c. the value is a ReferenceNumber with a kind, and the part before its first `-` is the key of a product → wrong_kind, message P04.
4. In every other case → bad_product, message P01.

POSTCONDITIONS:
- the key of the result is written as the account writes it, whatever the case of the value
- a product that exists when the statement runs is found, because the statement reads the list

ERRORS:
- bad_product: P01
- wrong_kind: P02, P03, P04
- not_authorized, rate_limited, network, api_error: X04, X01, X02, X03

## BEHAVIOR/INTERNAL: plan-fetch
Constraint: required

Chooses the requests for a read statement and returns the API records. It
uses the narrowest route that the WHERE condition allows. A list costs one
request per page and never one request per record.

INPUTS:
```
statement:   ReadStatement
used_fields: set of FieldName
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
   b. products: the product list, as resolve-product reads it. Scope: "all products". When used_fields has the field `description` or a custom field, the parameter `fields` of this reading also names `description` or `custom_fields`, and step 8 e applies to the custom fields. This reading is the product list of the statement: the statement does not read the list a second time.
   c. users: the list /users, read as step 8 says. Scope: "all users".
   d. statuses: the WHERE condition must have a conjunct `product = value`; otherwise → scope, message C01. Run resolve-product with hints for the value; on failure → the error of resolve-product. Call ApiGet on /products/{product} and ApiGet on /products/{product}/workflows, both with the NumericId of the product. The record of the product names the workflow of each record kind, and the entity statuses says which records the two answers give. Scope: "product <KEY>". The conjunct is applied.
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
5. Route by a list of values. This step applies when step 2 took no route. Take the first conjunct of the form `field IN (values)` that has at most MaxInValues values and whose field has a read route. The route of the field `product` counts here. A list with NULL or with empty text is not taken, because no route reads the records without a value. The conjunct is applied.
   a. First check every value of the list, in its order, as step 3 says for the field: the form of the value, and for the field `product` the result of resolve-product. The first value that fails gives its error. No record and no list of the entity is requested before every value has passed.
   b. A value that means the same as an earlier value of the list is left out: the same text without regard to case, or the same product. It is read once, and the scope names it once.
   c. Read every value that is left as step 3 says, each list to its last page; the steps 8 a and 8 b do not apply. Put the records together and leave out a record that was read before. Scope: the scopes of these values, joined by a comma and a space. A scope that equals an earlier one is named once.
6. Route by product. This step applies when no route was taken so far and the entity has a read route for the field `product`. When the WHERE condition has a conjunct `product = value`: read it as step 3 b says, and the conjunct is applied.
7. Account-wide list. This step applies when no route was taken so far.
   a. When the entity has no account-wide list → scope, message C01.
   b. When the entity has name search, look for a conjunct `name = text` or `name CONTAINS text` with a text that is not empty. The text of the first such conjunct is sent with the list request as the parameter `q`. This conjunct is not applied: read-statement still evaluates it on every record.
   c. Read the account-wide list. Scope: "all products".
8. Reading a list. Request it page by page with per_page = PageSize, from page 1 on, every page once.
   a. Count only. It applies when the select items are one `COUNT(*)` and nothing else, the statement has no GROUP BY, and every conjunct of the WHERE condition is applied. Request page 1 only. `total` is its total_records and count_only is true.
   b. Early end. It applies when every conjunct of the WHERE condition is applied and the statement has no ORDER BY, no GROUP BY, no DISTINCT and no aggregate item. Stop when enough records are read. Enough is the LIMIT of the statement, and default_limit without a LIMIT; it is never more than max_limit. When the list was not read to its end, `total` is its total_records.
   c. Too many pages. When page 1 reports more than max_pages pages, and neither a nor b ends the reading on page 1 → scope, message C02. No further page is requested.
   d. In every other case read the list to its last page.
   e. Custom fields. When used_fields has a custom field, the parameter `fields` also names `custom_fields`. When page 1 then has records and none of them has the member `custom_fields` → api_error, message X05. No further page is requested. The server does not deliver such records with empty custom fields, and it does not read them one by one.
   MECHANISM: every list request has the parameter `fields`. It has the API sources of all fields of the entity that are not heavy, and of the heavy fields among used_fields. Of a source with a dot it has the part before the dot (see INTERFACES). It has `custom_fields` only when used_fields has a custom field. One request per page delivers complete records. No request is sent per record of a list.
9. Lookups for the evaluation. When the WHERE condition compares a field of the type user with `me`, `myself` or `current_user`, read the current user with one ApiGet on /me. For a custom field of the kind users, one part of a value with commas that is such a word counts in the same way (evaluate-condition, step 11 c). When it compares a field of the type key outside the applied conjuncts, take the product list as resolve-product does. Each lookup is read at most once per statement.
10. A request of the steps 1 to 9 that fails in another way than the steps say: NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message X03. No partial result is returned.

POSTCONDITIONS:
- every record of the result belongs to the scope
- the scope is never empty text
- every page of a list was requested at most once, with increasing page numbers
- the number of requests does not grow with the number of records in a page
- at most max_pages pages of one list were requested
- a list request names `custom_fields` only when the statement uses a custom field

ERRORS:
- scope: an entity that needs a product or a parent in the WHERE condition (C01); too many pages (C02)
- bad_product: P01, from resolve-product
- wrong_kind: P02 to P04 from resolve-product; K01 to K06 and K08
- not_found: a parent that does not exist (K07)
- not_authorized, rate_limited, network, api_error: X04, X01, X02, X03
- api_error: a list that comes without the custom fields that the request asked for (X05)

## BEHAVIOR/INTERNAL: convert-record
Constraint: required

Turns one API record into a record with the fields of the catalog.

INPUTS:
```
entity:        EntityName
api_record:    ApiRecord
known_product: ProductKey or none   // the route was a product route
known_parent:  RecordId or none     // the route was a parent route
custom:        List<CustomField>    // the custom fields that the statement uses,
                                    // or the star custom fields of read-statement
```

OUTPUTS:
```
record:          Map<FieldName, Value>          // every field of the entity, and
                                                // the custom fields of the input
other_spellings: Map<FieldName, List<Value>>    // accepted by conditions, never shown
```

PRECONDITIONS:
- the API record was read with the members that the catalog gives as sources

STEPS:
1. For every field of the entity, take the value from the API source of the catalog. A member that is absent, and a source `a.b` whose member `a` is absent, give null.
2. Convert by the type of the field.
   a. text: when the value is a JSON object, take its member `body` (see INTERFACES). Turn HTML into plain text. The end of a paragraph, of a list item, of a heading, of a block and of a table row, and a line break tag, become a line break. Every other tag is removed. Character references are replaced by their characters. Several spaces in a row become one space. Empty lines and surrounding white space are removed. Empty text gives null.
   b. date: the first 10 characters of the value.
   c. boolean: true or false. An absent value gives null, except where the catalog says false.
   d. key: for the field `product`, the known product; without it, the key of the product whose NumericId the source has. The product list is taken for this at most once per statement, and only when the statement uses the field.
   e. every other type: the value as the API has it.
3. Apply the notes of the catalog entry. The tags of a feature are joined by a comma and a space, and a feature without tags has null. A requirement without a feature in the API record takes the known parent, and so does a comment without a commented record.
4. Custom fields. For every custom field of the input, take its entry from the member `custom_fields` of the API record. Convert the `value` of the entry by the kind of the field, as the table of custom values in custom-fields says. A record without the entry has null in the field.
5. Collect the other spellings of a value. Conditions accept them; no result shows them.
   - `id`: the NumericId of the record. For products, `key`: the NumericId of the product.
   - `product`: the NumericId of the product.
   - `assigned_to`: the name and the NumericId of the user.
   - a field of the type reference: the NumericId and the name of the referenced record, where the API record has them.
   - `user` of comments: the e-mail address and the NumericId of the user.
   - `parent` of comments: the NumericId of the commented record.
   - `entity` of statuses: the singular of the entity and the part of the member name before `_workflow`.
   - a custom field of the kind users: the names of its users, joined by a comma and a space. The NumericIds of its users, joined in the same way, are a second spelling.
   - a field with several values: each single value, as the API record lists it. These fields are `tags` of features, and a custom field of the kind tags, fixed_tags or users. For a users field the single values are the e-mail address, the name and the NumericId of each user. evaluate-condition, step 11, uses them.

POSTCONDITIONS:
- the record has every field of the entity and every custom field of the input, each with a value or null
- a field of the type text has no HTML tags, and a custom field of the kind note has none

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
- every value of the condition fits the type of its field: parse-statement has checked it against the table of values in a condition
- the lookups that the condition needs were read; update-statement and delete-statement read them in the same way before they check their guards, and a failed request there is the error of plan-fetch, step 10

STEPS:
1. AllOf is true when every member is true. AnyOf is true when at least one member is true. Negation is true when its member is not true.
2. NullTest: a field has no value when it is null or empty text. IS NULL is true for a field without a value, and IS NOT NULL for a field with a value.
3. InList is true when the comparison `field = value` is true for at least one of its values. NULL or empty text in the list stands for a field without a value: the list is then also true for such a field.
4. Comparison. First replace the value of the statement in three cases.
   a. The field has the type user and the value is `me`, `myself` or `current_user` in any case. The value becomes the e-mail address of the current user. Where the field is read as a name, it becomes the name of the current user.
   b. The field has the type key and resolve-product finds the value. The value becomes the key of that product.
   c. The field is `entity` of statuses and resolve-names accepts the value as an entity word. The value becomes the name of that entity.
5. Text is compared in its comparison form. The comparison form has no surrounding white space and no difference of case. Typographic apostrophes and quotation marks are replaced by ' and ", and the no-break space is replaced by a space.
6. `=` is true when the value equals the value of the field or one of its other spellings:
   a. when one side is a Boolean: the truth values are equal, where the texts `true` and `false` and the numbers 1 and 0 are read as truth values;
   b. when the field has the type number: the numbers are equal;
   c. when the field has the type date: the day of the field equals the day of the value. The day is the first 10 characters;
   d. when the field has the type datetime and the value is a DateText: the day of the field in UTC equals the value;
   e. when the field has the type datetime and the value is a timestamp: both are the same point in time;
   f. in every other case: the comparison forms are equal.
   A field without a value equals nothing.
7. `!=` is true when `=` is not true. It is true for a field without a value.
8. CONTAINS is true when the comparison form of the value is inside the comparison form of the field or of one of its other spellings. LIKE is true when the pattern covers the text from its first to its last character, where `%` stands for any text and `_` for one character. Both are false for a field without a value.
9. `<`, `<=`, `>` and `>=` are false for a field without a value. A field of the type number is compared as a number. A field of the type date is compared by its day with the day of the value, as text. A field of the type datetime is compared with a DateText by its day in UTC, and with a timestamp as a point in time. A value of a datetime field that cannot be read as a timestamp is compared as text. Every other field is compared in the order of text that build-rows, step 4, gives: a run of digits counts as a number.
10. Custom fields. A custom field has the field type that the table of kinds in custom-fields gives for its kind. The steps 1 to 9 and step 11 apply to it as to a catalog field of that type.
11. A field with several values: `tags` of features, and a custom field of the kind tags, fixed_tags or users. convert-record gives each single value of such a field as another spelling.
   a. `=` is therefore true when one value of the field equals the value of the statement. `tags = 'urgent'` is true for a feature with the tags urgent and customer.
   b. A value of the statement that has a comma is also read as several values: it is split as check-write-values, step 8, says. `=` is then true as well when the field has each of these values, in any order. The field can have more values. A value without any part after the split adds nothing.
   c. For a users field, step 4 a applies to each value of the statement: `me` stands for the current user. A value is compared with the e-mail address, the name and the NumericId of each user of the field.
   d. `!=` is true when `=` is not true, and IN is true when `=` is true for one value of its list.
   e. The steps 2, 8 and 9 are not changed for such a field. The test for no value, CONTAINS, LIKE and the operators `<`, `<=`, `>` and `>=` work on the joined text, and CONTAINS also on the other spellings. GROUP BY, ORDER BY and DISTINCT use the joined text.

POSTCONDITIONS:
- the result depends only on the record, the condition and the lookups
- `=` on `name` is equality of the complete name, not a search
- `=` on a field with several values is true for every record that has the value among the values of the field

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
star_custom: List<CustomField>   // the star custom fields of read-statement,
                                 // with their values in the kept record;
                                 // empty in most statements
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
   b. Records with equal values in all group fields are one group. Text values are equal when their comparison forms are equal (see evaluate-condition). Records without a value in a group field are one group for that field. The groups are in the order of their first record. Without group fields all records are one group, and there is one group also when there is no record.
   c. One row per group. First the group fields that the statement does not select, under their field names. Then the select items in the order of the statement: a field item has the value of the first record of the group, and an aggregate item has its result.
   d. COUNT(*) is the number of records of the group. COUNT(field) is the number of records with a value in the field. COUNT(DISTINCT field) is the number of different values. SUM and AVG use the values that are numbers. MIN and MAX use the order of step 4. Without a value the result is null, and 0 for COUNT. A result that is a whole number is written without decimals; every other result is rounded to two decimals.
3. Plain result: when the statement has no aggregate item and no GROUP BY. Sort the records as step 4 says. Make one row per record with the select items in the order of the statement, each under its label. With DISTINCT, leave out a row that equals an earlier row in all values.
   a. Star custom fields. After the select items, the row gets every star custom field that has a value in the record. Each one is under the name of the field, and they are in the order of these names. A star custom field without a value is not in the row.
4. ORDER BY. Sort by the first key. Rows that are equal in it are sorted by the next key, and so on. Rows that are equal in all keys keep their order. A key is a field, the label of a select item, or an aggregate, as parse-statement accepts it. A label stands for its select item. In a grouped result an aggregate key is computed for each group, also when the statement does not select it. Numbers come before text. Numbers are sorted by value. Text is sorted by its comparison form, character by character. Where both texts have a run of digits at the same place, the two runs are compared as numbers of any length: `DEMO-9` comes before `DEMO-10`. Two texts that this order does not separate are sorted character by character. A row without a value in the key comes last, with ASC and with DESC. A grouped result is sorted after step 2.
5. Limits. The limit is the LIMIT of the statement, and default_limit without a LIMIT. It is never more than max_limit. rows are the first rows up to the limit, and count is their number.
6. total has a value in two cases. When step 5 cut rows, it is the number of rows before the cut. When plan-fetch did not read the list to its end, it is list_total.
7. truncated is true when total has a value and the statement did not ask for the cut: it has no LIMIT, or a LIMIT above max_limit.

POSTCONDITIONS:
- an aggregate is computed over all matching records, also when step 5 cuts the rows of a grouped result
- every row has the labels of the statement in its order; in a grouped result the group fields that are not selected come first
- after these labels a row has the star custom fields that have a value, and no other label
- no row has a label twice
- count is never above max_limit

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
fields: Map<FieldName, Value | TextList> | AqlError
```

PRECONDITIONS:
- every field is a writable field of the entity, or a custom field of the entity that can be set

STEPS:
1. Go through the fields in the order of the statement and check each one by its type. A custom field is checked by its kind, as step 10 says. The first value that fails ends the check.
2. string and text: the value must be Text; otherwise → bad_value, message W28. The text is sent as the statement has it. Empty text is a value.
3. number: a Number, or Text that reads as a number; otherwise → bad_value, message W26.
4. boolean: a Boolean, or the text `true` or `false` in any case; otherwise → bad_value, message W27.
5. date: Text of the form YYYY-MM-DD that is a day of the calendar. Of a timestamp that starts with such a date, the first 10 characters are taken. Any other value → bad_value, message W22.
6. user: `me`, `myself` or `current_user` in any case is replaced by the e-mail address of the current user, read with ApiGet on /me. An e-mail address and a NumericId are sent as they are. Any other value → bad_value, message W23.
7. reference: the value must be a ReferenceNumber of the kind of the field, or a NumericId. It is sent in capital letters. For a value of another form: when resolve-product finds it as a product → wrong_kind, message W24. The records of that kind in the product are read for the message with one list request. Any other value → not_found, message W25.
8. The field `tags`: the text is split at commas into tags. White space around a tag is removed and an empty tag is left out.
9. status, unless form_only is true.
   a. Take the workflow statuses of the entity in the product, in workflow order, as plan-fetch reads them for the entity statuses. The record of the product names the workflow of the entity, and the list of workflows has its statuses. Both answers are read at most once per statement and are not kept after it.
   b. For a new record the product is the product of the parent. When the parent is a record, read it with ApiGet on its id route to learn its product; NotFound → not_found, message W25.
   c. Compare the value with the names of the statuses. The comparison is the equality of text of evaluate-condition. The name is sent as the workflow writes it.
   d. Without a match → bad_value, message W21. This is also the error when the product names no workflow for the entity.
10. A custom field, by its kind. The table of values to set in custom-fields gives the accepted value and the form that is sent.
   a. text, url, note and choice: the value must be Text; otherwise → bad_value, message W28.
   b. number: as step 3. date: as step 5.
   c. fixed_choice: the value must be Text; otherwise → bad_value, message W28. Empty text is accepted without a comparison. Any other text is compared with the texts of the options of the definition, which are read as custom-fields says. The comparison is the equality of text of evaluate-condition, and the text is sent as the option writes it. Without a match → bad_value, message W32.
   d. tags: the value must be Text; otherwise → bad_value, message W28. The text is split into tags as step 8 says. The checked value is the TextList of the tags.
   e. fixed_tags: as tags. Then each tag is compared with the options as step 10 c says, and it is sent as the option writes it. The first tag without a match → bad_value, message W32.
   f. users: the value must be Text; otherwise → bad_value, message W28. The text is split at commas into parts. White space around a part is removed, and an empty part is left out. `me`, `myself` or `current_user` in any case is replaced by the NumericId of the current user, read with ApiGet on /me. An e-mail address and a NumericId stay as they are. Any other part → bad_value, message W23. The checked value is the TextList of the parts.
11. A request of this behavior that fails in another way: NotAuthorized → not_authorized, message X04. RateLimited → rate_limited, message X01. Unreachable → network, message X02. Any other failure → api_error, message X03.

POSTCONDITIONS:
- no ApiCreate, ApiUpdate or ApiDelete call was made
- a status is sent with the name that the workflow has
- a value for a custom field of the kind fixed_choice or fixed_tags is sent only when it is empty or one of the options

ERRORS:
- bad_value: W21, W22, W23, W26, W27, W28, W32
- wrong_kind: a product where a record is needed (W24)
- not_found: W25
- not_authorized, rate_limited, network, api_error: X04, X01, X02, X03

## BEHAVIOR/INTERNAL: write-request
Constraint: required

Builds the body of a create request or an update request from the checked
fields of a write statement.

INPUTS:
```
entity: EntityName
fields: Map<FieldName, Value>    // the output of check-write-values
```

OUTPUTS:
```
body: JSON object
```

PRECONDITIONS:
- check-write-values has succeeded for the fields

STEPS:
1. The body is a JSON object with one member. The name of that member is the singular of the entity. Its value is a JSON object with one member per catalog field, in the order of the statement. All custom fields of the statement are together in one further member (step 6).
2. The name of a member is the request member that the catalog gives for the field. For a field without a request member it is the name of the field.
3. The value of a member depends on the type of the field.
   a. string and text: the text as a JSON string. Text is not converted to HTML.
   b. number: a JSON number. boolean: a JSON boolean.
   c. date: the date as a JSON string of the form YYYY-MM-DD.
   d. status: the name of the status as a JSON string.
   e. user: the e-mail address or the NumericId as a JSON string.
   f. reference: the id as a JSON string.
4. The field `tags`: the tags as a single JSON string, joined by a comma.
5. The parent of a new record is part of the route. It is not a member of the body.
6. Custom fields. When the statement assigns a custom field, the object of step 1 has the member `custom_fields`. Its place among the members is the place of the first custom field in the order of the statement. Its value is a JSON object with one member per custom field, in the order of the statement. The name of such a member is the key of the custom field, as its definition writes it. Its value is the value in the request that the table of values to set in custom-fields gives for the kind.
7. Goals. The field `success_metric` is in a create request only. It is the member `success_metric`, and its value is a JSON object with the member `name`. In a create request the status of a goal is the member `workflow_status` of that object, after `name`. The goal then has no member `workflow_status` of its own. In an update request the status is the member `workflow_status` of the goal, as for every other entity.

For the statement `UPDATE features SET status = 'In design', assigned_to =
'me' WHERE id = 'DEMO-1'` on the fixture account, the body is
`{"feature":{"workflow_status":"In design","assigned_to_user":"test.user@example.com"}}`.
For `UPDATE features SET name = 'CSV export 2', custom.priority = 'low'
WHERE id = 'DEMO-1'` it is
`{"feature":{"name":"CSV export 2","custom_fields":{"priority":"Low"}}}`.
For `INSERT INTO goals (parent, name, success_metric, status) VALUES
('DEMO', 'Win the cup', 'Podium places', 'At risk')` it is
`{"goal":{"name":"Win the cup","success_metric":{"name":"Podium places","workflow_status":"At risk"}}}`.

POSTCONDITIONS:
- the body has one member for every catalog field of the statement, the member `custom_fields` with one member for every custom field of the statement, and no other member
- exception: the status of a new goal is inside the member `success_metric` (step 7)
- the body has no member for the id, the parent or the product of the record

ERRORS:
- none

## BEHAVIOR/INTERNAL: aha-request
Constraint: required

Sends one operation of the AhaAPI as an HTTPS request and reads the
answer. Every request of the server to Aha! goes through this behavior.

INPUTS:
```
account:     Account
method:      GET | POST | PUT | DELETE
route:       string                 // from the catalog, with its parameters replaced
parameters:  Map<string, string>    // the parameters of ApiList
body:        JSON object or none    // from write-request
environment: the variables HTTPS_PROXY, https_proxy, NO_PROXY, no_proxy,
             SSL_CERT_FILE and SSL_CERT_DIR
```

OUTPUTS:
```
answer: JSON value | none | ApiFailure
```

PRECONDITIONS:
- the subdomain of the Account matches Subdomain, and its credential matches Credential

STEPS:
1. Build the address: `https://`, the subdomain, `.aha.io/api/v1`, the route, and the parameters as a query string. Every value that replaces a route parameter and every parameter value is percent-encoded. No address has another scheme or another host.
2. Send the request with the headers `Authorization: Bearer <credential>`, `Accept: application/json` and `User-Agent: mcp-server-aha-aql/<version> (https://github.com/mge1512/aha-aql)`. A request with a body also has `Content-Type: application/json`. Aha! asks every client for a way to reach its developer, and the address of the project in the User-Agent is that way.
3. Proxy. When the environment gives an HTTPS proxy and does not exclude the host of Aha! from it: connect to the proxy and ask it with CONNECT for a tunnel to `<subdomain>.aha.io`, port 443. In every other case connect to that host and port directly.
   MECHANISM: the standard library of the language reads the variables for an https address. HTTPS_PROXY or https_proxy gives the proxy. NO_PROXY or no_proxy gives the hosts without a proxy.
4. TLS. The connection to Aha!, direct or inside the tunnel, uses TLS. The certificate must be valid for the host name `<subdomain>.aha.io`, and its chain must end in the trust store of the platform. On Linux the variables SSL_CERT_FILE and SSL_CERT_DIR can name that trust store. No setting turns this check off.
5. Failures before an answer. In each of these cases → Unreachable, and the request is not sent again:
   a. the proxy cannot be reached, or it does not open the tunnel;
   b. the connection or the certificate check fails;
   c. no answer arrives within 60 seconds.
6. A redirect is not followed. An answer with a status from 300 to 399 → Rejected, with the status.
7. Status 429: wait, and send the same request again. The header `X-Ratelimit-Reset` of the answer gives the time when the limit ends, as UTC Unix time. The waiting time is the number of seconds from now to that time, at least 1 second and at most 30 seconds. Without the header it is 5 seconds. The request is sent again at most twice. If the third answer also has the status 429 → RateLimited.
8. Status 401 or 403 → NotAuthorized, with the status. Status 404 → NotFound.
9. Every other status outside 200 to 299 → Rejected, with the status and the text of the answer.
10. Status 200 to 299: the answer is the body, read as JSON. ApiDelete needs no body. If a body that is needed cannot be read as JSON → Rejected, with the status and the text `the answer is not JSON`. If it is JSON and does not have the form that INTERFACES gives for the operation → Rejected, with the status and the text `the answer has an unknown form`.

A proxy that ends TLS. A sandbox or a company network can use a proxy that
ends the TLS connection itself and opens its own connection to Aha!. The
server then gets the certificate of that proxy in step 4. It accepts that
certificate only when the trust store has the certificate authority of the
proxy. The operator of the environment decides this with the trust store.
In such an environment the credential can be a placeholder that the proxy
replaces on the way to Aha!. The server sends it as it is.

POSTCONDITIONS:
- the request was for the account of the Account and for no other host
- the server connected to that host or to the proxy of the environment, and to nothing else
- the credential was sent in the Authorization header of that request and nowhere else
- a request was sent again only after the status 429, and at most twice
- no failure has the credential or the address of the proxy in its text

ERRORS:
- Unreachable: no connection, a proxy that does not open the tunnel, a certificate that cannot be verified, or no answer in time
- NotAuthorized: status 401 or 403
- NotFound: status 404
- RateLimited: status 429 three times
- Rejected: a redirect, another status that is not a success, an answer that is not JSON, or an answer that does not have the form of INTERFACES

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
- none; the ExampleKey is chosen when a message first needs it (see choose-example-key)

STEPS:
1. Take the text of the ID from the tables below and replace every placeholder. The ID itself is not part of the message.
2. Write a value of the statement, shown below as `'<value>'`, as a statement has to write it: in apostrophes, with every apostrophe of the value written twice. `<VALUE>` is the value in capital letters.
3. Replace `<KEY>` by the ExampleKey. `<PRODUCT>` is the key of the product that the message is about.
4. Take lists of entities and of fields from the catalog, in catalog order, joined by a comma and a space.
5. Show at most MaxListed entries of a list of records, of product keys, of custom fields or of options. When there are more, end the list with `, and <n> more`.
6. The message of an error is the complete tool result of `query` and of `execute` for that error. A note is part of the result of a read statement.
7. Articles. Where a text below has the word `a` directly before `<kind>`, `<expected kind>` or `<singular>`, the message has `an` when the word that replaces the placeholder starts with a, e, i, o or u: `an epic`, `an idea`, `an initiative`.

The messages speak of the query tool and the execute tool. In manual mode
these are the two commands of the same names; the texts stay the same.

A hint that is not empty starts with one space, so that it can follow the
period of the sentence before it.

Placeholders that several messages use:

| Placeholder | Text |
|---|---|
| `<example of X>` | The example statement of the entity X, with {KEY} replaced |
| `<entity list>` | The names of all catalog entities |
| `<field list>` | The names of all fields of the entity |
| `<writable list>` | The names of the writable fields of the entity. A field that is writable in INSERT only has ` (INSERT only)` after its name |
| `<changeable list>` | The names of the entities that have an operation |
| `<read form>` | `SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]` |
| `<write forms>` | The three form lines of the description of `execute`, joined by a space, a vertical bar and a space |
| `<where>` | `the end of the statement`, or `position <n> (<text>)` with the position of the token that cannot be read and the next 24 characters of the statement from there, without surrounding white space |
| `<apostrophe hint>` | Empty, unless the statement has an apostrophe directly between a letter or digit and a letter. Then: ` The text contains an apostrophe (<word>). Inside text, write an apostrophe twice: <word doubled>`. `<word>` is the run of characters without a space around the first such apostrophe, without the characters ' " % ( ) , ; at its ends |
| `<example id>` | By kind: feature `<KEY>-23`, release `<KEY>-R-3`, initiative `<KEY>-S-1`, epic `<KEY>-E-1`, goal `<KEY>-G-1`, idea `<KEY>-I-1`, requirement `<KEY>-23-1`, product `<KEY>` |
| `<records>` | One entry `<id> (<name>)` per record, joined by a comma and a space; `none` when there is no record |
| `<singular>`, `<kind>` | The singular of the entity, or the name of the record kind. For the entity me the singular is `user` |
| `<custom list>` | The names of the custom fields of the entity, as custom-fields returns them without a key, joined by a comma and a space |
| `<writable custom list>` | The names of the custom list whose kind can be set |
| `<custom hint>` | Empty when the custom list is empty. Otherwise: ` Custom fields of <entity>: <custom list>.` |
| `<id field>` | The id field of the entity: `key` for products, `id` for every other entity |

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
| S05 | syntax | `<ID> is an id, not a statement. To read that record: SELECT * FROM <entity> WHERE id = '<ID>'` |
| S06 | syntax | `<word> is an entity, not a statement. To list its records: <example of entity>. To see its fields: DESCRIBE <entity>` |
| S07 | syntax | `query only reads. Use the execute tool for INSERT, UPDATE and DELETE.` |
| S08 | syntax | `JOIN is not supported. Read one entity per statement and filter by its parent, for example: <example of features>` |
| S09 | syntax | `HAVING is not supported. Use WHERE, GROUP BY, ORDER BY and LIMIT. Example: SELECT status, COUNT(*) FROM features WHERE product = '<KEY>' GROUP BY status` |
| S10 | syntax | `OFFSET is not supported. Use WHERE to narrow the result and LIMIT to shorten it.` |
| S11 | syntax | `subqueries are not supported. Send one statement per call. Form: <read form>` |
| S12 | syntax | `unknown function <NAME>. Functions: COUNT, SUM, AVG, MIN, MAX. Example: SELECT COUNT(*) FROM features WHERE product = '<KEY>'` |
| S13 | syntax | `text goes in single quotes: write '<word>' instead of <word>` |
| S14 | syntax | `DESCRIBE <entity> lists the fields of <entity> and takes nothing else. To read records: <example of entity>` |
| S15 | syntax | `two select items have the label '<label>'. Give one of them another label, in the form: field AS label` |
| S16 | syntax | `the label '<label>' is also the name of a field of <entity>. Choose another label, in the form: field AS label` |

In S05, `<ID>` is the id of the text in capital letters: the text `demo-3`
gets `DEMO-3`.

In S15 and S16, `<label>` is the label as the statement writes it. In S15
it is the label of the first pair of items that parse-statement refuses.

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
- `a field or * inside COUNT( )`
- `a field inside <FUNCTION>( )`, for SUM, AVG, MIN and MAX
- `a field after DISTINCT`
- `a field of GROUP BY, a selected field or an aggregate after ORDER BY`
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
| N02 | bad_name | `unknown field '<word>' on <entity>.<field hint> Fields of <entity>: <field list>.<custom hint> Example: <example of entity>` |
| N03 | bad_name | `users cannot be filtered by product. Fields of users: <field list>. Example: <example of users>` |
| N04 | bad_name | `unknown custom field '<word>' on <entity>. Custom fields of <entity>: <custom list>. To see their types: DESCRIBE <entity>` |
| N05 | bad_name | `<name> has the type <type>, and the server cannot read or set custom fields of that type.<custom hint>` |
| N06 | bad_name | `<entity> have no custom fields that can be used here. Fields of <entity>: <field list>. Example: <example of entity>` |
| N07 | bad_name | `the custom fields of the account cannot be read with this API key (status <n>). Fields of <entity>: <field list>. Example: <example of entity>` |
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

In N04, `<word>` is the word of the statement without a qualifier, for
example `custom.severity`. In N05, `<name>` is the name of the custom
field, `custom.` and its key, and `<type>` is the type name of its
definition, for example `EquationField`. In N07, `<n>` is the status that
Aha! gave for the list of the definitions.

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

Messages for a value in a condition or in a guard:

| ID | Kind | Text |
|---|---|---|
| V01 | bad_value | `<field> is <type text>: write the value as <form>. Got '<value>'.` |
| V02 | bad_value | `<field> is true or false: compare it with = or !=, for example <field> = TRUE` |

In V01 and V02, `<field>` is the name of the field; for a custom field it
is `custom.` and its key. In V01, `<type text>` and `<form>` depend on the
type of the field:

| Type of the field | `<type text>` | `<form>` |
|---|---|---|
| number | `a number` | `5 or 2.5` |
| boolean | `true or false` | `TRUE or FALSE` |
| date | `a date` | `'YYYY-MM-DD'` |
| datetime | `a timestamp` | `'YYYY-MM-DD' or 'YYYY-MM-DDThh:mm:ssZ'` |

Messages of write statements:

| ID | Kind | Text |
|---|---|---|
| W01 | syntax | `execute only changes data. Use the query tool for SELECT and DESCRIBE.` |
| W02 | syntax | `a statement starts with INSERT, UPDATE or DELETE.<apostrophe hint> Forms: <write forms>` |
| W03 | syntax | `<problem><apostrophe hint> Forms: <write forms>` |
| W04 | syntax | `<KEYWORD> needs WHERE id = '...' with the id of the one record to change.<apostrophe hint> Forms: <write forms>` |
| W05 | syntax | `one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record.` |
| W06 | syntax | `after WHERE id = '...', further conditions must have the form AND field = 'value'. For a field without a value write: AND field = ''` |
| W07 | syntax | `one INSERT creates one record. Send one statement per record.` |
| W08 | syntax | `<n> fields but <m> values` |
| W09 | syntax | `the parent is given twice with different values` |
| W10 | syntax | `a field can only be set to a value, not to an expression` |
| W11 | syntax | `NULL cannot be written. Leave the field out.` |
| W12 | bad_name | `unknown entity '<word>'. Entities that can be changed: <changeable list>` |
| W13 | bad_name | `<entity> cannot be changed. Entities that can be changed: <changeable list>` |
| W14 | bad_name | `<entity> cannot be <done>. Entities that can be <done>: <list>` |
| W15 | bad_name | `the id of a record cannot be set. Fields of <entity> that can be set: <writable list>` |
| W16 | bad_name | `<opening> Fields of <entity> that can be set: <writable list><parent note><custom write hint>` |
| W17 | bad_name | `unknown field '<word>' on <entity>. Fields of <entity>: <field list><custom guard hint>` |
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
| W31 | bad_name | `custom fields cannot be set in INSERT. Leave out <name>, create the record, and then send: UPDATE <entity> SET <name> = '...' WHERE id = '...'` |
| W32 | bad_value | `<field>: there is no option '<value>'. Options: <options>` |
| W33 | syntax | `the field <field> is given twice. A statement sets each field once.` |

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

In W04, `<KEYWORD>` is `UPDATE` or `DELETE`. In W14, `<done>` is
`created` for INSERT, `changed` for UPDATE and `deleted` for DELETE, and
`<list>` are the entities that have that operation.

In W16, `<opening>` is `unknown field '<word>' on <entity>.` for a word
that resolve-names does not know, and `the field <field> of <entity> cannot
be set.` for a field that is not writable. For a field that is writable in
INSERT only, in an UPDATE, it is `the field <field> of <entity> can be set
in INSERT only.` `<parent note>` is `, and parent
in INSERT` when the entity has the operation create, and empty otherwise.
`<custom write hint>` is `. Custom fields, with UPDATE only: <writable
custom list>` when the writable custom list is not empty, and empty
otherwise. In W17, `<custom guard hint>` is `. Custom fields of <entity>:
<custom list>` when the custom list is not empty, and empty otherwise.

In W33, `<field>` is the name of the field in the catalog, whatever the
statement writes. For a custom field it is `custom.` and its key.

In W31, `<name>` is the name of the custom field, `custom.` and its key. In
W32, `<field>` is that name too. `<options>` are the texts of the options
in the order of their list, joined by a comma and a space.

In W18, `<parent text>` is `a release`, `a product`, `a feature` or `the
record to comment on`. In W19, `<field>` is the field that is missing:
`name`, `body` for a comment, or `success_metric` for a goal. For a goal
without both it is `name`. In W21, `<names>` are the names of the workflow
statuses in workflow order, joined by a comma and a space. When the
product names no workflow for the entity, `<names>` is `(none)`.

In W25, `<PRODUCT>` is the key of the product of the record that the
statement changes. Without such a record it is the part of the value
before its first `-`, when that part is the key of a product. In every
other case it is the ExampleKey.

For the kind requirement the statement of W25 is `SELECT id, name FROM
requirements WHERE feature = '<FEATURE>'`, because requirements are listed
per feature and not per product. `<FEATURE>` is the value in capital
letters without its last hyphen and the number after it: `DEMO-1` for
`DEMO-1-9`.

In W29, for a field without a value the text is `nothing was changed:
<singular> <id> has no <field>, not '<value>'`.

Messages for failed requests:

| ID | Kind | Text |
|---|---|---|
| X01 | rate_limited | `aha: API error (status 429): too many requests` |
| X02 | network | `cannot reach Aha!: <reason>.<write hint>` |
| X03 | api_error | `aha: API error (status <n>) on <method> <route>: <answer>` |
| X04 | not_authorized | `Aha! refused the request (status <n>): the API key is not valid for the account <subdomain>, or it has no permission for this record. Repeating the statement does not help.` |
| X05 | api_error | `Aha! returned this list of <entity> without custom fields. A record that is read by its id has them: add <id field> = '...' to WHERE.` |

In X02, `<reason>` is one of these texts:

- `no answer within 60 seconds`
- `the certificate of <subdomain>.aha.io could not be verified`
- `the proxy did not open a connection to <subdomain>.aha.io`
- `the connection to <subdomain>.aha.io failed`

`<write hint>` is ` The change may have been made. Read the record before
you send the statement again.` for ApiCreate, ApiUpdate and ApiDelete. It
is empty for ApiList and ApiGet. It is also empty when the reason is the
certificate or the proxy, because then no request was sent.

In X03 and W30, `<answer>` is the first 300 characters of the text that
the API returned, and `(no text)` when it returned none. W30 has the text
of X03. It has its own ID because a write request causes it.

In X04, `<subdomain>` is the subdomain of the Account. The message has no
part of the credential.

Notes in the result of a read statement:

| ID | Text |
|---|---|
| H01 | `* shows <default fields>. Other fields of <entity>: <other fields>.` |
| H02 | `Showing <count> of <total> records. Narrow the result with WHERE, raise LIMIT (up to <max_limit>), or count with COUNT(*).` |
| H03 | `There is no <singular> with the <id field> '<value>'.` |
| H04 | `'<value>' is the key of a product, not its name. Use: SELECT key, name FROM products WHERE key = '<PRODUCT>'` |
| H05 | `'<value>' looks like an id, not a name. Use: WHERE id = '<VALUE>'` |
| H06 | `'<value>' looks like the id of a <kind>, not a name. Use: WHERE <kind> = '<VALUE>'` |
| H07 | `No <singular> has exactly the name '<value>'. name = compares the whole name; to search inside names use: name CONTAINS '<value>'` |
| H08 | `The <singular> <ID> exists, and another condition of the statement is not true for it. To see the record: SELECT * FROM <entity> WHERE <id field> = '<ID>'` |
| H09 | `<PRODUCT> has statuses for: <entities>.` |

In H08, `<ID>` is the id of the record as the record has it. In H09,
`<entities>` are the values of `entity` that the product has statuses for,
in the order of the entity statuses, joined by a comma and a space. For a
product without any status it is `(none)`.

POSTCONDITIONS:
- every complete statement that a message shows is accepted by parse-statement, after its placeholders are replaced
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
default_product: the setting default_product, or none
account:         Account
```

OUTPUTS:
```
key: ExampleKey
```

PRECONDITIONS:
- none

STEPS:
1. Choose the ExampleKey when it is first needed. That is when the tool list is answered, or when a DESCRIBE result, a message or a note shows it. In stdio mode and manual mode it is chosen once for the process. It is the one value from Aha! that the process keeps between statements; the product list that was read for it is not kept. In http mode it is chosen at most once for each request, with the Account of that request.
2. When the setting default_product has a value and resolve-product finds it: the ExampleKey is the key of that product.
3. In every other case the ExampleKey is PROD. While a product of the account has the ExampleKey as its key, in any case, add the letter X at its end.
4. When the setting default_product has a value and resolve-product does not find it: continue with step 3, and then write the warning G15 to the diagnostic output.
5. When the product list cannot be read: the ExampleKey is PROD. Write the warning G16 to the diagnostic output, and not G15. The failed reading is not an error of the call that needed the key.

POSTCONDITIONS:
- the ExampleKey is the key of the product of the setting default_product, or a key that no product of the account had when the key was chosen, or PROD when the product list could not be read
- a call that shows no example causes no request for the ExampleKey
- no warning has the credential

ERRORS:
- none

## PRECONDITIONS

- The Aha! API reports total_records and total_pages for every list. An empty list reports 0 for both. This was observed on one account on 2026-11-24.
- The answers of the Aha! API have the JSON forms that INTERFACES gives. The examples of the API documentation show these forms. The prototype of the test of 2026-11-12 read lists and single records in these forms on one account. A run of 2026-11-24 found them on one account. It read the lists, a feature, an epic, a goal, a product, the workflows and /me.
- The Aha! API accepts the parameter `fields` on the list routes of the catalog and a page size of PageSize. This was observed on one account on 2026-11-12, and on 2026-11-24 for 20 list routes of the catalog. Not requested then: /epics/{epic}/features and the comment lists of releases, epics, ideas, initiatives, goals and requirements.
- The Aha! API accepts an API key and an OAuth2 access token in the same way: as a bearer value in the Authorization header.
- The credential has the permissions in Aha! that a statement needs. The server gives no permission and takes none away.
- The Aha! API lists the custom field definitions of an account on /custom_field_definitions. Each definition has an id, a key, a name, a type and the record type that it belongs to. The API is documented so. One account answered so on 2026-11-24, with all its definitions in one answer without page information.
- The Aha! API returns the member `custom_fields` with a record that is read by its id. It also returns that member with the records of a list when the parameter `fields` names it. The first is documented. Both were observed on one account on 2026-11-24, the second on the feature list of a product and on the product list.
- The Aha! API accepts the member `custom_fields` in an update, next to other members, with one value per key. The documentation shows a text value. On 2026-11-24 an update with this member alone set lists with fixed options on one account. One option was sent as its text, and several options as a list of texts. An empty list removed them. The forms of write-request for numbers, dates, tags, users and notes are not documented and were not observed. This member next to other members was not observed either.
- The Aha! API accepts the values of write-request in an update of a feature. They are the status as its name, the user as an e-mail address, the release and the initiative as ids, and the tags as one text. This was observed on one account on 2026-11-24.
- The create route and the delete route of every entity that can be changed work as the catalog says. This was observed on one account on 2026-11-24; a comment was created on a feature only. An update was observed for features, epics, requirements and goals.
- The record of a product names the workflow of each record kind in a member whose name ends in `_workflow`. The list of workflows of the product has that workflow. This was observed on two products of one account on 2026-11-24, where features, epics and requirements named one workflow. The API documentation shows the same members.
- The Aha! API creates a goal only when the request has a success metric with a name. An update takes the status of a goal as `workflow_status`. Both were observed on one account on 2026-11-24. A new goal takes its status inside `success_metric`, as the documentation shows; this was observed on the same day. An update with a new text in `success_metric.name` is answered without an error and does not change the text. The documentation shows no other form for it.
- The Aha! API writes a timestamp as text in an ISO 8601 form. The runs of 2026-11-24 did not record which form: with or without a fraction of the second, with `Z` or with an offset. The server reads each of these forms, so no result depends on it. The fixture account uses the form with milliseconds and `Z`.

## POSTCONDITIONS

- After a read statement or a describe statement, no data in Aha! has changed.
- After a call of `execute` that returns a result, exactly one record was created, changed or deleted, and the result gives its id. With dry_run true, no record was created, changed or deleted.
- After a write statement that returns an error from parsing, from name resolution, from the id check, from a guard or from a value check, no data in Aha! has changed.
- Every result of a read statement gives its scope, and it says so when rows were cut.
- Every error gives the valid names or values, or a statement that can be sent as it is.
- A process in stdio mode or manual mode works with one Aha! account from its start to its end.

## INVARIANTS

- [observable] every row of a read result satisfies the WHERE condition of its statement, for every form of condition that parse-statement accepts, OR and NOT included
- [observable] a read result has every matching record, unless truncated is true or the statement has a LIMIT, and then total gives the number of matching rows
- [observable] every field and every custom field that describe-statement lists for an entity is delivered by read-statement with its value or with null when a statement names it, and every other field word gets an error with the list of the valid names
- [observable] every read result gives its scope: the record, the parent, the product or the account-wide list that was read
- [observable] a result row has the labels of the select items in the order of the statement, a grouped result has the fields of GROUP BY that are not selected before them, and a field without a value is null; the custom fields that `*` adds for one record that is selected by its id field follow these labels, and of them only those that have a value
- [observable] one call of execute creates, changes or deletes at most one record
- [observable] a write statement is checked completely before its write request: an unknown column, an unknown field, a value of the wrong form, an unknown status, a value that a fixed list of options does not have and a guard that is not true are errors, and none of them changes data
- [observable] an accepted write statement sends every field that it assigns; no column, assignment or condition is ignored without an error
- [observable] no JSON object of a tool result and no body of a request has two members with the same name
- [observable] the tool query sends only read requests, and with read_only true the server offers no tool that accepts INSERT, UPDATE or DELETE
- [observable] the tool list is query and execute, or query alone with read_only true; the server offers no other tool, no resource and no prompt
- [observable] text that follows a complete statement is an error, for every statement kind
- [observable] the error for an unknown entity or field lists the valid names, and every syntax error shows a form or a statement that parse-statement accepts
- [observable] every example statement in a tool description, in a DESCRIBE result, in an error message and in a note is accepted by parse-statement
- [observable] an example statement or example id in a tool description, a DESCRIBE result, an error message or a note has the ExampleKey, or the key of the product that the statement is about; the ExampleKey is the product of the setting default_product or a key that no product of the account has
- [observable] a statement that the strict reading accepts has the same result with and without the second reading
- [observable] in one reading of a list every page is requested once, with increasing page numbers, and the reading ends
- [observable] one statement requests at most max_pages pages of one list and returns at most max_limit rows
- [observable] an aggregate is computed over all matching records and never over a list that was cut
- [observable] a statement that fails is answered with a tool result that is marked as an error and has the message as its text; it is not answered with a JSON-RPC error
- [observable] manual mode writes to standard output the same text that the MCP tool returns for the same statement and the same Account
- [observable] the credential is in no tool result, no message, no note, no log line, no start message, no warning and no output of manual mode, and no option of the command line takes it
- [observable] every request of the server is for https://<subdomain>.aha.io/api/v1 with the subdomain of the Account of the call; the server follows no redirect, and the only other host that it connects to is the HTTPS proxy of the environment
- [observable] the settings of the server come from its arguments, from the key file and from the variables AHA_SUBDOMAIN, AHA_API_KEY, AHA_DEFAULT_PRODUCT and AHA_READ_ONLY, and no other variable changes a setting
- [observable] an option takes precedence over its environment variable, and the key file over AHA_API_KEY
- [observable] in http mode a request without a credential reaches no tool and no tool list, and the server uses no credential of its own
- [observable] in http mode nothing that was read with the credential of one request is part of the answer to another request
- [observable] one statement reads the product list, the custom field definitions, the options of one definition, and the record and the workflows of one product at most once each; the server keeps none of them after the statement, in stdio mode and in http mode, and the ExampleKey is the one value that it keeps between statements
- [observable] a read statement or a write statement whose field words are all catalog names causes no request for custom field definitions, and its list requests do not ask for custom fields; the one exception is a read statement with `*` that selects one record by its id field, which reads the definitions when it has found the record
- [observable] a result has a custom field only when the statement names it, or when the statement has `*` and selects one record by its id field: then the row also has the custom fields that have a value in that record; in every other statement `*` stands for catalog fields only
- [observable] a custom field is set with UPDATE only; INSERT with a custom field is an error and creates no record
- [observable] a list that Aha! returns without the custom fields of the request is an error of the statement; no result shows a custom field as empty for that reason
- [implementation] the certificate for the host of Aha! is checked against the trust store on every connection, also inside the tunnel of a proxy, and no setting turns the check off
- [implementation] in http mode the server keeps no credential and no data of Aha! after a request is answered
- [implementation] the code of the server reads the variables AHA_SUBDOMAIN, AHA_API_KEY, AHA_DEFAULT_PRODUCT and AHA_READ_ONLY and no other; the standard library of the language reads HTTPS_PROXY, https_proxy, NO_PROXY and no_proxy for the way to Aha!, and SSL_CERT_FILE and SSL_CERT_DIR for the trust store
- [implementation] the server has no direct dependency other than the MCP framework that the deployment template selects
- [implementation] the entity-catalog is the single source for name resolution, DESCRIBE, the parameter fields, record conversion, value checks, request bodies and the lists in messages; the server has no second list of entities or fields, and it has no built-in list of custom fields: these come from the definitions of the account
- [implementation] no list is read with one request per record

## EXAMPLES

The examples use FakeAha with the fixture account of the INTERFACES
section, and every example starts from the unchanged fixture account. The
INTERFACES section also says in which mode the server runs unless an
example says otherwise. A tool result is shown as it is returned, and the
order of its members is part of the expectation. A statement, a command
line and a message are shown between backticks, which are not part of
them. "No write request" means that FakeAha recorded no ApiCreate,
ApiUpdate and ApiDelete call. `stdout` is the standard output and `stderr`
the standard error of the process.

### EXAMPLE: query_result_envelope (of: query)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-3","count":3,"records":[{"id":"DEMO-3","name":"Live dashboard","status":"In development"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration"},{"id":"DEMO-5","name":"Rider's handbook","status":"Shipped"}]}`
  FakeAha recorded exactly one call: ApiList on /releases/DEMO-R-3/features, page 1

### EXAMPLE: query_rejects_product_parameter (of: query)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT id FROM features` and with a
  second parameter product = `DEMO`
THEN:
  the tool result is an error with the message
  `tool query has one parameter: query (one AQL statement). It does not accept: product. Name the product inside the statement: ... WHERE product = 'PROD'`
  FakeAha recorded no call

### EXAMPLE: query_requires_a_statement (of: query)
GIVEN:
  the fixture account
WHEN:
  the tool query is called without parameters
THEN:
  the tool result is an error with the message
  `query parameter is required: one AQL statement. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
WHEN:
  the tool query is called with a query that has three spaces and nothing else
THEN:
  the tool result is the same error

### EXAMPLE: query_refuses_write_statements (of: query)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `UPDATE features SET name = 'X' WHERE id = 'DEMO-1'`
THEN:
  the tool result is an error with the message
  `query only reads. Use the execute tool for INSERT, UPDATE and DELETE.`
  FakeAha recorded no call

### EXAMPLE: execute_result_envelope (of: execute)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO features (parent, name) VALUES ('DEMO-R-3', 'Export to PDF')`
THEN:
  the tool result is
  `{"id":"DEMO-7","name":"Export to PDF","parent":"DEMO-R-3","message":"Feature DEMO-7 created"}`
  FakeAha recorded exactly one call: ApiCreate on /releases/DEMO-R-3/features
  with the body `{"feature":{"name":"Export to PDF"}}`

### EXAMPLE: execute_rejects_unknown_parameter (of: execute)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called without the parameter statement and with a
  parameter query = `DELETE FROM features WHERE id = 'DEMO-6'`
THEN:
  the tool result is an error with the message
  `tool execute has one parameter: statement (one AQL statement). It does not accept: query. Pass the statement as statement.`
  FakeAha recorded no call

### EXAMPLE: execute_refuses_read_statements (of: execute)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `SELECT id FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is an error with the message
  `execute only changes data. Use the query tool for SELECT and DESCRIBE.`
  FakeAha recorded no call

### EXAMPLE: execute_writes_a_log_line (of: execute)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'shipped' WHERE id = 'demo-1'`
THEN:
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","status":"Shipped","message":"Feature DEMO-1 status changed to 'Shipped'"}`
  the diagnostic output has exactly one new line, and it has the form
  `<time> execute account=example op=update entity=features id=DEMO-1 result=ok`
  `<time>` has the form YYYY-MM-DDThh:mm:ssZ; it is not before the second
  in which the call started and not after the second in which it ended,
  both in UTC

### EXAMPLE: execute_logs_a_refused_statement (of: execute)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `DELETE FROM features WHERE name = 'CSV export'`
THEN:
  the tool result is an error whose message starts with
  `one statement changes one record: the WHERE must start with id = '...'.`
  the diagnostic output has exactly one new line, and it has the form
  `<time> execute account=example op=none entity=none id=none result=syntax`
  `<time>` has the form YYYY-MM-DDThh:mm:ssZ
  the diagnostic output does not contain `CSV export` and does not contain `fixture-key`
  FakeAha recorded no call

### EXAMPLE: tools_without_read_only_mode (of: tool-registration)
GIVEN:
  the setting read_only is false
WHEN:
  tool-registration runs and an MCP client lists the tools
THEN:
  the list has `query`, and its only parameter is `query`
  the list has `execute`, and its only parameter is `statement`
  the list has no other tool
  `query` is marked as a tool that only reads, and `execute` as a tool that
  can delete
  the description of `query` has the line
  `id is the reference number of a record, for example 'PROD-23'.`
  the description of `query` has the line
  `Custom fields are written custom.key. DESCRIBE entity lists them.`
  between the line that starts with `Other conditions:` and the line that
  starts with `Text goes in single quotes.`

### EXAMPLE: tools_in_read_only_mode (of: tool-registration)
GIVEN:
  the setting read_only is true
WHEN:
  tool-registration runs and an MCP client lists the tools
THEN:
  the list has `query`
  the list has no other tool

### EXAMPLE: manual_query_prints_the_tool_result (of: manual-mode)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql query query="SELECT id, name FROM features WHERE id = 'demo-1'"`
THEN:
  stdout is the line
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","name":"CSV export"}]}`
  stderr is empty
  exit_code = 0
  FakeAha recorded exactly one call: ApiGet on /features/DEMO-1
  the tool query returns the same text for the same statement in stdio mode

### EXAMPLE: manual_statement_from_standard_input (of: manual-mode)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
  standard input has the text `SELECT email FROM me` and a line break
WHEN:
  manual-mode runs for the command line `mcp-server-aha-aql query query=-`
THEN:
  stdout is the line
  `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`
  exit_code = 0

### EXAMPLE: manual_dry_run_sends_no_write_request (of: manual-mode)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET status = 'shipped' WHERE id = 'DEMO-1'"`
THEN:
  stdout is the line
  `{"id":"DEMO-1","status":"Shipped","message":"would update feature DEMO-1"}`
  stderr is empty
  exit_code = 0
  FakeAha recorded one ApiGet call on /features/DEMO-1
  no write request
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET status = 'done' WHERE id = 'DEMO-1'"`
THEN:
  stdout is empty
  stderr contains the line
  `error: workflow status 'done' not found for feature; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement`
  exit_code = 1
  no write request

### EXAMPLE: manual_error_goes_to_standard_error (of: manual-mode)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql query query="SELECT id, name FROM releases"`
THEN:
  stdout is empty
  stderr contains the line
  `error: releases are listed per product: add product = '...' to WHERE. Example: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`
  exit_code = 1

### EXAMPLE: manual_statement_argument_is_missing (of: manual-mode)
GIVEN:
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  manual-mode runs for the command line `mcp-server-aha-aql query`
THEN:
  stderr contains the line
  `error: query needs query=STATEMENT, or query=- to read the statement from standard input`
  exit_code = 2
  FakeAha recorded no call

### EXAMPLE: manual_execute_is_refused_in_read_only (of: manual-mode)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql execute read-only=true statement="DELETE FROM features WHERE id = 'DEMO-6'"`
THEN:
  stderr contains the line
  `error: execute is not available, because read-only is true (option read-only or variable AHA_READ_ONLY)`
  exit_code = 2
  FakeAha recorded no call
WHEN:
  the environment also has AHA_READ_ONLY = `true`
  and manual-mode runs for the command line
  `mcp-server-aha-aql execute statement="DELETE FROM features WHERE id = 'DEMO-6'"`
THEN:
  stderr contains the same line
  exit_code = 2
  FakeAha recorded no call

### EXAMPLE: account_from_the_environment (of: load-configuration)
GIVEN:
  the environment has AHA_SUBDOMAIN = `Example` and AHA_API_KEY = `fixture-key`
WHEN:
  load-configuration runs for the command line `mcp-server-aha-aql stdio`
THEN:
  the Account has the subdomain `example` and the credential `fixture-key`
  read_only is false and default_product has no value
  no request was sent

### EXAMPLE: arguments_take_precedence (of: load-configuration)
GIVEN:
  the environment has AHA_SUBDOMAIN = `other`, AHA_API_KEY = `environment-key`,
  AHA_DEFAULT_PRODUCT = `OTHER` and AHA_READ_ONLY = `false`
  the file /run/user/1000/aha-key has the content `fixture-key` and a line
  break, and only its owner can read it
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql stdio subdomain=example api-key-file=/run/user/1000/aha-key default-product=DEMO read-only=true`
THEN:
  the Account has the subdomain `example` and the credential `fixture-key`
  default_product is `DEMO` and read_only is true

### EXAMPLE: settings_from_the_environment (of: load-configuration)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example`, AHA_API_KEY = `fixture-key`,
  AHA_DEFAULT_PRODUCT = `demo` and AHA_READ_ONLY = `true`
WHEN:
  load-configuration runs for the command line `mcp-server-aha-aql stdio`
  and an MCP client lists the tools
THEN:
  the Account has the subdomain `example` and the credential `fixture-key`
  default_product is `demo` and read_only is true
  the tool list has `query` and no other tool
  the description of `query` has the line
  `id is the reference number of a record, for example 'DEMO-23'.`

### EXAMPLE: read_only_variable_of_the_wrong_form (of: load-configuration)
GIVEN:
  the environment has AHA_SUBDOMAIN = `example`, AHA_API_KEY = `fixture-key`
  and AHA_READ_ONLY = `1`
WHEN:
  load-configuration runs for the command line `mcp-server-aha-aql stdio`
THEN:
  stderr contains the line `error: AHA_READ_ONLY must be true or false`
  exit_code = 2
  no request was sent

### EXAMPLE: start_without_an_account (of: load-configuration)
GIVEN:
  the environment has neither AHA_SUBDOMAIN nor AHA_API_KEY
WHEN:
  load-configuration runs for the command line `mcp-server-aha-aql stdio`
THEN:
  stderr contains the line
  `error: no Aha! account. Use subdomain=NAME or set AHA_SUBDOMAIN`
  exit_code = 2
  no request was sent
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql stdio subdomain=example`
THEN:
  stderr contains the line
  `error: no credential. Use api-key-file=PATH or set AHA_API_KEY`
  exit_code = 2

### EXAMPLE: key_file_that_other_users_can_read (of: load-configuration)
GIVEN:
  the file /home/user/aha-key has the content `fixture-key`, and its
  permission bits are 0644
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql stdio subdomain=example api-key-file=/home/user/aha-key`
THEN:
  stderr contains the line
  `error: /home/user/aha-key can be read by other users. Run: chmod 600 /home/user/aha-key`
  stderr does not contain `fixture-key`
  exit_code = 2

### EXAMPLE: credential_is_not_an_option (of: load-configuration)
GIVEN:
  the environment has AHA_SUBDOMAIN = `example` and no AHA_API_KEY
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql stdio api-key=fixture-key`
THEN:
  stderr contains the line
  `error: unknown option 'api-key' for stdio. Options: subdomain, api-key-file, default-product, read-only`
  stderr does not contain `fixture-key`
  exit_code = 2

### EXAMPLE: values_of_the_wrong_form (of: load-configuration)
GIVEN:
  the environment has AHA_API_KEY = `fixture-key`
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql stdio subdomain=https://example.aha.io`
THEN:
  stderr contains the line
  `error: subdomain must be the account name before .aha.io, for example mycompany`
  exit_code = 2
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql stdio subdomain=example read-only=yes`
THEN:
  stderr contains the line `error: read-only must be true or false`
  exit_code = 2

### EXAMPLE: http_mode_uses_no_credential_of_the_process (of: load-configuration)
GIVEN:
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  load-configuration runs for the command line `mcp-server-aha-aql http`
THEN:
  load-configuration returns settings with the subdomain `example`, and no
  account
  stderr contains the line
  `warning: AHA_API_KEY is not read in http mode. Each request brings its credential in the header Aha-Api-Key`
  stderr does not contain `fixture-key`
  the server starts and sends no request to Aha!
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql http api-key-file=/home/user/aha-key`
THEN:
  stderr contains the line
  `error: http mode has no option api-key-file. Each request brings its credential in the header Aha-Api-Key`
  exit_code = 2
WHEN:
  the environment has neither AHA_SUBDOMAIN nor AHA_API_KEY
  and load-configuration runs for the command line
  `mcp-server-aha-aql http listen=127.0.0.1:8080`
THEN:
  load-configuration returns settings without a subdomain, and no account
  the diagnostic output has no warning

### EXAMPLE: listen_address_that_is_not_loopback (of: load-configuration)
GIVEN:
  the environment has AHA_SUBDOMAIN = `example`
  192.0.2.1 is an address for documentation, which the machine of the
  test does not have
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql http listen=192.0.2.1:8080`
THEN:
  stderr contains the line
  `warning: the listen address is not a loopback address. Every request has a credential: put a TLS endpoint in front of the server`
  FakeAha recorded no call
WHEN:
  load-configuration runs for the command line
  `mcp-server-aha-aql http listen=127.0.0.1:8080`
THEN:
  stderr has no line that starts with `warning:`

### EXAMPLE: unknown_first_word (of: load-configuration)
GIVEN:
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  load-configuration runs for the command line `mcp-server-aha-aql websocket`
THEN:
  stderr contains the line
  `error: unknown transport 'websocket'. Valid: stdio, http, query, execute`
  exit_code = 2
  no request was sent
WHEN:
  load-configuration runs for the command line `mcp-server-aha-aql`
THEN:
  stderr contains the line
  `error: unknown transport ''. Valid: stdio, http, query, execute`
  exit_code = 2

### EXAMPLE: http_request_brings_its_account (of: request-account)
GIVEN:
  the fixture account
  the server runs in http mode, and the setting subdomain has no value
WHEN:
  an MCP client calls the tool query with query = `SELECT email FROM me`
  and with the headers `Aha-Api-Key: fixture-key` and
  `Aha-Subdomain: example`, and request-account gives the Account
THEN:
  the tool result is
  `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`
  FakeAha recorded exactly one call: GET https://example.aha.io/api/v1/me
  with the header `Authorization: Bearer fixture-key`

### EXAMPLE: http_request_without_a_credential (of: request-account)
GIVEN:
  the server runs in http mode with the setting subdomain = `example`
WHEN:
  an MCP client asks for the tool list without the header Aha-Api-Key,
  and request-account checks the request
THEN:
  the answer has the status 400
  its body is a JSON-RPC error with the message
  `no credential. Send the header Aha-Api-Key with an Aha! API key`
  the answer has no tool list
  FakeAha recorded no call

### EXAMPLE: http_request_for_another_account (of: request-account)
GIVEN:
  the server runs in http mode with the setting subdomain = `example`
WHEN:
  an MCP client calls the tool query with the headers
  `Aha-Api-Key: fixture-key` and `Aha-Subdomain: other`,
  and request-account checks the request
THEN:
  the answer has the status 403
  its body is a JSON-RPC error with the message
  `this server works with one Aha! account only`
  FakeAha recorded no call

### EXAMPLE: http_request_from_a_web_page (of: request-account)
GIVEN:
  the server runs in http mode with the setting subdomain = `example`
WHEN:
  a request arrives with the headers `Origin: https://pages.example` and
  `Aha-Api-Key: fixture-key`, and request-account checks the request
THEN:
  the answer has the status 403
  its body is a JSON-RPC error with the message
  `requests from a web page are not accepted`
  FakeAha recorded no call

### EXAMPLE: http_requests_share_nothing (of: request-account)
GIVEN:
  the fixture account
  the server runs in http mode with the setting subdomain = `example`
WHEN:
  an MCP client calls the tool query with query = `SELECT key FROM products`
  and with the header `Aha-Api-Key: fixture-key`
THEN:
  the tool result is
  `{"entity":"products","scope":"all products","count":2,"records":[{"key":"DEMO"},{"key":"OTHER"}]}`
WHEN:
  a second MCP client calls the tool query with the same statement and with
  the header `Aha-Api-Key: other-key`, and request-account gives
  the Account of that request
THEN:
  the tool result is an error with the message
  `Aha! refused the request (status 403): the API key is not valid for the account example, or it has no permission for this record. Repeating the statement does not help.`
  FakeAha recorded an ApiList call on /products for each of the two requests

### EXAMPLE: count_per_status (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"In design","COUNT(*)":1},{"status":"Under consideration","COUNT(*)":3},{"status":"In development","COUNT(*)":1},{"status":"Shipped","COUNT(*)":1}]}`

### EXAMPLE: star_shows_default_fields (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT * FROM features WHERE product = 'DEMO' AND status = 'under consideration'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":3,"records":[{"id":"DEMO-2","name":"Offline maps","status":"Under consideration","release":"DEMO-R-2"},{"id":"DEMO-4","name":"Live video feed","status":"Under consideration","release":"DEMO-R-3"},{"id":"DEMO-6","name":"Route planner","status":"Under consideration","release":"DEMO-R-1"}],"note":"* shows id, name, status, release. Other fields of features: initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url."}`

### EXAMPLE: record_by_id_has_all_fields (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE id = 'demo-3'`
  and read-statement runs it
THEN:
  `scope` is `feature DEMO-3` and `count` is 1
  the row has the 17 fields of features in catalog order, and after them
  the six custom fields that have a value in DEMO-3
  id is `DEMO-3`, name is `Live dashboard`, status is `In development`,
  release is `DEMO-R-3`, initiative is `DEMO-S-1` and product is `DEMO`
  epic and assigned_to are null
  the result has no `note`
  FakeAha recorded one ApiGet call on /features/DEMO-3, one ApiList call on
  /products, one ApiList call on /custom_field_definitions and no other call

### EXAMPLE: field_without_value_is_null (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id, assigned_to FROM features WHERE release = 'DEMO-R-2'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","assigned_to":"test.user@example.com"},{"id":"DEMO-2","assigned_to":null}]}`

### EXAMPLE: label_is_the_word_of_the_statement (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT ref_num, title FROM features WHERE release_id = 'DEMO-R-2'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"ref_num":"DEMO-1","title":"CSV export"},{"ref_num":"DEMO-2","title":"Offline maps"}]}`

### EXAMPLE: name_equality_and_search (of: read-statement)
GIVEN:
  the fixture account, in which two feature names start with "Live"
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name = 'live'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":0,"records":[],"note":"No feature has exactly the name 'live'. name = compares the whole name; to search inside names use: name CONTAINS 'live'"}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name CONTAINS 'live'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name = 'offline MAPS'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-2"}]}`

### EXAMPLE: unknown_id_gives_an_empty_result (of: read-statement)
GIVEN:
  the fixture account, which has no feature DEMO-99
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE id = 'DEMO-99'`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-99","count":0,"records":[],"note":"There is no feature with the id 'DEMO-99'."}`
  FakeAha recorded no call on /custom_field_definitions

### EXAMPLE: record_that_exists_and_does_not_match (of: read-statement)
GIVEN:
  the fixture account, in which DEMO-1 has the status "In design"
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE id = 'demo-1' AND status = 'Shipped'`
  and read-statement explains the empty result
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":0,"records":[],"note":"The feature DEMO-1 exists, and another condition of the statement is not true for it. To see the record: SELECT * FROM features WHERE id = 'DEMO-1'"}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE id = 'DEMO-1' AND id = 'DEMO-2'`
THEN:
  the tool result has the same `note` and no other note, because the id
  route was read for DEMO-1 only
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE id = 'DEMO-1' LIMIT 0`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":0,"total":1,"records":[]}`
  it has no `note`, because the record was kept and LIMIT cut the row

### EXAMPLE: statuses_of_an_entity_that_has_none (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'comments'`
  and read-statement explains the empty result
THEN:
  the tool result is
  `{"entity":"statuses","scope":"product DEMO","count":0,"records":[],"note":"DEMO has statuses for: releases, features, epics, requirements, initiatives, goals, ideas."}`

### EXAMPLE: default_limit_is_reported (of: read-statement)
GIVEN:
  the fixture account and, in addition, a product BULK with one release
  BULK-R-1 and 60 features BULK-1 to BULK-60 in it
WHEN:
  the tool query is called with query =
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
  the tool query is called with query = `SELECT id, name FROM releases`
  and read-statement gets the error of plan-fetch
THEN:
  the tool result is an error with the message
  `releases are listed per product: add product = '...' to WHERE. Example: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`

### EXAMPLE: star_with_an_id_shows_custom_values (of: read-statement)
GIVEN:
  the fixture account, in which DEMO-1 has custom values
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE id = 'DEMO-1'`
  and read-statement adds the star custom fields
THEN:
  the row has the 17 fields of features in catalog order
  after them it has exactly these labels and values, in this order:
  `custom.customer` with `Acme`, `custom.effort_days` with 12.5,
  `custom.go_live` with `2026-04-15`, `custom.markets` with `EMEA, APAC`,
  `custom.priority` with `High`, `custom.release_notes` with the JSON
  string `"Adds the export.\nSee the handbook."`, `custom.reviewers` with
  `second.user@example.com`, and `custom.value_score` with 31
  the row has no label `custom.platforms`, because DEMO-1 has no value there
  the row has no label `custom.business_case`, because the server does not
  read a field of that type
  FakeAha recorded one ApiList call on /custom_field_definitions
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE id = 'DEMO-2'`
THEN:
  after the 17 fields the row has `custom.effort_days` with 8 and
  `custom.priority` with `Medium`, and no other label
WHEN:
  the tool query is called with query =
  `SELECT *, custom.platforms FROM features WHERE id = 'DEMO-2'`
THEN:
  the row has the 17 fields, then `custom.platforms` with null, then
  `custom.effort_days` with 8 and `custom.priority` with `Medium`
WHEN:
  the tool query is called with query =
  `SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`

### EXAMPLE: star_without_an_id_has_catalog_fields_only (of: read-statement)
GIVEN:
  the fixture account, in which the features of DEMO-R-2 and the product
  DEMO have custom values
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE release = 'DEMO-R-2'`
  and read-statement replaces the Star item
THEN:
  each of the two rows has the labels id, name, status and release, and no
  other label
  `note` starts with `* shows id, name, status, release.`
  FakeAha recorded no call on /custom_field_definitions, and the parameter
  `fields` of the list request does not name `custom_fields`
WHEN:
  the tool query is called with query = `SELECT * FROM products`
THEN:
  each of the two rows has the labels key and name, and no other label
  FakeAha recorded no call on /custom_field_definitions and no ApiGet call
  on a product

### EXAMPLE: star_shows_the_custom_values_of_a_product (of: read-statement)
GIVEN:
  the fixture account, in which the product DEMO has the cost_center
  CC-100 and the product OTHER has no custom value
WHEN:
  the tool query is called with query = `SELECT * FROM products WHERE key = 'DEMO'`
  and read-statement reads the product for its star custom fields
THEN:
  the row has the 7 fields of products in catalog order, and after them
  `custom.cost_center` with `CC-100`, and no other label
  FakeAha recorded one ApiList call on /custom_field_definitions and one
  ApiGet call on /products/7000000000000000001
WHEN:
  the tool query is called with query = `SELECT * FROM products WHERE key = 'OTHER'`
THEN:
  the row has the 7 fields of products and no other label
WHEN:
  FakeAha answers every ApiList call on /custom_field_definitions with
  the status 403, and the tool query is called with query =
  `SELECT * FROM products WHERE key = 'DEMO'`
THEN:
  the row has the 7 fields of products and no other label
  FakeAha recorded no ApiGet call on a product for this statement

### EXAMPLE: custom_fields_in_group_by_and_order_by (of: read-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT custom.priority, COUNT(*) FROM features WHERE product = 'DEMO' AND custom.priority IS NOT NULL GROUP BY custom.priority`
  and read-statement runs it
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"custom.priority":"High","COUNT(*)":2},{"custom.priority":"Medium","COUNT(*)":1}]}`
WHEN:
  the tool query is called with query =
  `SELECT SUM(custom.effort_days) FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"SUM(custom.effort_days)":23.5}]}`
WHEN:
  the tool query is called with query =
  `SELECT id, custom.effort_days FROM features WHERE product = 'DEMO' ORDER BY custom.effort_days DESC LIMIT 2`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"total":6,"records":[{"id":"DEMO-1","custom.effort_days":12.5},{"id":"DEMO-2","custom.effort_days":8}]}`

### EXAMPLE: describe_one_entity (of: describe-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `DESCRIBE features`
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
  `custom_fields` follows `fields` and has 9 objects with the names
  custom.customer, custom.effort_days, custom.go_live, custom.markets,
  custom.platforms, custom.priority, custom.release_notes, custom.reviewers,
  custom.value_score, in this order
  the object of custom.priority is
  `{"name":"custom.priority","label":"Priority","type":"one option of a list","writable":true}`
  the object of custom.reviewers is
  `{"name":"custom.reviewers","label":"Reviewers","type":"users (e-mail addresses), separated by commas","writable":true}`
  the object of custom.value_score is
  `{"name":"custom.value_score","label":"Value score","type":"number"}`
  no object has the name custom.business_case
  `custom_fields_note` is
  `* shows custom fields only with id = '...', and only those that have a value. To select one by name: SELECT id, custom.customer FROM features WHERE id = 'PROD-23'`
  `filter_by_parent` is `["product = '...'","release = '...'","initiative = '...'","epic = '...'"]`
  `status_values` is
  `SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'`
  `change_with_execute` is
  `["INSERT INTO features (parent, ...) VALUES (...): parent is the id of a release","UPDATE features SET field = 'value' WHERE id = '...'","DELETE FROM features WHERE id = '...'"]`
  `example` is `SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
  FakeAha recorded exactly one call: ApiList on /custom_field_definitions

### EXAMPLE: describe_when_aha_cannot_be_reached (of: describe-statement)
GIVEN:
  the fixture account
  FakeAha gives no answer to an ApiList call on /custom_field_definitions
WHEN:
  the tool query is called with query = `DESCRIBE features`
  and describe-statement gets the error of custom-fields
THEN:
  the tool result is an error with the message
  `cannot reach Aha!: no answer within 60 seconds.`
WHEN:
  the tool query is called with query = `DESCRIBE comments`
THEN:
  `entity` is `comments`, and the result has no member `custom_fields`
  FakeAha recorded no call for this statement

### EXAMPLE: describe_without_entity (of: describe-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `DESCRIBE`
  and describe-statement answers it
THEN:
  `entities` has 12 objects, one per catalog entity in catalog order
  the first object is
  `{"entity":"products","about":"Products (workspaces). The key of a product, for example PROD, starts the id of its records."}`
  `note` is
  `DESCRIBE entity lists the fields of one entity, for example: DESCRIBE features`
  every object has exactly the members `entity` and `about`
  FakeAha recorded no call

### EXAMPLE: describe_shows_a_field_that_is_set_in_insert_only (of: describe-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `DESCRIBE goals`
  and describe-statement answers it
THEN:
  `about` is `Goals of a product. A new goal needs a success metric.`
  the object of name in `fields` is
  `{"name":"name","type":"string","writable":true}`
  the object of success_metric in `fields` is
  `{"name":"success_metric","type":"string","writable":"INSERT only"}`
  the object of product in `fields` is `{"name":"product","type":"product key"}`

### EXAMPLE: insert_with_other_spellings (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO features (release_id, title, assignee) VALUES ('DEMO-R-2', 'Dark mode', 'me')`
  and insert-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-7","name":"Dark mode","parent":"DEMO-R-2","message":"Feature DEMO-7 created"}`
  FakeAha recorded one ApiCreate call on /releases/DEMO-R-2/features with
  the body `{"feature":{"name":"Dark mode","assigned_to_user":"test.user@example.com"}}`

### EXAMPLE: insert_comment_with_apostrophe (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO comments (parent, body) VALUES ('DEMO-3', 'It''s ready.')`
  and insert-statement runs it
THEN:
  FakeAha recorded one ApiCreate call on /features/DEMO-3/comments with
  the body `{"comment":{"body":"It's ready."}}`
  the tool result has, in this order, `id` with the NumericId of the new
  comment, `parent` with `DEMO-3`, `body` with `It's ready.`, `created_at`
  with a timestamp, and `message` with `Comment added to DEMO-3`

### EXAMPLE: insert_without_parent (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
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
  the tool execute is called with statement =
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
  the tool execute is called with statement =
  `INSERT INTO features (parent, name) VALUES ('DEMO-R-99', 'Dark mode')`
  and insert-statement runs it
THEN:
  the tool result is an error with the message
  `there is no release with the id 'DEMO-R-99'. To find the id, use the query tool: SELECT id, name FROM releases WHERE product = 'DEMO'`
  the account has no new feature

### EXAMPLE: insert_goal_without_a_success_metric (of: insert-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO goals (parent, name) VALUES ('DEMO', 'Win the cup')`
  and insert-statement finds no success metric
THEN:
  the tool result is an error with the message
  `success_metric is required to create a goal`
  FakeAha recorded no ApiCreate call
WHEN:
  the tool execute is called with statement =
  `INSERT INTO goals (parent, name, success_metric) VALUES ('DEMO', 'Win the cup', '')`
THEN:
  the tool result is an error with the same message
  FakeAha recorded no ApiCreate call

### EXAMPLE: update_status_in_any_case (of: update-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'in design' WHERE id = 'demo-2'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-2","name":"Offline maps","status":"In design","message":"Feature DEMO-2 status changed to 'In design'"}`
  FakeAha recorded one ApiGet call on /features/DEMO-2 and one ApiUpdate
  call on /features/DEMO-2 with the body `{"feature":{"workflow_status":"In design"}}`
  FakeAha recorded no ApiList call on a feature list

### EXAMPLE: update_assigned_to_me (of: update-statement)
GIVEN:
  the fixture account, in which the current user is test.user@example.com
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-2'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-2","name":"Offline maps","assigned_to":"test.user@example.com","message":"Feature DEMO-2 updated"}`

### EXAMPLE: update_with_a_true_guard (of: update-statement)
GIVEN:
  the fixture account, in which DEMO-3 has the status In development
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-3' AND status = 'in development'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-3","name":"Live dashboard","status":"Shipped","message":"Feature DEMO-3 status changed to 'Shipped'"}`

### EXAMPLE: update_with_a_false_guard (of: update-statement)
GIVEN:
  the fixture account, in which DEMO-2 has the name Offline maps
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-2' AND name = 'CSV export'`
  and update-statement runs it
THEN:
  the tool result is an error with the message
  `nothing was changed: feature DEMO-2 has name = 'Offline maps', not 'CSV export'`
  FakeAha recorded one ApiGet call on /features/DEMO-2
  no write request

### EXAMPLE: update_with_a_guard_for_no_value (of: update-statement)
GIVEN:
  the fixture account, in which DEMO-2 has no assignee and DEMO-1 is
  assigned to test.user@example.com
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-2' AND assigned_to = ''`
  and update-statement checks the guard for no value
THEN:
  the tool result is
  `{"id":"DEMO-2","name":"Offline maps","assigned_to":"test.user@example.com","message":"Feature DEMO-2 updated"}`
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET assigned_to = 'me' WHERE id = 'DEMO-1' AND assigned_to = ''`
THEN:
  the tool result is an error with the message
  `nothing was changed: feature DEMO-1 has assigned_to = 'test.user@example.com', not ''`
  no write request

### EXAMPLE: update_of_an_unknown_record (of: update-statement)
GIVEN:
  the fixture account, which has no feature DEMO-99
WHEN:
  the tool execute is called with statement =
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
  the tool execute is called with statement =
  `UPDATE features SET description = '' WHERE id = 'DEMO-1'`
  and update-statement runs it
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-1 with the body
  `{"feature":{"description":""}}`
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","description":null,"message":"Feature DEMO-1 updated"}`

### EXAMPLE: update_with_a_date_without_quotes (of: update-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET due_date = 2026-12-01 WHERE id = 'DEMO-1'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","due_date":"2026-12-01","message":"Feature DEMO-1 updated"}`

### EXAMPLE: update_of_the_name_shows_it_once (of: update-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET name = 'CSV export 2', due_date = '2026-05-01' WHERE id = 'DEMO-1'`
  and update-statement returns the WriteResult
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-1 with the body
  `{"feature":{"name":"CSV export 2","due_date":"2026-05-01"}}`
  the tool result is
  `{"id":"DEMO-1","name":"CSV export 2","due_date":"2026-05-01","message":"Feature DEMO-1 updated"}`
  the object of the result has the member `name` once

### EXAMPLE: update_sets_a_custom_field (of: update-statement)
GIVEN:
  the fixture account, in which DEMO-2 has the priority Medium
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.priority = 'low' WHERE id = 'demo-2'`
  and update-statement runs it
THEN:
  the tool result is
  `{"id":"DEMO-2","name":"Offline maps","custom.priority":"Low","message":"Feature DEMO-2 updated"}`
  FakeAha recorded these calls and no other: ApiList on
  /custom_field_definitions, ApiGet on /features/DEMO-2, ApiGet on
  /custom_field_definitions/7000000000000000801/options, and ApiUpdate on
  /features/DEMO-2 with the body
  `{"feature":{"custom_fields":{"priority":"Low"}}}`

### EXAMPLE: update_with_a_custom_guard (of: update-statement)
GIVEN:
  the fixture account, in which DEMO-1 has the priority High
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-1' AND custom.priority = 'Low'`
  and update-statement checks the guard
THEN:
  the tool result is an error with the message
  `nothing was changed: feature DEMO-1 has custom.priority = 'High', not 'Low'`
  no write request
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-1' AND custom.priority = 'high'`
THEN:
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","status":"Shipped","message":"Feature DEMO-1 status changed to 'Shipped'"}`

### EXAMPLE: dry_run_shows_a_list_as_text (of: update-statement)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET custom.markets = 'EMEA, LATAM' WHERE id = 'DEMO-1'"`
  and update-statement returns the checked value
THEN:
  stdout is the line
  `{"id":"DEMO-1","custom.markets":"EMEA, LATAM","message":"would update feature DEMO-1"}`
  exit_code = 0
  no write request

### EXAMPLE: delete_one_feature (of: delete-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement = `DELETE FROM features WHERE id = 'DEMO-6'`
  and delete-statement runs it
THEN:
  the tool result is `{"id":"DEMO-6","message":"Feature DEMO-6 deleted"}`
  FakeAha recorded one ApiGet call on /features/DEMO-6 and one ApiDelete
  call on /features/DEMO-6

### EXAMPLE: delete_a_release_through_its_product (of: delete-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement = `DELETE FROM releases WHERE id = 'DEMO-R-4'`
  and delete-statement runs it
THEN:
  the tool result is `{"id":"DEMO-R-4","message":"Release DEMO-R-4 deleted"}`
  FakeAha recorded one ApiDelete call on
  /products/7000000000000000001/releases/DEMO-R-4

### EXAMPLE: delete_with_a_false_guard (of: delete-statement)
GIVEN:
  the fixture account, in which DEMO-6 has the status Under consideration
WHEN:
  the tool execute is called with statement =
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
  the tool execute is called with statement = `DELETE FROM features WHERE id = 'DEMO-99'`
  and delete-statement runs it
THEN:
  the tool result is an error with the message
  `there is no feature with the id 'DEMO-99'`
  no write request

### EXAMPLE: delete_with_a_custom_guard (of: delete-statement)
GIVEN:
  the fixture account, in which DEMO-2 has the priority Medium
WHEN:
  the tool execute is called with statement =
  `DELETE FROM features WHERE id = 'DEMO-2' AND custom.priority = 'High'`
  and delete-statement checks the guard
THEN:
  the tool result is an error with the message
  `nothing was changed: feature DEMO-2 has custom.priority = 'Medium', not 'High'`
  no write request
WHEN:
  the tool execute is called with statement =
  `DELETE FROM features WHERE id = 'DEMO-2' AND custom.priority = 'medium'`
THEN:
  the tool result is `{"id":"DEMO-2","message":"Feature DEMO-2 deleted"}`

### EXAMPLE: doubled_apostrophe_in_text (of: tokenize)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND name = 'Rider''s handbook'`
  and tokenize reads the doubled apostrophe as one apostrophe
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-5"}]}`

### EXAMPLE: date_without_quotes (of: tokenize)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND release_date > 2026-06-01`
  and tokenize reads the date as a text
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":2,"records":[{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`

### EXAMPLE: text_that_is_not_closed (of: tokenize)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
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
  the tool query is called with query = `SELECT id FROM features WHERE name ~ 'x'`
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
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND name = 'Rider's cup release'`
  and parse-statement reads it a second time after the strict reading failed
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-4"}]}`

### EXAMPLE: strict_reading_is_kept (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
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
  the tool query is called with query = `FROM releases WHERE product = 'DEMO'`
  and parse-statement reads it as `SELECT * FROM releases WHERE product = 'DEMO'`
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","name":"What's next","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","name":"Spring release","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","name":"Summer release","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","name":"Rider's cup release","release_date":"2026-12-15","parking_lot":false}],"note":"* shows id, name, release_date, parking_lot. Other fields of releases: start_date, status, released, product, progress, description, created_at, url."}`
WHEN:
  the tool query is called with query = `COUNT(*) FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`

### EXAMPLE: text_after_the_statement (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' LIMIT 5 please`
  and parse-statement finds a token after the statement
THEN:
  the tool result is an error whose message starts with
  `cannot read the statement at position 56 (please): expected the end of the statement, or WHERE, GROUP BY, ORDER BY, LIMIT in this order.`
  FakeAha recorded no call

### EXAMPLE: join_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT f.name FROM features f JOIN releases r ON f.release = r.id`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `JOIN is not supported. Read one entity per statement and filter by its parent, for example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`

### EXAMPLE: id_or_entity_alone (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `DEMO-3` and parse-statement reads it
THEN:
  the tool result is an error with the message
  `DEMO-3 is an id, not a statement. To read that record: SELECT * FROM features WHERE id = 'DEMO-3'`
WHEN:
  the tool query is called with query = `features`
THEN:
  the tool result is an error with the message
  `features is an entity, not a statement. To list its records: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'. To see its fields: DESCRIBE features`

### EXAMPLE: describe_takes_nothing_else (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `DESCRIBE features WHERE product = 'DEMO'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `DESCRIBE features lists the fields of features and takes nothing else. To read records: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`

### EXAMPLE: apostrophe_hint_in_an_error (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE name = 'it's`
  and parse-statement can read it neither strictly nor a second time
THEN:
  the tool result is an error whose message contains
  `The text contains an apostrophe (it's). Inside text, write an apostrophe twice: it''s`

### EXAMPLE: write_without_an_id_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET name = 'X' WHERE status = 'Shipped'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `one statement changes one record: the WHERE must start with id = '...'. Further conditions are joined with AND and have the form field = 'value'. To change several records, find their ids with the query tool and send one statement per record.`
  FakeAha recorded no call
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET name = 'X' WHERE id = 'DEMO-1' OR id = 'DEMO-2'`
THEN:
  the tool result is the same error
  FakeAha recorded no call

### EXAMPLE: write_without_where_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement = `UPDATE features SET name = 'X'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `UPDATE needs WHERE id = '...' with the id of the one record to change. Forms: INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...) | UPDATE entity SET field = 'value', ... WHERE id = 'id' | DELETE FROM entity WHERE id = 'id'`
  FakeAha recorded no call
WHEN:
  the tool execute is called with statement = `DELETE FROM features`
THEN:
  the tool result is an error whose message starts with
  `DELETE needs WHERE id = '...' with the id of the one record to change.`
  FakeAha recorded no call

### EXAMPLE: insert_with_two_rows_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
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
  the tool execute is called with statement =
  `INSERT INTO features (parent, name, bogus) VALUES ('DEMO-R-3', 'A', 'x')`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `unknown field 'bogus' on features. Fields of features that can be set: name, status, release, initiative, assigned_to, start_date, due_date, tags, description, and parent in INSERT. Custom fields, with UPDATE only: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers`
  no write request

### EXAMPLE: update_of_a_field_that_is_set_in_insert_only (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE goals SET success_metric = 'Weekly viewers' WHERE id = 'DEMO-G-1'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `the field success_metric of goals can be set in INSERT only. Fields of goals that can be set: name, success_metric (INSERT only), status, progress, description, and parent in INSERT`
  FakeAha recorded no ApiGet call on /goals/DEMO-G-1
  no write request

### EXAMPLE: assignment_of_a_word_or_an_expression (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET name = name WHERE id = 'DEMO-1'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `text goes in single quotes: write 'name' instead of name`
  FakeAha recorded no call
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET name = name + ' (old)' WHERE id = 'DEMO-1'`
THEN:
  the tool result is an error with the message
  `unexpected character + at position 33.`
  FakeAha recorded no call

### EXAMPLE: null_cannot_be_written (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
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
  the tool execute is called with statement =
  `UPDATE users SET name = 'X' WHERE id = '7000000000000000901'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `users cannot be changed. Entities that can be changed: releases, features, epics, requirements, initiatives, goals, ideas, comments`
  FakeAha recorded no call

### EXAMPLE: products_cannot_be_changed (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO products (name, key) VALUES ('New product', 'NEW')`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `products cannot be changed. Entities that can be changed: releases, features, epics, requirements, initiatives, goals, ideas, comments`
  FakeAha recorded no call

### EXAMPLE: label_of_a_custom_field (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT f.custom.priority, f.custom.customer AS customer FROM features f WHERE id = 'DEMO-1'`
  and parse-statement reads the two custom field words
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"custom.priority":"High","customer":"Acme"}]}`

### EXAMPLE: labels_of_select_items (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT COUNT(1) FROM features WHERE product = 'DEMO'`
  and parse-statement reads `COUNT(1)` as `COUNT(*)`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`
WHEN:
  the tool query is called with query =
  `SELECT id AS 'feature id' FROM features WHERE release = 'DEMO-R-2'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"feature id":"DEMO-1"},{"feature id":"DEMO-2"}]}`

### EXAMPLE: keys_and_arguments_that_are_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status ORDER BY name`
  and parse-statement checks the key of ORDER BY
THEN:
  the tool result is an error whose message starts with
  `cannot read the statement at position 87 (name): expected a field of GROUP BY, a selected field or an aggregate after ORDER BY.`
  FakeAha recorded no call
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' ORDER BY COUNT(*)`
THEN:
  the tool result is an error whose message starts with
  `cannot read the statement at position 57 (COUNT(*)): expected a field name.`
WHEN:
  the tool query is called with query = `SELECT SUM(*) FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is an error whose message starts with
  `cannot read the statement at position 12 (*) FROM features WHERE p): expected a field inside SUM( ).`

### EXAMPLE: a_row_has_each_label_once (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id, id, f.name, name FROM features f WHERE release = 'DEMO-R-2'`
  and parse-statement leaves out the items that repeat an earlier item
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","name":"CSV export"},{"id":"DEMO-2","name":"Offline maps"}]}`
WHEN:
  the tool query is called with query =
  `SELECT COUNT(*), COUNT(1) FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":6}]}`
WHEN:
  the tool query is called with query =
  `SELECT COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status, workflow_status`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"In design","COUNT(*)":1},{"status":"Under consideration","COUNT(*)":3},{"status":"In development","COUNT(*)":1},{"status":"Shipped","COUNT(*)":1}]}`
WHEN:
  the tool query is called with query =
  `SELECT *, name FROM features WHERE release = 'DEMO-R-2'`
THEN:
  each of the two rows has the labels id, name, status and release, in
  this order, and no other label
WHEN:
  the tool query is called with query =
  `SELECT id, name AS id FROM features WHERE release = 'DEMO-R-2'`
THEN:
  the tool result is an error with the message
  `two select items have the label 'id'. Give one of them another label, in the form: field AS label`
  FakeAha recorded no call
WHEN:
  the tool query is called with query =
  `SELECT COUNT(*) AS status FROM features WHERE product = 'DEMO' GROUP BY status`
THEN:
  the tool result is an error with the message
  `the label 'status' is also the name of a field of features. Choose another label, in the form: field AS label`
WHEN:
  the tool query is called with query =
  `SELECT *, name AS id FROM features WHERE release = 'DEMO-R-2'`
THEN:
  the tool result is an error with the message
  `the label 'id' is also the name of a field of features. Choose another label, in the form: field AS label`

### EXAMPLE: a_field_that_is_given_twice (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'Shipped', status = 'In design' WHERE id = 'DEMO-2'`
  and parse-statement finds the second assignment of the field
THEN:
  the tool result is an error with the message
  `the field status is given twice. A statement sets each field once.`
  FakeAha recorded no call
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET release = 'DEMO-R-3', parent = 'DEMO-R-1' WHERE id = 'DEMO-2'`
THEN:
  the tool result is an error with the message
  `the field release is given twice. A statement sets each field once.`
WHEN:
  the tool execute is called with statement =
  `INSERT INTO comments (parent, body, text) VALUES ('DEMO-3', 'one', 'two')`
THEN:
  the tool result is an error with the message
  `the field body is given twice. A statement sets each field once.`
  FakeAha recorded no call
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.customer = 'Acme', custom.CUSTOMER = 'Globex' WHERE id = 'DEMO-2'`
THEN:
  the tool result is an error with the message
  `the field custom.customer is given twice. A statement sets each field once.`
  FakeAha recorded no ApiUpdate call

### EXAMPLE: values_that_do_not_fit_the_type (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND created_at > '01.02.2026'`
  and parse-statement checks the value against the type of the field
THEN:
  the tool result is an error with the message
  `created_at is a timestamp: write the value as 'YYYY-MM-DD' or 'YYYY-MM-DDThh:mm:ssZ'. Got '01.02.2026'.`
  FakeAha recorded no call
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND release_date = '2026-02-30'`
THEN:
  the tool result is an error with the message
  `release_date is a date: write the value as 'YYYY-MM-DD'. Got '2026-02-30'.`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND score > 'abc'`
THEN:
  the tool result is an error with the message
  `score is a number: write the value as 5 or 2.5. Got 'abc'.`
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND released > 0`
THEN:
  the tool result is an error with the message
  `released is true or false: compare it with = or !=, for example released = TRUE`

### EXAMPLE: insert_with_a_custom_field_is_refused (of: parse-statement)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO features (parent, name, custom.priority) VALUES ('DEMO-R-3', 'A', 'High')`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `custom fields cannot be set in INSERT. Leave out custom.priority, create the record, and then send: UPDATE features SET custom.priority = '...' WHERE id = '...'`
  no write request
WHEN:
  the tool execute is called with statement =
  `INSERT INTO features (parent, name, priority) VALUES ('DEMO-R-3', 'A', 'High')`
THEN:
  the tool result is an error with the same message
  no write request

### EXAMPLE: custom_field_that_cannot_be_set (of: parse-statement)
GIVEN:
  the fixture account, in which value_score is a scorecard
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.value_score = 40 WHERE id = 'DEMO-1'`
  and parse-statement reads it
THEN:
  the tool result is an error with the message
  `the field custom.value_score of features cannot be set. Fields of features that can be set: name, status, release, initiative, assigned_to, start_date, due_date, tags, description, and parent in INSERT. Custom fields, with UPDATE only: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers`
  FakeAha recorded no ApiGet call on /features/DEMO-1
  no write request

### EXAMPLE: describe_and_read_have_the_same_fields (of: entity-catalog)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `DESCRIBE releases`, and the
  entity-catalog gives the fields
THEN:
  the names in `fields` are id, name, release_date, start_date, status,
  parking_lot, released, product, progress, description, created_at, url,
  in this order
WHEN:
  the tool query is called with query = `SELECT * FROM releases WHERE id = 'DEMO-R-2'`
THEN:
  the row has exactly these labels, in the same order

### EXAMPLE: goal_status_from_the_success_metric (of: entity-catalog)
GIVEN:
  the fixture account, in which the API record of DEMO-G-1 has the object
  `success_metric` with the name Monthly viewers and the status On track,
  and no member `workflow_status` of its own
WHEN:
  the tool query is called with query =
  `SELECT id, success_metric, status FROM goals WHERE product = 'DEMO'`
  and the entity-catalog gives the sources of both fields
THEN:
  the tool result is
  `{"entity":"goals","scope":"product DEMO","count":1,"records":[{"id":"DEMO-G-1","success_metric":"Monthly viewers","status":"On track"}]}`
  the parameter `fields` of the ApiList call names `success_metric` once,
  and it does not name `workflow_status`

### EXAMPLE: release_of_an_epic (of: entity-catalog)
GIVEN:
  the fixture account, in which the API record of the epic DEMO-E-1 names
  its release in the object `release` and has no member
  `release_reference_num`
WHEN:
  the tool query is called with query =
  `SELECT id, release FROM epics WHERE product = 'DEMO'`
  and the entity-catalog gives the source of the release
THEN:
  the tool result is
  `{"entity":"epics","scope":"product DEMO","count":1,"records":[{"id":"DEMO-E-1","release":"DEMO-R-3"}]}`
  the parameter `fields` of the ApiList call on
  /products/7000000000000000001/epics names `release`, and it names
  neither `release_reference_num` nor `color`
WHEN:
  the tool query is called with query =
  `SELECT name FROM epics WHERE release = 'demo-r-3'`
THEN:
  the tool result is
  `{"entity":"epics","scope":"release DEMO-R-3","count":1,"records":[{"name":"Race day"}]}`
  FakeAha recorded one ApiList call on /releases/DEMO-R-3/epics

### EXAMPLE: statuses_of_an_entity (of: entity-catalog)
GIVEN:
  the fixture account, in which the record of DEMO names the workflow 401
  for features, for epics and for requirements
WHEN:
  the tool query is called with query =
  `SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'features'`
  and the entity-catalog gives the source of the entity statuses
THEN:
  the tool result is
  `{"entity":"statuses","scope":"product DEMO","count":6,"records":[{"name":"Under consideration"},{"name":"In design"},{"name":"Ready to develop"},{"name":"In development"},{"name":"Shipped"},{"name":"Will not implement"}]}`
  FakeAha recorded one ApiGet call on /products/7000000000000000001 and
  one ApiGet call on /products/7000000000000000001/workflows
WHEN:
  the tool query is called with query =
  `SELECT name FROM statuses WHERE product = 'demo' AND entity = 'feature'`
THEN:
  the tool result is the same
WHEN:
  the tool query is called with query =
  `SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'epics'`
THEN:
  the tool result has the same six records
WHEN:
  the tool query is called with query =
  `SELECT entity FROM statuses WHERE product = 'DEMO' AND name = 'Shipped'`
THEN:
  the tool result is
  `{"entity":"statuses","scope":"product DEMO","count":5,"records":[{"entity":"releases"},{"entity":"features"},{"entity":"epics"},{"entity":"requirements"},{"entity":"ideas"}]}`

### EXAMPLE: current_user (of: entity-catalog)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT email FROM me`
  and the entity-catalog gives the source of the entity me
THEN:
  the tool result is
  `{"entity":"me","scope":"current user","count":1,"records":[{"email":"test.user@example.com"}]}`

### EXAMPLE: comments_of_a_record (of: entity-catalog)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT body, user FROM comments WHERE parent = 'DEMO-3'`
  and the entity-catalog gives the comment route of a feature
THEN:
  the tool result is
  `{"entity":"comments","scope":"feature DEMO-3","count":1,"records":[{"body":"Verified with the provider.","user":"Test User"}]}`
WHEN:
  the tool query is called with query =
  `SELECT body, user FROM comments WHERE feature = 'demo-3'`
THEN:
  the tool result is the same

### EXAMPLE: custom_fields_per_record_type (of: entity-catalog)
GIVEN:
  the fixture account, in which the definition segment has the record
  type `Ideas::Idea` and the definition cost_center the record type
  `Project`
WHEN:
  the tool query is called with query = `DESCRIBE ideas`
  and the entity-catalog gives the record type Idea for ideas
THEN:
  `custom_fields` is
  `[{"name":"custom.segment","label":"Segment","type":"one option of a list","writable":true}]`
  `custom_fields_note` is
  `* shows custom fields only with id = '...', and only those that have a value. To select one by name: SELECT id, custom.segment FROM ideas WHERE id = 'PROD-I-1'`
WHEN:
  the tool query is called with query = `DESCRIBE products`
THEN:
  `custom_fields` is
  `[{"name":"custom.cost_center","label":"Cost center","type":"string"}]`
  `custom_fields_note` is
  `* shows custom fields only with key = '...', and only those that have a value. To select one by name: SELECT key, custom.cost_center FROM products WHERE key = 'PROD'`
WHEN:
  the tool query is called with query = `DESCRIBE goals`
THEN:
  the result has no member `custom_fields` and no member `custom_fields_note`

### EXAMPLE: custom_fields_are_read_by_name (of: custom-fields)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id, custom.priority, custom.go_live FROM features WHERE release = 'DEMO-R-2'`
  and custom-fields gives the two custom fields
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","custom.priority":"High","custom.go_live":"2026-04-15"},{"id":"DEMO-2","custom.priority":"Medium","custom.go_live":null}]}`
  FakeAha recorded exactly two calls: ApiList on /custom_field_definitions
  and ApiList on /releases/DEMO-R-2/features

### EXAMPLE: unknown_custom_field_lists_the_custom_fields (of: custom-fields)
GIVEN:
  the fixture account, which has no custom field with the key severity
WHEN:
  the tool query is called with query =
  `SELECT id, custom.severity FROM features WHERE product = 'DEMO'`
  and custom-fields finds no definition with that key
THEN:
  the tool result is an error with the message
  `unknown custom field 'custom.severity' on features. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score. To see their types: DESCRIBE features`
  FakeAha recorded exactly one call: ApiList on /custom_field_definitions

### EXAMPLE: custom_field_of_a_type_that_is_not_supported (of: custom-fields)
GIVEN:
  the fixture account, in which business_case is a worksheet
WHEN:
  the tool query is called with query =
  `SELECT id, custom.business_case FROM features WHERE id = 'DEMO-1'`
  and custom-fields finds the type name EquationField
THEN:
  the tool result is an error with the message
  `custom.business_case has the type EquationField, and the server cannot read or set custom fields of that type. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score.`
  FakeAha recorded no ApiGet call on /features/DEMO-1

### EXAMPLE: entity_without_custom_fields (of: custom-fields)
GIVEN:
  the fixture account, which defines no custom field for goals
WHEN:
  the tool query is called with query =
  `SELECT body, custom.priority FROM comments WHERE parent = 'DEMO-3'`
  and custom-fields finds no record type for comments in the catalog
THEN:
  the tool result is an error with the message
  `comments have no custom fields that can be used here. Fields of comments: id, body, user, parent, created_at, url. Example: SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'`
  FakeAha recorded no call
WHEN:
  the tool query is called with query =
  `SELECT id, custom.priority FROM goals WHERE product = 'DEMO'`
THEN:
  the tool result is an error with the message
  `goals have no custom fields that can be used here. Fields of goals: id, name, success_metric, status, product, progress, value, effort, description, created_at, updated_at, url. Example: SELECT id, name FROM goals WHERE product = 'PROD'`

### EXAMPLE: custom_fields_that_the_key_cannot_read (of: custom-fields)
GIVEN:
  the fixture account
  FakeAha answers every ApiList call on /custom_field_definitions with
  the status 403
WHEN:
  the tool query is called with query =
  `SELECT id, custom.priority FROM features WHERE product = 'DEMO'`
  and custom-fields cannot read the definitions
THEN:
  the tool result is an error with the message
  `the custom fields of the account cannot be read with this API key (status 403). Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
WHEN:
  the tool query is called with query =
  `SELECT id, severity FROM features WHERE product = 'DEMO'`
THEN:
  the tool result is an error with the message
  `unknown field 'severity' on features. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
WHEN:
  the tool query is called with query = `DESCRIBE features`
THEN:
  `fields` has 17 objects, and the result has no member `custom_fields`
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE id = 'DEMO-1'`
THEN:
  the tool result is no error
  the row has the 17 fields of features in catalog order and no other label
WHEN:
  the tool query is called with query =
  `SELECT id, name FROM features WHERE release = 'DEMO-R-2'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","name":"CSV export"},{"id":"DEMO-2","name":"Offline maps"}]}`

### EXAMPLE: custom_field_that_was_added_later (of: custom-fields)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`
  and custom-fields reads the definitions
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`
WHEN:
  a test adds to FakeAha a definition with the key `risk`, the name Risk,
  the type CustomFieldDefinitions::TextField and the record type Feature,
  and then the tool query is called with query =
  `SELECT id, custom.risk FROM features WHERE id = 'DEMO-1'`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.risk":null}]}`
  FakeAha recorded one ApiList call on /custom_field_definitions for the
  second statement, because the definitions of the first one are not kept

### EXAMPLE: definitions_are_not_kept_in_http_mode (of: custom-fields)
GIVEN:
  the fixture account
  the server runs in http mode with the setting subdomain = `example`
WHEN:
  an MCP client calls the tool query two times with query =
  `SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`
  and with the header `Aha-Api-Key: fixture-key`, and custom-fields reads
  the definitions for each request
THEN:
  both tool results are
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`
  FakeAha recorded two ApiList calls on /custom_field_definitions, one for
  each request

### EXAMPLE: own_name_of_the_entity (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
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
  the tool query is called with query =
  `SELECT id, severity FROM features WHERE product = 'DEMO'`
  and resolve-names does not find the word severity
THEN:
  the tool result is an error with the message
  `unknown field 'severity' on features. Fields of features: id, name, status, release, initiative, epic, assigned_to, product, start_date, due_date, tags, score, progress, description, created_at, updated_at, url. Custom fields of features: custom.customer, custom.effort_days, custom.go_live, custom.markets, custom.platforms, custom.priority, custom.release_notes, custom.reviewers, custom.value_score. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
  FakeAha recorded exactly one call: ApiList on /custom_field_definitions,
  because severity could be the key of a custom field

### EXAMPLE: unknown_entity_lists_the_entities (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT * FROM tickets`
  and resolve-names does not find the word tickets
THEN:
  the tool result is an error with the message
  `unknown entity 'tickets'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`

### EXAMPLE: users_cannot_be_filtered_by_product (of: resolve-names)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT name FROM users WHERE product = 'DEMO'`
  and resolve-names gets the word product on users
THEN:
  the tool result is an error with the message
  `users cannot be filtered by product. Fields of users: id, name, email, created_at. Example: SELECT name, email FROM users`

### EXAMPLE: bare_key_of_a_custom_field (of: resolve-names)
GIVEN:
  the fixture account, in which features have no catalog field priority
  and a custom field with the key priority
WHEN:
  the tool query is called with query =
  `SELECT id, priority FROM features WHERE id = 'demo-1'`
  and resolve-names reads the word priority as custom.priority
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","priority":"High"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE release = 'DEMO-R-2' AND Custom.Priority = 'medium'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":1,"records":[{"id":"DEMO-2"}]}`

### EXAMPLE: catalog_name_wins_over_a_custom_key (of: resolve-names)
GIVEN:
  the fixture account and, in addition, a custom field definition with
  the key `status`, the name Review status, the type
  CustomFieldDefinitions::TextField and the record type Feature; DEMO-1
  has the value Checked in it
WHEN:
  the tool query is called with query =
  `SELECT status, custom.status FROM features WHERE id = 'DEMO-1'`
  and resolve-names reads status as the catalog field and custom.status
  as the custom field
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"status":"In design","custom.status":"Checked"}]}`

### EXAMPLE: product_in_three_spellings (of: resolve-product)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT id FROM releases WHERE product = 'demo'`
  and resolve-product finds the product by its key in lower case
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1"},{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = '7000000000000000001'`
THEN:
  the tool result is the same
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'Demo Product'`
THEN:
  the tool result is the same

### EXAMPLE: the_product_list_is_not_kept (of: resolve-product)
GIVEN:
  the fixture account
WHEN:
  the tool query is called two times with query =
  `SELECT id FROM releases WHERE product = 'DEMO'`
  and resolve-product takes the product list for each statement
THEN:
  both tool results are
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1"},{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
  FakeAha recorded two ApiList calls on /products, one for each statement
  the parameter `fields` of both calls does not name `description`

### EXAMPLE: products_are_not_mixed (of: resolve-product)
GIVEN:
  the fixture account, in which DEMO and OTHER have one goal each
WHEN:
  the tool query is called with query = `SELECT id, name FROM goals WHERE product = 'other'`
  and resolve-product finds the product OTHER
THEN:
  the tool result is
  `{"entity":"goals","scope":"product OTHER","count":1,"records":[{"id":"OTHER-G-1","name":"Other goal"}]}`

### EXAMPLE: unknown_product_lists_the_keys (of: resolve-product)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT id FROM releases WHERE product = 'DEMOX'`
  and resolve-product finds no product
THEN:
  the tool result is an error with the message
  `unknown product 'DEMOX'. Product keys: DEMO, OTHER. To see their names: SELECT key, name FROM products`

### EXAMPLE: release_id_where_a_product_is_needed (of: resolve-product)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE product = 'DEMO-R-3'`
  and resolve-product gets the id of a release
THEN:
  the tool result is an error with the message
  `'DEMO-R-3' is a release, not a product. Use: WHERE release = 'DEMO-R-3'`

### EXAMPLE: in_list_reads_each_parent (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
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
  the tool query is called with query =
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
  the tool query is called with query =
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
  the tool query is called with query =
  `SELECT id, name FROM features WHERE product = 'HUGE' ORDER BY name`
  and plan-fetch learns from page 1 that the list has 16 pages
THEN:
  the tool result is an error with the message
  `this would read 3100 records. Narrow it down, for example with release = 'PROD-R-3'`
  FakeAha recorded exactly one ApiList call on the feature list of HUGE
WHEN:
  the tool query is called with query =
  `SELECT id, name FROM features WHERE product = 'HUGE'`
THEN:
  `count` is 50, `total` is 3100 and `truncated` is true
  FakeAha recorded exactly one more ApiList call on the feature list of HUGE

### EXAMPLE: release_by_name_lists_the_releases (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND release = 'Summer release'`
  and plan-fetch gets a name where the id of a release is needed
THEN:
  the tool result is an error with the message
  `release = needs the id of the release, not its name. The releases of DEMO: DEMO-R-1 (What's next), DEMO-R-2 (Spring release), DEMO-R-3 (Summer release), DEMO-R-4 (Rider's cup release)`

### EXAMPLE: parent_that_does_not_exist (of: plan-fetch)
GIVEN:
  the fixture account, which has no release DEMO-R-99
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE release = 'DEMO-R-99'`
  and plan-fetch gets NotFound for the list of the release
THEN:
  the tool result is an error with the message
  `there is no release with the id 'DEMO-R-99'. To find it: SELECT id, name, release_date FROM releases WHERE product = 'PROD'`

### EXAMPLE: release_id_where_a_feature_id_is_needed (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE id = 'DEMO-R-3'`
  and plan-fetch gets the id of a release for the field id of features
THEN:
  the tool result is an error with the message
  `'DEMO-R-3' is the id of a release, not of a feature. For the features of that release: SELECT * FROM features WHERE release = 'DEMO-R-3'`
  FakeAha recorded no ApiGet call

### EXAMPLE: comments_need_a_parent (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT body FROM comments`
  and plan-fetch finds no route
THEN:
  the tool result is an error with the message
  `comments are listed per record: add parent = '...' to WHERE. Example: SELECT body, user, created_at FROM comments WHERE parent = 'PROD-23'`

### EXAMPLE: answers_in_the_json_form_of_aha (of: plan-fetch)
GIVEN:
  the fixture account
  the answer of FakeAha to GET /releases/DEMO-R-2/features has the member
  `features` with two records and the member `pagination` with the value
  `{"total_records":2,"total_pages":1,"current_page":1}`
  the answer of FakeAha to GET /features/DEMO-1 is an object with the one
  member `feature`
WHEN:
  the tool query is called with query =
  `SELECT id, status FROM features WHERE release = 'DEMO-R-2'`
  and plan-fetch reads the list
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-1","status":"In design"},{"id":"DEMO-2","status":"Under consideration"}]}`
  the parameter `fields` of the list request names `reference_num` and
  `workflow_status`, and no name in it has a dot
WHEN:
  the tool query is called with query = `SELECT id, status FROM features WHERE id = 'DEMO-1'`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","status":"In design"}]}`

### EXAMPLE: answer_with_an_unknown_form (of: plan-fetch)
GIVEN:
  the fixture account
  FakeAha answers GET /releases/DEMO-R-2/features with the status 200 and
  the body `{"items":[]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE release = 'DEMO-R-2'`
  and plan-fetch reads the list
THEN:
  the tool result is an error with the message
  `aha: API error (status 200) on GET /releases/DEMO-R-2/features: the answer has an unknown form`

### EXAMPLE: custom_fields_are_requested_only_when_used (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id, name FROM features WHERE release = 'DEMO-R-2'`
  and plan-fetch builds the list request
THEN:
  FakeAha recorded exactly one call: ApiList on /releases/DEMO-R-2/features
  the parameter `fields` of that call does not name `custom_fields`
WHEN:
  the tool query is called with query =
  `SELECT id, custom.effort_days FROM features WHERE release = 'DEMO-R-2' ORDER BY custom.effort_days`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"id":"DEMO-2","custom.effort_days":8},{"id":"DEMO-1","custom.effort_days":12.5}]}`
  the parameter `fields` of the ApiList call on /releases/DEMO-R-2/features
  names `custom_fields`
  FakeAha recorded no ApiGet call on a feature

### EXAMPLE: list_without_custom_fields_is_an_error (of: plan-fetch)
GIVEN:
  the fixture account
  FakeAha leaves the member `custom_fields` out of every record of a list
WHEN:
  the tool query is called with query =
  `SELECT id, custom.priority FROM features WHERE release = 'DEMO-R-2'`
  and plan-fetch reads page 1 of the list
THEN:
  the tool result is an error with the message
  `Aha! returned this list of features without custom fields. A record that is read by its id has them: add id = '...' to WHERE.`
  FakeAha recorded no ApiGet call on a feature
WHEN:
  the tool query is called with query =
  `SELECT id, custom.priority FROM features WHERE id = 'DEMO-1'`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"id":"DEMO-1","custom.priority":"High"}]}`

### EXAMPLE: in_list_is_checked_before_the_first_request (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE id IN ('DEMO-1', 'DEMO-R-2')`
  and plan-fetch checks both values before it reads one
THEN:
  the tool result is an error with the message
  `'DEMO-R-2' is the id of a release, not of a feature. For the features of that release: SELECT * FROM features WHERE release = 'DEMO-R-2'`
  FakeAha recorded no call
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE id IN ('DEMO-1', 'demo-1', 'DEMO-2')`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1, feature DEMO-2","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`
  FakeAha recorded exactly two calls: ApiGet on /features/DEMO-1 and
  ApiGet on /features/DEMO-2

### EXAMPLE: description_of_a_product_is_requested_when_it_is_used (of: plan-fetch)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT key, name FROM products`
  and plan-fetch reads the product list
THEN:
  FakeAha recorded exactly one call: ApiList on /products
  the parameter `fields` of the call does not name `description`
WHEN:
  the tool query is called with query = `SELECT key, description FROM products`
THEN:
  FakeAha recorded exactly one call: ApiList on /products
  the parameter `fields` of the call names `description`
  each of the two rows has the labels key and description

### EXAMPLE: custom_field_of_a_product (of: plan-fetch)
GIVEN:
  the fixture account, in which the product DEMO has the cost_center CC-100
WHEN:
  the tool query is called with query = `SELECT key, custom.cost_center FROM products`
  and plan-fetch reads the product list with its custom fields
THEN:
  the tool result is
  `{"entity":"products","scope":"all products","count":2,"records":[{"key":"DEMO","custom.cost_center":"CC-100"},{"key":"OTHER","custom.cost_center":null}]}`
  FakeAha recorded an ApiList call on /products whose parameter `fields`
  names `custom_fields`

### EXAMPLE: parking_lot_and_missing_date (of: convert-record)
GIVEN:
  the fixture account, in which the API records of DEMO-R-2, DEMO-R-3 and
  DEMO-R-4 have no member parking_lot
WHEN:
  the tool query is called with query =
  `SELECT id, release_date, parking_lot FROM releases WHERE product = 'DEMO'`
  and convert-record converts the four API records
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-1","release_date":null,"parking_lot":true},{"id":"DEMO-R-2","release_date":"2026-03-31","parking_lot":false},{"id":"DEMO-R-3","release_date":"2026-06-30","parking_lot":false},{"id":"DEMO-R-4","release_date":"2026-12-15","parking_lot":false}]}`

### EXAMPLE: description_is_plain_text (of: convert-record)
GIVEN:
  the fixture account, in which the member `description` of the API record
  of DEMO-1 is the object
  `{"id":"7000000000000000601","body":"<p>Export <b>all</b> columns.</p><p>Keep the order.</p>"}`
WHEN:
  the tool query is called with query = `SELECT description FROM features WHERE id = 'DEMO-1'`
  and convert-record takes the text from the member `body`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"description":"Export all columns.\nKeep the order."}]}`

### EXAMPLE: custom_values_by_kind (of: convert-record)
GIVEN:
  the fixture account, in which the API record of DEMO-1 has these entries
  in its member `custom_fields`:
  `{"key":"effort_days","name":"Effort in days","type":"number","value":"12.5"}`,
  `{"key":"markets","name":"Markets","type":"array","value":["EMEA","APAC"]}`,
  `{"key":"reviewers","name":"Reviewers","type":"array","value":["7000000000000000901"],"email_value":["second.user@example.com"],"full_value":[{"id":"7000000000000000901","name":"Second User","email":"second.user@example.com"}]}`,
  `{"key":"release_notes","name":"Release notes","type":"note","value":"<p>Adds the export.</p><p>See the handbook.</p>"}`,
  `{"key":"value_score","name":"Value score","type":"scorecard","value":31}`
WHEN:
  the tool query is called with query =
  `SELECT custom.effort_days, custom.markets, custom.reviewers, custom.release_notes, custom.value_score FROM features WHERE id = 'DEMO-1'`
  and convert-record converts the entries
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-1","count":1,"records":[{"custom.effort_days":12.5,"custom.markets":"EMEA, APAC","custom.reviewers":"second.user@example.com","custom.release_notes":"Adds the export.\nSee the handbook.","custom.value_score":31}]}`
WHEN:
  the tool query is called with query =
  `SELECT custom.effort_days, custom.markets, custom.go_live FROM features WHERE id = 'DEMO-2'`
THEN:
  the tool result is
  `{"entity":"features","scope":"feature DEMO-2","count":1,"records":[{"custom.effort_days":8,"custom.markets":null,"custom.go_live":null}]}`

### EXAMPLE: or_and_not_are_evaluated (of: evaluate-condition)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND (status = 'Shipped' OR status = 'In design')`
  and evaluate-condition evaluates the OR on every record
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-5"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE release = 'DEMO-R-3' AND NOT status = 'Shipped'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-3","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`

### EXAMPLE: me_as_a_value (of: evaluate-condition)
GIVEN:
  the fixture account, in which the current user is test.user@example.com
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND assigned_to = 'me'`
  and evaluate-condition compares with the current user
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`

### EXAMPLE: fields_without_a_value_in_conditions (of: evaluate-condition)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND release_date IS NULL`
  and evaluate-condition tests the field
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":1,"records":[{"id":"DEMO-R-1"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND parking_lot = false`
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":3,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE release = 'DEMO-R-2' AND assigned_to != 'test.user@example.com'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":1,"records":[{"id":"DEMO-2"}]}`

### EXAMPLE: conditions_on_custom_fields (of: evaluate-condition)
GIVEN:
  the fixture account, in which the current user is test.user@example.com
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.priority = 'high'`
  and evaluate-condition compares the custom values
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-3"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.effort_days > 5`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.go_live < '2026-06-01'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers CONTAINS 'me'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers CONTAINS 'second user'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-3"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.priority IS NULL`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":3,"records":[{"id":"DEMO-4"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`

### EXAMPLE: fields_with_several_values (of: evaluate-condition)
GIVEN:
  the fixture account, in which the current user is test.user@example.com
  DEMO-1 has the markets EMEA and APAC and the reviewer Second User
  DEMO-3 has the market EMEA and the reviewers Test User and Second User
  DEMO-3 has the tags live and dashboard, and DEMO-4 has the tag live
WHEN:
  the tool query is called with query =
  `SELECT id, custom.markets FROM features WHERE product = 'DEMO' AND custom.markets = 'emea'`
  and evaluate-condition looks for the value among the values of the field
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1","custom.markets":"EMEA, APAC"},{"id":"DEMO-3","custom.markets":"EMEA"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.markets = 'APAC, EMEA'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.markets != 'APAC'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":5,"records":[{"id":"DEMO-2"},{"id":"DEMO-3"},{"id":"DEMO-4"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers = 'me'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers = 'second.user@example.com, me'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`
  FakeAha recorded one ApiGet call on /me
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND custom.reviewers IN ('Second User', 'nobody@example.com')`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-1"},{"id":"DEMO-3"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id, tags FROM features WHERE product = 'DEMO' AND tags = 'LIVE'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3","tags":"live, dashboard"},{"id":"DEMO-4","tags":"live"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND tags = 'dashboard, live'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND tags CONTAINS 'board'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-3"}]}`

### EXAMPLE: timestamps_are_points_in_time (of: evaluate-condition)
GIVEN:
  the fixture account, in which DEMO-1 has the updated_at
  `2026-02-02T08:00:00.000Z` and the other five features of DEMO have
  `2026-02-03T23:30:00.000Z`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND updated_at = '2026-02-02T08:00:00Z'`
  and evaluate-condition compares two points in time
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"id":"DEMO-1"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND updated_at <= '2026-02-02T09:00:00+01:00'`
THEN:
  the tool result is the same, because the value is the same point in time
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND updated_at > '2026-02-04T00:15:00+01:00'`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":5,"records":[{"id":"DEMO-2"},{"id":"DEMO-3"},{"id":"DEMO-4"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND updated_at = '2026-02-02'`
THEN:
  the tool result has the one record DEMO-1, because the value is a day
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' AND release_date >= '2026-06-30T00:00:00Z'`
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":2,"records":[{"id":"DEMO-R-3"},{"id":"DEMO-R-4"}]}`

### EXAMPLE: empty_text_and_null_stand_for_no_value (of: evaluate-condition)
GIVEN:
  the fixture account, in which DEMO-3 and DEMO-4 have the initiative
  DEMO-S-1 and the other four features of DEMO have none
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND initiative = ''`
  and evaluate-condition tests for no value
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":4,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"},{"id":"DEMO-5"},{"id":"DEMO-6"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND initiative != ''`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"id":"DEMO-3"},{"id":"DEMO-4"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' AND initiative IN ('DEMO-S-1', NULL)`
THEN:
  `scope` is `product DEMO` and `count` is 6
  FakeAha recorded no call on /initiatives/DEMO-S-1/features, because a
  list with NULL is not taken as a route

### EXAMPLE: grouping_without_group_by (of: build-rows)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT release, COUNT(*) FROM features WHERE product = 'DEMO'`
  and build-rows groups by the selected field
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":3,"records":[{"release":"DEMO-R-2","COUNT(*)":2},{"release":"DEMO-R-3","COUNT(*)":3},{"release":"DEMO-R-1","COUNT(*)":1}]}`

### EXAMPLE: rows_without_a_value_sort_last (of: build-rows)
GIVEN:
  the fixture account, in which DEMO-R-1 has no release date
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date`
  and build-rows sorts the rows
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-2"},{"id":"DEMO-R-3"},{"id":"DEMO-R-4"},{"id":"DEMO-R-1"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM releases WHERE product = 'DEMO' ORDER BY release_date DESC`
THEN:
  the tool result is
  `{"entity":"releases","scope":"product DEMO","count":4,"records":[{"id":"DEMO-R-4"},{"id":"DEMO-R-3"},{"id":"DEMO-R-2"},{"id":"DEMO-R-1"}]}`

### EXAMPLE: count_of_nothing_is_zero (of: build-rows)
GIVEN:
  the fixture account, in which no feature has the status Will not implement
WHEN:
  the tool query is called with query =
  `SELECT COUNT(*) FROM features WHERE product = 'DEMO' AND status = 'Will not implement'`
  and build-rows gets no record
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":1,"records":[{"COUNT(*)":0}]}`

### EXAMPLE: limit_of_the_statement_keeps_the_total (of: build-rows)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE product = 'DEMO' LIMIT 2`
  and build-rows cuts the rows
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"total":6,"records":[{"id":"DEMO-1"},{"id":"DEMO-2"}]}`
  the result has no member `truncated` and no `note`

### EXAMPLE: ids_are_sorted_by_their_numbers (of: build-rows)
GIVEN:
  the fixture account and, in addition, a product BULK with one release
  BULK-R-1 and 60 features BULK-1 to BULK-60 in it
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE release = 'BULK-R-1' ORDER BY id DESC LIMIT 2`
  and build-rows compares the numbers in the ids
THEN:
  the tool result is
  `{"entity":"features","scope":"release BULK-R-1","count":2,"total":60,"records":[{"id":"BULK-60"},{"id":"BULK-59"}]}`
WHEN:
  the tool query is called with query =
  `SELECT MIN(id), MAX(id) FROM features WHERE release = 'BULK-R-1'`
THEN:
  the tool result is
  `{"entity":"features","scope":"release BULK-R-1","count":1,"records":[{"MIN(id)":"BULK-1","MAX(id)":"BULK-60"}]}`
WHEN:
  the tool query is called with query =
  `SELECT id FROM features WHERE release = 'BULK-R-1' AND id > 'BULK-58' ORDER BY id`
THEN:
  the tool result is
  `{"entity":"features","scope":"release BULK-R-1","count":2,"records":[{"id":"BULK-59"},{"id":"BULK-60"}]}`

### EXAMPLE: order_by_a_label_and_by_an_aggregate (of: build-rows)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query =
  `SELECT name AS n FROM features WHERE release = 'DEMO-R-2' ORDER BY n DESC`
  and build-rows sorts by the field of the label
THEN:
  the tool result is
  `{"entity":"features","scope":"release DEMO-R-2","count":2,"records":[{"n":"Offline maps"},{"n":"CSV export"}]}`
WHEN:
  the tool query is called with query =
  `SELECT status FROM features WHERE product = 'DEMO' GROUP BY status ORDER BY COUNT(*) DESC, status`
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":4,"records":[{"status":"Under consideration"},{"status":"In design"},{"status":"In development"},{"status":"Shipped"}]}`

### EXAMPLE: records_without_a_value_are_one_group (of: build-rows)
GIVEN:
  the fixture account, in which four features of DEMO have no initiative
WHEN:
  the tool query is called with query =
  `SELECT initiative, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY initiative`
  and build-rows groups the records
THEN:
  the tool result is
  `{"entity":"features","scope":"product DEMO","count":2,"records":[{"initiative":null,"COUNT(*)":4},{"initiative":"DEMO-S-1","COUNT(*)":2}]}`

### EXAMPLE: tags_are_split_at_commas (of: check-write-values)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET tags = 'export, csv ,' WHERE id = 'DEMO-1'`
  and check-write-values prepares the value
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-1 with the body
  `{"feature":{"tags":"export,csv"}}`
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","tags":"export, csv","message":"Feature DEMO-1 updated"}`

### EXAMPLE: unknown_status_lists_the_statuses (of: check-write-values)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'Done' WHERE id = 'DEMO-2'`
  and check-write-values finds no workflow status with that name
THEN:
  the tool result is an error with the message
  `workflow status 'Done' not found for feature; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement`
  no write request

### EXAMPLE: status_of_an_epic (of: check-write-values)
GIVEN:
  the fixture account, in which the record of DEMO names the workflow 401
  as `epic_workflow`, and no workflow has the record type Epic
WHEN:
  the tool execute is called with statement =
  `UPDATE epics SET status = 'in development' WHERE id = 'DEMO-E-1'`
  and check-write-values takes the statuses of epics in DEMO
THEN:
  FakeAha recorded one ApiUpdate call on /epics/DEMO-E-1 with the body
  `{"epic":{"workflow_status":"In development"}}`
  the tool result is
  `{"id":"DEMO-E-1","name":"Race day","status":"In development","message":"Epic DEMO-E-1 status changed to 'In development'"}`
WHEN:
  the tool execute is called with statement =
  `UPDATE epics SET status = 'On track' WHERE id = 'DEMO-E-1'`
THEN:
  the tool result is an error with the message
  `workflow status 'On track' not found for epic; valid statuses: Under consideration, In design, Ready to develop, In development, Shipped, Will not implement`
  no write request

### EXAMPLE: date_that_is_not_a_date (of: check-write-values)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
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
  the tool execute is called with statement =
  `UPDATE features SET assigned_to = 'Second User' WHERE id = 'DEMO-2'`
  and check-write-values checks the user
THEN:
  the tool result is an error with the message
  `assigned_to: there is no user 'Second User'. Use an e-mail address, or 'me' for yourself`
  no write request

### EXAMPLE: custom_values_are_checked_by_kind (of: check-write-values)
GIVEN:
  the fixture account, in which platforms has the options Web, iOS and
  Android, and customer is a list that users can extend
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.effort_days = '4.5', custom.go_live = 2026-09-01, custom.customer = 'Initech', custom.platforms = 'ios, web' WHERE id = 'DEMO-2'`
  and check-write-values checks the four values
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-2 with the body
  `{"feature":{"custom_fields":{"effort_days":4.5,"go_live":"2026-09-01","customer":"Initech","platforms":["iOS","Web"]}}}`
  the tool result is
  `{"id":"DEMO-2","name":"Offline maps","custom.effort_days":4.5,"custom.go_live":"2026-09-01","custom.customer":"Initech","custom.platforms":"iOS, Web","message":"Feature DEMO-2 updated"}`

### EXAMPLE: option_that_is_not_in_the_list (of: check-write-values)
GIVEN:
  the fixture account, in which priority has the options High, Medium and
  Low
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.priority = 'Urgent' WHERE id = 'DEMO-2'`
  and check-write-values compares the value with the options
THEN:
  the tool result is an error with the message
  `custom.priority: there is no option 'Urgent'. Options: High, Medium, Low`
  no write request
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.platforms = 'web, Linux' WHERE id = 'DEMO-2'`
THEN:
  the tool result is an error with the message
  `custom.platforms: there is no option 'Linux'. Options: Web, iOS, Android`
  no write request
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.effort_days = 'many' WHERE id = 'DEMO-2'`
THEN:
  the tool result is an error with the message
  `custom.effort_days must be a number`
  no write request
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET custom.reviewers = 'Second User' WHERE id = 'DEMO-2'`
THEN:
  the tool result is an error with the message
  `custom.reviewers: there is no user 'Second User'. Use an e-mail address, or 'me' for yourself`
  no write request

### EXAMPLE: body_of_an_update_request (of: write-request)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET status = 'in design', assigned_to = 'me', due_date = '2026-07-01' WHERE id = 'DEMO-2'`
  and write-request builds the body
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-2 with the body
  `{"feature":{"workflow_status":"In design","assigned_to_user":"test.user@example.com","due_date":"2026-07-01"}}`

### EXAMPLE: body_of_a_create_request (of: write-request)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO releases (parent, name, release_date, description) VALUES ('demo', 'Winter release', '2027-01-31', 'Maintenance only')`
  and write-request builds the body
THEN:
  FakeAha recorded one ApiCreate call on
  /products/7000000000000000001/releases with the body
  `{"release":{"name":"Winter release","release_date":"2027-01-31","theme":"Maintenance only"}}`
  the body has no member for the parent

### EXAMPLE: body_with_custom_fields (of: write-request)
GIVEN:
  the fixture account, in which the current user has the id
  7000000000000000900
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET due_date = '2026-05-01', custom.markets = 'EMEA, LATAM ,', custom.reviewers = 'me, second.user@example.com', start_date = '2026-04-01', custom.release_notes = 'Ready.' WHERE id = 'DEMO-1'`
  and write-request builds the body
THEN:
  FakeAha recorded one ApiUpdate call on /features/DEMO-1 with the body
  `{"feature":{"due_date":"2026-05-01","custom_fields":{"markets":["EMEA","LATAM"],"reviewers":["7000000000000000900","second.user@example.com"],"release_notes":"Ready."},"start_date":"2026-04-01"}}`
  the tool result is
  `{"id":"DEMO-1","name":"CSV export","due_date":"2026-05-01","custom.markets":"EMEA, LATAM","custom.reviewers":"test.user@example.com, second.user@example.com","start_date":"2026-04-01","custom.release_notes":"Ready.","message":"Feature DEMO-1 updated"}`

### EXAMPLE: bodies_for_a_goal (of: write-request)
GIVEN:
  the fixture account
WHEN:
  the tool execute is called with statement =
  `INSERT INTO goals (parent, name, success_metric, status) VALUES ('DEMO', 'Win the cup', 'Podium places', 'at risk')`
  and write-request builds the body
THEN:
  FakeAha recorded one ApiCreate call on
  /products/7000000000000000001/goals with the body
  `{"goal":{"name":"Win the cup","success_metric":{"name":"Podium places","workflow_status":"At risk"}}}`
  the message of the tool result is `Goal DEMO-G-2 created`
WHEN:
  the tool execute is called with statement =
  `UPDATE goals SET status = 'achieved' WHERE id = 'DEMO-G-1'`
THEN:
  FakeAha recorded one ApiUpdate call on /goals/DEMO-G-1 with the body
  `{"goal":{"workflow_status":"Achieved"}}`
  the tool result is
  `{"id":"DEMO-G-1","name":"Grow the audience","status":"Achieved","message":"Goal DEMO-G-1 status changed to 'Achieved'"}`

### EXAMPLE: request_address_and_headers (of: aha-request)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE id = 'demo-3'`
  and aha-request sends the request
THEN:
  FakeAha recorded exactly one request:
  GET https://example.aha.io/api/v1/features/DEMO-3
  the request has the headers `Authorization: Bearer fixture-key` and
  `Accept: application/json`
  its `User-Agent` header starts with `mcp-server-aha-aql/` and ends with
  `(https://github.com/mge1512/aha-aql)`

### EXAMPLE: request_through_the_proxy_of_the_environment (of: aha-request)
GIVEN:
  the fixture account
  the environment has HTTPS_PROXY with the address of FakeAha, as in
  every example
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE id = 'DEMO-3'`
  and aha-request sends the request
THEN:
  FakeAha recorded one CONNECT request, for `example.aha.io:443`
  no header of the CONNECT request has `fixture-key`
  FakeAha recorded exactly one request inside the tunnel:
  GET https://example.aha.io/api/v1/features/DEMO-3
  the tool result is
  `{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`

### EXAMPLE: certificate_that_the_trust_store_does_not_have (of: aha-request)
GIVEN:
  the fixture account
  the environment has SSL_CERT_FILE with a certificate authority that did
  not sign the certificate of FakeAha
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE id = 'DEMO-3'`
  and aha-request checks the certificate inside the tunnel
THEN:
  the tool result is an error with the message
  `cannot reach Aha!: the certificate of example.aha.io could not be verified.`
  FakeAha recorded no request inside the tunnel, so the credential was
  not sent

### EXAMPLE: proxy_does_not_open_the_tunnel (of: aha-request)
GIVEN:
  the fixture account
  FakeAha is told to refuse CONNECT
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'`
  and aha-request sends the first request of the statement
THEN:
  the tool result is an error with the message
  `cannot reach Aha!: the proxy did not open a connection to example.aha.io.`
  the message does not contain the address of FakeAha
  FakeAha recorded no request inside a tunnel

### EXAMPLE: too_many_requests_is_sent_again (of: aha-request)
GIVEN:
  the fixture account
  FakeAha answers the route /features/DEMO-3 once with the status 429 and
  with the header `X-Ratelimit-Reset`, whose value is the UTC Unix time of
  that answer plus 2, and after that as usual
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE id = 'DEMO-3'`
  and aha-request sends the request
THEN:
  FakeAha recorded two calls on /features/DEMO-3
  the second call arrived at least 1 second and less than 5 seconds after
  the first
  the tool result is
  `{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`

### EXAMPLE: too_many_requests_three_times (of: aha-request)
GIVEN:
  the fixture account
  FakeAha answers the route /features/DEMO-3 every time with the status 429
  and without the header X-Ratelimit-Reset
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE id = 'DEMO-3'`
  and aha-request sends the request
THEN:
  FakeAha recorded three calls on /features/DEMO-3
  the second call arrived at least 5 seconds after the first, and the
  third at least 5 seconds after the second
  the tool result is an error with the message
  `aha: API error (status 429): too many requests`

### EXAMPLE: redirect_is_not_followed (of: aha-request)
GIVEN:
  the fixture account
  FakeAha answers the route /features/DEMO-3 with the status 302, the header
  `Location: https://other.example/api/v1/features/DEMO-3` and no text
WHEN:
  the tool query is called with query = `SELECT id FROM features WHERE id = 'DEMO-3'`
  and aha-request sends the request
THEN:
  FakeAha recorded exactly one request, and it went to example.aha.io
  the tool result is an error with the message
  `aha: API error (status 302) on GET /features/DEMO-3: (no text)`

### EXAMPLE: credential_that_aha_does_not_accept (of: aha-request)
GIVEN:
  the fixture account
  the server runs in stdio mode with the Account of the subdomain `example`
  and the credential `wrong-key`
WHEN:
  the tool query is called with query = `SELECT email FROM me`
  and aha-request sends the request
THEN:
  the tool result is an error with the message
  `Aha! refused the request (status 403): the API key is not valid for the account example, or it has no permission for this record. Repeating the statement does not help.`
  the tool result does not contain `wrong-key`

### EXAMPLE: write_request_without_an_answer (of: aha-request)
GIVEN:
  the fixture account
  FakeAha gives no answer to an ApiUpdate call on /features/DEMO-1
WHEN:
  the tool execute is called with statement =
  `UPDATE features SET name = 'CSV export 2' WHERE id = 'DEMO-1'`
  and aha-request sends the write request
THEN:
  FakeAha recorded exactly one ApiUpdate call
  the tool result is an error with the message
  `cannot reach Aha!: no answer within 60 seconds. The change may have been made. Read the record before you send the statement again.`

### EXAMPLE: statements_in_messages_can_be_sent (of: error-messages)
GIVEN:
  the fixture account
  the ExampleKey PROD
WHEN:
  parse-statement gets each of these statements of
  the error-messages tables and of the catalog after the placeholders are
  replaced: the example statement of every catalog entity; the five
  example statements of the description of `query`; the three example
  statements of the description of `execute`;
  `SELECT status, COUNT(*) FROM features WHERE product = 'PROD' GROUP BY status`;
  `SELECT COUNT(*) FROM features WHERE product = 'PROD'`;
  `SELECT key, name FROM products`;
  `SELECT id, name FROM releases WHERE product = 'PROD'`;
  `SELECT id, name FROM requirements WHERE feature = 'PROD-23'`;
  `SELECT * FROM features WHERE name CONTAINS 'x'`;
  `SELECT id, body FROM comments WHERE parent = 'PROD-23'`;
  `SELECT name FROM statuses WHERE product = 'PROD' AND entity = 'features'`;
  `DESCRIBE features`;
  `SELECT id, custom.customer FROM features WHERE id = 'PROD-23'`;
  `UPDATE features SET custom.priority = '...' WHERE id = '...'`
THEN:
  parse-statement returns a statement for each of them

### EXAMPLE: articles_and_ids_in_messages (of: error-messages)
GIVEN:
  the fixture account
WHEN:
  the tool query is called with query = `SELECT * FROM features WHERE epic = 'DEMO'`
  and error-messages writes the article before the kind
THEN:
  the tool result is an error with the message
  `'DEMO' is a product, not an epic. Use: WHERE product = 'DEMO'`
WHEN:
  the tool query is called with query = `SELECT * FROM epics WHERE id = 'DEMO-S-1'`
THEN:
  the tool result is an error with the message
  `'DEMO-S-1' is the id of an initiative, not of an epic. For the epics of that initiative: SELECT * FROM epics WHERE initiative = 'DEMO-S-1'`
WHEN:
  the tool execute is called with statement = `INSERT INTO ideas (parent) VALUES ('DEMO')`
THEN:
  the tool result is an error with the message
  `name is required to create an idea`
WHEN:
  the tool query is called with query = `SELECT * FROM products WHERE key = 'NOPE'`
THEN:
  the tool result is
  `{"entity":"products","scope":"all products","count":0,"records":[],"note":"There is no product with the key 'NOPE'."}`
WHEN:
  the tool query is called with query = `demo-3`
THEN:
  the tool result is an error with the message
  `DEMO-3 is an id, not a statement. To read that record: SELECT * FROM features WHERE id = 'DEMO-3'`
WHEN:
  the tool query is called with query = `SELECT * FROM me WHERE name = 'x'`
THEN:
  the note of the tool result ends with
  `No user has exactly the name 'x'. name = compares the whole name; to search inside names use: name CONTAINS 'x'`
WHEN:
  the tool execute is called with statement =
  `INSERT INTO comments (parent, body) VALUES ('DEMO-1-9', 'c')`
THEN:
  the tool result is an error with the message
  `there is no requirement with the id 'DEMO-1-9'. To find the id, use the query tool: SELECT id, name FROM requirements WHERE feature = 'DEMO-1'`

### EXAMPLE: example_key_is_the_default_product (of: choose-example-key)
GIVEN:
  the fixture account
  the setting default_product is `demo`
WHEN:
  an MCP client lists the tools for the first time and choose-example-key
  runs
THEN:
  the description of `query` has the line
  `id is the reference number of a record, for example 'DEMO-23'.`
  no description contains `PROD` or `OTHER`

### EXAMPLE: example_key_without_a_default_product (of: choose-example-key)
GIVEN:
  the fixture account
  the setting default_product has no value
WHEN:
  an MCP client lists the tools for the first time and choose-example-key
  runs
THEN:
  the description of `query` has the line
  `Filter by parent: product = 'PROD', release = 'PROD-R-3', feature = 'PROD-23', initiative = 'PROD-S-1'.`
  no description contains `DEMO` or `OTHER`

### EXAMPLE: example_key_is_not_a_product_of_the_account (of: choose-example-key)
GIVEN:
  the fixture account and, in addition, a product with the key PROD
  the setting default_product has no value
WHEN:
  an MCP client lists the tools for the first time and choose-example-key
  runs
THEN:
  the description of `execute` has the line
  `INSERT INTO features (parent, name) VALUES ('PRODX-R-3', 'Export to CSV')`
  no description contains `'PROD-`

### EXAMPLE: default_product_that_the_account_does_not_have (of: choose-example-key)
GIVEN:
  the fixture account
  the setting default_product is `nope`
WHEN:
  an MCP client lists the tools for the first time and choose-example-key
  runs
THEN:
  the description of `query` has the line
  `id is the reference number of a record, for example 'PROD-23'.`
  stderr contains the line
  `warning: default-product names no product of the account. Examples use the key PROD`

### EXAMPLE: example_key_is_chosen_when_it_is_needed (of: choose-example-key)
GIVEN:
  the fixture account
  the environment has AHA_SUBDOMAIN = `example` and AHA_API_KEY = `fixture-key`
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql query query="SELECT id FROM features WHERE id = 'DEMO-3'"`
THEN:
  stdout is the line
  `{"entity":"features","scope":"feature DEMO-3","count":1,"records":[{"id":"DEMO-3"}]}`
  FakeAha recorded no ApiList call on /products, because the result shows
  no example
WHEN:
  manual-mode runs for the command line
  `mcp-server-aha-aql query query="SELECT id FROM bugs"`
  and choose-example-key runs for the message
THEN:
  stderr contains the line
  `error: unknown entity 'bugs'. Entities: products, releases, features, epics, requirements, initiatives, goals, ideas, comments, users, statuses, me. Example: SELECT id, name, status FROM features WHERE release = 'PROD-R-3'`
  exit_code = 1
  FakeAha recorded exactly one call: ApiList on /products

### EXAMPLE: example_key_when_the_product_list_cannot_be_read (of: choose-example-key)
GIVEN:
  the fixture account
  the setting default_product is `demo`
  FakeAha gives no answer to an ApiList call on /products
WHEN:
  an MCP client lists the tools for the first time and choose-example-key
  runs
THEN:
  the tool list is returned, with `query` and `execute`
  the description of `query` has the line
  `id is the reference number of a record, for example 'PROD-23'.`
  stderr contains the line
  `warning: the product list could not be read. Examples use the key PROD`
  stderr does not contain `fixture-key`

## DEPENDENCIES

The only direct dependency is the MCP framework that the deployment
template selects. HTTPS, TLS and JSON come from the standard library of
the language. convert-record describes the conversion of HTML to text
completely, so that it needs no library.

The server uses no part of aha-studio: no source file, no package and no
generated client.

## DEPLOYMENT

Runtime: one static binary, `mcp-server-aha-aql`. It has three modes: an
MCP server on standard input and output, an MCP server on HTTP, and a
command that makes one tool call and ends.

Invocation:

```
mcp-server-aha-aql stdio   [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql http    [listen=HOST:PORT] [subdomain=NAME] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql query   query=STATEMENT [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql execute statement=STATEMENT [dry-run=true] [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
```

The first word:

| Word | Effect |
|---|---|
| stdio | MCP on standard input and output, as the deployment template says |
| http | MCP on HTTP, as the deployment template says |
| query | Manual mode: one call of the tool `query` |
| execute | Manual mode: one call of the tool `execute` |

The options and their defaults are in the table of load-configuration.

Where the account name and the credential come from:

| Mode | Account name | Credential |
|---|---|---|
| stdio, manual | The option `subdomain`. Without it, the variable AHA_SUBDOMAIN | The file of the option `api-key-file`. Without it, the variable AHA_API_KEY |
| http | The option `subdomain` or the variable AHA_SUBDOMAIN. Without both, the header `Aha-Subdomain` of each request | The header `Aha-Api-Key` of each request |

The settings default_product and read_only have an option and a variable
in the same way. The table of environment variables in load-configuration
lists the variables that the server uses: four for its settings, and the
variables that the standard library reads for the proxy and for the
trust store.

Manual mode at a shell. The statement is in double quotes, because AQL
writes text in single quotes:

```
mcp-server-aha-aql query query="SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'"
mcp-server-aha-aql query query=- < statement.aql
mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-23'"
```

Use with an MCP host. The host starts the server in stdio mode, and the
configuration of the host gives the settings. It can give them as
arguments of the command or as environment variables of the process. Both
ways lead to the same Account. An example for the MCP host kit, with the
credential in a file that only the user can read:

```yaml
mcpServers:
  aha:
    type: local
    command: ["mcp-server-aha-aql", "stdio", "subdomain=mycompany", "api-key-file=/home/me/.config/aha/api-key", "default-product=DEMO"]
```

The same with the credential in the environment of the server process:

```yaml
mcpServers:
  aha:
    type: local
    command: ["mcp-server-aha-aql", "stdio", "default-product=DEMO"]
    environment:
      AHA_SUBDOMAIN: mycompany
      AHA_API_KEY: "<the API key>"
```

With the second form the configuration file of the host has a secret, and
it needs the protection of a key file.

In a sandbox. A sandbox that gives a process its credentials as
environment variables needs no key file and no option: the server takes
AHA_SUBDOMAIN and AHA_API_KEY from its environment. When the sandbox
sends all traffic through a proxy, the server uses the proxy of the
variable HTTPS_PROXY. When that proxy ends the TLS connection itself, its
certificate authority has to be in the trust store, for example through
SSL_CERT_FILE. The value of AHA_API_KEY can then be a placeholder that the
sandbox replaces on the way to Aha!.

HTTP mode. The server starts without a credential, for example from the
systemd unit of the package:

```
mcp-server-aha-aql http listen=127.0.0.1:8080 subdomain=mycompany
```

Every MCP client then sends its own credential in the header
`Aha-Api-Key`. This version does not use the authorization of the MCP
specification. The http transport of the deployment template speaks plain
HTTP. On an address that is not a loopback address, a TLS endpoint in
front of the server is necessary, because every request has a credential.

Container image. The image has the certificate authorities of its build
as its trust store, because the server checks the certificate of Aha!. An
image that is built again has the current list. An operator can also
give a trust store of their own with SSL_CERT_FILE. The server in the
image listens on `127.0.0.1:8080`, the default of the deployment
template. A TLS endpoint in the same pod, or in the same network
namespace, reaches it at that address. An operator who publishes the
port of the container sets `listen=0.0.0.0:8080` in the command of the
container. The server then writes the warning G13, and the TLS endpoint
in front of it is the task of the operator.

Authentication with OAuth: this version has no OAuth flow. Aha! accepts an
OAuth2 access token wherever it accepts an API key. A token that was
obtained in another way can therefore be used as the credential.

Output streams:

| Stream | Content |
|---|---|
| standard output | The MCP messages in stdio mode. The tool result in manual mode. Not used in http mode |
| standard error | The diagnostic output: start messages, warnings, and the log lines of `execute`. In manual mode also the error line |

Exit codes:

| Code | Meaning |
|---|---|
| 0 | Manual mode: the tool returned a result. stdio and http: the server ended as the deployment template says |
| 1 | Manual mode: the statement failed with an AqlError. http: see the deployment template |
| 2 | The command line or the configuration is wrong: start messages G01 to G12 |

External services:

| Service | Address | Use |
|---|---|---|
| Aha! REST API, version 1 | `https://<subdomain>.aha.io/api/v1`, port 443 | Every read and every write. The server sends requests for no other host |
| HTTPS proxy, only when the environment gives one | The address in HTTPS_PROXY | The tunnel to Aha! |

Declarations for the deployment template. The template allows each of
these only when a specification declares it:

| Template key | Declared here |
|---|---|
| COMMAND-WORDS | The command words are `query` and `execute`. Each makes one tool call and ends, and it selects no transport (see manual-mode). Their arguments are in the table of options in load-configuration. Their output and their exit codes are in the tables of output streams and exit codes above. An unknown first word gets the message G12, which lists the two transports and then the two command words |
| CONFIG-ENV-VARS | The settings of the server come from the four AHA_ variables of the table in load-configuration, and from no other variable. An option takes precedence over its variable. The same table names the variables for the proxy and for the trust store, which the standard library of the language reads |
| TOOL-EXECUTION-ERRORS | Every AqlError of `query` and of `execute` is a tool execution error. The tool result is marked as an error and has the message as its only text, so that the language model reads it. Protocol errors and internal errors of the server are JSON-RPC error responses |

Platform: as the deployment template says. The check of the permission
bits of the key file applies on Linux and macOS.

## MILESTONE: 0.0.0
Status: pending
Scaffold: true

Included BEHAVIORs:
  query, execute, tool-registration, manual-mode, load-configuration, request-account, read-statement, describe-statement, insert-statement, update-statement, delete-statement, tokenize, parse-statement, entity-catalog, custom-fields, resolve-names, resolve-product, plan-fetch, convert-record, evaluate-condition, build-rows, check-write-values, write-request, aha-request, error-messages, choose-example-key

Acceptance criteria:
  test -x ./mcp-server-aha-aql
  ./mcp-server-aha-aql --version | grep -Eq '[0-9a-f]{64}'

## MILESTONE: 0.1.0
Status: pending

Included BEHAVIORs:
  load-configuration, aha-request, entity-catalog, error-messages, choose-example-key, resolve-product, custom-fields

Deferred BEHAVIORs:
  tokenize, resolve-names, parse-statement, plan-fetch, convert-record, evaluate-condition, build-rows, read-statement, describe-statement, check-write-values, write-request, insert-statement, update-statement, delete-statement, query, execute, tool-registration, request-account, manual-mode

Acceptance criteria:
  make test
  ./mcp-server-aha-aql bogus 2>&1 | grep -qx "error: unknown transport 'bogus'. Valid: stdio, http, query, execute"
  ./mcp-server-aha-aql bogus >/dev/null 2>&1; test $? -eq 2
  env -i ./mcp-server-aha-aql stdio < /dev/null 2>&1 | grep -qx 'error: no Aha! account. Use subdomain=NAME or set AHA_SUBDOMAIN'
  env -i ./mcp-server-aha-aql stdio subdomain=example < /dev/null 2>&1 | grep -qx 'error: no credential. Use api-key-file=PATH or set AHA_API_KEY'
  env -i ./mcp-server-aha-aql stdio bogus=1 < /dev/null 2>&1 | grep -qx "error: unknown option 'bogus' for stdio. Options: subdomain, api-key-file, default-product, read-only"
  env -i AHA_READ_ONLY=maybe ./mcp-server-aha-aql stdio subdomain=example < /dev/null 2>&1 | grep -qx 'error: AHA_READ_ONLY must be true or false'
  f=$(mktemp) && printf 'k\n' > "$f" && chmod 644 "$f" && env -i ./mcp-server-aha-aql stdio subdomain=example api-key-file="$f" < /dev/null 2>&1 | grep -q 'can be read by other users. Run: chmod 600'; r=$?; rm -f "$f"; test $r -eq 0

## MILESTONE: 0.2.0
Status: pending

Included BEHAVIORs:
  tokenize, resolve-names, parse-statement

Deferred BEHAVIORs:
  plan-fetch, convert-record, evaluate-condition, build-rows, read-statement, describe-statement, check-write-values, write-request, insert-statement, update-statement, delete-statement, query, execute, tool-registration, request-account, manual-mode

Acceptance criteria:
  make test

## MILESTONE: 0.3.0
Status: pending

Included BEHAVIORs:
  plan-fetch, convert-record, evaluate-condition, build-rows, read-statement, describe-statement

Deferred BEHAVIORs:
  check-write-values, write-request, insert-statement, update-statement, delete-statement, query, execute, tool-registration, request-account, manual-mode

Acceptance criteria:
  make test

## MILESTONE: 0.4.0
Status: pending

Included BEHAVIORs:
  check-write-values, write-request, insert-statement, update-statement, delete-statement

Deferred BEHAVIORs:
  query, execute, tool-registration, request-account, manual-mode

Acceptance criteria:
  make test

## MILESTONE: 0.5.0
Status: pending

Included BEHAVIORs:
  query, execute, tool-registration, request-account, manual-mode

Deferred BEHAVIORs:

Acceptance criteria:
  make test
  env -i AHA_SUBDOMAIN=example AHA_API_KEY=x ./mcp-server-aha-aql query query="DELETE FROM features WHERE id = 'DEMO-1'" 2>&1 | grep -qx 'error: query only reads. Use the execute tool for INSERT, UPDATE and DELETE.'
  env -i AHA_SUBDOMAIN=example AHA_API_KEY=x ./mcp-server-aha-aql query query="DELETE FROM features WHERE id = 'DEMO-1'" >/dev/null 2>&1; test $? -eq 1
  printf '%s\n' "DELETE FROM features WHERE id = 'DEMO-1'" | env -i AHA_SUBDOMAIN=example AHA_API_KEY=x ./mcp-server-aha-aql query query=- 2>&1 | grep -qx 'error: query only reads. Use the execute tool for INSERT, UPDATE and DELETE.'
  env -i AHA_SUBDOMAIN=example AHA_API_KEY=x ./mcp-server-aha-aql execute statement="SELECT id FROM features" 2>&1 | grep -qx 'error: execute only changes data. Use the query tool for SELECT and DESCRIBE.'
  env -i AHA_SUBDOMAIN=example AHA_API_KEY=x ./mcp-server-aha-aql execute read-only=true statement="DELETE FROM features WHERE id = 'DEMO-1'" 2>&1 | grep -qx 'error: execute is not available, because read-only is true (option read-only or variable AHA_READ_ONLY)'
  env -i AHA_SUBDOMAIN=example AHA_API_KEY=x ./mcp-server-aha-aql execute read-only=true statement="DELETE FROM features WHERE id = 'DEMO-1'" >/dev/null 2>&1; test $? -eq 2
  env -i AHA_SUBDOMAIN=example AHA_API_KEY=x HTTPS_PROXY=http://127.0.0.1:1 ./mcp-server-aha-aql query query="SELECT id FROM features WHERE id = 'DEMO-1'" 2>&1 | grep -qx 'error: cannot reach Aha!: the proxy did not open a connection to example.aha.io.'
  env -i AHA_SUBDOMAIN=example AHA_API_KEY=x HTTPS_PROXY=http://127.0.0.1:1 ./mcp-server-aha-aql query query="SELECT id FROM features WHERE id = 'DEMO-1'" 2>/dev/null | grep -c . | grep -qx 0
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"check","version":"0"}}}' '{"jsonrpc":"2.0","method":"notifications/initialized"}' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | env -i AHA_SUBDOMAIN=example AHA_API_KEY=x HTTPS_PROXY=http://127.0.0.1:1 ./mcp-server-aha-aql stdio 2>/dev/null | grep -q 'Change ONE record in Aha! with one AQL statement'
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"check","version":"0"}}}' '{"jsonrpc":"2.0","method":"notifications/initialized"}' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | env -i AHA_SUBDOMAIN=example AHA_API_KEY=x AHA_READ_ONLY=true HTTPS_PROXY=http://127.0.0.1:1 ./mcp-server-aha-aql stdio 2>/dev/null | grep -q 'Read from Aha! with one AQL statement'
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"check","version":"0"}}}' '{"jsonrpc":"2.0","method":"notifications/initialized"}' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | env -i AHA_SUBDOMAIN=example AHA_API_KEY=x AHA_READ_ONLY=true HTTPS_PROXY=http://127.0.0.1:1 ./mcp-server-aha-aql stdio 2>/dev/null | grep -c 'Change ONE record in Aha!' | grep -qx 0
