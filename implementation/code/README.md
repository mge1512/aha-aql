<!-- generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262 -->
<!-- variant: us -->

# mcp-server-aha-aql

mcp-server-aha-aql is a small MCP (Model Context Protocol) server for the
Aha! product management service. A language model works with Aha! through
one query language, AQL (Aha Query Language), and two tools:

- `query` reads with one `SELECT` or `DESCRIBE` statement.
- `execute` changes one record with one `INSERT`, `UPDATE` or `DELETE`
  statement.

The same two tools can also be called from a shell (manual mode). The
server calls the Aha! REST API, version 1, directly. It has no OAuth flow:
the credential is an Aha! API key, or an OAuth2 access token that was
obtained in another way.

AQL comes from the open source project aha-studio. This server implements a
dialect of AQL and shares no code with aha-studio. The dialect differs from
the original in four points:

- `id` is the reference number of a record.
- `=` on a name is equality.
- A statement reads one entity.
- A write statement changes one record, and it finds that record by its id.

## Installation

The packages are built on the Open Build Service (OBS). There is no
installation with curl.

```
zypper install mcp-server-aha-aql      # openSUSE, SUSE Linux Enterprise
dnf install mcp-server-aha-aql         # Fedora
apt install mcp-server-aha-aql         # Debian, Ubuntu
```

The package installs the binary `/usr/bin/mcp-server-aha-aql`, the man page
`mcp-server-aha-aql(1)` and the systemd unit `mcp-server-aha-aql.service`
for the http mode.

To build from source (Go 1.24 or later, no network access needed, because
the dependencies are in `vendor/`):

```
make build        # the binary ./mcp-server-aha-aql
make test         # the test suite in independent_tests/
make man          # the man page from mcp-server-aha-aql.1.md (needs pandoc)
make install DESTDIR=/tmp/root
```

A container image is built from the `Containerfile`. It starts the http
mode on port 8080.

## Invocation

```
mcp-server-aha-aql stdio   [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql http    [listen=HOST:PORT] [subdomain=NAME] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql query   query=STATEMENT [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql execute statement=STATEMENT [dry-run=true] [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql --version
```

The first word selects the mode:

| Word | Mode |
|---|---|
| `stdio` | MCP over standard input and standard output. An MCP host starts the server as a subprocess |
| `http` | MCP over HTTP (Streamable HTTP) on the endpoint `/mcp`. The default address is `127.0.0.1:8080`; `listen=HOST:PORT` changes it |
| `query` | Manual mode: one call of the tool `query`, then the process ends |
| `execute` | Manual mode: one call of the tool `execute`, then the process ends |

Any other first word gets the message
`error: unknown transport '<word>'. Valid: stdio, http, query, execute`
and the exit code 2.

### stdio mode

An example for an MCP host configuration (here the host kit), with the
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

With the second form the configuration file of the host contains a secret.
Protect that file as you would protect a key file.

The key file must be readable by its owner only. Run `chmod 600` on it;
otherwise the server does not start.

### http mode

The server starts without a credential:

```
mcp-server-aha-aql http listen=127.0.0.1:8080 subdomain=mycompany
```

Every MCP client sends its own credential in the header `Aha-Api-Key` with
each request. Without the option `subdomain` (and without `AHA_SUBDOMAIN`)
the client also sends the header `Aha-Subdomain`. The server keeps neither
the credential nor any data that it read from Aha! after a request is
answered.

The server refuses a request before any MCP method runs:

| Status | Case |
|---|---|
| 403 | The request has an `Origin` header (a request from a web page) |
| 400 | No `Aha-Api-Key` header, or a value of the wrong form |
| 400 | An `Aha-Subdomain` header of the wrong form |
| 403 | An `Aha-Subdomain` other than the subdomain of the server |
| 400 | No subdomain in the settings and no `Aha-Subdomain` header |

The body is a JSON-RPC error with the id `null` and the code `-32600`.

The http transport speaks plain HTTP. On an address that is not a loopback
address, put a TLS endpoint in front of the server, because every request
carries a credential. The server writes a warning when `listen` is not a
loopback address.

The systemd unit of the package runs
`/usr/bin/mcp-server-aha-aql http listen=127.0.0.1:8080`. To add a
subdomain, override `ExecStart` with `systemctl edit mcp-server-aha-aql`.

### Manual mode

The statement goes in double quotes, because AQL writes text in single
quotes:

```
mcp-server-aha-aql query query="SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'"
mcp-server-aha-aql query query=- < statement.aql
mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-23'"
```

With `query=-` or `statement=-` the statement is read from standard input.
The tool result goes to standard output as one line of JSON. An error goes
to standard error as one line that starts with `error: `.

With `dry-run=true` the command `execute` checks the statement completely
(the form, the names, the id, the guards and the values). It sends read
requests for these checks and no write request, and it shows what the
statement would do.

## Tools

### query

Reads from Aha! with one AQL statement. Its only parameter is `query`
(text, required): one AQL statement. It is marked as a tool that only reads.

```
SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]
DESCRIBE entity
```

Entities: products, releases, features, epics, requirements, initiatives,
goals, ideas, comments, users, statuses, me. `DESCRIBE entity` lists the
fields of an entity, including the custom fields of the account, which are
written `custom.key`.

Examples:

```
SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'
SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status
SELECT id, name, release_date FROM releases WHERE product = 'DEMO' AND parking_lot = false ORDER BY release_date
SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'features'
SELECT email FROM me
```

The result is a JSON object with the members `entity`, `scope`, `count`,
`total` (when rows were cut), `truncated` (when the cut was not asked for),
`records` and `note`. One call returns at most 200 rows, and 50 rows
without `LIMIT`.

### execute

Changes ONE record in Aha! with one AQL statement. Its only parameter is
`statement` (text, required). It is marked as a tool that changes data and
can delete. The tool is not offered when the setting read_only is true.

```
INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)
UPDATE entity SET field = 'value', ... WHERE id = 'id'
DELETE FROM entity WHERE id = 'id'
```

`parent` is the record that the new one belongs to: a release for features
and epics, a product for releases, initiatives, goals and ideas, a feature
for requirements, and any record for comments. Further conditions after
`WHERE id = '...'` are guards: when one of them is not true for the record,
nothing is changed. Custom fields are set with `UPDATE` only.

In stdio mode and in http mode every call of `execute` writes one log line
to standard error:

```
2026-11-13T09:30:00Z execute account=example op=update entity=features id=DEMO-1 result=ok
```

The line has no field value, no text of the statement and no credential.

A statement that fails is a normal tool result that is marked as an error
(`isError`), with the message as its only text. Protocol errors and
internal errors of the server are JSON-RPC error responses.

## Configuration

Options (key=value arguments):

| Key | Value | Accepted for | Variable | Default |
|---|---|---|---|---|
| `subdomain` | The account name: the part of the address before `.aha.io` | stdio, http, query, execute | `AHA_SUBDOMAIN` | none |
| `api-key-file` | The path of a file that contains the credential | stdio, query, execute | none; without the option the credential comes from `AHA_API_KEY` | none |
| `default-product` | The key, the numeric id or the name of a product. Examples show it | stdio, http, query, execute | `AHA_DEFAULT_PRODUCT` | none |
| `read-only` | `true` or `false`. With `true` the tool `execute` is not offered | stdio, http, query, execute | `AHA_READ_ONLY` | `false` |
| `listen` | host:port for the http transport | http | none | `127.0.0.1:8080` |
| `dry-run` | `true` or `false`, see manual mode | execute | none | `false` |
| `query` | One read statement or describe statement, or `-` | query | none | required |
| `statement` | One write statement, or `-` | execute | none | required |

An option takes precedence over its environment variable, and the key file
over `AHA_API_KEY`. No option takes the credential itself, because other
users of the same system can see the arguments of a process.

Environment variables. The server reads these variables and no other:

| Variable | Use |
|---|---|
| `AHA_SUBDOMAIN` | The setting subdomain, when the option is absent |
| `AHA_API_KEY` | The credential in stdio mode and manual mode, when the option `api-key-file` is absent. It is not read in http mode |
| `AHA_DEFAULT_PRODUCT` | The setting default_product, when the option is absent |
| `AHA_READ_ONLY` | The setting read_only, when the option is absent |
| `HTTPS_PROXY`, `https_proxy` | The proxy for the connection to Aha! (HTTP CONNECT tunnel) |
| `NO_PROXY`, `no_proxy` | The hosts that are reached without the proxy |
| `SSL_CERT_FILE`, `SSL_CERT_DIR` | The trust store, on Linux |

In a sandbox that gives a process its credentials as environment variables,
the server needs no key file and no option. When the proxy of the sandbox
ends the TLS connection itself, its certificate authority must be in the
trust store, for example through `SSL_CERT_FILE`. The certificate check
cannot be turned off.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Manual mode: the tool returned a result. stdio and http: the server ended normally (end of input, SIGTERM or SIGINT) |
| 1 | Manual mode: the statement failed. http: the listen address cannot be bound |
| 2 | The command line or the configuration is wrong |

## External services

| Service | Address | Use |
|---|---|---|
| Aha! REST API, version 1 | `https://<subdomain>.aha.io/api/v1`, port 443 | Every read and every write. The server sends requests to no other host and follows no redirect |
| HTTPS proxy, only when the environment names one | The address in `HTTPS_PROXY` | The tunnel to Aha! |

## License

MIT. See the file `LICENSE` and https://opensource.org/license/mit.
