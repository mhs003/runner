# AGENTS.md — AI Agent Context for `runner`

## Project Identity

**runner** is a zero-dependency Go task runner (like `just`/`Make`). It reads a `.runner` config
file, parses it with a custom lexer/parser, resolves task dependencies via topological sort, and
executes commands through `/bin/sh -c`.

- **Module:** `mhs003/runner`
- **Language:** Go 1.25.5
- **Dependencies:** None — Go standard library only
- **Entry point:** `cmd/run/main.go`

---

## Build, Test & Run Commands

| Command | Action |
|---------|--------|
| `make` / `make all` | Builds `build/run` |
| `go build -o build/run ./cmd/run` | Manual build |
| `go test ./...` | Run all tests |
| `go test ./... -v` | Run all tests verbosely |
| `go vet ./...` | Static analysis |
| `go run ./cmd/run [task]` | Run without installing |
| `./build/run --list` | List all tasks |
| `./build/run --dry <task>` | Dry-run (print commands, no exec) |
| `./build/run install` | Installs to `~/.config/hypr/bin/run` |

There are **106 test cases** across 6 test files. There is no CI.

---

## Code Style & Conventions

### General
- Standard Go project layout: `cmd/` for entry point, `internal/` for library packages.
- No external packages — stdlib only.
- No dependency injection, no interfaces for extensibility, no generics.
- Concrete types throughout.

### Naming
- Go-idiomatic `camelCase` for variables and functions, `PascalCase` for exported types.
- Acronyms are all-caps (`ParseError`, not `ParseErr`).

### Error Handling
- `panic` only in `main()` for truly fatal errors (config file not found).
- `fmt.Println(err); os.Exit(1)` for semantic errors (parse, resolve, exec failures).
- Custom error type `config.ParseError` with `Line` field for parse errors.
- `fmt.Errorf` with lowercase messages for resolver errors.

### Comments
- `//` line comments. Some commented-out code blocks preserved for reference.
- No doc comments on exported types or functions.

### Imports
- Grouped in stdlib-only blocks. No blank imports.
- Uses `maps.Copy` (Go 1.21+) for merging maps.

### Data Structures
- Prefer `map[string]*T` over slices when keyed lookup is needed.
- Pointer receivers only for data mutation (e.g., resolver appending to `*[]*Task`).
- Structs are pure data; only `ParseError` has a method (to implement `error`).

### Tests
- Standard Go `_test.go` files in each package.
- Table-driven tests where appropriate.
- `interpolate()` and `resolveLazyValue()` are tested with shell execution commands (e.g., `$(echo ...)`, `$(sh -c 'exit 1')`).

---

## Package Map

```
cmd/run/main.go            — CLI entry, flag parsing, orchestration
internal/config/
  ast.go                   — Type definitions: File, Task, Dep, BodyLine, RunArgs, ParseError
  lexer.go                 — Text → []Line (indent tracking, comment stripping)
  lexer_test.go            — Lexer unit tests
  parser.go                — []Line → *File AST (dispatched parser methods)
  parser_test.go           — Parser unit tests
  args.go                  — ParseArgs CLI argument parser
  args_test.go             — ParseArgs unit tests
  loader.go                — Reads .runner from given path
  vars.go                  — Resolves {{VAR}} references in @vars values
  vars_test.go             — ResolveVars unit tests
internal/display/
  list.go                  — --list output formatting (pretty text + JSON)
internal/engine/
  resolver.go              — Topological sort of task DAG with cycle detection
  resolver_test.go         — Resolver unit tests
  executor.go              — Collects task commands recursively, lazy $(cmd) + caching, executes via /bin/sh -c
  executor_test.go         — Interpolate & resolveLazyValue unit tests
```

---

## Data Flow

```
.runner file ──→ Load ──→ Lex ──→ Parse ──→ ResolveVars ──→ *File (AST)
                                                                 │
CLI args ──→ ParseArgs ──→ vars map ────────────────────────────┤
                                                                 ▼
                                                          Resolve ──→ Execute ──→ /bin/sh -c
```

1. **Load** — read `.runner` from CWD (`loader.go`)
2. **Lex** — split text into lines with indent info, strip comments (`lexer.go`)
3. **Parse** — build AST (`@vars`, tasks with deps/commands) from lines (`parser.go`)
4. **ResolveVars** — resolve `{{VAR}}` references in `@vars` values via `config.ResolveVars(file.Vars)`. Skips values containing `$(` (deferred to lazy execution).
5. **Inject** — merge file vars (pre-resolved), built-ins (`CWD`, `OS`, `ARCH`), and CLI args into `vars` map
6. **Resolve** — topological sort of task DAG (DFS with cycle detection)
7. **Execute** — collect all commands (including recursively inlined deps with lazy `$(cmd)` resolution + caching) into one script per task, run via `/bin/sh -c`

---

## Important Gotchas

1. **Indentation is space-based** — the lexer counts individual space characters. Tabs would be
   treated as indent level 1. The convention is 2-space indentation.

2. **Quotes in `.runner` are literal** — `BIN="run"` stores `"run"` (with quotes) as the variable
   value. It's up to the shell command to handle quotes.

3. **No `--keep-going` on error** — if any command fails, execution stops immediately.
   Subsequent tasks in the resolved order are skipped.

4. **Stdio is fully passthrough** — commands inherit stdin/stdout/stderr from the parent process.
   Good for interactive commands but means there's no output capture.

5. **Variable interpolation uses regex** — the `interpolate()` function in `executor.go` uses
   `regexp.MustCompile(\{\{(.+?)\}\})` to find all `{{...}}` tokens and resolve them individually
   via `resolveLazyValue()`. This avoids the substring corruption problem of naive `ReplaceAll`.
   The `{{key||default}}` syntax provides a fallback: if `key` is not found or empty, `default`
   is used (resolved as a variable key first, then as a literal string). `interpolate` returns
   `(string, error)` — a failed `$(cmd)` halts execution. File vars are pre-resolved by
   `config.ResolveVars()` before execution; values containing `$(` are kept as-is for lazy
   evaluation at runtime with a `shellCache` to avoid repeated execution.

---

## Modifying `.runner` Format

The parser currently supports `@vars` as the only meta block. To add a new meta block
(`@something:`):

1. Add a `parseXxx()` method to the `parser` struct following the `parseVars()` pattern.
2. Add a branch in `parser.dispatchHeader()` for the new block name.
3. The `parser.parse()` loop will dispatch automatically.

---

## Common Pitfalls When Editing

- **Adding a new type?** Put it in `ast.go` — that's the single source of truth for types.
- **New meta block?** Add a `parseXxx()` method to the `parser` struct and a branch in `dispatchHeader()`.
- **New CLI flag?** Use `flag` stdlib in `main.go`, then add it to the vars map before resolve.
- **New built-in variable?** Add it in `main.go` around lines 87-89 where `CWD`/`OS`/`ARCH` are set.
- **Changing the lexer?** Make sure to preserve the comment stripping rules (full-line `#` vs inline at indent 0).
- **Adding execution features?** Modify `executor.go` — that's the sole execution engine.
- **Adding tests?** Put them in the same package with `_test.go` suffix. Use table-driven tests.
- **ParseArgs lives in `args.go`** — it's a standalone function with its own test file `args_test.go`.
- **Display formatting lives in `internal/display/list.go`** — add new `--list` output modes there, not in `main.go`. The `PrintTasks` function receives a `*config.File` and a `jsonOutput bool`. Add new flags for custom formatting in `main.go` and pass them through.
- **Adding `--list` sub-features** — the display package imports only `encoding/json`, `fmt`, `sort`, and `strings` (all stdlib). No imports from `engine` or `config` beyond the types in `ast.go`.
