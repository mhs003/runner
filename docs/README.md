# runner — Cheatsheet

A zero-dependency Go task runner with a minimal DSL. Think `just` meets `Make`, written in Go.

```
$ ./build/run [flags] <task> [--key val] [-f] [args...]
```

---

## Build & Install

```bash
go build -o build/run ./cmd/run      # build binary
make                                  # build (make)
./build/run install                   # install to ~/.local/bin
go test ./...                         # run all tests (106 tests, 6 files)
go vet ./...                          # static analysis
```

**Requirements:** Go 1.25+ — zero external dependencies.

---

## Quick Start

```bash
$ ./build/run --init           # scaffolds .runner in CWD
$ ./build/run                  # runs the "main" task
```

Scaffolded `.runner`:

```yaml
main:
    echo "I am running"
```

---

## CLI Reference

| Flag | Description |
|------|-------------|
| `--list` | List all tasks with body, deps, and vars |
| `--json` | JSON output (use with `--list`) |
| `--dry` | Print interpolated commands without executing |
| `--file <path>` | Config file path (default: `.runner`) |
| `--init` | Scaffold a `.runner` in the current directory |

**Config sources:** With the default path, Runner loads both `.runner` in the current directory
and `~/.runner.global`. Local tasks and variables override global entries with the same name;
entries unique to either file remain available, and tasks and variables can reference entries in
the other file. If only one file exists, it is used on its own. An explicit `--file <path>` loads
only that file and does not merge the global config.

```bash
# ALL of these work
./build/run test Hello World
./build/run test -p "conf" --verbose
./build/run test --entry=./cmd/app
./build/run test Hello World -p conf -v
./build/run test -abc              # three boolean flags: -a, -b, -c
```

---

## `.runner` File Syntax

### Structure

```yaml
@vars:                # ← file-level variables (optional, at top)
  KEY = value

[exit-on-error]       # ← annotation (scoped to next task only)

taskname:             # ← task header (indent 0)
  shell command       # ← body lines (indent 2+)
  !echo verbose       # ← verbose: prints "> echo verbose" before running
  @ depname           # ← inline dependency
  @ build --target x  # ← inline dependency with args

taskname: dep1 dep2:  # ← task with header dependencies (dep1, dep2 run first)
  echo done
```

### `@vars:` Block

```yaml
@vars:
  BIN = app
  PORT = 8080
  BUILD_DIR = ./build
```

- `KEY = value` — split on first `=`, both sides trimmed
- **`{{VAR}}` references** — values can reference other `@vars`:
  ```yaml
  @vars:
    NAME = world
    GREET = hello {{NAME}}
  ```
  `{{GREET}}` → `hello world`. Cross-references work in any order. Cycles detected.
- **`$(cmd)` shell substitution** — values containing `$(...)` are executed via `/bin/sh -c` lazily (on first use) and cached:
  ```yaml
  @vars:
    HOST = $(hostname)
    DESC = {{HOST}} is live
  ```
  `{{HOST}}` executes `hostname` once, caches the result. `{{DESC}}` → `coder71 is live`. `$VAR` env vars from `@vars` are available inside `$(cmd)`.
- **Quotes stored literally:** `BIN = "app"` → `{{BIN}}` yields `"app"` (with quotes)
- Empty lines and comments (`#`) allowed within the block

### Tasks

```yaml
# ↓ header deps (resolved before this task runs)
build: clean deps:
  go build ./...

# ↓ shell variable persists across body lines
deploy:
  MSG="Starting deploy"
  echo $MSG
  @ docker-build --tag latest

# ↓ verbose (!) — prints the command before running it
install:
  !cp ./build/run ~/.local/bin/
```

### Dependencies

| Type | Syntax | When resolved |
|------|--------|---------------|
| **Header dep** | `taskname: dep1 dep2:` | Before execution (topological sort) |
| **Inline dep** | `  @ depname` (indented) | At execution time (inlined into script) |

**Header dep** — runs once, independently, in resolved order.
**Inline dep** — commands are recursively collected and inlined into the caller's `/bin/sh -c` script, so shell variables (`$MSG`) persist across boundaries.

```yaml
# header dep: wait_db runs once, before deploy
deploy: wait_db:
  @ docker-build --tag latest

# inline dep: docker-build commands are inlined into deploy's script
```

### Inline Dep Arguments

```yaml
main:
  @ build --target x86_64 --release

build:
  echo "building for {{--target}}"
  test "{{--release}}" = "true" && echo "release mode"
```

- Args passed to `@ <dep>` are injected as variables into the dep's scope
- Dep args only override variables that don't already exist
- After the dep completes, newly-introduced vars are cleaned up
- **CLI-level args (`{{@}}`) are NOT overridden by dep-call-site args** — they always resolve to the complete top-level CLI task arguments

### Comments

```yaml
# full line comment — completely ignored

taskname: # inline comment at top level → taskname:

  echo hello # keeps # in shell — passed literally
```

- `#` at start of trimmed line → skipped entirely
- `#` at indent 0 (top level) with preceding text → stripped from `#` onward
- `#` in indented body lines → **preserved** (passed to shell as-is)

### Annotations

```yaml
[exit-on-error]       # ← ONLY valid annotation

test:
  false
  echo "never runs"   # ← set -e stops script on first error
```

- Scoped to the very next task only
- Prepends `set -e` to the combined shell script
- Unknown annotations → parse error

---

## Variables & Interpolation

All `{{TOKEN}}` patterns are resolved at execution time. Tokens can appear anywhere in command text.

### Variable Sources

| Source | Syntax | Example |
|--------|--------|---------|
| `@vars` block | `{{KEY}}` | `{{BIN}}` |
| `@vars` shell cmd | `$(cmd)` in var value | `{{HOST}}` where `HOST=$(hostname)` → `coder71` |
| Built-in | `{{CWD}}`, `{{OS}}`, `{{ARCH}}` | `{{OS}}-{{ARCH}}` |
| All positional CLI args | `{{ARGS}}` | `{{ARGS}}` → `Hello World` |
| Individual positional | `{{1}}`, `{{2}}`, `{{3}}`... | `{{1}}` → first arg |
| Named CLI arg | `{{--key}}`, `{{-k}}` | `{{--entry}}` |
| Boolean CLI flag | `{{--flag}}`, `{{-f}}` | → `"true"` / `"false"` |
| **Positional patterns** | `{{@}}`, `{{@N}}`, `{{N@}}`, `{{M@N}}` | see below |

### `{{@}}` Positional Arg Patterns

```
test:
  echo {{@}}             # all args
  echo {{@2}}            # first 2 args
  echo {{2@}}            # last 2 args
  echo {{2@4}}           # args from position 2 to 4 (1-indexed, inclusive)
  echo {{@||no args}}    # fallback if no args
```

| Pattern | Args: `a b c d e` | Result |
|---------|-------------------|--------|
| `{{@}}` | | `a b c d e` |
| `{{@3}}` | | `a b c` |
| `{{2@}}` | | `d e` |
| `{{2@4}}` | | `b c d` |
| `{{@}}` (no args) | | empty string |
| `{{@\|\|fallback}}` (no args) | | `fallback` |
| `{{4@2}}` (M > N) | | empty string |

- Negative/zero N → empty string (invalid)
- Exceeds length → clamped to available args
- `{{@}}` includes all task arguments in their original order, including named arguments and flags

### Fallback Syntax

```
{{VAR||default}}
```

- If `VAR` is missing or empty, falls back to `default`
- `default` is first looked up as a variable key, then treated as literal string
- Common pattern: `{{--entry||ENTRY}}` — CLI arg if provided, file var otherwise

### Variable Precedence

```
File vars (@vars block)           ← lowest
  └─ Built-ins (CWD, OS, ARCH)
      └─ CLI positional (1, 2, ..., @, ARGS)
          └─ CLI named args (--key, -k)
              └─ CLI flags (--flag, -f)  ← highest
```

Each subsequent layer overwrites the previous. CLI flags always win.

### CLI Argument Parsing

| Input | Result | Access as |
|-------|--------|-----------|
| `--entry=./cmd/app` | Named | `{{--entry}}` |
| `--entry ./cmd/app` | Named | `{{--entry}}` |
| `-e ./cmd/app` | Named | `{{-e}}` |
| `--verbose` | Flag | `{{--verbose}}` → `"true"` |
| `-v` | Flag | `{{-v}}` → `"true"` |
| `-abc` | 3 flags | `{{-a}}`, `{{-b}}`, `{{-c}}` → `"true"` |
| `foo bar --path=x -v` | Mixed | `{{1}}`=foo, `{{2}}`=bar, `{{@}}`=`foo bar --path=x -v` |

**Limitations:**
- `--key` followed by `--other` → `--key` treated as boolean flag
- Combined short flags (`-abc`) cannot include a value-taking flag (e.g. `-o val` must be separate)

---

## Execution Model

```
Parse → ResolveVars → Inject vars → Resolve (topological sort) → Execute (/bin/sh -c)
```

- **ResolveVars** — resolves `{{VAR}}` cross-references within `@vars` block (pre-process)
- **Inject** — merges file vars, built-ins, and CLI args
- **Resolve** — topological sort of header dependencies
- **Execute** — collects commands (recursively inlining deps with lazy `$(cmd)` resolution + caching), runs via `/bin/sh -c`

### Resolution

- **DFS topological sort** on header dependencies only
- Cycle detection via DFS stack — self-referencing tasks (`a: a:`) caught
- Missing dependency → error
- Empty task (no body, no deps) → warning, continues

### Execution

- **One `/bin/sh -c` per task** — all body lines and recursively inlined deps are collected into one shell script
- **Shell state persists** across body lines and inline `@ dep` boundaries:
  ```yaml
  main:
    MSG="Hello"        # set in shell
    echo $MSG           # → Hello
    @ print_msg         # $MSG still available
  ```
- **Lazy `$(cmd)` execution** — `$(cmd)` in `@vars` values runs only when the variable is first used. Result is cached in `shellCache` shared across all tasks and inline deps.
- **`[exit-on-error]`** prepends `set -e` to the script — first non-zero exit stops everything
- **No `--keep-going`** — first failing command (or `$(cmd)`) in any task stops all subsequent tasks
- **Stdio passthrough** — commands inherit stdin/stdout/stderr from the terminal
- **Verbose mode (`!`)** — prepends `echo '> <command>'` in the script, works with dry-run

### `--dry` Run

Prints the flat collected command list for each task without executing. **Note:** `$(cmd)` values are still executed during interpolation (they're needed to produce the display output).

```bash
$ ./build/run --dry build
echo '> go build -o build/run ./cmd/run'
go build -o build/run ./cmd/run
```

No visual distinction between commands from the parent task vs commands from inlined deps.

---

## Pro Tips & Gotchas

| Gotcha | Detail |
|--------|--------|
| **Quotes are literal** | `@vars:` stores quotes as-is. If you write `BIN = "run"`, then `{{BIN}}` expands to `"run"` (with quotes). The shell handles quoting. |
| **Tabs = 1 indent** | The lexer counts each space/tab as one indent. Convention is **2-space indentation**. Tabs break the indent model. |
| **`#` in body = literal** | Only top-level `#` comments are stripped. Inside a task body, `#` is passed to the shell. |
| **`@` with nothing after** | `@` on a body line with no following text → silently skipped |
| **Fallback resolved as var** | `{{X\|\|Y}}` — if `Y` is a valid variable, its value is used. Use `{{X\|\| literal}}` for literals. |
| **`--dry` prints flat** | No visual separation between parent and dep commands. Also: `$(cmd)` in vars still runs during dry-run (needed for interpolation). |
| **No `--help`** | Go's `flag` package auto-generates basic help via `-h` |
| **`{{@}}` in deps** | `{{@}}` inside an inline dep's body resolves to all **top-level CLI** task args, not dep-call-site args |
| **Empty `@`** | `{{@}}` with no args resolves to an empty string. Use `{{@\|\|fallback}}` for a default. |
| **Range `M@N` clamp** | If N > len(args), clamps to max. If M > N, the token becomes empty. |

---

## Full Example

```yaml
@vars:
  BIN = app
  PORT = 8080

# $ run                    → builds + deploys
main: test:
  @ build --release
  @ deploy

# $ run build --target x86_64   → builds for target
build:
  echo "building {{BIN}} for {{--target||default}}"
  go build -o ./build/{{BIN}} ./cmd/{{BIN}}

# $ run test -v                    → runs tests
[exit-on-error]
test:
  echo "running tests..."
  go test ./...

# $ run deploy             → deploys
deploy:
  echo "deploying to {{CWD}} on {{OS}}-{{ARCH}}"
  cp ./build/{{BIN}} ~/.local/bin/

# $ run greet Hello World  → prints 'Hello World'
greet:
  echo {{@}}

# $ run whoami             → prints OS/ARCH
whoami:
  echo {{OS}}-{{ARCH}}

# $ run server             → starts server
server dev:
  @ dev:backend
  @ dev:frontend

dev:backend:
  echo "backend on :{{PORT}}"

dev:frontend:
  echo "frontend on :3000"
```

**Usage with args:**

```bash
$ ./build/run                    # → main (default task)
$ ./build/run greet Hello World  # → Hello World
$ ./build/run test -v            # → runs tests with --verbose flag
$ ./build/run server             # → starts backend + frontend
```

---

## See Also

- [Source & Issues](https://github.com/mhs003/runner)
- `sample.runner` — annotated example in the repo root
