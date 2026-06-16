package config

import (
	"reflect"
	"testing"
)

func TestLexEmptyInput(t *testing.T) {
	lines := Lex("")
	// strings.Split("", "\n") returns [""], so we get 1 empty line
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "" {
		t.Fatalf("expected empty text, got '%s'", lines[0].Text)
	}
}

func TestLexSingleLine(t *testing.T) {
	lines := Lex("hello")
	expected := []Line{{No: 1, Indent: 0, Text: "hello"}}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("expected %+v, got %+v", expected, lines)
	}
}

func TestLexIndent(t *testing.T) {
	lines := Lex("  hello")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Indent != 2 {
		t.Fatalf("expected indent 2, got %d", lines[0].Indent)
	}
	if lines[0].Text != "hello" {
		t.Fatalf("expected text 'hello', got '%s'", lines[0].Text)
	}
}

func TestLexTrailingWhitespace(t *testing.T) {
	lines := Lex("hello  ")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "hello" {
		t.Fatalf("expected text 'hello', got '%s'", lines[0].Text)
	}
}

func TestLexCommentLine(t *testing.T) {
	lines := Lex("# this is a comment")
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestLexIndentedCommentLine(t *testing.T) {
	lines := Lex("  # indented comment")
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestLexInlineCommentAtIndentZero(t *testing.T) {
	lines := Lex("task: # this is a task")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "task:" {
		t.Fatalf("expected text 'task:', got '%s'", lines[0].Text)
	}
}

func TestLexInlineCommentWithLeadingSpaces(t *testing.T) {
	lines := Lex("task:  # indented comment")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "task:" {
		t.Fatalf("expected text 'task:', got '%s'", lines[0].Text)
	}
}

func TestLexInlineCommentPreservedInIndented(t *testing.T) {
	lines := Lex("  echo hello # keep this")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "echo hello # keep this" {
		t.Fatalf("expected text 'echo hello # keep this', got '%s'", lines[0].Text)
	}
}

func TestLexEmptyLines(t *testing.T) {
	lines := Lex("hello\n\n\nworld")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if lines[1].Text != "" {
		t.Fatalf("expected empty text for line 2, got '%s'", lines[1].Text)
	}
	if lines[2].Text != "" {
		t.Fatalf("expected empty text for line 3, got '%s'", lines[2].Text)
	}
}

func TestLexLineNumbers(t *testing.T) {
	lines := Lex("first\n  second\n# comment\nthird")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].No != 1 {
		t.Fatalf("expected line 1 to be No 1, got %d", lines[0].No)
	}
	if lines[1].No != 2 {
		t.Fatalf("expected line 2 to be No 2, got %d", lines[1].No)
	}
	if lines[2].No != 4 {
		t.Fatalf("expected line 4 to be No 4 (comment skipped), got %d", lines[2].No)
	}
}

func TestLexMultipleLines(t *testing.T) {
	input := "@vars:\n  KEY = val\n\nbuild:\n  echo hello"
	lines := Lex(input)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %+v", len(lines), lines)
	}
}
