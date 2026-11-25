<!-- generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30 -->

# mcp-server-aha-aql

mcp-server-aha-aql is an MCP server for the Aha! product management service.
MCP (Model Context Protocol) is the protocol that a language model host uses
to call tools. A language model works with Aha! through one query language,
AQL (Aha Query Language), and two tools:

- `query` reads records.
- `execute` creates, changes or deletes one record.

You can also call the two tools from a shell, without a language model.

The server calls the Aha! REST API, version 1, at
`https://<subdomain>.aha.io/api/v1`. It sends requests to no other host. The
only other host that it connects to is an HTTPS proxy from the environment.

## Installation

The server is one static binary. Packages are built on the Open Build
Service (OBS). There is no installation with `curl`.

On openSUSE and SUSE Linux Enterprise:

```
sudo zypper install mcp-server-aha-aql
```

On Fedora:

```
sudo dnf install mcp-server-aha-aql
```

On Debian and Ubuntu:

```
sudo apt install mcp-server-aha-aql
```

To build from source, you need Go 1.25.5 or later and `pandoc`:

```
make build
make man
make test
sudo make install
```

`make build` runs `go mod vendor` the first time. A package build uses the
vendor tarball (`make vendor-tarball`) and needs no network.

## Invocation

The first word of the command line selects the mode.

| Word | Effect |
|---|---|
| `stdio` | MCP on standard input and standard output |
| `http` | MCP on HTTP, endpoint `/mcp` |
| `query` | One call of the tool `query`, then the program ends |
| `execute` | One call of the tool `execute`, then the program ends |

```
mcp-server-aha-aql stdio   [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql http    [listen=HOST:PORT] [subdomain=NAME] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql query   query=STATEMENT [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql execute statement=STATEMENT [dry-run=true] [subdomain=NAME] [api-key-file=PATH] [default-product=PRODUCT] [read-only=true]
mcp-server-aha-aql --version
```

An unknown first word gets this message, and the exit code is 2:

```
error: unknown transport 'websocket'. Valid: stdio, http, query, execute
```

### stdio mode

An MCP host starts the server as a subprocess. The host gives the settings
as arguments or as environment variables. An example for a host
configuration, with the credential in a file that only you can read:

```yaml
mcpServers:
  aha:
    type: local
    command: ["mcp-server-aha-aql", "stdio", "subdomain=mycompany", "api-key-file=/home/me/.config/aha/api-key", "default-product=DEMO"]
```

The key file must not be readable by the group or by other users. If it is,
the server does not start. Run `chmod 600` on the file.

### http mode

```
mcp-server-aha-aql http listen=127.0.0.1:8080 subdomain=mycompany
```

The default listen address is `127.0.0.1:8080`. In http mode the server has
no credential of its own. Every HTTP request must have these headers:

| Header | Content |
|---|---|
| `Aha-Api-Key` | The Aha! API key (or an OAuth2 access token) for this request |
| `Aha-Subdomain` | The account name. Optional when the server has the setting subdomain |

The server refuses a request:

- with status 403 when it has an `Origin` header (a request from a web page),
- with status 400 when the credential is missing,
- with status 400 when there is no account name, or the account name has the wrong form,
- with status 403 when the account name is not the account of the server.

The http transport speaks plain HTTP. If the listen address is not a
loopback address, put a TLS endpoint in front of the server. The server then
writes a warning. The package has the systemd unit
`mcp-server-aha-aql.service` for this mode.

### Manual mode

The statement is in double quotes, because AQL writes text in single quotes:

```
mcp-server-aha-aql query query="SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'"
mcp-server-aha-aql query query=- < statement.aql
mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-23'"
```

With `query=-` or `statement=-` the server reads the statement from standard
input. With `dry-run=true` the tool `execute` checks the statement
completely and sends no write request.

The tool result goes to standard output. An error goes to standard error as
one line that starts with `error: `.

| Exit code | Meaning |
|---|---|
| 0 | The tool returned a result. In stdio and http mode: the server ended normally |
| 1 | The statement failed. In http mode: the listen address cannot be bound |
| 2 | The command line or the configuration is wrong |

## Tools

### query

Reads from Aha! with one AQL statement. It has one parameter, `query`. It
only reads (annotation readOnlyHint).

```
SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n]
DESCRIBE entity
```

Entities: products, releases, features, epics, requirements, initiatives,
goals, ideas, comments, users, statuses, me. `DESCRIBE entity` lists the
fields of an entity, also the custom fields of your account. Custom fields
are written `custom.key`.

Examples:

```
SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'
SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status
SELECT name FROM statuses WHERE product = 'DEMO' AND entity = 'features'
SELECT email FROM me
```

A result is a JSON object with `entity`, `scope`, `count`, `total` (when
rows were cut), `truncated` (when the cut was not asked for), `records` and
`note`. A read returns at most 200 rows, and 50 without a LIMIT.

### execute

Changes ONE record in Aha! with one AQL statement. It has one parameter,
`statement`. It can delete data (annotation destructiveHint). With
`read-only=true` the server does not offer this tool.

```
INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value', ...)
UPDATE entity SET field = 'value', ... WHERE id = 'id'
DELETE FROM entity WHERE id = 'id'
```

A condition after `WHERE id = '...'` is a guard: `AND status = 'In design'`.
If a guard is not true, nothing is changed. Custom fields are set with
UPDATE only.

In stdio and http mode, every call of `execute` writes one log line to
standard error:

```
2026-11-25T08:00:00Z execute account=mycompany op=update entity=features id=DEMO-1 result=ok
```

The line has no value of a field, no text of the statement and no
credential.

A statement that fails is a normal tool result. The result is marked as an
error (`isError`), and its text tells the language model what to write
instead. Protocol errors are JSON-RPC errors.

## Configuration

Every setting has an option. Some settings also have an environment
variable. An option takes precedence over its variable.

| Option | Variable | Value | Default |
|---|---|---|---|
| `subdomain` | `AHA_SUBDOMAIN` | The account name before `.aha.io` | none |
| `api-key-file` | none | The path of a file with the credential (not in http mode) | none |
| `default-product` | `AHA_DEFAULT_PRODUCT` | Key, numeric id or name of a product; examples use its key | none |
| `read-only` | `AHA_READ_ONLY` | `true` or `false`. With `true` the tool `execute` is not offered | `false` |
| `listen` | none | `host:port` for http mode | `127.0.0.1:8080` |
| `dry-run` | none | `true` or `false`, word `execute` only | `false` |
| `query` | none | The statement for the word `query`, or `-` | required |
| `statement` | none | The statement for the word `execute`, or `-` | required |

The credential comes from the key file. Without the option `api-key-file`,
it comes from the variable `AHA_API_KEY` (stdio mode and manual mode only).
No option takes the credential itself, because other users of the system can
see the arguments of a process.

The server reads these environment variables and no others:

| Variable | Use |
|---|---|
| `AHA_SUBDOMAIN` | The setting subdomain |
| `AHA_API_KEY` | The credential in stdio mode and manual mode |
| `AHA_DEFAULT_PRODUCT` | The setting default_product |
| `AHA_READ_ONLY` | The setting read_only |
| `HTTPS_PROXY`, `https_proxy` | The proxy for the connection to Aha! (read by the Go standard library) |
| `NO_PROXY`, `no_proxy` | Hosts that are reached without the proxy (read by the Go standard library) |
| `SSL_CERT_FILE`, `SSL_CERT_DIR` | The trust store on Linux (read by the Go standard library) |

The server always checks the certificate of `<subdomain>.aha.io`, also inside
the tunnel of a proxy. If a proxy ends the TLS connection itself, put the
certificate authority of the proxy into the trust store, for example with
`SSL_CERT_FILE`. No setting turns the check off.

## Container image

The `Containerfile` builds an image `FROM scratch`. The image has the
certificate authorities of its build as its trust store. The server in the
image runs in http mode on `127.0.0.1:8080`. To publish the port of the
container, set `listen=0.0.0.0:8080` in the command, and put a TLS endpoint in
front of the server.

## License

MIT. See the file `LICENSE`.
