import { existsSync } from "node:fs";
import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";
import { resolveServer } from "./server";

let client: LanguageClient | undefined;

// The language server lives in the chik binary, so the extension starts it as
// a child process and speaks LSP over stdio. No second runtime is installed.
export function activate(context: vscode.ExtensionContext): void {
  const configured = vscode.workspace.getConfiguration("chik").get<string>("server.path", "");
  const command = resolveServer(configured, process.env.PATH ?? "", existsSync);

  if (!command) {
    void vscode.window.showErrorMessage(
      "The chik language server was not found. Set chik.server.path or put chik on PATH.",
    );
    return;
  }

  const serverOptions: ServerOptions = {
    command,
    args: ["lsp", "--stdio"],
    transport: TransportKind.stdio,
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "chik" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.chik"),
    },
  };

  client = new LanguageClient("chik", "Chik", serverOptions, clientOptions);
  context.subscriptions.push(client);
  void client.start();
}

export async function deactivate(): Promise<void> {
  await client?.stop();
  client = undefined;
}
