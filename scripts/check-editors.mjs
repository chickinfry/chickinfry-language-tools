#!/usr/bin/env node
// Fails when a copy of the TextMate grammar differs from the source in
// editors/textmate/. Run node scripts/sync-editors.mjs to update the copies.
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const source = join(root, "editors", "textmate");
const copies = [
  join(root, "editors", "vscode", "syntaxes"),
  join(root, "editors", "jetbrains", "src", "main", "resources", "textmate"),
];
const files = ["chik.tmLanguage.json", "language-configuration.json"];

const failures = [];
for (const copy of copies) {
  for (const name of files) {
    const want = readFileSync(join(source, name), "utf8");
    let got;
    try {
      got = readFileSync(join(copy, name), "utf8");
    } catch {
      failures.push(copy.slice(root.length) + "/" + name + ": missing");
      continue;
    }
    if (got !== want) failures.push(copy.slice(root.length) + "/" + name + ": differs from editors/textmate/" + name);
  }
}

if (failures.length > 0) {
  console.error("editor grammars are out of sync:");
  for (const failure of failures) console.error("  " + failure);
  process.exit(1);
}
console.log("both editors use the grammar in editors/textmate");

