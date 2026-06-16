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

There are **54 test cases** across 4 test files. There is no CI.

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
- `interpolate()` is tested as a pure function (no shell execution in unit tests).

---

## Package Map

```
cmd/run/main.go            — CLI entry, flag parsing, orchestration
internal/config/
  ast.go                   — Type definitions: File, Task, Condition, RunArgs
  lexer.go                 — Text → []Line (indent tracking, comment stripping)
  lexer_test.go            — Lexer unit tests
  parser.go                — []Line → *File AST + ParseArgs helper
  parser_test.go           — Parser + ParseArgs unit tests
  loader.go                — Reads .runner from CWD
internal/engine/
  resolver.go              — Topological sort of task DAG with cycle detection
  resolver_test.go         — Resolver unit tests
  executor.go              — Runs commands via /bin/sh -c, interpolates vars
  executor_test.go         — Interpolate unit tests
```

---

## Data Flow

```
.runner file ──→ Load ──→ Lex ──→ Parse ──→ *File (AST)
                                                 │
CLI args ──→ ParseArgs ──→ vars map ─────────────┤
                                                 ▼
                                          Resolve ──→ Execute ──→ /bin/sh -c
```

1. **Load** — read `.runner` from CWD (`loader.go`)
2. **Lex** — split text into lines with indent info, strip comments (`lexer.go`)
3. **Parse** — build AST (`@vars`, tasks with deps/commands) from lines (`parser.go`)
4. **Inject** — merge file vars, built-ins (`CWD`, `OS`, `ARCH`), and CLI args into `vars` map
5. **Resolve** — topological sort of task DAG (DFS with cycle detection)
6. **Execute** — run each command in order with variable interpolation

---

## Important Gotchas

1. **`ParseArgs` is marked "SH!T solution"** (`parser.go:94-131`) with a TODO. It doesn't support
   `--key=value` syntax, combined short flags, or `=` in flag values. Be careful when modifying.

2. **`Condition` field in `Task` is dead code** — defined in `ast.go` but never populated or
   checked during execution. It's a planned feature (conditional execution based on env vars).

3. **`loader.go` returns a path that is ignored** — `Load()` returns `([]byte, string, error)`
   but `main.go` discards the path with `_`. The path return exists but isn't used.

4. **Indentation is space-based** — the lexer counts individual space characters. Tabs would be
   treated as indent level 1. The convention is 2-space indentation.

5. **Quotes in `.runner` are literal** — `BIN="run"` stores `"run"` (with quotes) as the variable
   value. It's up to the shell command to handle quotes.

6. **No `--keep-going` on error** — if any command fails, execution stops immediately.
   Subsequent tasks in the resolved order are skipped.

7. **Stdio is fully passthrough** — commands inherit stdin/stdout/stderr from the parent process.
   Good for interactive commands but means there's no output capture.

8. **Variable interpolation uses regex** — the `interpolate()` function in `executor.go` uses
   `regexp.MustCompile(\{\{(.+?)\}\})` to find all `{{...}}` tokens and resolve them individually.
   This avoids the substring corruption problem of naive `ReplaceAll`. The `{{key||default}}`
   syntax provides a fallback: if `key` is not found or empty, `default` is used (resolved as
   a variable key first, then as a literal string).

---

## Modifying `.runner` Format

The parser currently only supports `@vars:` as a meta block. To add a new meta block
(`@something:`), follow the existing pattern at `parser.go:28-45`:

1. Detect the block header by name at indent 0
2. Scan forward consuming indented lines until the next indent-0 line
3. Reset `current = nil`
4. Advance the outer loop index past the block

---

## Common Pitfalls When Editing

- **Adding a new type?** Put it in `ast.go` — that's the single source of truth for types.
- **New meta block?** Add the parsing logic in `parser.go` in the `@` block section.
- **New CLI flag?** Use `flag` stdlib in `main.go`, then add it to the vars map before resolve.
- **New built-in variable?** Add it in `main.go` around lines 96-98 where `CWD`/`OS`/`ARCH` are set.
- **Changing the lexer?** Make sure to preserve the comment stripping rules (full-line `<literal>#</literal>` vs inline at indent 0).
- **Adding execution features?** Modify `executor.go` — that's the sole execution engine.
- **Adding tests?** Put them in the same package with `_test.go` suffix. Use table-driven tests.
