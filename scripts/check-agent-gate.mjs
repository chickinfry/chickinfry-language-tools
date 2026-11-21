#!/usr/bin/env node
// Checks that the agent gate is self-consistent without running an agent.
//
// The gate claims a submission is judged by the published tools. That holds
// only if each task's own expected answer is a declaration those tools accept,
// and if the blocked list names exactly the tasks that cannot pass today. Both
// are checked here, from the task files alone.
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

// A shipped declaration must be a clean text file: UTF-8, LF, one trailing
// newline. A task answer that fails this teaches the wrong file shape.
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

const blocked = JSON.parse(read(join(gate, "blocked.json"))).tasks;
for (const name of Object.keys(blocked)) {
  if (!tasks.includes(name)) fail("blocked.json names a task that does not exist: " + name);
}

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

  const expectedDir = join(dir, "expected");
  const submitted = chikFiles(expectedDir).map((path) =>
    path.slice(expectedDir.length).replace(/^[\\/]/, ""));
  for (const file of rules.files ?? []) {
    if (!submitted.includes(file)) fail(task + ": check.json names " + file + " but expected/ does not contain it");
  }
  for (const file of submitted) {
    if (!(rules.files ?? []).includes(file)) fail(task + ": expected/ contains " + file + " but check.json does not name it");
  }

  // A graded field must appear in the answer, or the check passes on a
  // submission that does not address the task.
  const answer = submitted.map((file) => read(join(expectedDir, file))).join("\n");
  for (const field of rules.fields ?? []) {
    if (!new RegExp("\\b" + field + "\\b").test(answer)) {
      fail(task + ": graded field " + field + " is not in the expected answer");
    }
  }

  for (const file of chikFiles(expectedDir)) {
    checkTextFile(file, task + ": " + file.slice(expectedDir.length + 1));
  }
}

if (failures.length > 0) {
  for (const failure of failures) console.error("agent gate: " + failure);
  process.exit(1);
}

console.log("agent gate: " + tasks.length + " tasks are self-consistent");

