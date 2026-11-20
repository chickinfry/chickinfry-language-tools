#!/usr/bin/env node
// Copies the TextMate grammar into both editors. The grammar is one file, so
// VS Code and the JetBrains plugin cannot highlight the same code differently.
//
// Run after editing editors/textmate/.
import { copyFileSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const source = join(root, "editors", "textmate");
const targets = [
  join(root, "editors", "vscode", "syntaxes"),
  join(root, "editors", "jetbrains", "src", "main", "resources", "textmate"),
];
const files = ["chik.tmLanguage.json", "language-configuration.json"];

for (const target of targets) {
  mkdirSync(target, { recursive: true });
  for (const name of files) {
    copyFileSync(join(source, name), join(target, name));
  }
  console.log("copied the grammar into " + target.slice(root.length));
}

