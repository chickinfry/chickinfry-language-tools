#!/usr/bin/env node
// Parses the beta conformance corpus with the tree-sitter grammar.
//
// The grammar is the one editors use for highlighting, so every file the beta
// accepts must parse cleanly. That is the first loop, and it covers format/,
// roundtrip/, and convert/.
//
// errors/ needs a narrower rule. Those inputs are rejected by the language, but
// for different reasons:
//
//   CHK00xx   lexical. The text cannot be tokenized at all, so the grammar must
//             report an ERROR.
//   anything else
//             the input is well formed and a semantic rule rejects it, or the
//             beta refuses later-release grammar. The grammar must accept it:
//             it covers the language so that routes and types still
//             highlight, and the parser is what refuses them for the beta. A
//             file whose only problem is a name would otherwise lose its
//             highlighting too.
//
// A retired keyword is different, and the grammar rejects it rather than
// highlighting it. The corpus cannot say "this must fail", so the case named
// below is asserted directly instead of being trusted to a clean parse.
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const grammarDir = join(root, "tree-sitter-chik");
const corpus = join(root, "conformance");
const localBinary = join(grammarDir, "node_modules", ".bin", "tree-sitter");
const treeSitter = existsSync(localBinary) ? localBinary : "tree-sitter";

const accepted = ["format", "roundtrip", "convert"];

function chikFiles(dir) {
  const found = [];
  for (const entry of readdirSync(dir).sort()) {
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) found.push(...chikFiles(path));
    else if (entry.endsWith(".chik")) found.push(path);
  }
  return found;
}

function parse(path) {
  const result = spawnSync(treeSitter, ["parse", "--quiet", path], {
    cwd: grammarDir,
    encoding: "utf8",
  });
  if (result.error) throw result.error;
  return { status: result.status, output: (result.stdout ?? "") + (result.stderr ?? "") };
}

function detail(output) {
  return (
    output
      .split("\n")
      .find((line) => line.includes("(ERROR") || line.includes("(MISSING"))
      ?.trim() ?? "parse failed"
  );
}

const failures = [];
let parsed = 0;
for (const group of accepted) {
  for (const path of chikFiles(join(corpus, group))) {
    parsed += 1;
    const { status, output } = parse(path);
    if (status !== 0) failures.push(relative(root, path) + ": " + detail(output));
  }
}

let lexical = 0;
for (const path of chikFiles(join(corpus, "errors"))) {
  const expected = JSON.parse(readFileSync(join(path, "..", "expected.json"), "utf8"));
  const allLexical = expected.diagnostics.every((entry) => entry.code.startsWith("CHK00"));
  if (!allLexical) continue;
  lexical += 1;
  const { status } = parse(path);
  if (status === 0) {
    failures.push(relative(root, path) + ": a lexical case parses cleanly, so it tests nothing");
  }
}

// later/ holds grammar the beta refuses but the language defines: routes, a
// types block, unions. The parser rejects it for now, and the grammar has to
// keep covering it, or those files would lose highlighting the day the beta
// ends. Only the .chik inputs are parsed; expected.json is compared by an
// implementation, not by a grammar.
let later = 0;
for (const path of chikFiles(join(root, "later"))) {
  if (path.endsWith("expected.chik")) continue;
  later += 1;
  const { status, output } = parse(path);
  if (status !== 0) {
    failures.push(relative(root, path) + ": later-release grammar must still parse: " + detail(output));
  }
}

if (parsed === 0) failures.push("no accepted .chik file under conformance/, so this check is blind");
if (lexical === 0) failures.push("no lexical case under conformance/errors/, so that half of this check is blind");
if (later === 0) failures.push("no .chik file under later/, so the later-release surface is unchecked");

// A grammar that accepts everything would pass the loops above. This snippet is
// missing two closing braces, so it must fail.
const scratch = mkdtempSync(join(tmpdir(), "chik-grammar-"));
try {
  const broken = join(scratch, "broken.chik");
  writeFileSync(broken, "services {\n  posts {\n    operations {\n      get() -> {};\n");
  if (parse(broken).status === 0) {
    failures.push("an unterminated file parsed cleanly, so this check is blind to errors");
  }

  // The roles decision retired extensions.chik and moved its one entry into
  // project.chik under `capabilities`. The corpus case pins the parser's
  // CHK0601, and the grammar has to agree: a file that still starts with the
  // old keyword must not highlight as a valid block.
  const retiredCase = join(corpus, "errors", "extensions_block_is_retired", "input.chik");
  if (!existsSync(retiredCase)) {
    failures.push("conformance/errors/extensions_block_is_retired is missing, so nothing pins the retired keyword");
  } else if (parse(retiredCase).status === 0) {
    failures.push("the retired extensions block still parses, so the grammar accepts a file the language refuses");
  }
} finally {
  rmSync(scratch, { recursive: true, force: true });
}

if (failures.length > 0) {
  console.error("grammar check failed:");
  for (const failure of failures) console.error("  " + failure);
  process.exit(1);
}
console.log(
  "grammar accepts " + parsed + " beta files and " + later + " later-release files, " +
    "rejects " + lexical + " lexical cases, and rejects a broken file",
);

