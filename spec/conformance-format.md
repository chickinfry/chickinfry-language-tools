# Conformance corpus format

The corpus under `conformance/` is the executable definition of `.chik`. This
page fixes how a case is laid out and compared. Implementations are tested
against these files, not against prose.

## Case layout

Each case is one directory named `lower_snake_case`. The directory name
appears in test output, so it should describe the behavior being pinned.

| Group | Required files | Meaning |
| --- | --- | --- |
| `format/` | `input.chik`, `expected.chik` | Formatting the input produces exactly the expected bytes. |
| `errors/` | `input.chik`, `expected.json` | Parsing the input fails with exactly the expected diagnostics. |
| `convert/` | `input.yml`, `expected/<name>.chik` | Converting YAML produces the named `.chik` files. |
| `roundtrip/` | `input.chik` | Formatting the input returns the input unchanged. |
| `definition/` | `input.chik`, `expected.json` | Go-to-definition from a position resolves to the declarations listed. |
| `completion/` | `input.chik`, `expected.json` | Completion at a position offers at least the listed labels. |
| `invalid/` | `reason.md` | The corpus entry itself is wrong. A checker must reject it. |

`convert/` may also carry `expected/reason.md` instead of `expected/*.chik`
when the conversion must be refused. In that case the YAML is readable but
cannot be expressed in `.chik` without loss.

## Beta scope and the later groups

`conformance/` holds only cases inside the beta scope in
[beta scope](beta-scope.md). Every case there is required: an implementation
that cannot pass one is not ready.

`later/` mirrors the same four groups and holds cases for grammar the language
defines but the beta does not accept, such as a `routes` block, a named
`types` declaration, a union, or `Page<T>`. Those cases are kept so the work is
not lost. They are not part of the beta specification, and an implementation is
not required to pass them.

An implementation rejects a later-scope construct with `CHK0601` rather than
accepting it silently. That refusal is what keeps a beta file portable: a
declaration that parses today must parse in the next release too, and a
construct the beta cannot generate for must fail where the author can see it.

## File rules

Every file is UTF-8 with LF line endings and ends with exactly one newline.
No byte order mark. No CR. Tabs are not allowed except inside string literals.
Case names use `[a-z][a-z0-9_]*`.

A case directory contains no other files.

## Error output

`expected.json` is one object:

```json
{
  "diagnostics": [
    {
      "code": "CHK0101",
      "severity": "error",
      "path": "services.posts.operations",
      "range": {
        "start": { "line": 4, "column": 7 },
        "end": { "line": 4, "column": 12 }
      }
    }
  ]
}
```

- `code` is a stable identifier from [error codes](error-codes.md).
- `severity` is `error` for every case in this corpus.
- `path` is the declaration path the diagnostic belongs to. Use `""` when the
  diagnostic is about the file as a whole.
- `line` and `column` are 1-based. `column` counts Unicode code points, not
  bytes.
- `range` is half-open: `start` is included, `end` is not.
- Diagnostics are ordered by start position, then by code.

Comparison ignores the human message. Two implementations may word the same
problem differently, and a corpus that pinned wording would block an otherwise
correct replacement. Message quality is reviewed in the implementation.

## Definition output

A `definition/` case pins a cursor and what it resolves to. The same rule as
the error format applies to positions: 1-based, columns in code points.

```json
{
  "position": { "line": 7, "column": 24 },
  "targets": [
    { "line": 2, "column": 3 }
  ]
}
```

- `position` is the cursor. A client sends a zero-based UTF-16 position; the
  corpus stays in the same code-point unit as every other position, and the
  implementation converts once at its protocol boundary.
- `targets` lists the declaration starts the name resolves to. It is empty
  when the name is not declared, which is the correct answer for a typo and
  not an error.
- Order is by file, then by position. A single-file case has at most one entry
  unless the same name is declared twice.

## Completion output

A `completion/` case pins a cursor and the labels that must appear there.

```json
{
  "position": { "line": 6, "column": 20 },
  "includes": ["Post", "id", "string", "timestamp"]
}
```

- `includes` is a lower bound. A server may offer more labels, and a client
  filters as the user types, so pinning the full list would forbid a better
  completion set.
- Every listed label must appear. A missing one is a failure.

Both groups live in `conformance/` only. A definition or completion case
cannot be later scope: it asks what the parser already knows.

## Cascading errors

A lexical error makes the rest of the file unreliable: an unterminated string
swallows the following lines, so the parser then reports missing tokens that
are not the author's mistake.

For that reason a case in `errors/` pins **the first diagnostic only**. An
implementation must report that diagnostic first, at that position. It may
report further diagnostics after it, and it may stop after the first; both are
accepted. This keeps the corpus from pinning a particular recovery strategy.

A case that needs to pin a later diagnostic, such as a semantic error, uses an
input that has no earlier error.

Within that first diagnostic, a lexical problem is reported before a
syntactic one even when it starts later, because the syntactic report is its
consequence. After the lexical stage, order is start position, then code.


## Format output

`expected.chik` is compared byte for byte. The formatter is therefore fully
determined by this corpus: indentation, spacing, line breaks, and where a
trailing comma appears are all decided here.

## Conversion output

Each file under `expected/` is compared byte for byte with the same name.
A conversion that must be refused has `expected/reason.md` instead, and the
reason states which declaration cannot be expressed and why.

## Roundtrip

A `roundtrip/` case asserts that the formatter is a fixed point on its own
output. It also pins that comments survive: the input contains the comments,
and formatting must return the same bytes.

## Why input and output, not expectations in code

A case has no implementation-specific hooks, no golden-file generator, and no
version field. An implementation reads the files and compares. Adding a case
never requires editing an implementation.

## Adding a case

1. Create the directory under the matching group.
2. Add the required files.
3. Run `node scripts/check-corpus.mjs`.

A case that an implementation disagrees with is a specification question, not
a test to skip.
