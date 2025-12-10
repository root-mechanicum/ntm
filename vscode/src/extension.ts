import * as vscode from 'vscode';

export function activate(context: vscode.ExtensionContext) {
	console.log('Congratulations, your extension "ntm-vscode" is now active!');

	let disposable = vscode.commands.registerCommand('ntm.showStatus', () => {
		vscode.window.showInformationMessage('NTM Status: Active');
	});

	context.subscriptions.push(disposable);
}

export function deactivate() {}
