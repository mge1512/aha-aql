% MCP-SERVER-AHA-AQL(1) mcp-server-aha-aql 0.23.0 | User Commands
% Matthias G. Eckermann <mge@mailbox.org>
% November 2026

<!-- generated from spec: aha-aql.spec.md sha256:b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268 -->

# NAME

mcp-server-aha-aql - MCP server for Aha! with the query language AQL

# SYNOPSIS

**mcp-server-aha-aql stdio** [subdomain=*NAME*] [api-key-file=*PATH*] [default-product=*PRODUCT*] [read-only=true]

**mcp-server-aha-aql http** [listen=*HOST:PORT*] [subdomain=*NAME*] [default-product=*PRODUCT*] [read-only=true]

**mcp-server-aha-aql query** query=*STATEMENT* [subdomain=*NAME*] [api-key-file=*PATH*] [default-product=*PRODUCT*] [read-only=true]

**mcp-server-aha-aql execute** statement=*STATEMENT* [dry-run=true] [subdomain=*NAME*] [api-key-file=*PATH*] [default-product=*PRODUCT*] [read-only=true]

**mcp-server-aha-aql --version**

# DESCRIPTION

**mcp-server-aha-aql** is an MCP (Model Context Protocol) server for the
Aha! product management service. A language model reads and changes Aha!
records through one query language, AQL, and two tools. The tool **query**
reads records. The tool **execute** creates, changes or deletes one record.

The server calls the Aha! REST API at https://*subdomain*.aha.io/api/v1. It
sends requests to no other host. The only other host that it connects to is
the HTTPS proxy of the environment.

# MODES

The first word of the command line selects the mode.

**stdio**
:   MCP over standard input and standard output. The process works with one
    Aha! account.

**http**
:   MCP over HTTP on the endpoint /mcp. Every request brings its credential
    in the header **Aha-Api-Key**, and its account name in the header
    **Aha-Subdomain** when the server has no setting subdomain. A request
    with an **Origin** header is refused with status 403. The server keeps
    no MCP session: no answer has the header **Mcp-Session-Id**.

**query**
:   Command word. One call of the tool query, then the program ends. Selects
    no transport.

**execute**
:   Command word. One call of the tool execute, then the program ends.
    Selects no transport.

Another first word gives the message
"error: unknown transport '*word*'. Valid: stdio, http, query, execute"
and the exit code 2.

# OPTIONS

All options have the form *key*=*value*.

**subdomain**=*NAME*
:   The account name: the part of the address before .aha.io. Variable:
    **AHA_SUBDOMAIN**.

**api-key-file**=*PATH*
:   The path of a file that has the credential. Only the owner may read the
    file. Not in http mode.

**default-product**=*PRODUCT*
:   The key, the numeric id or the name of a product. Tool descriptions and
    examples use its key. Variable: **AHA_DEFAULT_PRODUCT**.

**read-only**=true|false
:   With true the tool execute is not offered. Variable: **AHA_READ_ONLY**.

**listen**=*HOST:PORT*
:   The listen address in http mode. Default: 127.0.0.1:8080. A warning is
    written when the host is not a loopback address.

**dry-run**=true|false
:   Word execute only. The statement is checked completely, and no write
    request is sent.

**query**=*STATEMENT*
:   The statement for the word query. With **-** the statement is read
    from standard input.

**statement**=*STATEMENT*
:   The statement for the word execute. With **-** the statement is read
    from standard input.

**--version**
:   Writes the version and the SHA256 of the specification, then ends.

An option takes precedence over its environment variable. No option takes
the credential itself.

# TOOLS

**query**
:   One parameter, *query*: one AQL statement.
    SELECT fields FROM entity [WHERE field = 'value' [AND ...]] [GROUP BY
    field] [ORDER BY field [DESC]] [LIMIT n], or DESCRIBE entity. Entities:
    products, releases, features, epics, requirements, initiatives, goals,
    ideas, comments, users, statuses, me. Custom fields are written
    custom.key.

**execute**
:   One parameter, *statement*: one AQL statement.
    INSERT INTO entity (parent, field, ...) VALUES ('parent id', 'value',
    ...), UPDATE entity SET field = 'value', ... WHERE id = 'id', or DELETE
    FROM entity WHERE id = 'id'. In stdio and http mode each call writes one
    log line to standard error.

The answers to *initialize* and *server/discover* have server instructions.
The one MCP prompt, **start** ("How to start with AQL"), has no argument and
gives a short guide to the first statements.

# ENVIRONMENT

**AHA_SUBDOMAIN**
:   The setting subdomain.

**AHA_API_KEY**
:   The credential in stdio mode and manual mode, when the option
    api-key-file is absent. Not read in http mode.

**AHA_DEFAULT_PRODUCT**
:   The setting default_product.

**AHA_READ_ONLY**
:   The setting read_only: true or false.

**HTTPS_PROXY**, **https_proxy**, **NO_PROXY**, **no_proxy**
:   The proxy for the connection to Aha!. The Go standard library reads them.

**SSL_CERT_FILE**, **SSL_CERT_DIR**
:   The trust store on Linux. The Go standard library reads them.

# EXIT STATUS

**0**
:   Manual mode: the tool returned a result. stdio and http: the server ended
    normally, after end of input, SIGTERM or SIGINT.

**1**
:   Manual mode: the statement failed; the message is on standard error.
    http: the listen address cannot be bound.

**2**
:   The command line or the configuration is wrong.

# EXAMPLES

    mcp-server-aha-aql query query="SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'"
    mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-23'"
    mcp-server-aha-aql http listen=127.0.0.1:8080 subdomain=mycompany

# SEE ALSO

https://github.com/mge1512/aha-aql

# SPECIFICATION

This program is a translation of the specification aha-aql 0.23.0, with
the SHA256 b5c2ed33da65d1486080b50fd0a29bd64fef4f69ec258e8971393437c2fd7268.
