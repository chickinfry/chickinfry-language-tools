# Error presentation

A declaration error is shown the way a Rust compiler shows one: a number, a
file and position, the source line, an underline, a help line, and the
corrected shape where one is obvious. The same content is available as JSON so
an agent can read it and fix the file without parsing the human form.

## Human form

```text
error[CHK0007]: unknown type `Post`
  --> chik/api.chik:4:21
   |
 4 |       get(id: id) -> { post: Post };
   |                            ^^^^ not declared
   |
   = help: declare `types { Post = { ... }; }` or use a built-in type
   = note: built-in types are id, string, and timestamp
```

The parts:

| Part | Rule |
| --- | --- |
| `error[CHK0007]` | Severity and number. Numbers are stable and never reused. |
| Message | One line, lower case, no trailing period. |
| `--> file:line:column` | Column counts Unicode code points, 1-based. |
| Gutter | Source line with its number, aligned. |
| Underline | `^` under the range, with a short label when it adds meaning. |
| `= help:` | What to do. At most two lines. |
| `= note:` | Extra context, optional. |
| `= suggested:` | A corrected line or block, when the fix is unambiguous. |

Long ranges that span lines underline from the start position to the end of
the first line, then show the remaining lines without underlines.

## Numbering

Numbers are `CHK` followed by four digits, assigned once. A code that stops
being produced keeps its number reserved so an old message never changes
meaning. The list is in [error codes](error-codes.md).

## JSON form

`chik check --format json` prints one object:

```json
{
  "version": 1,
  "file": "chik/api.chik",
  "diagnostics": [
    {
      "code": "CHK0007",
      "severity": "error",
      "message": "unknown type `Post`",
      "range": {
        "start": { "line": 4, "column": 21 },
        "end": { "line": 4, "column": 25 }
      },
      "path": "services.posts.operations.get.output.post",
      "source_line": "      get(id: id) -> { post: Post };",
      "help": "declare `types { Post = { ... }; }` or use a built-in type",
      "notes": ["built-in types are id, string, and timestamp"],
      "suggestion": null
    }
  ]
}
```

Fields:

| Field | Rule |
| --- | --- |
| `version` | `1`. Changes only if this shape breaks. |
| `file` | Repository-relative path. |
| `code` | The same number as the human form. |
| `severity` | `error` in this release. |
| `message` | The same line as the human form. |
| `range` | Half-open, 1-based, columns in code points. |
| `path` | Declaration path, `""` for the whole file. |
| `source_line` | The line the range starts on, without a trailing newline. |
| `help` | One string or null. |
| `notes` | Array, possibly empty. |
| `suggestion` | `{ "text": "..." }` or null. Never a diff. |

Diagnostics are ordered by start position, then by code.

## `chik explain`

```text
$ chik explain CHK0007
CHK0007 — unknown type

A type name appears in a field, a list element, or a map value, but no
`types` block declares it and it is not a built-in.

Built-in types are `id`, `string`, and `timestamp`.
```

`chik explain` with no argument lists every number with its one-line summary.
An unknown number exits non-zero and says so.

## Comparison rules

The conformance corpus compares `code`, `path`, and `range` only. Message,
help, notes, and suggestion are reviewed in the implementation, because wording
will differ between implementations and pinning it would block a correct
replacement.

## Why numbers

A number survives translation and rewording. A user can search for it, an
agent can key on it, and documentation can link to it. A bare message cannot do
any of those.
