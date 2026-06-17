# ARCHITECTURE.md — runner Architecture

## Overview

**runner** is a zero-dependency Go task runner. It reads a `.runner` configuration file (a
custom DSL inspired by `just`/`Make`), parses it with a hand-written lexer and parser, resolves
inter-task dependencies via topological sort, and executes shell commands through `/bin/sh -c`.

The entire project is ~760 lines of Go, spread across 9 source files in 4 packages (plus 5 test files).
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
                                      │  (--list,    │     ┌──────────────────┐
                                      │   --dry,     │     │   *File AST      │
                                      │   <task>,    │     │  (vars, tasks)   │
                                      │   --k v, ...)│     └──┬───────┬───────┘
                                      └──────┬───────┘        │       │
                                             │            ┌───┘       └───┐
                                             ▼            ▼               ▼
                                      ┌──────────┐  ┌─────────────┐ ┌──────────┐
                                      │  --list  │  │  Variable   │ │ --list   │
                                      │  (text)  │  │  Injection  │ │  (JSON)  │
                                      │    ↓     │  └──────┬──────┘ │    ↓     │
                                      │ display  │         │        │ display  │
                                      │.PrintTasks│        ▼        │.PrintTasks│
                                      └──────────┘  ┌──────────┐  └──────────┘
                                                     │ Resolve  │
                                                     └────┬─────┘
                                                          │
                                                          ▼
                                                   ┌──────────────┐
                                                   │   Execute    │
                                                   │ /bin/sh -c   │
                                                   └──────────────┘
```

### Stage Details

| Stage | File | Input | Output | Description |
|-------|------|-------|--------|-------------|
| 1. Load | `loader.go` | file system | `[]byte` | Reads config file from given path. |
| 2. Lex | `lexer.go` | raw text | `[]Line` | Splits by newlines, counts leading spaces as indent, strips comments. |
| 3. Parse | `parser.go` | `[]Line` | `*File` | Builds AST: detects `@vars` meta block, task headers (with `HeaderDeps`), body lines (typed `cmd` or `dep` with args), `[exit-on-error]` annotations. |
| 4. Display | `list.go` | `*File` + `--json` flag | stdout (text/JSON) | Formats and prints all tasks, deps, and vars. Used by `--list`. Branches off after Parse; the main flow continues to Inject. |
| 5. Inject | `main.go` | `*File` + CLI | `map[string]string` | Merges file vars, built-ins (`CWD`, `OS`, `ARCH`), CLI positional/named/flag args. |
| 6. Resolve | `resolver.go` | `*File` + task name | `[]*Task` (ordered) | DFS topological sort on `HeaderDeps` only with cycle detection via `stack` map. |
| 7. Execute | `executor.go` | ordered tasks + vars | exit code | `collectCommands` recursively gathers commands from body lines (inlining `@ dep` calls), builds one combined script per task, runs via `/bin/sh -c`. |

---

## Package Map

### `cmd/run/main.go` (114 lines) — Entry Point

**Responsibility:** CLI orchestration.

```
main()
  ├── flag.Parse()              // --list, --dry, --file, --init, --json
  ├── [--init: scaffold .runner, exit]
  ├── config.Load(path)         // read .runner (or ~/.runner.global fallback)
  ├── config.Lex()              // tokenize
  ├── config.Parse()            // build AST
  ├── [--list: display.PrintTasks(file, *showJSON), exit]
  ├── [validate task exists]
  ├── build vars map
  │   ├── maps.Copy(file.Vars)            // from .runner @vars
  │   ├── CWD, OS, ARCH                   // built-in env vars
  │   ├── config.ParseArgs(args)          // CLI args → RunArgs
  │   ├── maps.Copy(ra.Named)             // --key value pairs
  │   └── ra.Flags → "true"/"false"       // boolean flags
  ├── engine.Resolve()                    // topological sort
  └── engine.Execute(file, order, vars)   // run commands
```

**Key design choices:**
- `panic` for load errors (file not found) — considered unrecoverable.
- `fmt.Println(err); os.Exit(1)` for parse/resolve/exec errors — user-facing.
- Variables are accumulated in a single `map[string]string` with progressive `maps.Copy` calls:
  Built-ins `CWD`, `OS`, `ARCH` are set first, then file vars overwrite them, then CLI named/flag
  vars overwrite again. This gives CLI args highest precedence.

---

### `internal/config/ast.go` (39 lines) — Type Definitions

**Responsibility:** Single source of truth for all data types in the project.

```go
type File struct {
    Vars  map[string]string    // @vars: key=value pairs
    Tasks map[string]*Task     // task name → Task (pointer)
}

type Dep struct {
    Name string               // dependency task name
    Args []string             // per-call dep arguments (e.g. --target x86_64)
}

type BodyLine struct {
    Type string               // "cmd" or "dep"
    Text string               // command text (with ! prefix) or dep name
    Args []string             // dep arguments (only for "dep" type)
}

type Task struct {
    Name        string       // unique identifier
    HeaderDeps  []Dep        // dependency task names (resolved before execution)
    BodyLines   []BodyLine   // body lines in order (commands + inline deps)
    ExitOnError bool         // [exit-on-error] annotation
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
`parser` struct. It maintains a `current *Task` pointer, an index `i` for scanning,
and a `pendingAnnotations` slice for inter-line annotations.

```
parse(lines):
  f = &File{}
  p = parser{f: f, lines: lines}

  for p.i over lines:
    if line is empty → continue

    if line is "[annotation]" at indent 0:
      → ANNOTATION
      push to pendingAnnotations

    if line ends with ':' at indent 0:
      → BLOCK HEADER
      dispatchHeader(name)  // delegates to parseVars or parseTaskHeader

    if indent > 0:
      if current is nil → error: command outside task
      if line starts with '@':
        → INLINE DEPENDENCY
        strip '@', split parts → BodyLine{Type:"dep", Text:parts[0], Args:parts[1:]}
        append to current.BodyLines
      else:
        → COMMAND
        append BodyLine{Type:"cmd", Text:raw_line} to current.BodyLines
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

### `internal/display/list.go` (138 lines) — Task List Formatter

**Responsibility:** Format and print the `--list` output (plain text or JSON).

```go
func PrintTasks(f *config.File, jsonOutput bool)
```

Called from `main.go` when `--list` is passed. Two output modes:

- **Pretty text** — aligned columns listing tasks, their body lines, dependencies, and vars.
- **JSON** — structured output consumable by external tools (VS Code extension, scripts).

The package sorts tasks alphabetically, computes column alignment padding, and
prints directly to stdout. No return value — the caller handles `os.Exit(0)`.

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
  for each dep in task.HeaderDeps:
    Resolve(f, dep.Name, seen, stack, out)
  remove name from stack
  mark name as seen
  append task to out
```

**Edge cases:**
- Tasks with no BodyLines and no HeaderDeps produce a warning (`fmt.Printf`), but don't halt.
- Missing dependency check happens **before** any field access on the task pointer,
  avoiding nil dereference.
- Self-referencing tasks (`a: a:`) are caught by the stack check.

**Bug fixed:** The resolver previously accessed task fields (`t.BodyLines`, `t.HeaderDeps`) before
checking if the task existed, which panicked on references to undefined task names. The
existence check was moved before any field access.

**Note:** `HeaderDeps` only includes deps from the task header line (`taskname: dep1 dep2:`).
Inline deps (`@ dep`) inside the body are not resolved by the resolver — they are inlined
at execution time by `collectCommands`.

---

### `internal/engine/executor.go` (143 lines) — Command Executor

**Responsibility:** Collect all commands (including recursively inlined deps) into one script per task, execute via `/bin/sh -c`.

```
Execute(file, order, vars, dry):
  for each task in resolved order:
    runTask(file, task, vars, dry, empty_stack)

runTask(file, task, vars, dry, stack):
  cmds = collectCommands(file, task, vars, stack)
  if cmds empty → return

  if dry → print each cmd, return

  script = set -e (if ExitOnError) + join cmds with newlines
  run /bin/sh -c script with passthrough stdio
  if error → return error immediately

collectCommands(file, task, vars, stack):
  if task.Name in stack → CYCLE DETECTED → return error
  add task.Name to stack

  cmds = []
  for each BodyLine:
    case "cmd":
      strip '!' if present (verbose)
      cmd = interpolate(text, vars)
      if verbose → prepend echo '> cmd'
      append cmd to cmds

    case "dep":
      apply line.Args to vars (per-call dep arguments)
      dep = file.Tasks[line.Text]
      sub = collectCommands(file, dep, vars, stack)
      append sub to cmds
      revert line.Args from vars

  remove task.Name from stack
  return cmds
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
- **Single `/bin/sh -c` per task** — all commands (including recursively inlined deps) are
  collected into one shell script. Shell variables (`$MSG`) persist across `@ dep` boundaries.
- **Verbose mode (`!` prefix)** — emits `echo '> <cmd>'` inline in the combined script before
  the command itself, so verbose output appears correctly in the single-shell session.
- **Dry-run** prints the flat list of collected commands in order, without execution.
- **Inline dep recursion** — `collectCommands` walks `BodyLines`, and for each `"dep"` line
  recursively collects the target task's commands. Per-call dep args are applied to vars during
  collection and cleaned up afterward.
- **Cycle detection** — a `stack` map tracks the current recursion chain in `collectCommands`,
  catching cycles in inline dep references.
- **No `--keep-going`** — first failing command stops execution entirely.
- **Stdio passthrough** — commands inherit the terminal.

---

## `.runner` File Format

### Full Grammar

```
file           = { meta_block | annotation | task_definition | empty_line | comment }
meta_block     = "@vars:" newline { var_line newline }
var_line       = indent identifier "=" value
annotation     = "[" identifier "]"
task_definition = header newline { body_line newline }
header         = taskname [ ":" deps ] ":"
               | taskname deps ":"
body_line      = indent command
               | indent "@" dependency { arg }
deps           = dependency { dependency }
dependency     = identifier
arg            = identifier (args are passed as {{--key}} vars to the dep)
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

# session sharing — shell state persists across body lines
main:
    MSG="Starting build"
    echo $MSG
    @ build --target x86_64
    @ deploy

# dep arguments — {{--target}} injected into build's vars
build:
    echo "building '{{BIN}}' for '{{--target}}'"

deploy:
    echo "deploying to {{CWD}} on {{OS}}-{{ARCH}}"

# [exit-on-error] — shell exits on first non-zero command
[exit-on-error]
test:
    echo "running tests..."
    false
    echo "this never runs"
```

---

## Design Decisions & Trade-offs

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| Zero external dependencies | No vendor overhead, trivially buildable | Must hand-write parser, no dependency injection |
| Custom file format (not YAML/TOML) | Minimal, task-runner-specific syntax | Steep learning curve, no ecosystem tooling |
| `/bin/sh -c` execution | Shell features work (pipes, redirects) | Shell injection, platform-specific |
| Regex-based variable interpolation | Avoids substring corruption, enables fallback syntax | Regex compile cost (one-time) vs naive ReplaceAll |
| 73 unit tests across 5 files | Safety net for refactoring | No integration or end-to-end tests |
| Single `/bin/sh -c` per task with command inlining | Shell variable persistence across `@ dep` boundaries | Larger script per task, harder to debug individual failures |
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

4. **--dry with inline deps**: Dry-run prints the flat command list; there's no visual
   distinction between commands from the parent task vs. commands from an inlined dep.

5. **Dep args from multiple callers**: If two tasks call the same dep with different args,
   the header-dep resolved order runs the dep once (without args). Each inline call site
   inlines its own copy with per-call args. No last-wins ambiguity.
