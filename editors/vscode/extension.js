const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

function activate(context) {
  const bin = process.env.KIRO_LSP_BIN || 'kiro-lsp';
  const serverOptions = {
    run: { command: bin, transport: TransportKind.stdio },
    debug: { command: bin, transport: TransportKind.stdio }
  };
  const clientOptions = {
    documentSelector: [{ scheme: 'file', language: 'kiro' }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.ki')
    }
  };
  client = new LanguageClient('kiro-lsp', 'Kiro LSP', serverOptions, clientOptions);
  context.subscriptions.push(client.start());
}

function deactivate() {
  return client ? client.stop() : undefined;
}

module.exports = { activate, deactivate };
