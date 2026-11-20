import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveServer } from "../server";

const never = () => false;
const always = () => true;

test("a configured path wins over PATH", () => {
  assert.equal(resolveServer("/opt/chik/bin/chik", "/usr/bin", always), "/opt/chik/bin/chik");
});

test("a configured absolute path that does not exist is reported as missing", () => {
  assert.equal(resolveServer("/opt/chik/bin/chik", "/usr/bin", never), undefined);
});

test("an empty configuration searches PATH in order", () => {
  const exists = (path: string) => path === "/second/chik";
  assert.equal(resolveServer("", "/first:/second", exists), "/second/chik");
});

test("an empty PATH yields no command", () => {
  assert.equal(resolveServer("", "", always), undefined);
});

