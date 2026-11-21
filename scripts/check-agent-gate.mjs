#!/usr/bin/env node
// Checks that the agent gate is self-consistent before anyone grades an agent.
//
// The gate's whole claim is that a submission is judged by the published
// tools. That only holds if each task's own expected answer is a declaration
// the tools accept, and if the fields it grades are fields the task asked for.
// Both are checked here, without running an agent.
import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const gate = join(root, "agent-gate");
const failures = [];

function fail(message) {
  failures.push(message);
}

function read(path) {
  return readFileSync(path, "utf8");
}

function chikFiles(dir) {
  const found = [];
  if (!existsSync(dir)) return found;
  for (const entry of readdirSync(dir).sort()) {
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) found.push(...chikFiles(path));
    else if (entry.endsWith(".chik")) found.push(path);
  }
  return found;
}

// A declaration the gate ships must be well formed in the same way the corpus
// is: UTF-8, LF, one trailing newline. A task answer that fails this teaches
// the wrong file shape.
function checkTextFile(path, label) {
  const bytes = readFileSync(path);
  if (bytes[0] === 0xef && bytes[1] === 0xbb && bytes[2] === 0xbf) fail(label + ": has a byte order mark");
  const text = bytes.toString("utf8");
  if (text.includes("\r")) fail(label + ": uses CR line endings");
  if (!text.endsWith("\n")) fail(label + ": does not end with a newline");
  if (text.endsWith("\n\n")) fail(label + ": ends with a blank line");
  if (text.length === 0) fail(label + ": is empty");
}

const tasks = readdirSync(join(gate, "tasks")).sort();
if (tasks.length === 0) fail("agent-gate/tasks is empty");

for (const task of tasks) {
  const dir = join(gate, "tasks", task);
  for (const required of ["task.md", "check.json"]) {
    if (!existsSync(join(dir, required))) fail(task + ": missing " + required);
  }
  if (!existsSync(join(dir, "expected"))) {
    fail(task + ": missing expected/");
    continue;
  }

  let rules;
  try {
    rules = JSON.parse(read(join(dir, "check.json")));
  } catch (error) {
    fail(task + ": check.json is not JSON: " + error.message);
    continue;
  }

  const submitted = chikFiles(join(dir, "expected")).map((path) =>
    path.slice(join(dir, "expected").length).replace(/^[\\/]/, ""));
  for (const file of rules.files ?? []) {
    if (!submitted.includes(file)) fail(task + ": check.json names " + file + " but expected/ does not contain it");
  }
  for (const file of submitted) {
    if (!(rules.files ?? []).includes(file)) fail(task + ": expected/ contains " + file + " but check.json does not name it");
  }

  // A graded field must be a word the task or the answer actually contains.
  // Otherwise the check passes on an answer that does not address the task.
  const prompt = read(join(dir, "task.md"));
  const answer = submitted.map((file) => read(join(dir, "expected", file))).join("\n");
  for (const field of rules.fields ?? []) {
    const word = new RegExp("\\b" + field + "\\b");
    if (!word.test(answer)) fail(task + ": graded field " + field + " is not in the expected answer");
    if (!word.test(prompt) && !word.test(answer)) fail(task + ": graded field " + field + " is not asked for");
  }

  for (const file of chikFiles(join(dir, "expected"))) {
    checkTextFile(file, task + ": " + file.slice(join(dir, "expected").length + 1));
  }
}

if (failures.length > 0) {
  for (const failure of failures) console.error("agent gate: " + failure);
  process.exit(1);
}

console.log("agent gate: " + tasks.length + " tasks are self-consistent");
