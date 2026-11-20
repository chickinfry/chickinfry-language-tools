# Editors

Syntax highlighting and language server wiring for `.chik` in VS Code and
JetBrains IDEs.

Both extensions start the language server as `chik lsp --stdio`. The server
ships inside the `chik` binary, so neither extension bundles a parser or a
Node runtime for the server.

| Directory | Editor |
| --- | --- |
| `vscode/` | Visual Studio Code 1.91 and later. |
| `jetbrains/` | IntelliJ IDEA and other commercial IntelliJ IDEs. |

The shared TextMate grammar is `textmate/chik.tmLanguage.json`. VS Code reads
it from the extension's `package.json`. The JetBrains plugin ships it as a
VS Code-format bundle, which the TextMate Bundles plugin reads natively.

Neither extension is published. See the repository report for what a
publisher needs.

