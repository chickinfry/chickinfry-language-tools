#!/usr/bin/env node
// Grades one agent submission for one task.
//
// The judgment comes from the published tools, not from comparing text: the
// submitted files are parsed and checked by the same binary a customer runs,
// and the task's check.json names the fields that must be present or the error
// code that must be reported. Spacing, comments, and field order do not matter.
//
// Usage:
//   node agent-gate/grade.mjs --server <chik binary> --task <id> --submission <dir>
//
// Exit code 0 means the submission passes, 1 means it fails, 2 means the
// invocation was wrong.
import { execFileSync } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL(".", import.meta.url));

function parseArgs(argv) {
  const args = { server: process.env.CHIK_SERVER ?? "", task: "", submission: "" };
  for (let index = 0; index < argv.length; index += 1) {
    const flag = argv[index];
    if (flag === "--server") args.server = argv[++index];
    else if (flag === "--task") args.task = argv[++index];
    else if (flag === "--submission") args.submission = argv[++index];
  }
  return args;
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

// runCheck copies the submission into a project directory and runs the
// customer command. Copying keeps the submission untouched and gives the tool
// the chik/ layout it expects.
// baselineProject is the smallest declaration chik check accepts as a project.
// A submission that declares its own project.chik overwrites it.
const baselineProject = 'project {\n  name: "gate"\n  server: "typescript"\n}\n';

function runCheck(server, submission) {
  const project = mkdtempSync(join(tmpdir(), "chik-gate-"));
  try {
    const chikDir = join(project, "chik");
    mkdirSync(chikDir, { recursive: true });
    // chik check reads a whole project, so a task that asks only for api.chik
    // needs the project file a real project always has. The baseline is not
    // graded; it only lets the customer command run.
    writeFileSync(join(chikDir, "project.chik"), baselineProject);
    for (const path of chikFiles(submission)) {
      const relative = path.slice(submission.length).replace(/^[\\/]/, "");
      const target = join(project, relative.startsWith("chik/") ? relative : join("chik", relative));
      mkdirSync(dirname(target), { recursive: true });
      copyFileSync(path, target);
    }
    // chik check exits 1 when a declaration is refused, but it still writes the
    // JSON report to stdout. The report is the result, not the exit code.
    let output = "";
    try {
      output = execFileSync(server, ["check", "--format", "json"], {
        cwd: project,
        encoding: "utf8",
        stdio: ["ignore", "pipe", "pipe"],
      });
    } catch (error) {
      output = error.stdout ?? "";
      if (output.trim() === "") {
        // Not every failure produces a JSON report (a missing project file, for
        // example). That is a grading failure, not a crash: report it.
        return { unreadable: (error.stderr ?? error.message).trim() };
      }
    }
    return output
      .split("\n")
      .filter((line) => line.trim().length > 0)
      .map((line) => JSON.parse(line));
  } finally {
    rmSync(project, { recursive: true, force: true });
  }
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  if (!args.server || !existsSync(args.server)) {
    console.error("grade: --server must name a chik binary (or set CHIK_SERVER)");
    return 2;
  }
  if (!args.task) {
    console.error("grade: --task is required");
    return 2;
  }
  if (!args.submission || !existsSync(args.submission)) {
    console.error("grade: --submission must name a directory");
    return 2;
  }
  const checkPath = join(root, "tasks", args.task, "check.json");
  if (!existsSync(checkPath)) {
    console.error("grade: no such task " + args.task);
    return 2;
  }
  const rules = JSON.parse(readFileSync(checkPath, "utf8"));

  const submitted = chikFiles(args.submission).map((path) => path.slice(args.submission.length).replace(/^[\\/]/, ""));
  const missing = rules.files.filter((file) => !submitted.includes(file));
  if (missing.length > 0) {
    console.log(JSON.stringify({ pass: false, reason: "missing files: " + missing.join(", ") }));
    return 1;
  }

  const reports = runCheck(args.server, args.submission);
  if (!Array.isArray(reports)) {
    console.log(JSON.stringify({ pass: false, reason: "the tool could not read the project", detail: reports.unreadable }));
    return 1;
  }
  const diagnostics = reports.flatMap((report) => report.diagnostics.map((entry) => ({ ...entry, file: report.file })));

  if (rules.error) {
    const match = diagnostics.find((entry) => entry.code === rules.error);
    if (!match) {
      console.log(JSON.stringify({ pass: false, reason: "expected " + rules.error + ", saw " + (diagnostics.map((entry) => entry.code).join(",") || "none") }));
      return 1;
    }
    console.log(JSON.stringify({ pass: true, code: match.code, file: match.file }));
    return 0;
  }

  if (diagnostics.length > 0) {
    console.log(JSON.stringify({
      pass: false,
      reason: "the declaration is refused",
      diagnostics: diagnostics.map((entry) => ({ code: entry.code, path: entry.path, file: entry.file })),
    }));
    return 1;
  }

  // With no diagnostics, presence is proven by the parsed text: every name in
  // fields must appear, and the parser already accepted the file.
  const text = submitted.map((file) => readFileSync(join(args.submission, file), "utf8")).join("\n");
  const missingFields = rules.fields.filter((name) => !new RegExp("\\b" + name + "\\b").test(text));
  if (missingFields.length > 0) {
    console.log(JSON.stringify({ pass: false, reason: "missing declarations: " + missingFields.join(", ") }));
    return 1;
  }
  if (rules.comment && !text.includes(rules.comment)) {
    console.log(JSON.stringify({ pass: false, reason: "the comment the task asked for is missing" }));
    return 1;
  }
  console.log(JSON.stringify({ pass: true, fields: rules.fields }));
  return 0;
}

process.exit(main());
