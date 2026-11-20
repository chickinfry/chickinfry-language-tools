# Concrete syntax tree

The parser produces a concrete syntax tree (CST), not an abstract one. Every
byte of the source is reachable from the tree: comments, blank lines, and the
exact spelling of a number or string all survive. The formatter, the language
server, and the console editor all work on this tree.

## The one requirement that decides the design

Serializing a parsed tree must return the original bytes, exactly. That single
requirement forces trivia into the tree instead of a side channel, and it is
what the `roundtrip/` corpus group measures.

## Node

```text
Node {
  kind    : Kind
  range   : Range        // half-open, 1-based, columns in code points
  children: [Node]       // ordered as in the source
  tokens  : [Token]      // only for leaf kinds
  trivia  : { leading: [Trivia], trailing: [Trivia] }
  id      : string       // stable identity, see below
}
```

A node either has children or tokens, never both. A leaf carries the tokens it
owns, including punctuation.

## Kinds

| Kind | Children | Notes |
| --- | --- | --- |
| `File` | blocks | One per role file. |
| `ProjectBlock` | decls | |
| `SchemaBlock` | decls | |
| `ApiBlock` | decls | |
| `ClientsBlock` | decls | |
| `AuthBlock` | decls | |
| `ExtensionsBlock` | decls | |
| `TypesBlock` | type decls | Later release; parsed but rejected by beta. |
| `ServicesBlock` | services | |
| `RoutesBlock` | routes | Later release. |
| `Service` | operations | |
| `Operation` | params, result | |
| `Param` | type | |
| `Result` | fields | |
| `Field` | type | |
| `TypeRef` | — | Leaf: `id`, `string`, `timestamp`, a name. |
| `TypeList` | element | `T[]`. |
| `TypeNullable` | inner | `T?`. |
| `TypeRecord` | fields | `{ ... }`. |
| `Table` | columns, keys | |
| `Column` | type decl | |
| `Error` | — | Recovery node, see below. |

Kinds are additive: a new language feature adds a kind. An implementation that
does not know a kind must reject the file, not ignore the node.

## Trivia

```text
Trivia { kind: Comment | Whitespace | Newline, text: string, range: Range }
```

Attachment rules, applied in this order:

1. Trivia on the same line after a node is that node's trailing trivia.
2. Trivia after that, up to the next node, is the next node's leading trivia.
3. Trivia at end of file belongs to `File`.

These three rules are what make formatting idempotent: a formatter rewrites
only the whitespace it owns and leaves comments attached where they were.

## Ranges

A range is half-open: `start` is included, `end` is not. Lines and columns are
1-based, and a column counts Unicode code points, matching the error format.
A node's range covers its first token through the end of its last token, and
excludes leading trivia. Trailing trivia is excluded too.

## Stable identity

`id` is a path from the file root, built from kinds and declaration names:

```text
File/ApiBlock/ServicesBlock/Service[posts]/Operation[get]/Param[id]
```

An index suffix disambiguates repeated unnamed children:

```text
File/ApiBlock/ServicesBlock/Service[posts]/Operation[get]/Result/Field[title]
```

An `id` survives formatting and comment edits, because neither changes kinds
or names. It does not survive a rename, which is what the editor wants: a
rename is a new identity, and a merge treats it as a delete plus an add.

## Error recovery

A parser that fails still returns a tree. The failing region becomes an
`Error` node holding the tokens it consumed, so the rest of the file is parsed
and every diagnostic has a range. An implementation must not stop at the first
error.

## What the tree is not

- Not an AST. There is no normalized form here; normalization is a later step.
- Not a document model with undo. History belongs to the editor.
- Not a wire format. The tree is an in-memory contract between components of
  one implementation; nothing here is published to customers.

## How this contract is tested

Two corpus groups:

| Group | What it proves |
| --- | --- |
| `roundtrip/` | Parse then serialize returns the input bytes. |
| `format/` | Formatting a parsed tree produces the expected bytes. |

A CST implementation that cannot round-trip fails `roundtrip/` before it can
pass `format/`, because formatting starts from a parsed tree.
