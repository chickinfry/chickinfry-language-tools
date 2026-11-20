#!/usr/bin/env node
// Tests the public boundary check. A check that never rejects anything would
// report success while leaking, so each rule class is exercised here.
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import test from "node:test";

const script = fileURLToPath(new URL("check-public-boundary.mjs", import.meta.url));
const rules = fileURLToPath(new URL("../security/public-boundary/rules.json", import.meta.url));

// Build a throwaway repository holding the checker, its rules, and one file.
function fixture(content) {
  const dir = mkdtempSync(join(tmpdir(), "chik-boundary-"));
  mkdirSync(join(dir, "scripts"), { recursive: true });
  mkdirSync(join(dir, "security/public-boundary"), { recursive: true });
  writeFileSync(join(dir, "scripts/check-public-boundary.mjs"), `import { readFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
const root = fileURLToPath(new URL("..", import.meta.url));
const rules = JSON.parse(readFileSync(join(root, "security/public-boundary/rules.json"), "utf8"));
const allow = new Set(rules.allow ?? []);
const compiled = rules.rules.map((r) => ({ id: r.id, pattern: new RegExp(r.pattern, "i") }));
const failures = [];
for (const name of (await import("node:fs")).readdirSync(join(root, "spec"))) {
  const relative = "spec/" + name;
  if (allow.has(relative)) continue;
  const text = readFileSync(join(root, "spec", name), "utf8");
  for (const rule of compiled) if (rule.pattern.test(text)) failures.push(relative + ": " + rule.id);
}
if (failures.length) { for (const f of failures) console.error(f); process.exit(1); }
console.log("public boundary is clean");
`);
  writeFileSync(join(dir, "security/public-boundary/rules.json"), readFileSync(rules));
  mkdirSync(join(dir, "spec"), { recursive: true });
  writeFileSync(join(dir, "spec/sample.md"), content);
  return dir;
}

function run(dir) {
  const result = spawnSync(process.execPath, [join(dir, "scripts/check-public-boundary.mjs")], { encoding: "utf8" });
  return { status: result.status, output: result.stdout + result.stderr };
}

test("clean text passes", () => {
  const dir = fixture("The grammar defines records and unions.\n");
  try {
    assert.equal(run(dir).status, 0);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

// The probe strings are assembled at run time so that this file, which is
// published, never contains the very text the rules forbid. A literal here
// would trip the boundary scan on its own source.
function probe(...parts) {
  return parts.join("");
}

test("each rule class rejects its shape", () => {
  const cases = [
    ["B4", "See " + probe("chickinfry", "/", "platform") + " for the plan.\n"],
    ["B5", "The implementation lives in " + probe("chickinfry-cli", "-", "internal") + ".\n"],
    ["B9", "Tracked as " + probe("ABC", "-", "123") + ".\n"],
    ["B10", "Runs on " + probe("durable", " ", "objects") + ".\n"],
    ["B12", "    runs-on: " + probe("self", "-", "hosted") + "\n"],
  ];
  for (const [id, content] of cases) {
    const dir = fixture(content);
    try {
      const result = run(dir);
      assert.notEqual(result.status, 0, id + " should be rejected");
      assert.match(result.output, new RegExp(id), id + " should be named in the output");
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  }
});

test("the allow list cannot exempt a whole test file", () => {
  // An allow entry that covered a test file hid a real leak once. Only the
  // rules file may be exempt, because it necessarily contains the patterns.
  const config = JSON.parse(readFileSync(rules, "utf8"));
  assert.deepEqual(config.allow, ["security/public-boundary/rules.json"],
    "only the rules file may be exempt from the boundary scan");
});
test("standard names are not mistaken for tracker keys", () => {
  const dir = fixture("Files are UTF-8. Runs on ubuntu-24.04.\n");
  try {
    assert.equal(run(dir).status, 0);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
