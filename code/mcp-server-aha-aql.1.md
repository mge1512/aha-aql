---
title: MCP-SERVER-AHA-AQL
section: 1
header: User Commands
footer: mcp-server-aha-aql 0.13.0
date: 2026-11-24
---
<!-- generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262 -->

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
Aha! product management service. A language model works with Aha! through
one query language, AQL, and two tools: **query** reads with one SELECT or
DESCRIBE statement, and **execute** changes one record with one INSERT,
UPDATE or DELETE statement. The server calls the Aha! REST API, version 1,
at https://*subdomain*.aha.io/api/v1.

The first word selects the mode:

**stdio**
: MCP over standard input and standard output.

**http**
: MCP over HTTP on the endpoint /mcp. Every request brings its credential
  in the header **Aha-Api-Key**, and without a configured subdomain also
  the header **Aha-Subdomain**. A request with an **Origin** header is
  refused with status 403.

**query**
: Manual mode: one call of the tool query. The result goes to standard
  output as one line of JSON; an error goes to standard error as one line
  that starts with "error: ".

**execute**
: Manual mode: one call of the tool execute. With **dry-run=true** the
  statement is checked completely and no write request is sent.

A value of **-** for **query** or **statement** reads the statement from
standard input.

# TOOLS

**query**
: Read from Aha! with one AQL statement. Parameter **query**: one AQL
  statement. Forms: SELECT fields FROM entity [WHERE field = 'value' [AND
  ...]] [GROUP BY field] [ORDER BY field [DESC]] [LIMIT n], and DESCRIBE
  entity. Entities: products, releases, features, epics, requirements,
  initiatives, goals, ideas, comments, users, statuses, me.

**execute**
: Change ONE record in Aha! with one AQL statement. Parameter
  **statement**: one AQL statement. Forms: INSERT INTO entity (parent,
  field, ...) VALUES ('parent id', 'value', ...); UPDATE entity SET field =
  'value', ... WHERE id = 'id'; DELETE FROM entity WHERE id = 'id'. Not
  offered when read-only is true. Each call in stdio and http mode writes
  one log line to standard error.

# OPTIONS

**subdomain=***NAME*
: The account name, the part of the address before .aha.io.

**api-key-file=***PATH*
: A file that contains the credential. Only its owner may read it
  (chmod 600). Not accepted in http mode.

**default-product=***PRODUCT*
: The key, numeric id or name of a product. Examples in tool descriptions
  and messages use its key.

**read-only=true|false**
: With true the tool execute is not offered. Default false.

**listen=***HOST:PORT*
: The address of the http transport. Default 127.0.0.1:8080. A
  non-loopback address needs a TLS endpoint in front of the server.

**dry-run=true|false**
: For execute in manual mode. Default false.

**--version**
: Print the version and the SHA256 of the specification, and exit.

# ENVIRONMENT

The server reads these variables and no other. An option takes precedence
over its variable, and the key file over AHA_API_KEY.

**AHA_SUBDOMAIN**
: The setting subdomain.

**AHA_API_KEY**
: The credential in stdio and manual mode. Not read in http mode.

**AHA_DEFAULT_PRODUCT**
: The setting default-product.

**AHA_READ_ONLY**
: The setting read-only, true or false.

**HTTPS_PROXY**, **https_proxy**, **NO_PROXY**, **no_proxy**
: The HTTPS proxy for the connection to Aha!, and the hosts without it.

**SSL_CERT_FILE**, **SSL_CERT_DIR**
: The trust store, on Linux.

# EXIT STATUS

**0**
: Manual mode: the tool returned a result. stdio and http: normal end.

**1**
: Manual mode: the statement failed. http: the address cannot be bound.

**2**
: The command line or the configuration is wrong.

# EXAMPLES

    mcp-server-aha-aql query query="SELECT id, name, status FROM features WHERE release = 'DEMO-R-3'"
    mcp-server-aha-aql query query=- < statement.aql
    mcp-server-aha-aql execute dry-run=true statement="UPDATE features SET status = 'Shipped' WHERE id = 'DEMO-23'"
    mcp-server-aha-aql http listen=127.0.0.1:8080 subdomain=mycompany

# FILES

*/usr/lib/systemd/system/mcp-server-aha-aql.service*
: The systemd unit for the http mode.

# SEE ALSO

https://github.com/mge1512/aha-aql

This program is generated from the specification aha-aql.spec.md,
sha256 9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262.
