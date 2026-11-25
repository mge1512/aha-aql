---
name: aha-aql
description: Read and change records of Aha! through the MCP server mcp-server-aha-aql and its query language AQL. Use when the user asks about Aha! products, workspaces, releases, milestones, features, epics, ideas, goals or initiatives.
---

# Aha! with AQL

The MCP server mcp-server-aha-aql has two tools. kit shows each tool
with the server name in front, for example `aha__query`.

- `query` reads, with one SELECT or DESCRIBE statement in each call.
- `execute` changes one record, with one INSERT, UPDATE or DELETE
  statement. It is missing when the server runs read-only.

## Start

1. `SELECT key, name FROM products`. Most other lists need a product key.
2. `DESCRIBE features` lists the fields of features. `DESCRIBE` alone
   lists the entities.
3. `SELECT id, name, status FROM features WHERE product = 'KEY'`, with a
   product key from step 1 in place of KEY.

## Search by word

- In names:
  `SELECT id, name FROM features WHERE product = 'KEY' AND name CONTAINS 'word'`
- In names and descriptions, for one release (this reads every page of the list):
  `SELECT id, name FROM features WHERE release = 'KEY-R-3' AND (name CONTAINS 'word' OR description CONTAINS 'word')`

A workspace or project is a product, a milestone or version is a
release, and an objective is a goal.

## Rules

- Text goes in single quotes. Write an apostrophe inside text twice:
  `'Year''s end'`.
- Most error messages show the statement to send instead. Send it.
- A write names its record by the id, for example
  `UPDATE features SET status = 'In design' WHERE id = 'KEY-23'`.
  Ask the user before every write.
