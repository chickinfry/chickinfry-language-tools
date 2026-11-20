#!/usr/bin/env node
// Tests the corpus checker itself. A checker that cannot reject a broken case
// is worse than no checker, because CI would report success on a broken corpus.
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import test from "node:test";

const checker = fileURLToPath(new URL("check-corpus.mjs", import.meta.url));

function corpus(files) {
  const dir = mkdtempSync(join(tmpdir(), "chik-corpus-"));
  for (const [path, content] of Object.entries(files)) {
    const full = join(dir, path);
    mkdirSync(join(full, ".."), { recursive: true });
    writeFileSync(full, content);
  }
  return dir;
}

function run(dir) {
  const result = spawnSync(process.execPath, [checker, dir], { encoding: "utf8" });
  return { status: result.status, output: result.stdout + result.stderr };
}

const base = {
  "format/one/input.chik": "services {\n}\n",
  "format/one/expected.chik": "services {\n}\n",
  "errors/one/input.chik": "services {\n}\n",
  "errors/one/expected.json": JSON.stringify({
    diagnostics: [{
      code: "chik_expected_token", severity: "error", path: "",
      range: { start: { line: 1, column: 1 }, end: { line: 1, column: 2 } },
    }],
  }, null, 2) + "\n",
  "convert/one/input.yml": "services: {}\n",
  "convert/one/expected/api.chik": "services {\n}\n",
  "roundtrip/one/input.chik": "services {\n}\n",
};

test("a complete corpus passes", () => {
  const dir = corpus(base);
  try {
    const result = run(dir);
    assert.equal(result.status, 0, result.output);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test("each structural problem is rejected", () => {
  const cases = [
    ["missing group", (files) => { delete files["errors/one/input.chik"]; delete files["errors/one/expected.json"]; delete files["errors/one"]; }],
    ["missing expected", (files) => { delete files["format/one/expected.chik"]; }],
    ["unexpected file", (files) => { files["format/one/notes.txt"] = "x\n"; }],
    ["CR line endings", (files) => { files["format/one/input.chik"] = "services {\r\n}\r\n"; }],
    ["byte order mark", (files) => { files["format/one/input.chik"] = "\ufeffservices {\n}\n"; }],
    ["no trailing newline", (files) => { files["format/one/input.chik"] = "services {\n}"; }],
    ["message in expected", (files) => {
      const doc = JSON.parse(files["errors/one/expected.json"]);
      doc.diagnostics[0].message = "not compared";
      files["errors/one/expected.json"] = JSON.stringify(doc, null, 2) + "\n";
    }],
    ["warning severity", (files) => {
      const doc = JSON.parse(files["errors/one/expected.json"]);
      doc.diagnostics[0].severity = "warning";
      files["errors/one/expected.json"] = JSON.stringify(doc, null, 2) + "\n";
    }],
    ["reversed range", (files) => {
      const doc = JSON.parse(files["errors/one/expected.json"]);
      doc.diagnostics[0].range.end = { line: 0, column: 0 };
      files["errors/one/expected.json"] = JSON.stringify(doc, null, 2) + "\n";
    }],
    ["bad case name", (files) => {
      files["format/Bad-Name/input.chik"] = "services {\n}\n";
      files["format/Bad-Name/expected.chik"] = "services {\n}\n";
    }],
    ["empty convert output", (files) => { files["convert/one/expected/api.chik"] = ""; }],
  ];
  for (const [name, mutate] of cases) {
    const files = structuredClone(base);
    mutate(files);
    const dir = corpus(files);
    try {
      const result = run(dir);
      assert.notEqual(result.status, 0, name + " should be rejected");
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  }
});
