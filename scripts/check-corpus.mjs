#!/usr/bin/env node
// Checks that the conformance corpus is well formed. It does not run any
// implementation: the corpus is data, and this script keeps it honest.
import assert from "node:assert/strict";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const corpus = process.argv[2] ?? join(root, "conformance");

const groups = {
  format: { required: ["input.chik", "expected.chik"] },
  errors: { required: ["input.chik", "expected.json"] },
  convert: { required: ["input.yml"], expectedDir: "expected" },
  roundtrip: { required: ["input.chik"] },
};

const caseName = /^[a-z][a-z0-9_]*$/;
const failures = [];

function fail(message) {
  failures.push(message);
}

function read(path) {
  return readFileSync(path, "utf8");
}

// Every text file in the corpus is UTF-8, LF, no BOM, and ends with one newline.
function checkTextFile(path, label) {
  const bytes = readFileSync(path);
  if (bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) fail(label + ": has a byte order mark");
  const text = bytes.toString("utf8");
  if (text.includes("\r")) fail(label + ": uses CR line endings");
  if (!text.endsWith("\n")) fail(label + ": does not end with a newline");
  if (text.endsWith("\n\n")) fail(label + ": ends with a blank line");
  if (text.length === 0) fail(label + ": is empty");
}

function checkErrorOutput(path, label) {
  let document;
  try {
    document = JSON.parse(read(path));
  } catch (error) {
    fail(label + ": is not JSON: " + error.message);
    return;
  }
  if (!Array.isArray(document.diagnostics) || document.diagnostics.length === 0) {
    fail(label + ": needs at least one diagnostic");
    return;
  }
  for (const [index, entry] of document.diagnostics.entries()) {
    const where = label + " diagnostic " + index;
    if (typeof entry.code !== "string" || !/^chik_[a-z_]+$/.test(entry.code)) fail(where + ": bad code");
    if (entry.severity !== "error") fail(where + ": severity must be error");
    if (typeof entry.path !== "string") fail(where + ": path must be a string");
    const range = entry.range;
    if (!range || !range.start || !range.end) {
      fail(where + ": missing range");
      continue;
    }
    for (const point of [range.start, range.end]) {
      if (!Number.isInteger(point.line) || point.line < 1) fail(where + ": bad line");
      if (!Number.isInteger(point.column) || point.column < 1) fail(where + ": bad column");
    }
    if (range.end.line < range.start.line ||
        (range.end.line === range.start.line && range.end.column < range.start.column)) {
      fail(where + ": range ends before it starts");
    }
    if (Object.hasOwn(entry, "message")) fail(where + ": message is not compared and must be omitted");
  }
}

function checkCase(group, name) {
  const dir = join(corpus, group, name);
  const label = group + "/" + name;
  if (!caseName.test(name)) fail(label + ": case name is not lower_snake_case");
  const spec = groups[group];
  const entries = readdirSync(dir).sort();
  for (const required of spec.required) {
    if (!entries.includes(required)) fail(label + ": missing " + required);
  }
  const allowed = new Set([...spec.required, spec.expectedDir].filter(Boolean));
  for (const entry of entries) {
    if (!allowed.has(entry)) fail(label + ": unexpected file " + entry);
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) {
      if (entry !== spec.expectedDir) fail(label + ": unexpected directory " + entry);
      continue;
    }
    checkTextFile(path, label + "/" + entry);
    if (entry === "expected.json") checkErrorOutput(path, label + "/" + entry);
  }
  if (spec.expectedDir) {
    const expectedDir = join(dir, spec.expectedDir);
    let names = [];
    try {
      names = readdirSync(expectedDir).sort();
    } catch {
      fail(label + ": missing " + spec.expectedDir + "/");
      return;
    }
    if (names.length === 0) fail(label + ": " + spec.expectedDir + "/ is empty");
    const refused = names.length === 1 && names[0] === "reason.md";
    for (const entry of names) {
      const path = join(expectedDir, entry);
      if (statSync(path).isDirectory()) {
        fail(label + ": expected/ has a nested directory " + entry);
        continue;
      }
      checkTextFile(path, label + "/expected/" + entry);
      if (refused && entry === "reason.md") continue;
      if (!entry.endsWith(".chik")) fail(label + ": expected file is not .chik: " + entry);
    }
  }
}

for (const group of Object.keys(groups)) {
  const dir = join(corpus, group);
  let names;
  try {
    names = readdirSync(dir).sort();
  } catch {
    fail("missing group " + group + "/");
    continue;
  }
  if (names.length === 0) fail("group " + group + "/ has no cases");
  for (const name of names) {
    if (!statSync(join(dir, name)).isDirectory()) {
      fail(group + "/" + name + ": not a directory");
      continue;
    }
    checkCase(group, name);
  }
}

if (failures.length > 0) {
  for (const failure of failures) console.error(failure);
  console.error(failures.length + " corpus problem(s)");
  process.exit(1);
}
console.log("corpus is well formed");
