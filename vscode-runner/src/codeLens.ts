import * as vscode from 'vscode'

const TASK_HEADER_RE = /^([a-zA-Z_][\w.:-]*)\s*:/

export class RunnerCodeLensProvider implements vscode.CodeLensProvider {
  provideCodeLenses(document: vscode.TextDocument): vscode.CodeLens[] {
    const lenses: vscode.CodeLens[] = []
    const taskNames: string[] = []

    for (let i = 0; i < document.lineCount; i++) {
      const line = document.lineAt(i)
      const match = line.text.match(TASK_HEADER_RE)
      if (match && match[1] !== '@vars') {
        const name = match[1]
        taskNames.push(name)
        const range = line.range
        lenses.push(
          new vscode.CodeLens(range, {
            title: `▶ Run ${name}`,
            command: 'runner.runTask',
            arguments: [name],
          })
        )
        lenses.push(
          new vscode.CodeLens(range, {
            title: `☐ Dry Run`,
            command: 'runner.dryRunTask',
            arguments: [name],
          })
        )
      }
    }

    if (taskNames.length > 0) {
      lenses.unshift(
        new vscode.CodeLens(new vscode.Range(0, 0, 0, 0), {
          title: `Tasks: ${taskNames.length}`,
          command: '',
        })
      )
    }

    return lenses
  }
}
