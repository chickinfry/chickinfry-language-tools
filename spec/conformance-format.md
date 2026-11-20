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
| `invalid/` | `reason.md` | The corpus entry itself is wrong. A checker must reject it. |

`convert/` may also carry `expected/reason.md` instead of `expected/*.chik`
when the conversion must be refused. In that case the YAML is readable but
cannot be expressed in `.chik` without loss.

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
