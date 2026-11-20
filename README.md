> Draft: the grammar may change.

# Chickinfry language tools

The specification and conformance corpus for `.chik`, the Chickinfry
declaration language, and the editor tooling built on it.

This repository holds the definition of the language. The Go implementation
lives in the Chickinfry CLI so that `chik` stays a single binary.

## Contents

| Path | What it holds |
| --- | --- |
| `spec/` | The grammar, the type system, and the conformance corpus format. |
| `conformance/` | Input and expected output pairs. This corpus is the specification in executable form. |
| `scripts/` | Checks that keep the corpus well-formed. |

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
node --test scripts/
```

CI runs the same commands on GitHub-hosted runners. The first command checks
the corpus structure; the second checks that the checker itself rejects broken
cases. Neither runs an implementation.

## Contributing

A case that an implementation disagrees with is a specification question. Open
an issue describing the input, the expected output, and what the
implementation does instead.

## Not here yet

The tree-sitter grammar and the editor extensions arrive with the first real
grammar. Empty directories are not committed.
