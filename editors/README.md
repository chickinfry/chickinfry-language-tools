# Editors

Syntax highlighting and language server wiring for `.chik` in VS Code and
JetBrains IDEs.

Both extensions start the language server as `chik lsp --stdio`. The server
ships inside the `chik` binary, so neither extension bundles a parser or a
Node runtime for the server.

| Directory | Editor |
| --- | --- |
| [`vscode/`](vscode/) | Visual Studio Code 1.91 and later, and compatible editors that use Open VSX. |
| [`jetbrains/`](jetbrains/) | IntelliJ Platform 2026.1 and later, with the commercial LSP API. |

The shared TextMate grammar is `textmate/chik.tmLanguage.json`. VS Code reads
it from the extension's `package.json`. The JetBrains plugin ships it as a
VS Code-format bundle, which the TextMate Bundles plugin reads natively.

See [packaging and publishing](PUBLISHING.md) to build the installable
packages or release them to the marketplaces.

