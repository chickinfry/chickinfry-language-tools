#!/usr/bin/env node
// Parses the .chik examples printed in spec/*.md with the tree-sitter grammar.
//
// The prose and the corpus can disagree, and they did: the api.chik example in
// spec/beta-scope.md ended a result field with ";" while both implementations
// require ",". A reader copying that example gets a parse error. This check
// makes every example in the specification executable.
//
// A block that names a retired keyword is reported, not skipped. The roles
// decision retired extensions.chik, so a page that prints one describes a file
// the language no longer has, and the grammar refuses it.
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const grammarDir = join(root, "tree-sitter-chik");
const specDir = join(root, "spec");
const localBinary = join(grammarDir, "node_modules", ".bin", "tree-sitter");
const treeSitter = existsSync(localBinary) ? localBinary : "tree-sitter";

const blockKeywords = ["project", "database", "auth", "clients", "types", "services", "routes"];
const wholeFile = new RegExp("^(" + blockKeywords.join("|") + ")\\s*\\{", "m");
const operation = /^[a-z_][a-z0-9_]*\s*\(/;

// The roles decision retired extensions.chik. It is still parsed so a page
// that prints one fails with the reason instead of being skipped.
const retired = /^(extensions)\s*\{/m;

// A fragment has no block around it, so it is placed where it belongs.
function asFile(text) {
  const trimmed = text.trimStart();
  if (wholeFile.test(trimmed) || retired.test(trimmed)) return text;
  if (!operation.test(trimmed)) return null;
  return "services {\n  example {\n    operations {\n" +
    text.replace(/^/gm, "      ") +
    "    }\n  }\n}\n";
}

const fence = "```";
const blocksOf = (source) => {
  const pattern = new RegExp(fence + "text\\n([\\s\\S]*?)" + fence, "g");
  return [...source.matchAll(pattern)].map((match) => match[1]);
};

const scratch = mkdtempSync(join(tmpdir(), "chik-spec-"));
const failures = [];
let checked = 0;
let skipped = 0;
try {
  for (const name of readdirSync(specDir).sort()) {
    if (!name.endsWith(".md")) continue;
    const source = readFileSync(join(specDir, name), "utf8");
    for (const [index, block] of blocksOf(source).entries()) {
      const file = asFile(block);
      if (file === null) {
        skipped += 1;
        continue;
      }
      const path = join(scratch, name + "-" + index + ".chik");
      writeFileSync(path, file);
      const result = spawnSync(treeSitter, ["parse", "--quiet", path], {
        cwd: grammarDir,
        encoding: "utf8",
      });
      if (result.error) throw result.error;
      checked += 1;
      if (result.status === 0) continue;
      const output = (result.stdout ?? "") + (result.stderr ?? "");
      const detail = output
        .split("\n")
        .find((line) => line.includes("(ERROR") || line.includes("(MISSING"));
      const where = "spec/" + name + " block " + (index + 1);
      failures.push(where + ": " + (detail ?? "parse failed").trim());
    }
  }
} finally {
  rmSync(scratch, { recursive: true, force: true });
}

if (checked === 0) failures.push("no .chik example was found in spec/, so this check is blind");

if (failures.length > 0) {
  console.error("spec examples do not parse:");
  for (const failure of failures) console.error("  " + failure);
  process.exit(1);
}
console.log("all " + checked + " .chik examples in spec/ parse (" + skipped + " data or transcript blocks skipped)");

