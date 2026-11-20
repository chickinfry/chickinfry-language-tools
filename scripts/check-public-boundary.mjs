#!/usr/bin/env node
// Public boundary check. This repository is public: every branch, commit, and
// pull request is visible immediately. Reject text that would leak a private
// repository name, a private issue reference, an internal work-session
// identifier, a vendor name, or a credential.
//
// The rule classes match the release scanner: private repository names,
// private issue numbers, internal session identifiers, vendor names, and
// credential shapes. The rules live in rules.json so that a change to them is
// reviewed like any other change.
//
// Usage:
//   node scripts/check-public-boundary.mjs           # tracked files
//   node scripts/check-public-boundary.mjs --history # every commit, for a first push
import { execFileSync } from "node:child_process";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = process.argv[2] && !process.argv[2].startsWith("--") ? process.argv[2] : fileURLToPath(new URL("..", import.meta.url));
const rulesPath = join(root, "security/public-boundary/rules.json");
const rules = JSON.parse(readFileSync(rulesPath, "utf8"));
if (rules.version !== 1 || !Array.isArray(rules.rules)) {
  console.error("rules.json is not version 1");
  process.exit(2);
}
const allow = new Set(rules.allow ?? []);
const compiled = rules.rules.map((rule) => ({
  id: rule.id,
  pattern: new RegExp(rule.pattern, "i"),
}));

const history = process.argv.includes("--history");
const failures = [];

function check(relative, text) {
  if (allow.has(relative)) return;
  for (const rule of compiled) {
    if (rule.pattern.test(text)) failures.push(relative + ": " + rule.id);
  }
}

function walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === ".git" || entry.name === "node_modules") continue;
    const path = join(dir, entry.name);
    const info = statSync(path);
    if (info.isSymbolicLink()) {
      failures.push(path.slice(root.length) + ": symlink");
      continue;
    }
    if (info.isDirectory()) {
      walk(path);
      continue;
    }
    if (!info.isFile()) {
      failures.push(path.slice(root.length) + ": not a regular file");
      continue;
    }
    if (info.size > 1_000_000) {
      failures.push(path.slice(root.length) + ": larger than 1 MB");
      continue;
    }
    check(path.slice(root.length), readFileSync(path, "utf8"));
  }
}

if (history) {
  // A first push publishes every commit, so scan each one, not just the tip.
  const commits = execFileSync("git", ["rev-list", "--all"], { cwd: root, encoding: "utf8" })
    .split("\n")
    .filter(Boolean);
  for (const commit of commits) {
    const files = execFileSync("git", ["ls-tree", "-r", "--name-only", commit], { cwd: root, encoding: "utf8" })
      .split("\n")
      .filter(Boolean);
    for (const file of files) {
      const text = execFileSync("git", ["show", commit + ":" + file], { cwd: root, encoding: "utf8", maxBuffer: 4_000_000 });
      check(file, text);
    }
  }
} else {
  walk(root);
}

if (failures.length > 0) {
  for (const failure of failures) console.error(failure);
  console.error(failures.length + " boundary problem(s)");
  process.exit(1);
}
console.log(history ? "public boundary is clean across all commits" : "public boundary is clean");
