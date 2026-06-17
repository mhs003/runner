import * as vscode from 'vscode'
import { RunnerCodeLensProvider } from './codeLens'
import { RunnerDefinitionProvider } from './definition'

export function activate(context: vscode.ExtensionContext) {
  const selector: vscode.DocumentSelector = { language: 'runner', scheme: 'file' }

  context.subscriptions.push(
    vscode.languages.registerCodeLensProvider(selector, new RunnerCodeLensProvider())
  )

  context.subscriptions.push(
    vscode.languages.registerDefinitionProvider(selector, new RunnerDefinitionProvider())
  )

  context.subscriptions.push(
    vscode.commands.registerCommand('runner.runTask', (taskName: string) => {
      const terminal = vscode.window.createTerminal(`Runner: ${taskName}`)
      terminal.show()
      terminal.sendText(`run ${taskName}`)
    })
  )

  context.subscriptions.push(
    vscode.commands.registerCommand('runner.dryRunTask', (taskName: string) => {
      const terminal = vscode.window.createTerminal(`Runner: ${taskName} (dry)`)
      terminal.show()
      terminal.sendText(`run --dry ${taskName}`)
    })
  )
}

export function deactivate() {}
