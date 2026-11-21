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
  definition: { required: ["input.chik", "expected.json"] },
  completion: { required: ["input.chik", "expected.json"] },
};

// later/ holds cases for grammar the beta does not accept. They are kept so the
// work is not lost, but they are not part of the beta specification and an
// implementation is not required to pass them. The directory mirrors the four
// beta groups.
const laterGroups = ["format", "errors", "convert", "roundtrip"];

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
    if (typeof entry.code !== "string" || !/^CHK[0-9]{4}$/.test(entry.code)) fail(where + ": bad code");
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
  return checkCaseIn(join(corpus, group), group, name);
}

// A definition case pins one cursor position and the declarations it resolves
// to. Line and column are 1-based code points, the same as the error format, so
// an implementation converts once for both.
function checkDefinitionOutput(path, label) {
  let document;
  try {
    document = JSON.parse(read(path));
  } catch (error) {
    fail(label + ": is not JSON: " + error.message);
    return;
  }
  const position = document.position;
  if (!position || !Number.isInteger(position.line) || !Number.isInteger(position.column)) {
    fail(label + ": needs a position with an integer line and column");
  } else if (position.line < 1 || position.column < 1) {
    fail(label + ": position is 1-based, so line and column must be at least 1");
  }
  if (!Array.isArray(document.targets)) {
    fail(label + ": targets must be an array, empty when the name is not a declaration");
    return;
  }
  for (const [index, target] of document.targets.entries()) {
    if (!Number.isInteger(target.line) || !Number.isInteger(target.column) || target.line < 1 || target.column < 1) {
      fail(label + " target " + index + ": line and column are 1-based integers");
    }
  }
  if (Object.hasOwn(document, "message")) fail(label + ": message is not compared and must be omitted");
}

// A completion case pins a cursor position and the labels an implementation
// must offer there. Only the listed labels are required: a server may offer
// more, and a client filters as the user types.
function checkCompletionOutput(path, label) {
  let document;
  try {
    document = JSON.parse(read(path));
  } catch (error) {
    fail(label + ": is not JSON: " + error.message);
    return;
  }
  const position = document.position;
  if (!position || !Number.isInteger(position.line) || !Number.isInteger(position.column)) {
    fail(label + ": needs a position with an integer line and column");
  } else if (position.line < 1 || position.column < 1) {
    fail(label + ": position is 1-based, so line and column must be at least 1");
  }
  if (!Array.isArray(document.includes) || document.includes.length === 0) {
    fail(label + ": includes must list at least one required label");
    return;
  }
  for (const [index, required] of document.includes.entries()) {
    if (typeof required !== "string" || required.length === 0) {
      fail(label + " include " + index + ": must be a non-empty string");
    }
  }
  if (Object.hasOwn(document, "message")) fail(label + ": message is not compared and must be omitted");
}

function checkCaseIn(root, group, name) {
  const dir = join(root, name);
  const label = (root === corpus ? "" : "later/") + group + "/" + name;
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
    if (entry === "expected.json") {
      if (group === "definition") checkDefinitionOutput(path, label + "/" + entry);
      else if (group === "completion") checkCompletionOutput(path, label + "/" + entry);
      else checkErrorOutput(path, label + "/" + entry);
    }
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

// later/ mirrors the beta groups. Its cases are checked for shape so they stay
// usable, but an implementation is not required to pass them.
const laterRoot = join(corpus, "..", "later");
for (const group of laterGroups) {
  const dir = join(laterRoot, group);
  let names;
  try {
    names = readdirSync(dir).sort();
  } catch {
    continue;
  }
  for (const name of names) {
    if (!statSync(join(dir, name)).isDirectory()) {
      fail("later/" + group + "/" + name + ": not a directory");
      continue;
    }
    checkCaseIn(dir, group, name);
  }
}

if (failures.length > 0) {
  for (const failure of failures) console.error(failure);
  console.error(failures.length + " corpus problem(s)");
  process.exit(1);
}
console.log("corpus is well formed");
