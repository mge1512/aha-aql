# aha-aql 0.3.0: notes

Status: not normative. The specification is `aha-aql.spec.md`, and a
translator does not need this file. The notes record what changed, what
was tested, what to check against Aha! itself, what the deployment
template has to allow, and the plan for OAuth as an optional later step.

Date: 2026-11-13

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
- The specification has two other deviations: the first words `query` and `execute`, and a failed statement as a tool result. They do not depend on this change.

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

- Custom fields, in reading and in writing.
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
- Aha! OAuth2: https://www.aha.io/api/oauth2
- MCP authorization, version 2025-11-25: https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization
- kit: https://github.com/mark3labs/kit
- OpenShell, providers: https://docs.nvidia.com/openshell/sandboxes/manage-providers
- OpenShell, security best practices: https://docs.nvidia.com/openshell/latest/security/best-practices
- OpenShell, repository: https://github.com/NVIDIA/OpenShell
- Deployment template `mcp-server`, version 0.3.23: in the workspace archive, directory `pcd-spec/templates`
- Test results of 2026-11-12 and analysis of 2026-11-11: in the workspace archive, directory `results`
