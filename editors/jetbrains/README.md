# Chik

Syntax highlighting and language server support for `.chik`, the Chickinfry
declaration language.

## What you get

- Syntax highlighting for `.chik` files, from a bundled TextMate grammar.
- Diagnostics, completion, and navigation from the `chik` language server.

## Requirements

The language server is the `chik` binary, started as `chik lsp --stdio`.
Highlighting works without it. The other features need the CLI installed and
`chik` on your `PATH`, or its location set in **Settings | Tools | Chik**.

## Supported IDEs

The plugin uses the IntelliJ Platform LSP API, which the commercial
IntelliJ-based IDEs provide. It needs 2026.1 or later. IntelliJ IDEA
Community and Android Studio do not include that API, so the plugin does not
load there.

## Settings

| Setting | Meaning |
| --- | --- |
| **Settings > Tools > Chik** | Path to the `chik` binary. Empty means look on `PATH`. |

## License

Apache-2.0. The TextMate grammar is shared with the VS Code extension so both
editors highlight the same code the same way.

Report issues in the [Chik language repository](https://github.com/chickinfry/chik-lang/issues).
