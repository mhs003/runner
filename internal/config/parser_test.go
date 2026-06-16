package config

import (
	"reflect"
	"testing"
)

func lex(s string) []Line {
	return Lex(s)
}

func TestParseEmpty(t *testing.T) {
	f, err := Parse([]Line{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Vars == nil {
		t.Fatal("Vars map should not be nil")
	}
	if f.Tasks == nil {
		t.Fatal("Tasks map should not be nil")
	}
	if len(f.Vars) != 0 {
		t.Fatalf("expected 0 vars, got %d", len(f.Vars))
	}
	if len(f.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(f.Tasks))
	}
}

func TestParseVars(t *testing.T) {
	f, err := Parse(lex("@vars:\n  KEY=val\n  FOO=bar"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Vars["KEY"] != "val" {
		t.Fatalf("expected KEY=val, got KEY=%s", f.Vars["KEY"])
	}
	if f.Vars["FOO"] != "bar" {
		t.Fatalf("expected FOO=bar, got FOO=%s", f.Vars["FOO"])
	}
}

func TestParseVarsWithSpaces(t *testing.T) {
	f, err := Parse(lex("@vars:\n  KEY = val with spaces"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Vars["KEY"] != "val with spaces" {
		t.Fatalf("expected KEY='val with spaces', got KEY='%s'", f.Vars["KEY"])
	}
}

func TestParseVarsWithQuotes(t *testing.T) {
	f, err := Parse(lex(`@vars:
  BIN = "run"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// quotes are stored literally
	if f.Vars["BIN"] != `"run"` {
		t.Fatalf("expected BIN='\"run\"', got BIN='%s'", f.Vars["BIN"])
	}
}

func TestParseSingleTask(t *testing.T) {
	f, err := Parse(lex("build:\n  echo hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task, ok := f.Tasks["build"]
	if !ok {
		t.Fatal("expected task 'build' to exist")
	}
	if task.Name != "build" {
		t.Fatalf("expected task name 'build', got '%s'", task.Name)
	}
	if len(task.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(task.Commands))
	}
	if task.Commands[0] != "echo hello" {
		t.Fatalf("expected command 'echo hello', got '%s'", task.Commands[0])
	}
	if len(task.Deps) != 0 {
		t.Fatalf("expected 0 deps, got %d", len(task.Deps))
	}
}

func TestParseTaskWithDeps(t *testing.T) {
	f, err := Parse(lex("build: dep1 dep2:\n  echo hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// TrimSuffix only removes final ':', so "build: dep1 dep2:" → name="build: dep1 dep2"
	// Then SplitN(" ", 2) → ["build:", "dep1 dep2"], taskName="build:"
	task, ok := f.Tasks["build:"]
	if !ok {
		t.Fatalf("expected task 'build:' to exist, got tasks: %v", f.Tasks)
	}
	expected := []string{"dep1", "dep2"}
	if !reflect.DeepEqual(task.Deps, expected) {
		t.Fatalf("expected deps %v, got %v", expected, task.Deps)
	}
}

func TestParseInlineDeps(t *testing.T) {
	f, err := Parse(lex("build:\n  @ dep1\n  @dep2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := f.Tasks["build"]
	expected := []string{"dep1", "dep2"}
	if !reflect.DeepEqual(task.Deps, expected) {
		t.Fatalf("expected deps %v, got %v", expected, task.Deps)
	}
}

func TestParseMultipleCommands(t *testing.T) {
	f, err := Parse(lex("build:\n  echo first\n  echo second"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := f.Tasks["build"]
	expected := []string{"echo first", "echo second"}
	if !reflect.DeepEqual(task.Commands, expected) {
		t.Fatalf("expected commands %v, got %v", expected, task.Commands)
	}
}

func TestParseMultipleTasks(t *testing.T) {
	input := "build:\n  echo build\n\ntest:\n  echo test"
	f, err := Parse(lex(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(f.Tasks))
	}
	if _, ok := f.Tasks["build"]; !ok {
		t.Fatal("expected task 'build'")
	}
	if _, ok := f.Tasks["test"]; !ok {
		t.Fatal("expected task 'test'")
	}
}

func TestParseVarsAndTasks(t *testing.T) {
	input := "@vars:\n  BIN=app\n\nbuild:\n  echo {{BIN}}"
	f, err := Parse(lex(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Vars["BIN"] != "app" {
		t.Fatalf("expected BIN=app, got BIN=%s", f.Vars["BIN"])
	}
	if _, ok := f.Tasks["build"]; !ok {
		t.Fatal("expected task 'build'")
	}
}

func TestParseVerboseCommand(t *testing.T) {
	// verbose prefix ! is part of the command text — parsed, not processed
	f, err := Parse(lex("build:\n  !echo hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := f.Tasks["build"]
	if task.Commands[0] != "!echo hello" {
		t.Fatalf("expected command '!echo hello', got '%s'", task.Commands[0])
	}
}

func TestParseErrorUnknownKeyword(t *testing.T) {
	_, err := Parse(lex("unknown text"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Line != 1 {
		t.Fatalf("expected line 1, got %d", pe.Line)
	}
}

func TestParseErrorCommandOutsideTask(t *testing.T) {
	_, err := Parse(lex("  command without task"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*ParseError); !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
}

// ParseArgs tests

func TestParseArgsEmpty(t *testing.T) {
	ra := ParseArgs([]string{})
	if len(ra.Positional) != 0 {
		t.Fatalf("expected 0 positional, got %d", len(ra.Positional))
	}
	if len(ra.Named) != 0 {
		t.Fatalf("expected 0 named, got %d", len(ra.Named))
	}
	if len(ra.Flags) != 0 {
		t.Fatalf("expected 0 flags, got %d", len(ra.Flags))
	}
}

func TestParseArgsPositional(t *testing.T) {
	ra := ParseArgs([]string{"foo", "bar"})
	expected := []string{"foo", "bar"}
	if !reflect.DeepEqual(ra.Positional, expected) {
		t.Fatalf("expected positional %v, got %v", expected, ra.Positional)
	}
}

func TestParseArgsNamed(t *testing.T) {
	ra := ParseArgs([]string{"--entry", "./cmd/app"})
	if ra.Named["--entry"] != "./cmd/app" {
		t.Fatalf("expected --entry=./cmd/app, got --entry=%s", ra.Named["--entry"])
	}
}

func TestParseArgsShortNamed(t *testing.T) {
	ra := ParseArgs([]string{"-e", "./cmd/app"})
	if ra.Named["-e"] != "./cmd/app" {
		t.Fatalf("expected -e=./cmd/app, got -e=%s", ra.Named["-e"])
	}
}

func TestParseArgsFlag(t *testing.T) {
	ra := ParseArgs([]string{"--verbose"})
	if !ra.Flags["--verbose"] {
		t.Fatal("expected --verbose flag to be true")
	}
}

func TestParseArgsShortFlag(t *testing.T) {
	ra := ParseArgs([]string{"-v"})
	if !ra.Flags["-v"] {
		t.Fatal("expected -v flag to be true")
	}
}

func TestParseArgsFlagAfterFlag(t *testing.T) {
	ra := ParseArgs([]string{"--verbose", "--dry"})
	if !ra.Flags["--verbose"] || !ra.Flags["--dry"] {
		t.Fatal("expected both flags to be true")
	}
}

func TestParseArgsFlagThenNamed(t *testing.T) {
	ra := ParseArgs([]string{"--dry", "--entry", "main"})
	if !ra.Flags["--dry"] {
		t.Fatal("expected --dry flag")
	}
	if ra.Named["--entry"] != "main" {
		t.Fatalf("expected --entry=main, got %s", ra.Named["--entry"])
	}
}

func TestParseArgsMixed(t *testing.T) {
	// --dry followed by --entry treats --dry as a flag (next is another flag)
	ra := ParseArgs([]string{"--dry", "--entry", "./cmd/app", "build", "-v"})
	if !ra.Flags["--dry"] {
		t.Fatal("expected --dry flag")
	}
	if ra.Named["--entry"] != "./cmd/app" {
		t.Fatalf("expected --entry=./cmd/app, got %s", ra.Named["--entry"])
	}
	if !ra.Flags["-v"] {
		t.Fatal("expected -v flag")
	}
	expected := []string{"build"}
	if !reflect.DeepEqual(ra.Positional, expected) {
		t.Fatalf("expected positional %v, got %v", expected, ra.Positional)
	}
}
