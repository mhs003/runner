# ARCHITECTURE.md — runner Architecture

## Overview

**runner** is a zero-dependency Go task runner. It reads a `.runner` configuration file (a
custom DSL inspired by `just`/`Make`), parses it with a hand-written lexer and parser, resolves
inter-task dependencies via topological sort, and executes shell commands through `/bin/sh -c`.

The entire project is ~600 lines of Go, spread across 9 files in 3 packages (plus 5 test files).
There are no external dependencies and no CI.

---

## Data Flow

```
┌──────────────┐     ┌──────────┐     ┌──────────┐     ┌──────────────────┐
│  .runner     │ ──→ │  Load   │ ──→ │  Lex    │ ──→ │     Parse        │
│  (text file) │     │loader.go│     │lexer.go │     │   parser.go      │
└──────────────┘     └──────────┘     └──────────┘     └────────┬─────────┘
                                                                 │
                                     ┌──────────────┐            │
                                     │   CLI args   │            ▼
                                     │  (--list,    │     ┌──────────────┐
                                     │   --dry,     │     │   *File AST  │
                                     │   <task>,    │     │  (vars,      │
                                     │   --k v, ...)│     │   tasks)     │
                                     └──────┬───────┘     └──────┬───────┘
                                            │                    │
                                            ▼                    ▼
                                     ┌──────────────────────────────┐
                                     │     Variable Injection       │
                                     │  file vars ← built-ins ← CLI │
                                     └──────────────┬───────────────┘
                                                    │
                                                    ▼
                                     ┌──────────────────────────────┐
                                     │  Resolve (Topological Sort)  │
                                     │   DFS + cycle detection     │
                                     └──────────────┬───────────────┘
                                                    │
                                                    ▼
                                     ┌──────────────────────────────┐
                                     │        Execute               │
                                     │  /bin/sh -c per command      │
                                     │  interpolation + verbose     │
                                     └──────────────────────────────┘
```

### Stage Details

| Stage | File | Input | Output | Description |
|-------|------|-------|--------|-------------|
| 1. Load | `loader.go` | file system | `[]byte` | Reads config file from given path. |
| 2. Lex | `lexer.go` | raw text | `[]Line` | Splits by newlines, counts leading spaces as indent, strips comments. |
| 3. Parse | `parser.go` | `[]Line` | `*File` | Builds AST: detects `@vars` meta block, task headers, inline deps, commands. |
| 4. Inject | `main.go` | `*File` + CLI | `map[string]string` | Merges file vars, built-ins (`CWD`, `OS`, `ARCH`), CLI positional/named/flag args. |
| 5. Resolve | `resolver.go` | `*File` + task name | `[]*Task` (ordered) | DFS topological sort with cycle detection via `stack` map. |
| 6. Execute | `executor.go` | ordered tasks + vars | exit code | For each command: interpolate vars, optionally print, run via `/bin/sh -c`. |

---

## Package Map

### `cmd/run/main.go` (130 lines) — Entry Point

**Responsibility:** CLI orchestration.

```
main()
  ├── flag.Parse()              // --list, --dry, --file
  ├── config.Load(path)         // read .runner (or custom --file path)
  ├── config.Lex()              // tokenize
  ├── config.Parse()            // build AST
  ├── [--list: print tasks, exit]
  ├── [validate task exists]
  ├── build vars map
  │   ├── maps.Copy(file.Vars)            // from .runner @vars
  │   ├── CWD, OS, ARCH                   // built-in env vars
  │   ├── config.ParseArgs(args)          // CLI args → RunArgs
  │   ├── maps.Copy(ra.Named)             // --key value pairs
  │   └── ra.Flags → "true"/"false"       // boolean flags
  ├── engine.Resolve()                    // topological sort
  └── engine.Execute()                    // run commands
```

**Key design choices:**
- `panic` for load errors (file not found) — considered unrecoverable.
- `fmt.Println(err); os.Exit(1)` for parse/resolve/exec errors — user-facing.
- Variables are accumulated in a single `map[string]string` with progressive `maps.Copy` calls:
  Built-ins `CWD`, `OS`, `ARCH` are set first, then file vars overwrite them, then CLI named/flag
  vars overwrite again. This gives CLI args highest precedence.

---

### `internal/config/ast.go` (25 lines) — Type Definitions

**Responsibility:** Single source of truth for all data types in the project.

```go
type File struct {
    Vars  map[string]string    // @vars: key=value pairs
    Tasks map[string]*Task     // task name → Task (pointer)
}

type Task struct {
    Name     string           // unique identifier
    Deps     []string         // dependency task names
    Commands []string         // shell commands (in order)
}

type RunArgs struct {
    Positional []string         // non-flag arguments
    Named      map[string]string // --key value / -k value
    Flags      map[string]bool   // --flag / -f (no value)
}

type ParseError struct {
    Line int
    Msg  string
}
func (e *ParseError) Error() string
```

**Design notes:**
- `File` uses `map[string]*Task` not `map[string]Task` — pointers enable mutation via shared
  references (used in resolver when appending to `[]*Task`).
- `ParseError` implements the `error` interface. The `Line` field is populated by the parser
  but never read by the caller (`main.go` just prints `err.Error()`).
- `RunArgs` is the result of `ParseArgs()`, consumed immediately in `main.go`. It's not stored
  in the AST.

---

### `internal/config/lexer.go` (48 lines) — Lexer/Tokenizer

**Responsibility:** Convert raw text to structured `[]Line` with indent tracking and comment stripping.

```go
type Line struct {
    No     int    // 1-indexed line number (matches original file)
    Indent int    // number of leading space characters
    Text   string // trimmed content (leading/trailing whitespace removed)
}
```

**Algorithm:**
1. Split input by `\n`
2. For each line:
   a. Count leading spaces → `indent`
   b. `strings.TrimSpace` → `content`
   c. If content starts with `#` → skip (comment line)
   d. If indent == 0 and content contains `#` → strip everything from `#` onward,
      then trim again (inline comment at top level)
   e. Otherwise → append `Line{No, Indent, content}`

**Comment rules:**
- Lines whose trimmed content starts with `#` are skipped entirely.
- Top-level lines (indent 0) can have inline comments: `taskname: # comment` → `taskname:`
- Indented lines (commands) keep `#` as literal text — any `#` in a command is passed to the
  shell.
- After inline comment stripping, the line is trimmed again to remove trailing spaces.

**Notable:**
- Tabs count as indent level 1 (one character), but the convention is 2-space indentation.
- Empty lines produce `Line{Text: ""}` — the parser handles skipping them.
- Indent-based scoping: indent > 0 means "inside a block" (task body or meta block).

---

### `internal/config/parser.go` (95 lines) — Parser

**Responsibility:** Convert `[]Line` into `*File` AST.

**Architecture:**

The parser is a hand-written single-pass state machine using dispatched methods on a
`parser` struct. It maintains a `current *Task` pointer and an index `i` for scanning.

```
parse(lines):
  f = &File{}
  p = parser{f: f, lines: lines}

  for p.i over lines:
    if line is empty → continue

    if line ends with ':' at indent 0:
      → BLOCK HEADER
      dispatchHeader(name)  // delegates to parseVars or parseTaskHeader

    if indent > 0:
      if current is nil → error: command outside task
      if line starts with '@':
        → INLINE DEPENDENCY
        append strip('@') to current.Deps
      else:
        → COMMAND
        append raw line to current.Commands
```

**Parsing methods:**
- `parseVars()` — scans forward, parses `key=value`, stores in `f.Vars`.
- `parseTaskHeader()` — parses `"name: dep1 dep2:"` into task name and deps.
  Uses `strings.TrimSuffix(parts[0], ":")` so task name never includes trailing `:`.
- `dispatchHeader()` — routes `@vars` to `parseVars()`, all other headers to `parseTaskHeader()`.

**`@vars:` block parsing:**
- Lines within `@vars:` are split on first `=`.
- Both key and value are `TrimSpace`-d.
- Quotes in values are **not** stripped — `BIN = "app"` stores `"app"` (with quotes).
- The shell handles quoting at execution time.

**Task header parsing:**
- A task header `taskname:` at indent 0 creates a task.
- If the header has spaces: `taskname: dep1 dep2`: the parts after
  the first word are treated as dependency names (split by `strings.Fields`).
- Trailing colons on dep names are trimmed (`build: dep1 dep2:` → name=`"build"`, deps=`["dep1", "dep2"]`).
- Duplicate task names return a `ParseError`.

**Inline dependencies:**
- Within a task body, `@depname` adds `depname` to the current task's `Deps` list.
- Multiple deps can be on one line: `@ dep1 dep2` (after stripping `@`, `strings.Fields`
  splits the rest).

---

### `internal/config/args.go` (45 lines) — CLI Argument Parser

**Responsibility:** Parse CLI arguments into `RunArgs` struct (positional, named, flags).

```go
func ParseArgs(args []string) RunArgs
```

**Features:**
- `--key value` and `-k value` → `Named["--key"]` / `Named["-k"]`
- `--key=value` → `Named["--key"]` (via `strings.Cut`)
- `--flag` and `-f` → `Flags["--flag"]` / `Flags["-f"]` (boolean)
- Combined short flags: `-abc` → `Flags["-a"]`, `Flags["-b"]`, `Flags["-c"]`
- `=` in values: `--path=/usr/bin` works correctly (value is `/usr/bin`)
- Positional args: anything that doesn't start with `-`

**Limitations:**
- No combined short flags with values (`-k value` must be separate tokens).
- If `--key` is followed by another flag (`--key --other`), `--key` is treated as a flag
  (boolean), not as a named arg. This is intentional but restrictive.
- Named keys preserve the `--` or `-` prefix, so accessing them in `.runner` requires
  `{{--entry}}` not `{{entry}}`.

---

### `internal/config/loader.go` (15 lines) — File Loader

**Responsibility:** Read config file from given path.

```go
func Load(path string) ([]byte, error) {
    return os.ReadFile(path)
}
```

**Design choices:**
- Takes explicit path parameter (set by `--file` flag or default `.runner` in `main.go`).
- No upward directory search — `--file` enables custom paths.
- No fallback if file doesn't exist — `os.ReadFile` error propagates directly.

---

### `internal/engine/resolver.go` (40 lines) — Dependency Resolver

**Responsibility:** Topological sort of tasks with cycle detection.

```go
func Resolve(f *File, name string, seen map[string]bool,
             stack map[string]bool, out *[]*Task) error
```

**Algorithm:** Recursive DFS with an explicit call stack for cycle detection.

```
Resolve(f, name, seen, stack, out):
  if name is in stack → CYCLE DETECTED → return error
  if name is in seen  → already processed → return nil

  task = f.Tasks[name]
  if task missing → return error "Unknown dependency"

  add name to stack
  for each dep in task.Deps:
    Resolve(f, dep, seen, stack, out)
  remove name from stack
  mark name as seen
  append task to out
```

**Edge cases:**
- Tasks with no commands and no deps produce a warning (`fmt.Printf`), but don't halt.
- Missing dependency check happens **before** any field access on the task pointer,
  avoiding nil dereference.
- Self-referencing tasks (`a: @ a` or `a: a`) are caught by the stack check.

**Bug fixed:** The resolver previously accessed task fields (`t.Commands`, `t.Deps`) before
checking if the task existed, which panicked on references to undefined task names. The
existence check was moved before any field access.

---

### `internal/engine/executor.go` (67 lines) — Command Executor

**Responsibility:** Run resolved tasks in order, handling interpolation, verbose mode, and dry-run.

```
Execute(tasks, vars, dry):
  for each task in resolved order:
    for each command:
      if command starts with '!':
        strip '!' → shouldVerbose = true
      cmd = interpolate(command, vars)

      if dry → print cmd, continue

      if shouldVerbose → print "> cmd"
      run /bin/sh -c cmd with passthrough stdio
      if error → return error immediately
```

**Variable interpolation (`interpolate`):**

```go
var tokenRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

func interpolate(s string, vars map[string]string) string {
    return tokenRe.ReplaceAllStringFunc(s, func(match string) string {
        inner := match[2 : len(match)-2]

        var primary, fallback string
        if idx := strings.Index(inner, "||"); idx >= 0 {
            primary = inner[:idx]
            fallback = inner[idx+2:]
        } else {
            primary = inner
        }

        if v, ok := vars[primary]; ok && v != "" {
            return v
        }

        if fallback != "" {
            if v, ok := vars[fallback]; ok {
                return v
            }
            return fallback
        }

        return match
    })
}
```

**Key characteristics:**
- **Regex-based token matching** — finds each `{{...}}` token individually, avoiding the
  substring corruption problem of naive `ReplaceAll`. Config: `\{\{(.+?)\}\}`.
- **`{{key||default}}` fallback syntax** — if `key` is not found or is empty string, the
  `default` is tried first as a variable key, then as a literal string. This allows patterns
  like `{{--entry||ENTRY}}` (CLI override with file-var fallback).
- **Unknown tokens preserved** — `{{MISSING}}` with no matching variable is left as-is in
  the output.
- **`/bin/sh -c` invocation** — all commands run through the shell. Shell features (pipes,
  redirects, variable expansion) work natively.
- **Verbose mode (`!` prefix)** prints the interpolated command to stdout before running it.
- **Dry-run** prints interpolated commands without executing them.
- **No `--keep-going`** — first failing command stops execution entirely.
- **Stdio passthrough** — commands inherit the terminal.

---

## `.runner` File Format

### Full Grammar

```
file           = { meta_block | task_definition | empty_line | comment }
meta_block     = "@vars:" newline { var_line newline }
var_line       = indent identifier "=" value
task_definition = header newline { body_line newline }
header         = taskname [ ":" deps ] ":"
               | taskname deps ":"
body_line      = indent command
               | indent "@" dependency { dependency }
deps           = dependency { dependency }
dependency     = identifier
command        = [ "!" ] text
comment        = "#" text { text }
empty_line     = ""
```

### Variable Interpolation Syntax

```
{{VAR_NAME}}        — from @vars block
{{CWD}}             — built-in: working directory
{{OS}}              — built-in: runtime.GOOS
{{ARCH}}            — built-in: runtime.GOARCH
{{ARGS}}            — all positional CLI args joined by space
{{1}}, {{2}}, ...   — individual positional CLI args
{{--key}}           — named CLI arg (--key value)
{{-k}}              — named short CLI arg (-k value)
{{--flag}}          — boolean CLI flag (→ "true" or "false")
{{-f}}              — boolean short CLI flag
{{key||default}}    — use `key` if non-empty, fall back to `default`
                      (default resolved as var key first, then literal)
```

### Variable Precedence (highest wins)

```
Built-ins (CWD, OS, ARCH)        ← lowest
  └─ File vars (@vars block)
      └─ CLI positional (1, 2, ...)
          └─ CLI named args (--key, -k)
              └─ CLI flags (--flag, -f)  ← highest
```

### Example

```yaml
@vars:
  BIN = app
  PORT = 8080

build:
  @ lint
  echo "Building {{BIN}} for {{OS}}-{{ARCH}}"

build:dev:
  !go build -o ./bin/{{BIN}} ./cmd/app

lint:
  go vet ./...
```

---

## Design Decisions & Trade-offs

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| Zero external dependencies | No vendor overhead, trivially buildable | Must hand-write parser, no dependency injection |
| Custom file format (not YAML/TOML) | Minimal, task-runner-specific syntax | Steep learning curve, no ecosystem tooling |
| `/bin/sh -c` execution | Shell features work (pipes, redirects) | Shell injection, platform-specific |
| Regex-based variable interpolation | Avoids substring corruption, enables fallback syntax | Regex compile cost (one-time) vs naive ReplaceAll |
| 62 unit tests across 5 files | Safety net for refactoring | No integration or end-to-end tests |
| Sequential execution (no `--keep-going`) | Simplicity | First failure aborts all remaining tasks |
| `--file` flag for custom config paths | Flexibility without breaking default | No upward directory search |
| 2-space indentation | Visual clarity | Tab = 1 indent (breaks convention) |

---

## Known Issues & TODOs

1. **No `--help` flag**: `flag` package auto-generates basic help, but there's no explicit
   `--help` handling.

2. **Fallback syntax edge case**: `{{key||default}}` resolves `default` as a var key first,
   then as a literal. If a var exists with the same name as the intended literal, the var
   takes precedence. This is intentional but could surprise users.

3. **ParseArgs combined flag limitation**: Combined short flags (`-abc`) set all as boolean
   flags. There's no support for combined short flags with values (e.g., `-o value` must be
   separate tokens).
