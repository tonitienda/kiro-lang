const vscode = require('vscode');
const cp = require('child_process');
const path = require('path');
const {
  LanguageClient,
  RevealOutputChannelOn,
} = require('vscode-languageclient/node');

let client;

function activate(context) {
  const outputChannel = vscode.window.createOutputChannel('Kiro Language Server');
  context.subscriptions.push(outputChannel);

  const serverOptions = () => startLanguageServer(outputChannel);
  const clientOptions = {
    documentSelector: [
      { scheme: 'file', language: 'kiro' },
      { scheme: 'untitled', language: 'kiro' },
    ],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.ki')
    },
    outputChannel,
    revealOutputChannelOn: RevealOutputChannelOn.Never,
  };

  client = new LanguageClient(
    'kiro-lsp',
    'Kiro Language Server',
    serverOptions,
    clientOptions,
  );

  context.subscriptions.push(client.start());
  client.onReady().catch((error) => {
    handleStartupError(resolveServerCommand(), outputChannel, error);
  });
}

function deactivate() {
  return client ? client.stop() : undefined;
}

function startLanguageServer(outputChannel) {
  const serverCommand = resolveServerCommand();
  outputChannel.appendLine(`Starting Kiro language server: ${formatCommand(serverCommand)}`);

  return new Promise((resolve, reject) => {
    const child = cp.spawn(serverCommand.command, serverCommand.args, {
      stdio: 'pipe',
      env: process.env,
    });
    let settled = false;

    child.stderr.on('data', (chunk) => {
      outputChannel.append(chunk.toString());
    });

    child.once('spawn', () => {
      settled = true;
      resolve(child);
    });

    child.once('error', (error) => {
      if (settled) {
        return;
      }
      settled = true;
      handleStartupError(serverCommand, outputChannel, error);
      reject(error);
    });

    child.once('exit', (code, signal) => {
      if (settled) {
        return;
      }
      settled = true;
      const reason = signal ? `signal ${signal}` : `exit code ${code}`;
      const error = new Error(`language server exited before initialization with ${reason}`);
      handleStartupError(serverCommand, outputChannel, error);
      reject(error);
    });
  });
}

function resolveServerCommand() {
  const config = vscode.workspace.getConfiguration('kiro');
  const configuredPath = String(config.get('lsp.path', '') || '').trim();
  const configuredArgs = config.get('lsp.args', []);
  const envPath = String(process.env.KIRO_LSP_BIN || '').trim();

  if (configuredPath) {
    return {
      command: configuredPath,
      args: inferServerArgs(configuredPath, configuredArgs),
      source: 'setting `kiro.lsp.path`',
    };
  }

  if (envPath) {
    return {
      command: envPath,
      args: inferServerArgs(envPath, configuredArgs),
      source: 'environment variable `KIRO_LSP_BIN`',
    };
  }

  return {
    command: 'kiro',
    args: ['lsp'],
    source: 'default `kiro lsp` entrypoint',
  };
}

function inferServerArgs(command, configuredArgs) {
  if (Array.isArray(configuredArgs) && configuredArgs.length > 0) {
    return configuredArgs;
  }
  return looksLikeKiroCLI(command) ? ['lsp'] : [];
}

function looksLikeKiroCLI(command) {
  const base = path.basename(command).toLowerCase();
  return base === 'kiro' || base === 'kiro.exe';
}

function formatCommand(serverCommand) {
  return [serverCommand.command, ...serverCommand.args].join(' ');
}

function handleStartupError(serverCommand, outputChannel, error) {
  const message = [
    `Could not start the Kiro language server with ${formatCommand(serverCommand)}.`,
    'Install the `kiro` CLI and confirm `kiro lsp` works in a terminal, or set `kiro.lsp.path` for an advanced override.',
    `Source: ${serverCommand.source}.`,
    error && error.message ? `Details: ${error.message}` : '',
  ].filter(Boolean).join(' ');

  outputChannel.appendLine(message);
  vscode.window.showErrorMessage(message, 'Open Settings', 'Show Output').then((selection) => {
    if (selection === 'Open Settings') {
      vscode.commands.executeCommand('workbench.action.openSettings', 'kiro.lsp.path');
    }
    if (selection === 'Show Output') {
      outputChannel.show(true);
    }
  });
}

module.exports = { activate, deactivate };
