# Runner

Zero-dependency Go task runner — a simple command orchestration tool.

```
git clone https://github.com/mhs003/runner.git
cd runner
go build -o run ./cmd/run
```

## Getting Started

Create a `.runner` file in your project root:

```yaml
@vars:
  BIN = app

build:
    go build -o {{BIN}} .

test:
    go test ./...
```

Run a task:

```
$ run build
$ run test
```

If no task is specified, `run` defaults to `main`.

## File Format

### Tasks

```
taskname:
    command
    @ depname
```

A task header at indent 0, followed by body lines at indent 2+.

### Variables

```
@vars:
  KEY = value
```

Access them in commands: `{{KEY}}`. Built-in variables: `{{CWD}}`, `{{OS}}`, `{{ARCH}}`,
`{{ARGS}}`, `{{1}}`, `{{2}}`, ... (positional CLI args), `{{--key}}` (named CLI args),
`{{-f}}` (boolean flags).

Fallback syntax: `{{--entry||ENTRY}}` — use `--entry` if non-empty, else `ENTRY` var,
else literal `ENTRY`.

### Dependencies

Header deps — run before the task:

```
deploy: build test:
    echo "deploying"
```

Inline deps — run at that point in the body (commands share the same shell session,
so `$MSG` and other shell variables persist):

```
main:
    MSG="Starting build"
    @ build --target x86_64
    echo $MSG
```

Pass arguments to inline deps:

```
build:
    echo "building for {{--target}}"
```

### Annotations

```
[exit-on-error]
test:
    false
    echo "this never runs"
```

Applies to the immediately following task only.

### Comments

```
# this is a comment
```

## CLI

| Flag | Description |
|------|-------------|
| `-list` | List all tasks with their body lines |
| `-dry`  | Print commands without executing them |
| `-file <path>` | Use a config file other than `.runner` |
| `-init` | Scaffold a `.runner` file in the current directory |

With the default config path, Runner loads both `.runner` in the current directory and
`~/.runner.global`. Local tasks and variables override global entries with the same name, while
entries unique to either file remain available. Tasks and variables can reference entries from
the other file. If only one file exists, it is used on its own. An explicit `-file <path>` loads
only that file and does not merge the global config.

## Install

```
./run install
```

Copies the binary to `~/.local/bin/run`.

---

## VS Code Extension

A companion extension (`vscode-runner/`) provides syntax highlighting and CodeLens
**Run** / **Dry Run** buttons for `.runner` files. Open the `vscode-runner/` folder
in VS Code and press `F5` to try it, or package it with `vsce`:

```bash
cd vscode-runner
npm install
npm run build
```

---

## License

MIT (See [LICENSE](https://github.com/mhs003/runner/blob/master/LICENSE) file).
