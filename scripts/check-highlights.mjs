#!/usr/bin/env node
// Checks the tree-sitter highlight queries.
//
// Two failure modes matter. A query that does not compile leaves the editor
// with no highlighting at all, and a non-standard capture is silently ignored
// by most editors. tree-sitter highlight --check reports the second one as
// text on stdout and still exits 0, so this script reads the output rather
// than the exit code.
import { spawnSync } from "node:child_process";
import { copyFileSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const grammarDir = join(root, "tree-sitter-chik");
const query = join(grammarDir, "queries", "highlights.scm");
// A beta format case: types/ moved to later/ when the beta scope was pinned,
// so the sample has to come from what the beta actually accepts.
const sample = join(root, "conformance", "format", "single_operation", "input.chik");
const localBinary = join(grammarDir, "node_modules", ".bin", "tree-sitter");
const treeSitter = existsSync(localBinary) ? localBinary : "tree-sitter";

// tree-sitter only highlights files inside a configured parser directory, and
// the list has to name the grammar's parent.
const scratch = mkdtempSync(join(tmpdir(), "chik-highlight-"));
const config = join(scratch, "config.json");
writeFileSync(config, JSON.stringify({ "parser-directories": [root] }));

// The sample has to sit inside the grammar directory, because the parser
// directory above names the repository root. A copy avoids moving a corpus file.
const copy = join(grammarDir, "highlight-sample.chik");
copyFileSync(sample, copy);

function highlight(file) {
  const result = spawnSync(treeSitter, ["highlight", "--config-path", config, "--check", file], {
    cwd: grammarDir,
    encoding: "utf8",
  });
  if (result.error) throw result.error;
  return (result.stdout ?? "") + (result.stderr ?? "");
}

const original = readFileSync(query, "utf8");
const failures = [];
try {
  const output = highlight(copy);
  if (output.includes("Non-standard highlight capture")) {
    const names = output
      .split("\n")
      .filter((line) => line.trim().startsWith("*"))
      .map((line) => line.trim());
    failures.push("non-standard captures: " + names.join(" "));
  }
  if (/Query error|Invalid node type|Invalid capture name/.test(output)) {
    failures.push("the query does not compile: " + output.split("\n").slice(0, 4).join(" "));
  }
  if (!output.includes("conform to standards") && failures.length === 0) {
    failures.push("the check printed no verdict, so it did not run");
  }

  // A check that never notices a bad capture would pass a clean query. Prove
  // it notices one by adding a capture that cannot be standard, then restore
  // the file from the bytes read above.
  writeFileSync(query, original + "\n(number) @not.a.standard.capture\n");
  const badOutput = highlight(copy);
  if (!badOutput.includes("not.a.standard.capture")) {
    failures.push("a deliberately non-standard capture went unreported, so this check is blind");
  }
} finally {
  writeFileSync(query, original);
  rmSync(copy, { force: true });
  rmSync(scratch, { recursive: true, force: true });
}

if (failures.length > 0) {
  console.error("highlight check failed:");
  for (const failure of failures) console.error("  " + failure);
  process.exit(1);
}
console.log("highlight queries compile and use only standard captures");

