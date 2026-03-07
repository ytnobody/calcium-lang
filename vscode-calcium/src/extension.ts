import * as path from 'path';
import { workspace, ExtensionContext, window } from 'vscode';
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: ExtensionContext): void {
  const config = workspace.getConfiguration('calcium');
  const lspPath: string = config.get('lsp.path', 'calcium-lsp');
  const logFile: string = config.get('lsp.logFile', '');

  const args: string[] = [];
  if (logFile) {
    args.push('--log', logFile);
  }

  const serverOptions: ServerOptions = {
    command: lspPath,
    args,
    transport: TransportKind.stdio,
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: 'file', language: 'calcium' }],
    synchronize: {
      fileEvents: workspace.createFileSystemWatcher('**/*.ca'),
    },
    outputChannelName: 'Calcium Language Server',
  };

  client = new LanguageClient(
    'calcium-lsp',
    'Calcium Language Server',
    serverOptions,
    clientOptions,
  );

  client.start();

  window.showInformationMessage('Calcium Language Server started.');
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
