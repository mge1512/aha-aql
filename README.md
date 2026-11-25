# aha-aql

An MCP server for Aha! with one interface: the query language AQL.

<!-- variant: us -->

## What it is

`mcp-server-aha-aql` is a small server for the Model Context Protocol
(MCP). It connects a language model to the Aha! product management
service. The model does not get one tool per API call. It gets two tools
and one language:

| Tool | Statements | Effect |
|---|---|---|
| `query` | `SELECT`, `DESCRIBE` | Reads from Aha! |
| `execute` | `INSERT`, `UPDATE`, `DELETE` | Changes one record |

The language is AQL (Aha Query Language), a small SQL dialect:

```sql
SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'
SELECT status, COUNT(*) FROM features WHERE product = 'DEMO' GROUP BY status
DESCRIBE features
UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-23'
```

AQL knows twelve entities: products, releases, features, epics,
requirements, initiatives, goals, ideas, comments, users, statuses and me.
`DESCRIBE` lists the fields of an entity, including the custom fields of
the account.

The server calls the Aha! REST API, version 1, and no other host. The
credential is an Aha! API key.

This project is not affiliated with the makers of Aha!.

## How to use it

### Build

The code is in `implementation/code`. It needs Go 1.24 or later.

```sh
cd implementation/code
go mod vendor
make build
```

The result is the binary `mcp-server-aha-aql`.

### Run

The first word selects the mode:

| Word | Mode |
|---|---|
| `stdio` | MCP over standard input and output. An MCP host starts the server |
| `http` | MCP over HTTP on `/mcp`, by default on `127.0.0.1:8080` |
| `query` | One call of the tool `query` from a shell |
| `execute` | One call of the tool `execute` from a shell |

Put the API key in a file that only you can read, and try a statement from
the shell:

```sh
chmod 600 ~/.config/aha/api-key
mcp-server-aha-aql query subdomain=mycompany api-key-file="$HOME/.config/aha/api-key" \
    query="SELECT id, name FROM products"
mcp-server-aha-aql execute dry-run=true subdomain=mycompany api-key-file="$HOME/.config/aha/api-key" \
    statement="UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-23'"
```

`subdomain` is the part of your Aha! address before `.aha.io`. With
`dry-run=true` the server checks the statement and sends no write request.

For an MCP host, add the server to its configuration. An example in the
form of the host kit:

```yaml
mcpServers:
  aha:
    type: local
    command: ["mcp-server-aha-aql", "stdio", "subdomain=mycompany",
              "api-key-file=/home/me/.config/aha/api-key", "default-product=DEMO"]
```

In http mode the server starts without a credential, and every client
sends its own key in the header `Aha-Api-Key`:

```sh
mcp-server-aha-aql http listen=127.0.0.1:8080 subdomain=mycompany
```

### Options

| Option | Environment variable | Meaning |
|---|---|---|
| `subdomain` | `AHA_SUBDOMAIN` | The Aha! account |
| `api-key-file` | `AHA_API_KEY` (the key itself) | The credential, for stdio and shell use |
| `default-product` | `AHA_DEFAULT_PRODUCT` | The product that examples in tool descriptions show |
| `read-only=true` | `AHA_READ_ONLY` | The tool `execute` is not offered |

No option takes the key itself, because other users of a system can see
the arguments of a process. The full reference is in
[`implementation/code/README.md`](implementation/code/README.md) and in
the man page next to it.

## Why we do it

A server that maps an API call by call offers dozens of tools. Every tool
description uses context, and a model has to choose among all of them. A
small model that runs on your own hardware does this badly.

One query language has these effects:

- Two tools fit into the context of a small local model. Your product
  data can stay in your network and still be useful to a model.
- Models already know SQL. A statement that is wrong gets a message that
  says what to write instead, so the model can correct itself.
- One statement reads what would otherwise need several tool calls.
- Writes are narrow. A write statement changes one record, and it finds
  that record by its id. Further conditions are guards. `read-only=true`
  removes the write tool, and `dry-run=true` shows what a statement would
  do.
- The server keeps no data from Aha! between statements, and in http mode
  it keeps no credential.

The second reason is the method. This repository is a project of
[Post-Coding Development (PCD)](https://github.com/mge1512/pcd). The
specification is the source of truth, and a language model translates it
into code. When the code is wrong, we change the specification and
translate again.

## Where the idea for AQL comes from

AQL comes from the open source project
[aha-studio](https://github.com/grokify/aha-studio) (MIT license). Its
MCP server offers AQL as one of its tools, next to tools for single API
calls. Thank you to its author.

This server takes the idea and the name of the language. It shares no
code with aha-studio, and it implements a dialect that differs in four
points:

- `id` is the reference number of a record, for example `DEMO-23`.
- `=` on a name is equality.
- A statement reads one entity.
- A write statement changes one record, and it finds that record by its
  id.

Four environment variables have the names that aha-studio uses, and the
binary has another name (`mcp-server-aha-aql`), so both servers can be
installed together.

## Repository layout

| Directory | Content |
|---|---|
| `specification/` | The specification `aha-aql.spec.md`, the source of truth |
| `documentation/` | The changelog of the specification and the review report of the first implementation |
| `pcd-artefacts/` | The deployment template and the library hints for the next translation |
| `implementation/` | The first implementation (`code/`) and the inputs it was translated from (`spec/`) |

## Status

- The specification is at version 0.20.0.
- The code in `implementation/code` is the translation of version 0.13.0.
  Its 187 tests pass against a test double of Aha!. The review found 30
  points, and the versions 0.14.0 to 0.20.0 answer most of them. See
  [`documentation/REVIEW_REPORT.md`](documentation/REVIEW_REPORT.md).
- The next step is a new translation of the current specification.
- Use it on a test product first. Write statements change data in Aha!.

## License

MIT. See [`implementation/code/LICENSE`](implementation/code/LICENSE).
