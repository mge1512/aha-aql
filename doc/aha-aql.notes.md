# aha-aql 0.5.0: notes

Status: not normative. The specification is `aha-aql.spec.md`, and a
translator does not need this file. The notes record what changed, what
was tested, what to check against Aha! itself, what the deployment
template has to allow, and the plan for OAuth as an optional later step.

Date: 2026-11-24

## 1. What changed

### 1.1 From version 0.1.0 to version 0.2.0

Version 0.1.0 specified an amendment of aha-studio 0.12.0, as input for
upstream patches. Version 0.2.0 specifies a new MCP server,
`mcp-server-aha-aql`, that shares no code with aha-studio.

| Topic | 0.1.0 | 0.2.0 |
|---|---|---|
| Deployment | enhance-existing, on aha-studio 0.12.0 | mcp-server, a new project |
| Callers | An agent profile and an operator profile | One profile. Manual mode runs the same two tools |
| Access to Aha! | The client library of aha-studio | Own HTTPS requests to the REST API (aha-request) |
| Removed | | Operator statements, writes by predicate, custom field references, the entity tags, the default release, the offline cache, GraphQL calls, the DELTA section |
| New | | manual-mode, load-configuration, request-account, write-request, aha-request, the log line of execute, message X04, start messages G01 to G11 |
| Products | Create and update | Read only |
| The tested language | Two tools, the catalog, the read path, writes of one record | Unchanged, with the same tool descriptions and messages |

### 1.2 From version 0.2.0 to version 0.3.0

The environment is now a regular source of settings, in addition to the
options and the key file. The reason is the use in a sandbox (section 5).

| Topic | 0.2.0 | 0.3.0 |
|---|---|---|
| Variables for settings | AHA_SUBDOMAIN and AHA_API_KEY, as a deviation from the template | These two, and AHA_DEFAULT_PRODUCT and AHA_READ_ONLY. An option takes precedence over its variable |
| Way to Aha! | A direct connection only | Through the proxy of HTTPS_PROXY when the environment gives one, with NO_PROXY for exceptions |
| Trust store | The trust store of the system | The same. The specification now says that SSL_CERT_FILE and SSL_CERT_DIR can name it on Linux |
| AHA_API_KEY in http mode | Stops the start | Is not read. A warning says so |
| List of variables | In the steps | One table in load-configuration, and an invariant: the server reads these variables and no other |
| Messages | | X02 has the new reason that the proxy did not open a connection. G09 and G11 have new texts |

### 1.3 From version 0.3.0 to version 0.3.1

Three small changes after the review of the open questions on 2026-11-24
(section 3.1):

- The author line has the address mge@mailbox.org.
- The User-Agent has the address of the project, https://github.com/mge1512/aha-aql.
- load-configuration takes the mode from the first word. An unknown first word gets the new start message G12, which lists stdio, http, query and execute.

### 1.4 From version 0.3.1 to version 0.4.0

Custom fields can be read and set. The new behavior `custom-fields` reads
their definitions from the account, because every account has its own.

| Topic | 0.3.1 | 0.4.0 |
|---|---|---|
| Name | `custom.priority` was an unknown field | `custom.<key>` is the name of a custom field. The key alone works too, when no catalog name has that word |
| Entities | | features, releases, epics, requirements, initiatives, goals and ideas: read and set. products: read |
| Reading | | SELECT, WHERE, GROUP BY, ORDER BY and the aggregates. `*` shows catalog fields only |
| Setting | | UPDATE only. A value for a list with fixed options is checked against the options first |
| Types | | Text, web address, note, number, date, both choice lists, both tags fields and users fields. A scorecard can be read. Every other type gets an error that names the type |
| DESCRIBE | Fields of the catalog | Also `custom_fields`, with name, label, type and writable |
| Tool description | | One new line in `query` |
| Requests | | One list request for the definitions, only for a statement that has a custom field or an unknown field word. A list asks for custom fields only when the statement uses one |
| Messages | N02, W16, W17 | They also list the custom fields. New: N04 to N07, W31, W32, X05 |

### 1.5 From version 0.4.0 to version 0.5.0

The specification has MILESTONE sections. Nothing else changed. A
translator builds the server in six passes:

| Milestone | Behaviors | Acceptance |
|---|---|---|
| 0.0.0, scaffold | All 26, as stubs | The binary exists, and `--version` shows the hash of the specification |
| 0.1.0 | load-configuration, aha-request, entity-catalog, error-messages, choose-example-key, resolve-product, custom-fields | `make test`, and seven start errors at the command line |
| 0.2.0 | tokenize, resolve-names, parse-statement | `make test` |
| 0.3.0 | plan-fetch, convert-record, evaluate-condition, build-rows, read-statement, describe-statement | `make test` |
| 0.4.0 | check-write-values, write-request, insert-statement, update-statement, delete-statement | `make test` |
| 0.5.0 | query, execute, tool-registration, request-account, manual-mode | `make test`, eight commands in manual mode and three tool lists over stdio |

What a maintainer should know about this design:

- The order follows the rule of the milestone mode: a milestone uses only behaviors of itself or of earlier milestones. The command line (manual-mode) therefore comes last, and the milestones 0.2.0 to 0.4.0 have `make test` as their only acceptance command. That command is as strong as the tests of the translator. The examples whose heading names an included behavior are the tests to expect.
- No acceptance command needs an Aha! account or a network. The two commands that reach aha-request set HTTPS_PROXY to a local port where nothing listens. The request then fails at once, with the message for a proxy that does not open a connection. The commands start with `env -i`, so that no variable of the build environment changes the result.
- The three tool lists over stdio send the MCP handshake as three lines of JSON. If an MCP framework is strict about the protocol version of that handshake, these three commands are the first to adjust.
- The milestone 0.5.0 has an empty list under `Deferred BEHAVIORs:`, because nothing is left. RULE-15 asks for the field and RULE-16 for known names. pcd-lint decides whether it accepts the empty list.
- The skill has no milestones hints file for the template mcp-server, so the scaffold milestone has no `Hints-file:`.
- An alternative order puts query and manual-mode directly after the read path. Reading could then be tried at the command line, also against a test account, before the write path exists. It breaks the dependency rule in one point, because manual-mode also calls execute.

## 2. Evidence and its limits

The numbers come from the test of 2026-11-12 with the model lfm3, a
Python prototype of the read path and one Aha! account:

| Tool set | 50 requests | 15 held-out requests | Size of the tool definitions |
|---|---|---|---|
| The two AQL tools | 48 correct | 15 correct | 2.1 KB |
| The 74 tools of mcp-server-aha | 48 correct | 14 correct | 29.3 KB |

Limits of this evidence:

- The write statements ran against a mock, not against Aha!.
- Nothing in this list was tested: manual mode, the configuration rules, http mode, the HTTPS requests, the proxy, the request bodies, the log line.
- The prototype is a Python program that belongs to the tests. No program has been built from this specification yet.
- Custom fields were not tested at all: not with a language model and not against Aha!. Section 2.1 says what the unit rests on.

### 2.1 What the custom fields unit rests on

| Topic | Source | Status |
|---|---|---|
| The list of definitions and its members | API documentation of Aha!, OpenAPI file | Documented. The example has no page information and no request parameter |
| Names of the types | OpenAPI file (10 names in examples). GraphQL schema of Aha!, as aha-studio 0.12.0 ships it (33 names) | Documented. That NoteField, NumberField and the two SelectMultiple types have the forms of the table of kinds follows from their names |
| Forms of the entries in a record | OpenAPI file | Documented by examples for text, number, date, web address, note, scorecard, tags and users |
| Custom fields with one record | API documentation: by default only with a specific record | Documented |
| Custom fields in a list | Follows from the documentation of the parameter `fields` | Assumed. When Aha! returns a list without them, the statement fails with X05 |
| Update with `custom_fields` | API documentation, for features, requirements, ideas and others | Documented with a text value |
| Options of a list | API documentation: `GET /custom_field_definitions/:id/options` | Documented |
| Request forms for numbers, tags, users and notes | None | A guess, as agreed on 2026-11-24 |
| Custom fields when a record is created | API documentation, for ideas | Documented. Not used: the decision is UPDATE only |
| A filter `custom_fields` on the lists of features, epics and initiatives | API documentation | Documented. Not used: the server evaluates conditions itself |
| The layouts of a product with their custom fields, `api_type` and options | OpenAPI file, example of `GET /products/{id}` | Documented. Not used, see section 9 |

## 3. Decisions that the versions 0.2.0 and 0.3.0 add

Each row is a decision that the test results did not give. The right
column has the alternative.

| No. | Decision | Reason | Alternative |
|---|---|---|---|
| 1 | Settings and the credential can come from environment variables, in addition to the options and the key file. The deployment template forbids this today and will be changed (section 7) | A host or a sandbox gives values to a process in this way, and in a sandbox it is the way for the credential (section 5). The MCP specification says that a stdio server takes credentials from the environment | Options and the key file only |
| 2 | No option takes the credential itself. A key file that the group or other users can read stops the start | Arguments are visible in the process list. A key file that other users can read exposes the key | A warning in place of the refusal |
| 3 | In http mode every request brings its credential in the header `Aha-Api-Key`. The server uses no credential of its own | A server with its own credential serves everybody who reaches its port. With a credential per request, Aha! sees the key of the user who made the request | One credential for the process, on a loopback address only |
| 4 | Http mode does not use the authorization of the MCP specification. The header Authorization and the status 401 are not used | That authorization is optional. It forbids a server to pass the token of a client on to another service, so the Aha! key cannot be sent as an MCP bearer token | See section 8, step B |
| 5 | A request with an Origin header is refused | A web page in a browser is not a client of this server | A list of accepted origins |
| 6 | A statement that fails is a tool result that is marked as an error. This deviates from the wording of the template | The language model reads tool results, and the messages are written so that it can correct its statement from them | JSON-RPC errors, which many hosts do not show to the model |
| 7 | Request bodies use plain values: `workflow_status` is the status name, `assigned_to_user` the e-mail address | The client library of aha-studio 0.12.0 sends them so | The object forms `{"name": ...}` and `{"email": ...}` of the request examples of Aha! |
| 8 | Products are read only | A workspace is an administrative object, and 0.1.0 was not consistent about its key | Create and update, with a rule for the key |
| 9 | The server uses the proxy and the trust store that the environment gives. It has no option for either | A sandbox sends all traffic through a proxy that ends TLS with its own certificate authority. With a direct connection only, the server cannot reach Aha! there | A direct connection only, as in 0.2.0 |
| 10 | After the status 429 a request is sent again, at most twice. After every other failure it is not sent again | Aha! documents the limit and the header X-Ratelimit-Reset. A write that is sent twice can create two records | No repetition at all |
| 11 | The ExampleKey is chosen when it is first needed, and in http mode for each request | Http mode has no account before a request | The key of the option `default-product` without a check |
| 12 | Every call of execute writes one log line without values | A record of what a language model changed | No log |
| 13 | AHA_DEFAULT_PRODUCT and AHA_READ_ONLY are read too | They are names of aha-studio. A host configuration with AHA_READ_ONLY=true must not start a server that can write | Only the two variables for the account |
| 14 | In http mode AHA_API_KEY is not read, and a warning says so | In a sandbox the variable is in the environment of every process, so its presence is not a mistake of the operator | The start error of 0.2.0 |

### 3.1 Answers of 2026-11-24

The maintainer went through the open questions on 2026-11-24. The left
column has the number of the decision in the table above, where there is
one.

| Decision | Question | Answer |
|---|---|---|
| 3 | Http mode: where the key comes from | From each request, as specified |
| 14 | Http mode: AHA_API_KEY is set | It is ignored, with a warning, as specified |
| 6 | A failed statement | A tool result that is marked as an error, as specified. The template gets a clarification |
| | First words `query` and `execute` | They stay. The template gets a rule for further first words, and the message for an unknown word lists all four (0.3.1) |
| 7 | Values in write requests | Plain values, until a test on an account decides |
| 8 | Products | Read only |
| | Custom fields | Reading and writing are wanted (0.4.0, section 3.2) |
| | NULL in a write statement | Stays refused for now |
| 2 | A key file that other users can read | The start is refused, as specified |
| 13 | AHA_DEFAULT_PRODUCT and AHA_READ_ONLY | They stay, with strict values |
| 9 | Proxy | From the environment only |
| | Names | aha-aql and mcp-server-aha-aql |
| | License | MIT |
| | Author line | Matthias G. Eckermann, mge@mailbox.org (0.3.1) |
| | Contact in the User-Agent | https://github.com/mge1512/aha-aql (0.3.1) |
| | `### ENTITY:` subsections | The maintainer runs pcd-lint and pcd-slice and reports the findings |

### 3.2 Decisions of the custom fields unit (0.4.0)

The first six rows are the answers of the maintainer of 2026-11-24. The
other rows are choices of the session that the maintainer did not decide.
Each of them takes the cautious reading.

| No. | Decision | Reason | Alternative |
|---|---|---|---|
| 15 | The name is `custom.<key>`. The key alone is accepted when no catalog name has that word | A language model writes the key alone. `custom.` is the form without doubt, and a catalog name always wins | `custom.<key>` only |
| 16 | All record entities have custom fields. On products they can be read | Answer of the maintainer | Features only |
| 17 | Tags fields, users fields and notes can be set, in addition to text, numbers, dates, web addresses and choice lists | Answer of the maintainer. The request forms are a guess and are in the checks of section 6 | Only the types with a documented form |
| 18 | A value for a list with fixed options is checked against the options, and the error lists them | The same rule as for a workflow status. Aha! may ignore an unknown value without an error | Send the value unchecked |
| 19 | The tool description of `query` gets one line. `execute` stays as it is | The descriptions are the tested part of the language. Every further line costs context in every call | A line in both descriptions |
| 20 | A custom field is set with UPDATE only | Answer of the maintainer. INSERT stays as tested | Custom fields in INSERT. Aha! documents this for ideas |
| 21 | `*` shows catalog fields only. DESCRIBE lists custom fields in an own member | The result of `*` stays as tested, and it stays small in an account with many custom fields | `*` with an id also shows the custom fields that have a value |
| 22 | The definitions come from the list of the account, not from the layouts of a product | DESCRIBE and name resolution have no product. One request serves every statement | The layouts in the product record, section 9 |
| 23 | When Aha! cannot give the definitions to the credential (status 403 or 404), the server works as if the account had none. An unknown field word then gets the same message as in 0.3.1 | A typing error in a field name still gets the list of the valid names, as in the tested version | An error for every statement that needs the definitions |
| 24 | A list that comes without the custom fields that the request asked for is an error (X05) | Rows with empty custom fields would be a wrong answer that looks right. Reading the records one by one would break the rule of one request per page | Read each record by its id |
| 25 | A users field takes `me`, an e-mail address or a user id. `me` is sent as the id | Aha! shows the users of such a field by their ids. Whether it also takes an e-mail address is check 15 | Look up every e-mail address in the user list |

## 4. Use with an MCP host

The specification has two examples for the MCP host kit in its DEPLOYMENT
section. What the README of kit documents, and what follows from it:

| Topic | The README of kit | Consequence |
|---|---|---|
| Local server | `type: local`, a `command` list and an `environment` map | Both ways of the specification work: options in the command list, or the variables in the map |
| Remote server | `type: remote` with `url`, `noOAuth` and `tasksMode`. A key for request headers is not documented | Http mode of this server cannot be used from kit today. Use `type: local` |
| Variables in the configuration file | A substitution such as `${env://NAME}` is not documented | A key under `environment` is clear text in `~/.kit.yml`. The option `api-key-file` keeps it out of that file |
| OAuth | kit starts OAuth when a remote server answers with HTTP 401 | This server never answers 401, so kit does not start a flow that cannot succeed |

## 5. Use in a sandbox

Some sandboxes for agents give a process its credentials through the
environment and send all its traffic through a proxy. This section takes
NVIDIA OpenShell as the example. The facts come from its documentation.
Nothing of this was tested with the server.

| Topic | The documentation of OpenShell | What the server needs |
|---|---|---|
| Credentials | A provider puts them into the environment of the sandbox. The process gets a placeholder in the variable and never the value itself | The credential from a variable. A key file would put the key itself into the sandbox |
| Replacement | The proxy replaces a placeholder in header values, in query parameters and in path segments. It does not change request bodies. It has to see plain HTTP, so the policy must let it end TLS for the endpoint | The server sends the credential in the Authorization header only, and it sends the placeholder unchanged |
| Network | Every outbound connection goes through a CONNECT proxy, and nothing is allowed by default. A process that ignores the proxy variables does not get past the proxy | The proxy of the variable HTTPS_PROXY |
| TLS | The proxy ends TLS with a certificate authority of the sandbox. It gives that authority to processes through variables, and SSL_CERT_FILE is one of them | The trust store that SSL_CERT_FILE names |
| Policy | A rule has an endpoint and the programs that may use it. The quick start shows a rule that allows GET and blocks POST | A rule for `<subdomain>.aha.io`, port 443, with the binary of the server. A rule that only allows reading blocks every write at the border of the sandbox, whatever the server is told |

A provider for the key, after the generic example of the documentation:

```
openshell provider create --name aha --type generic --credential AHA_API_KEY=<the API key>
```

The account name is not a secret. It can be the option `subdomain` in the
command list of the MCP host.

To check in the version of the sandbox that is in use:

- which proxy variables the sandbox sets; the server reads HTTPS_PROXY and https_proxy
- that the placeholder has only printable characters and no space, as the type Credential requires
- whether a credential of a generic provider needs a provider profile or a binding to the endpoint
- the form of the policy rule for the endpoint and for the binary

## 6. To check on a test account

Before the first use with production data, check these points on an account
or a product that can be thrown away. Manual mode is made for this.

| No. | Check | Why |
|---|---|---|
| 1 | Every list route of the catalog accepts `fields` and `per_page=200` | Documented in general, observed on one account |
| 2 | An update with `workflow_status`, `assigned_to_user`, `release`, `initiative` and `tags` as plain values changes the record | Decision 7. If Aha! rejects or ignores a value, write-request has to use the object form |
| 3 | A release takes its description from `theme` | From the client library of aha-studio |
| 4 | The status of a goal can be written as `workflow_status` | Aha! returns it under the success metric |
| 5 | An idea can be created with POST on `/products/{product}/ideas` | aha-studio creates ideas through GraphQL |
| 6 | Plain text with line breaks in `description` and `body` is shown as written | The fields are HTML in Aha!, and the server sends text unchanged |
| 7 | The answer to an update has the changed record | The result of execute shows the values after the write |
| 8 | A wrong key gets the status 403, and the status 429 comes with X-Ratelimit-Reset | Documented, not observed |
| 9 | A record without access gets the status 404 | Documented. The messages then say that the record does not exist |

Checks for custom fields (version 0.4.0):

| No. | Check | Why |
|---|---|---|
| 10 | `GET /custom_field_definitions` answers for the key of a normal user, and with all definitions in one answer | The specification covers the statuses 403 and 404 and a list in pages. The documentation shows none of them |
| 11 | The values of `type` and `custom_fieldable_type` in the account | The table of kinds and the record types of the catalog come from documentation. The command below shows what the account has |
| 12 | A list with `custom_fields` in the parameter `fields` returns that member for every record, as an empty list where a record has no value | Decision 24. If the member is missing, every list statement with a custom field fails with X05 |
| 13 | An update with `custom_fields` next to other members changes both | Documented for the creation of an idea, not for an update |
| 14 | A number as a JSON number, tags as a JSON list of texts, and a note as plain text are stored as sent | Decision 17. These forms are a guess |
| 15 | A users field takes a list of user ids. Does it also take e-mail addresses? | Decision 25 |
| 16 | Empty text clears a text field or a choice list, and an empty list clears a tags field or a users field | Not documented. NULL cannot be written, so this is the way to remove a value |
| 17 | `GET /custom_field_definitions/{id}/options` has the member `text` for every option, in the order of the list. Are hidden options in it? | The check of a value and the message W32 use these texts |
| 18 | A value for a custom field that is not in the layout of the product | The definitions are account-wide (decision 22). Aha! shows a field on a record only when the layout has it |

A command for check 11. It only reads:

```
curl -sS -H "Authorization: Bearer $AHA_API_KEY" -H "Accept: application/json" \
  "https://$AHA_SUBDOMAIN.aha.io/api/v1/custom_field_definitions?per_page=200" \
  | jq -r '.custom_field_definitions[] | [.custom_fieldable_type, .type] | @tsv' \
  | sort | uniq -c
```

A type that the table of kinds does not have is no error. The server
refuses that field with message N05, and the table can get a row later.

## 7. What the deployment template has to allow

The template `mcp-server` in version 0.3.23 forbids environment variables
in three places: the row CONFIG-ENV-VARS of its table, one PRECONDITION and
one INVARIANT. The specification conforms when the changed template allows
these four points:

| No. | Rule | Where the specification uses it |
|---|---|---|
| 1 | Environment variables are a source of settings and of credentials, in addition to key=value options and a key file | load-configuration |
| 2 | An option takes precedence over its variable | load-configuration |
| 3 | A server that calls an external service uses the conventional variables for the proxy and for the trust store | aha-request |
| 4 | A specification lists every variable that its server reads | The table in load-configuration and one invariant |

Three more points that the template could settle at the same time:

- The template has no rule for secrets. Candidates are: no secret as a key=value argument, and a key file that only its owner can read.
- The row CONFIG-ENV-VARS also allows "preset files". The template does not say what a preset file is at run time.
- The specification has two other deviations: the first words `query` and `execute`, and a failed statement as a tool result. Both were decided on 2026-11-24, so the template needs a rule for further first words and a sentence on tool results. The file `mcp-server.template.diff` is a draft of all three changes.

## 8. OAuth as an optional later step

### 8.1 Facts

| Source | What it says |
|---|---|
| Aha! API | An API key goes into the header `Authorization: Bearer`. An OAuth2 access token is used in the same way |
| Aha! OAuth2 | Authorization: `GET https://<subdomain>.aha.io/oauth/authorize` with `client_id`, `redirect_uri` and `response_type`. Token: `POST https://<subdomain>.aha.io/oauth/token` with `code`, `client_id`, `client_secret`, `grant_type` and `redirect_uri`. The answer has `access_token` and `token_type` |
| Aha! OAuth2 | An application is registered at `secure.aha.io/oauth/applications` and gets a client ID, a client secret and a redirect URL. The page does not mention refresh tokens, an expiry time, PKCE, scopes or a state parameter |
| MCP authorization, 2025-11-25 | Authorization is optional. It is defined for HTTP transports. A stdio server takes credentials from the environment |
| MCP authorization, 2025-11-25 | A server that supports it serves protected resource metadata (RFC 9728), accepts only tokens that were issued for itself, and does not pass the token of a client on to another API |
| MCP authorization, 2025-11-25 | The authorization server needs metadata discovery. A client must use PKCE and must stop when the metadata does not announce it |
| kit | For a remote server that answers 401, kit runs dynamic client registration, PKCE and the token exchange, and it stores the tokens |

### 8.2 What the specification already has

- The Credential is a bearer value that the server does not interpret. An access token of Aha! that was obtained in another way works in every mode, without a change of the specification.
- Http mode does not use the header Authorization and the status 401. Both stay free for MCP authorization.

### 8.3 Step A: a login command for stdio mode and manual mode

A third command, `login`, runs the authorization code flow of Aha! in the
browser of the user. It listens on a loopback address for the redirect
and exchanges the code. It writes the access token to a file that only
the user can read. The server then starts with `api-key-file` on that
file.

| Topic | Content |
|---|---|
| Needs | An OAuth application that the organization registers in Aha!. Its client ID, client secret and redirect URL are settings of the command |
| Changes in the specification | One new behavior and its options. No change of query, execute, request-account and aha-request |
| Open | How long a token is valid, and how it is renewed. Whether Aha! accepts a loopback address as redirect URL. The token request needs the client secret, so the secret is on the computer of the user |
| Gain | The user needs no personal API key in a file. For one user on one computer the gain is small |

### 8.4 Step B: MCP authorization in http mode

A remote MCP host that only knows OAuth needs the server to follow the MCP
authorization. Aha! does not document metadata discovery, PKCE or client
registration, so Aha! cannot be the authorization server of an MCP client
directly. There are two ways.

| Way | Content | Consequence |
|---|---|---|
| B1: a gateway in front of the server | An existing gateway does the MCP authorization. On the way to this server it sets `Aha-Api-Key` and `Aha-Subdomain` for the user | This server stays as specified. The gateway has the credentials |
| B2: an authorization layer in the server | The server is the authorization server for MCP clients, sends the user to Aha! for the login, and keeps one Aha! token per user | The server is no longer without state: a token store, signing keys and consent per client. This is a separate specification, and probably a separate component |

### 8.5 When to do it

Nothing has to be done while a personal API key is acceptable. Step A is
the answer when a policy forbids personal API keys. Step B is the answer
when an MCP host must reach the server over HTTP and can only authenticate
with OAuth. B1 comes before B2.

## 9. Not in this version

- Custom fields in INSERT, and custom fields in the result of `*` (decisions 20 and 21).
- Custom fields of the types worksheet, attachment and checkbox, and relations to other records or to custom tables. A checkbox is the nearest candidate: the documentation has no example of its entry in a record.
- The filter `custom_fields` of Aha! on the lists of features, epics and initiatives. With it, a condition on a custom field could be applied by Aha!, and a large product would need fewer pages.
- Custom fields per layout. The product record lists the layouts of the product with their custom fields, an `api_type` and the options of choice lists. With it, DESCRIBE could show only the fields that the records of one product have. The documentation says that the list of all layouts needs an administrator; for the product record it says nothing.
- An option for the proxy, and the variable ALL_PROXY.
- JOIN, HAVING, OFFSET, subqueries and relative dates.
- Writes that find their records by a condition.
- Writes on products.
- One credential for all clients of an http server.
- MCP resources and MCP prompts.

## 10. Relation to aha-studio

- The language and its name come from aha-studio (MIT license, project grokify/aha-studio). This server uses no code of it.
- The binary of aha-studio is `mcp-server-aha`. This one is `mcp-server-aha-aql`, so both can be installed together.
- The defects of aha-studio 0.12.0 that the analysis found are in the DELTA section of version 0.1.0. They can be reported upstream as issues.
- Four variables have the names of aha-studio: AHA_SUBDOMAIN, AHA_API_KEY, AHA_DEFAULT_PRODUCT and AHA_READ_ONLY. Two of them differ in their effect. In this server AHA_DEFAULT_PRODUCT only gives the ExampleKey, and a statement still has to say its product. This server accepts `true` and `false` for AHA_READ_ONLY, and aha-studio also accepts `1` and `0`.
- The server does not read the other variables of aha-studio, for example AHA_TOOLSETS, AHA_DOMAIN and AHA_API_TOKEN.

## Sources

- Aha! API: https://www.aha.io/api
- Aha! API, custom fields: https://www.aha.io/api/resources/custom_fields/list_all_custom_fields and https://www.aha.io/api/resources/custom_fields/list_options_for_a_custom_field
- Aha! API, update of custom fields: https://www.aha.io/api/resources/features/update_a_features_custom_fields and https://www.aha.io/api/resources/requirements/update_a_requirements_custom_fields
- Aha! API, an idea with custom fields: https://www.aha.io/api/resources/ideas/create_an_idea_with_custom_fields
- Aha! API, lists of features with the filter `custom_fields`: https://www.aha.io/api/resources/features/list_features_in_a_release
- Aha! OpenAPI file: in the workspace archive, file `aha/openapi.json`
- Aha! GraphQL schema: in the workspace archive, file `aha-studio-0.12.0/vendor/github.com/grokify/aha-go/graphql/schema.graphql`
- Aha! OAuth2: https://www.aha.io/api/oauth2
- MCP authorization, version 2025-11-25: https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization
- kit: https://github.com/mark3labs/kit
- OpenShell, providers: https://docs.nvidia.com/openshell/sandboxes/manage-providers
- OpenShell, security best practices: https://docs.nvidia.com/openshell/latest/security/best-practices
- OpenShell, repository: https://github.com/NVIDIA/OpenShell
- Deployment template `mcp-server`, version 0.3.23: in the workspace archive, directory `pcd-spec/templates`
- Test results of 2026-11-12 and analysis of 2026-11-11: in the workspace archive, directory `results`
