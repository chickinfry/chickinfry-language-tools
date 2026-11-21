// Tokenizes a sample with the TextMate grammar and checks the scopes. A
// grammar with a broken regex still loads, so tokenizing is the only way to
// know the file highlights.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { test } from "node:test";
import vsctm from "vscode-textmate";
import oniguruma from "vscode-oniguruma";

const require = createRequire(import.meta.url);
const root = fileURLToPath(new URL("..", import.meta.url));
const grammarPath = root + "syntaxes/chik.tmLanguage.json";

const wasm = readFileSync(require.resolve("vscode-oniguruma/release/onig.wasm")).buffer;
await oniguruma.loadWASM(wasm);

const registry = new vsctm.Registry({
  onigLib: Promise.resolve({
    createOnigScanner: (patterns) => new oniguruma.OnigScanner(patterns),
    createOnigString: (text) => new oniguruma.OnigString(text),
  }),
  loadGrammar: async (scopeName) => {
    if (scopeName !== "source.chik") return null;
    return vsctm.parseRawGrammar(readFileSync(grammarPath, "utf8"), grammarPath);
  },
});

const grammar = await registry.loadGrammar("source.chik");
assert.ok(grammar, "the grammar must load for source.chik");

// Returns the scope of the token that starts at the first character of needle.
function scopeAt(source, needle) {
  const offset = source.indexOf(needle);
  assert.notEqual(offset, -1, "sample does not contain " + JSON.stringify(needle));
  let ruleStack = vsctm.INITIAL;
  let lineStart = 0;
  for (const line of source.split("\n")) {
    const tokens = grammar.tokenizeLine(line, ruleStack);
    if (offset >= lineStart && offset < lineStart + line.length) {
      const column = offset - lineStart;
      const token = tokens.tokens.find((entry) => entry.startIndex <= column && column < entry.endIndex);
      return token?.scopes ?? [];
    }
    ruleStack = tokens.ruleStack;
    lineStart += line.length + 1;
  }
  return [];
}

const sample = [
  "// header",
  "types {",
  "  Post = { id: id, title: string };",
  "}",
  "services {",
  "  posts {",
  "    operations {",
  '      get(id: id) -> { post: Post } at GET "/v1/posts/{id}";',
  "    }",
  "  }",
  "}",
  "project {",
  '  name: "demo";',
  "  browser_session: false;",
  "}",
].join("\n");

const cases = [
  ["// header", "comment.line.double-slash.chik"],
  ["types", "keyword.control.chik"],
  ["Post =", "entity.name.type.chik"],
  ["id: id", "variable.other.property.chik"],
  ["services", "keyword.control.chik"],
  ["posts {", "entity.name.section.chik"],
  ["get(", "entity.name.function.chik"],
  ["GET", "constant.language.method.chik"],
  ['"/v1/posts/{id}"', "string.quoted.double.chik"],
  ["project", "keyword.other.role.chik"],
  ["false", "constant.language.boolean.chik"],
];

for (const [needle, expected] of cases) {
  test(needle + " is scoped " + expected, () => {
    const scopes = scopeAt(sample, needle);
    assert.ok(
      scopes.includes(expected),
      "scopes were " + JSON.stringify(scopes) + " for " + JSON.stringify(needle),
    );
  });
}

