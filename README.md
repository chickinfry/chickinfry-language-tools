> Draft: the grammar may change.

# Chik language

The specification and conformance corpus for `.chik`, the Chickinfry
declaration language, and the editor tooling built on it.

The Go implementation is in [`go/`](go/) under the module path
`github.com/chickinfry/chik-lang/go`. The module can be linked into the
`chik` binary, including its language server.

## Contents

| Path | What it holds |
| --- | --- |
| `go/` | The Go parser, formatter, converter, diagnostics, and language server. |
| `spec/` | The grammar, the type system, and the conformance corpus format. |
| `conformance/` | Input and expected output pairs. This corpus is the specification in executable form. |
| `scripts/` | Checks that keep the corpus well-formed. |
| `tree-sitter-chik/` | The tree-sitter grammar and its highlight queries. |
| `editors/` | The VS Code and JetBrains extensions. |

## The corpus is the specification

Any implementation of `.chik` — a parser, formatter, error reporter, or
converter — must pass every case in `conformance/` before it can replace
another. The prose in `spec/` explains intent; the corpus decides.

A case names its input, its expected output, and nothing else. Cases do not
name an implementation, a language, or a version. You can copy cases into your
own test suite to check an independent implementation.

## Scope

`.chik` describes the same application contract that `chik/*.yml` describes
today. It adds no new contract fields. Reading a declaration produces the same
normalized contract either way, so generators, the control plane, and client
SDKs are unchanged.

During the transition a project may use YAML or `.chik`. Converting YAML to
`.chik` is explicit and never silent.

## Working in this repository

```sh
node scripts/check-corpus.mjs
node --test scripts/*.test.mjs
node scripts/check-grammar.mjs
node scripts/check-editors.mjs
node scripts/check-spec-examples.mjs
node scripts/check-keywords.mjs
```

CI runs the same commands on GitHub-hosted runners. The first command checks
the corpus structure; the second checks that the checker itself rejects broken
cases. `check-grammar.mjs` parses every corpus file with the tree-sitter
grammar, and `check-editors.mjs` fails when an editor copy of the TextMate
grammar drifts from `editors/textmate/`.
`check-spec-examples.mjs` parses the `.chik` examples printed in `spec/`, so a
copyable example cannot drift from the grammar.
`check-keywords.mjs` compares the block keywords in the grammar, the highlight
queries, the TextMate grammar, and the role file table in `spec/grammar.md`.

Run the Go conformance suite from a checkout of this repository:

```sh
cd go
go test -mod=readonly ./...
```

The tests read the format, errors, conversion, roundtrip, definition, and
completion cases from the same checkout.

The grammar and the editors are built in their own CI jobs:

```sh
cd tree-sitter-chik && npm ci && npx tree-sitter test
cd editors/vscode && npm ci && npm run compile && node --test test/grammar.test.mjs
cd editors/jetbrains && ./gradlew compileKotlin
```

## Contributing

A case that an implementation disagrees with is a specification question. Open
an issue describing the input, the expected output, and what the
implementation does instead.

## Editors

`tree-sitter-chik/` is a tolerant grammar for highlighting: it decides where
tokens and blocks begin and end, and leaves name casing, unknown types, and
address overlap to the Go parser so a file with a semantic error still
highlights.

`editors/vscode/` and `editors/jetbrains/` both start the language server as
`chik lsp --stdio` from the binary. They share one TextMate grammar in
`editors/textmate/`, copied into each extension by
`node scripts/sync-editors.mjs`. Neither extension is published yet.
