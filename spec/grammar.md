# The `.chik` grammar

This page states the grammar an implementation must accept. It follows the
approved D1 draft and the decisions that settled the open questions. Where a
draft and a later decision disagree, the decision wins and this page records it.

## Files

A project declares up to five role files under `chik/`:

| File | Declares |
| --- | --- |
| `api.chik` | Types, services, and routes. |
| `schema.chik` | The database profile and tables. |
| `auth.chik` | Authentication policy. |
| `project.chik` | Project name, generation targets, storage, vault, realtime, and the control-plane key. |
| `clients.chik` | Named client applications. |

Every file is UTF-8. A file may use either line ending, and a formatter
normalizes to LF.

## Lexical rules

- Whitespace is space, tab, CR, and LF.
- Line comments start with `//` and run to end of line.
- Block comments start with `/*` and end with `*/`. Nested block comments
  are not allowed.
- Comments are preserved. They are not dropped by the formatter.
- Identifiers match `[A-Za-z_][A-Za-z0-9_]*`.
- Strings use double quotes and JSON escapes. `\u` escapes are accepted.
- Keywords are not reserved: a field may be named `type` or `service`.
  Position decides meaning.

## API grammar

```ebnf
api        = [types], [services], [routes] ;
types      = "types", "{", {type_decl, [semi]}, "}" ;
services   = "services", "{", {service}, "}" ;
routes     = "routes", "{", {route, [semi]}, "}" ;

type_decl  = TypeName, "=", (record | union) ;
service    = name, "{", "operations", "{", {operation, [semi]}, "}", "}" ;
operation  = ["public"], name, "(", [fields], ")", "->", record, ["at", method, path] ;
route      = name, method, path ;

fields     = field, {",", field}, [","] ;
field      = name, ":", type ;
record     = "{", [fields], "}" ;
union      = "union", "(", name, ")", "{", variant, {variant, [semi]}, "}" ;
variant    = name, ":", record ;

type       = primary, {"[]" | "?"} ;
primary    = "id" | "string" | "timestamp" | TypeName | record
           | "map", "<", "string", ",", type, ">"
           | "Page", "<", type, ">"
           | "json" ;

method     = "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS" ;
path       = string ;
semi       = ";" ;
```

`types`, `services`, and `routes` may each appear at most once per file.

`->` separates an operation's parameters from its result. It is not a field
colon: `title: string` declares a field, `get(...) -> ...` declares what an
operation returns.

## Separators

The separators follow the Go style the decision fixed:

- Inside a block, a line break ends an entry. Nothing else is required.
- A construct may span lines inside `{ }`, `( )`, or `[ ]`. That is the only
  place a line break does not end the construct.
- Several items on one line separate with commas.
- A semicolon is accepted where an entry ends. It is not part of the canonical
  form: the formatter removes it, as `gofmt` does.
- A trailing comma is accepted in a list, an object, or a parameter list. The
  formatter keeps it when the construct spans lines and removes it when the
  construct stays on one line.

A comment line is not an item, so it takes no trailing comma.

## Types

| Type | Meaning |
| --- | --- |
| `id` | A UUID. |
| `string` | A UTF-8 string. |
| `timestamp` | An RFC 3339 instant. |
| `TypeName` | A declared record or union. |
| `{ ... }` | An inline record. |
| `map<string, T>` | A map with string keys. |
| `Page<T>` | A page of `T`. Output only. |
| `json` | Any JSON value except `null`. |

### Modifiers

`T?` means the field is present and its value may be null. It does not mean
the field may be omitted. An omitted required field is an error.

`optional` written after a type means the caller may leave the field out. It is
a separate marker from `?`, and the two combine: `string? optional` is a
nullable field the caller may also omit. `optional` is a word in that position,
not a reserved keyword, so a field may be named `optional`.

`optional` applies to input fields only. An output field is always present, so
`optional` there is `CHK0303`.

`T[]` is a list. `string?[]` is a list of nullable strings.
`string[]?` is a nullable list. `T[][]` is a list of lists. `T??` is an
error.

### Records

A record declares fields. A record used as a named type, a union variant, or a
list element must declare at least one field. An operation input or output may
be an empty record `{}`.
A record may be an operation parameter, an output field, a field within
another record, or a list element. An object field preserves its own name and
contains its declared fields.

### Unions

`union(kind)` produces a tagged union on the wire. Each variant contributes a
`kind` field whose value is the variant name:

```json
{"kind": "text", "text": "hello"}
```

A union has at least two variants. Tags are unique. A variant may not declare a
field named `kind`. An unknown tag is rejected; the implementation never
guesses a variant from field shape.

### Named types

A named type may refer forward. Direct and mutual recursion are rejected. Type
aliases, inheritance, and user generics are not part of the language.

### Page

`Page<T>` is a builtin that expands to `{ items: T[], next_cursor: string }`
when generated. The end of a page is `next_cursor: ""`. `Page` is allowed only
as a direct operation output field.

## Services and operations

```text
services {
  posts {
    operations {
      public get(id: id) -> { id: id, title: string } at GET "/v1/posts/{id}"
      publish(id: id) -> { id: id }
      list() -> { rows: Post[] }
    }
  }
}
```

An operation without `public` is protected when the project uses
authentication. In a project without authentication, `public` is an error.

Names: type names are `UpperCamelCase`. Service, operation, field, route, and
union tag names are `lower_snake_case`.

### Addresses

`at METHOD "path"` replaces the default `POST /service.operation` address. It
does not add a second address.

The allowed methods for an addressed operation are `GET`, `POST`, `PUT`,
`PATCH`, and `DELETE`. `HEAD` and `OPTIONS` are allowed only on routes.

A path starts with `/`. Each `{name}` is a whole segment and must match an
input of type `id`, `string`, or `timestamp` that is required and not
nullable. Host, query, fragment, wildcard, regex, empty segments, dot segments,
duplicate slashes, and a trailing slash other than the root are rejected.

### Input placement

For `GET` and `DELETE`, fields other than path placeholders are sent as query
parameters. An empty string is sent as `name=`; a missing value is different
and is rejected.

For `POST`, `PUT`, and `PATCH`, fields other than path placeholders are sent as
a JSON body. A path placeholder repeated in the body, or an unexpected query
parameter, is a 400 at request time.

## Routes

```text
routes {
  payment_webhook POST "/hooks/payment"
  feed GET "/feed.xml"
}
```

A route declares a name, a method, and a path. It has no typed input or output
and no `public` selector. It receives the same context as an operation, with a
nullable session. A route is never automatically rejected for a missing
session.

Allowed methods are `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, and
`OPTIONS`.

## Overlap

Every address, operation or route, is checked in one table. Two addresses
overlap when one can match a request the other also matches. Examples that are
rejected:

- `/posts/{id}` and `/posts/new`
- `/posts/{id}` and `/posts/{slug}`
- `/{x}/b` and `/a/{y}`

Different methods do not overlap. There is no automatic `HEAD` fallback. An
overlap reports both declarations with file, line, column, method, and path.
Registration order never resolves a conflict.

## Comments and formatting

The formatter preserves comments and blank-line structure where it can. The
conformance corpus decides the exact bytes; see
[conformance format](conformance-format.md).

## What is not in the language

- A top-level `version` or `format` selector.
- Executable expressions, environment substitution, or imports.
- User-defined generics, type aliases, or inheritance.
- Numeric types beyond `id`, `string`, and `timestamp`.

Adding any of these is a new decision, not an implementation detail.
