# Chik

Syntax highlighting and language server support for `.chik`, the Chickinfry
declaration language.

## What you get

- Syntax highlighting for `.chik` files: comments, strings, block keywords,
  type names, operations, fields, and HTTP methods.
- Diagnostics, completion, and go-to-definition from the `chik` language
  server, when it is installed.

## Requirements

The extension starts the language server from the `chik` binary as
`chik lsp --stdio`. Highlighting works without it. The other features need it:

1. Install the Chickinfry CLI so `chik` is on your `PATH`.
2. Or set `chik.server.path` to the binary.

## Settings

| Setting | Meaning |
| --- | --- |
| `chik.server.path` | Path to the `chik` binary. Empty means look on `PATH`. |

## Scope

The extension covers the beta surface of the language. Grammar that a later
release adds is still highlighted, but the parser refuses it for now.

## License

Apache-2.0. The grammar is shared with the JetBrains plugin so both editors
highlight the same code the same way.
