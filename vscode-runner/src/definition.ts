import * as vscode from 'vscode'

const TASK_RE = /^([a-zA-Z_][\w.:-]*)\s*:/
const VAR_RE = /\{\{\s*([\w.:-]+)(?:\|\|[^}]*)?\s*\}\}/

export class RunnerDefinitionProvider implements vscode.DefinitionProvider {
  provideDefinition(
    document: vscode.TextDocument,
    position: vscode.Position,
  ): vscode.Definition | undefined {
    const wordRange = document.getWordRangeAtPosition(position, /[\w.:-]+/)
    if (!wordRange) return undefined

    const word = document.getText(wordRange)
    const lineText = document.lineAt(position.line).text

    if (this.isInsideInterpolation(lineText, wordRange.start.character, word.length)) {
      return this.findVarDefinition(document, word)
    }

    return this.findTaskDefinition(document, word)
  }

  private isInsideInterpolation(line: string, offset: number, wordLen: number): boolean {
    const before = line.slice(0, offset)
    const after = line.slice(offset + wordLen)
    if (before.includes('{{') && after.includes('}}')) {
      return true
    }
    return false
  }

  private findVarDefinition(
    doc: vscode.TextDocument,
    name: string,
  ): vscode.Location | undefined {
    for (let i = 0; i < doc.lineCount; i++) {
      const line = doc.lineAt(i).text
      const m = line.match(/^\s*([\w.-]+)\s*=\s*/)
      if (m && m[1] === name) {
        return new vscode.Location(doc.uri, new vscode.Position(i, 0))
      }
    }
    return undefined
  }

  private findTaskDefinition(
    doc: vscode.TextDocument,
    name: string,
  ): vscode.Location | undefined {
    for (let i = 0; i < doc.lineCount; i++) {
      const line = doc.lineAt(i).text
      const m = line.match(TASK_RE)
      if (m && m[1] === name) {
        return new vscode.Location(doc.uri, new vscode.Position(i, 0))
      }
    }
    return undefined
  }
}
