// Starts the real language server and speaks LSP to it over stdio.
//
// This is the check the extension's own tests cannot make: resolveServer only
// proves a path was found, and a path that starts a process which does not
// speak the protocol would still pass. Here the server has to answer
// initialize, publish diagnostics for a file, and resolve a definition.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { test } from "node:test";

// CHIK_SERVER points at a built chik binary. CI builds one and sets it; a
// local run without it skips rather than fails.
const server = process.env.CHIK_SERVER;
const enabled = Boolean(server) && existsSync(server);

function frame(message) {
  const body = JSON.stringify(message);
  return "Content-Length: " + Buffer.byteLength(body) + "\r\n\r\n" + body;
}

// readFrames parses whatever complete messages the server has written.
function readFrames(buffer) {
  const messages = [];
  for (;;) {
    const headerEnd = buffer.indexOf("\r\n\r\n");
    if (headerEnd < 0) return { messages, rest: buffer };
    const header = buffer.slice(0, headerEnd).toString("ascii");
    const match = /Content-Length:\s*(\d+)/i.exec(header);
    if (!match) return { messages, rest: buffer.slice(headerEnd + 4) };
    const length = Number(match[1]);
    const start = headerEnd + 4;
    if (buffer.length < start + length) return { messages, rest: buffer };
    messages.push(JSON.parse(buffer.slice(start, start + length).toString("utf8")));
    buffer = buffer.slice(start + length);
  }
}

test("the server answers initialize", { skip: !enabled && "set CHIK_SERVER to a built chik binary" }, async () => {
  const child = spawn(server, ["lsp", "--stdio"], { stdio: ["pipe", "pipe", "pipe"] });
  let buffer = Buffer.alloc(0);
  const seen = [];
  child.stdout.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    const { messages, rest } = readFrames(buffer);
    buffer = rest;
    seen.push(...messages);
  });
  child.stdin.write(frame({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} }));

  const answer = await waitFor(() => seen.find((message) => message.id === 1));
  assert.ok(answer, "the server did not answer initialize");
  assert.equal(answer.result.capabilities.definitionProvider, true);
  assert.equal(answer.result.capabilities.documentFormattingProvider, true);
  assert.ok(answer.result.capabilities.completionProvider);
  child.kill();
});

test("a document gets diagnostics with a corpus code", { skip: !enabled && "set CHIK_SERVER to a built chik binary" }, async () => {
  const child = spawn(server, ["lsp", "--stdio"], { stdio: ["pipe", "pipe", "pipe"] });
  let buffer = Buffer.alloc(0);
  const seen = [];
  child.stdout.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    const { messages, rest } = readFrames(buffer);
    buffer = rest;
    seen.push(...messages);
  });
  const uri = "file:///project/chik/api.chik";
  child.stdin.write(frame({
    jsonrpc: "2.0",
    method: "textDocument/didOpen",
    params: {
      textDocument: { uri, version: 1, text: "types {\n  Post = { author: Author }\n}\n" },
    },
  }));

  const published = await waitFor(() =>
    seen.find((message) => message.method === "textDocument/publishDiagnostics"),
  );
  assert.ok(published, "no diagnostics were published");
  assert.equal(published.params.uri, uri);
  assert.ok(published.params.diagnostics.length > 0, "the unknown type should be reported");
  // The number is the parser's, so the editor and chik check agree.
  assert.equal(published.params.diagnostics[0].code, "CHK0301");
  child.kill();
});

test("definition resolves a type to its declaration", { skip: !enabled && "set CHIK_SERVER to a built chik binary" }, async () => {
  const child = spawn(server, ["lsp", "--stdio"], { stdio: ["pipe", "pipe", "pipe"] });
  let buffer = Buffer.alloc(0);
  const seen = [];
  child.stdout.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    const { messages, rest } = readFrames(buffer);
    buffer = rest;
    seen.push(...messages);
  });
  const uri = "file:///project/chik/api.chik";
  const text = [
    "types {",
    "  Post = { id: id }",
    "}",
    "services {",
    "  posts {",
    "    operations {",
    "      get() -> { post: Post }",
    "    }",
    "  }",
    "}",
    "",
  ].join("\n");
  child.stdin.write(frame({
    jsonrpc: "2.0",
    method: "textDocument/didOpen",
    params: { textDocument: { uri, version: 1, text } },
  }));
  child.stdin.write(frame({
    jsonrpc: "2.0",
    id: 2,
    method: "textDocument/definition",
    params: { textDocument: { uri }, position: { line: 6, character: 23 } },
  }));

  const answer = await waitFor(() => seen.find((message) => message.id === 2));
  assert.ok(answer, "the server did not answer definition");
  assert.ok(Array.isArray(answer.result), "definition should be an array");
  assert.equal(answer.result.length, 1);
  assert.equal(answer.result[0].range.start.line, 1, "the Post declaration is on line 2 (zero-based 1)");
  child.kill();
});

// waitFor polls until the predicate returns a value, then resolves with it.
// A protocol reply is asynchronous, so the test cannot read it synchronously.
function waitFor(predicate) {
  return new Promise((resolve) => {
    const started = Date.now();
    const tick = () => {
      const found = predicate();
      if (found) return resolve(found);
      if (Date.now() - started > 10000) return resolve(undefined);
      setTimeout(tick, 20);
    };
    tick();
  });
}
