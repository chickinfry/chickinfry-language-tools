#!/usr/bin/env node
// Checks that the block keywords agree in every place they are written down.
//
// The same four names appear in four files: the grammar's block_role, the
// tree-sitter highlight queries, the TextMate grammar, and the role file table
// in spec/grammar.md. They drifted once already: the roles decision retired
// extensions.chik, and the keyword stayed in three of them until a reviewer
// found it. Nothing compared them, so this check does.
//
// The keywords are read from the generated grammar.json, not from grammar.js,
// so a stale parser.c is caught too.
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));

function read(path) {
  return readFileSync(join(root, path), "utf8");
}

function grammarKeywords() {
  const grammar = JSON.parse(read("tree-sitter-chik/src/grammar.json"));
  const rule = grammar.rules.block_role;
  if (!rule || rule.type !== "CHOICE") throw new Error("block_role is not a choice in src/grammar.json");
  return rule.members.map((member) => member.value);
}

function queryKeywords() {
  // The role keywords are quoted inside the query's bracket lists. Other
  // keywords share those lists (types, services, routes, ...), so only the
  // ones the grammar declares as block roles are compared.
  const query = read("tree-sitter-chik/queries/highlights.scm");
  const roles = new Set(grammarKeywords());
  const quoted = [...query.matchAll(/^\s+"([a-z_]+)"$/gm)].map((match) => match[1]);
  return [...new Set(quoted.filter((word) => roles.has(word)))];
}

function textmateKeywords() {
  const grammar = JSON.parse(read("editors/textmate/chik.tmLanguage.json"));
  const patterns = grammar.repository["block-keywords"].patterns;
  const role = patterns.find((pattern) => pattern.name === "keyword.other.role.chik");
  if (!role) throw new Error("the TextMate grammar has no role keyword pattern");
  const match = role.match.match(/\(([a-z_|]+)\)/);
  if (!match) throw new Error("the role keyword pattern has no alternation: " + role.match);
  return match[1].split("|");
}

function specKeywords() {
  // The role file table is the prose definition. api.chik declares services,
  // which is a separate block rather than a config block, so it is skipped;
  // schema.chik's block is database.
  const spec = read("spec/grammar.md");
  const table = spec.slice(spec.indexOf("A project declares"), spec.indexOf("Every file is UTF-8"));
  const files = [...table.matchAll(/`([a-z]+)\.chik`/g)].map((match) => match[1]);
  return files.filter((file) => file !== "api").map((file) => (file === "schema" ? "database" : file));
}

const failures = [];
function same(label, want, got) {
  const a = [...want].sort().join(",");
  const b = [...got].sort().join(",");
  if (a !== b) failures.push(label + ": expected [" + a + "] but found [" + b + "]");
}

const grammar = grammarKeywords();
if (grammar.length === 0) failures.push("the grammar declares no block role");

same("tree-sitter-chik/queries/highlights.scm", grammar, queryKeywords());
same("editors/textmate/chik.tmLanguage.json", grammar, textmateKeywords());
same("spec/grammar.md role file table", grammar, specKeywords());

if (failures.length > 0) {
  console.error("block keywords disagree:");
  for (const failure of failures) console.error("  " + failure);
  process.exit(1);
}
console.log("block keywords agree in all four places: " + [...grammar].sort().join(", "));
