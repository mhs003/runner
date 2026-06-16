package config

import (
	"reflect"
	"strings"
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
	task, ok := f.Tasks["build"]
	if !ok {
		t.Fatalf("expected task 'build' to exist, got tasks: %v", f.Tasks)
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

func TestParseExitOnErrorAnnotation(t *testing.T) {
	f, err := Parse(lex("[exit-on-error]\nbuild:\n  false"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := f.Tasks["build"]
	if !task.ExitOnError {
		t.Fatal("task should have ExitOnError=true")
	}
}

func TestParseExitOnErrorAnnotationScoped(t *testing.T) {
	input := "[exit-on-error]\nbuild:\n  false\n\ntest:\n  true"
	f, err := Parse(lex(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Tasks["build"].ExitOnError {
		t.Fatal("build task should have ExitOnError=true")
	}
	if f.Tasks["test"].ExitOnError {
		t.Fatal("test task should NOT have ExitOnError=true")
	}
}

func TestParseErrorUnknownAnnotation(t *testing.T) {
	_, err := Parse(lex("[foobar]\nbuild:\n  echo hi"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if !strings.Contains(pe.Msg, "Unknown annotation") {
		t.Fatalf("expected 'Unknown annotation' in error, got '%s'", pe.Msg)
	}
}

func TestParseDuplicateTask(t *testing.T) {
	_, err := Parse(lex("build:\n  echo first\nbuild:\n  echo second"))
	if err == nil {
		t.Fatal("expected error for duplicate task, got nil")
	}
	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if !strings.Contains(pe.Msg, "Duplicate task") {
		t.Fatalf("expected 'Duplicate task' in error, got '%s'", pe.Msg)
	}
}
