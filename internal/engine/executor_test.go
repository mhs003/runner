package engine

import (
	"testing"
)

func TestInterpolateNoMatch(t *testing.T) {
	result := interpolate("hello world", map[string]string{}, nil)
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", result)
	}
}

func TestInterpolateSimple(t *testing.T) {
	result := interpolate("{{NAME}}", map[string]string{"NAME": "world"}, nil)
	if result != "world" {
		t.Fatalf("expected 'world', got '%s'", result)
	}
}

func TestInterpolateMultiple(t *testing.T) {
	result := interpolate("{{A}}-{{B}}", map[string]string{"A": "foo", "B": "bar"}, nil)
	if result != "foo-bar" {
		t.Fatalf("expected 'foo-bar', got '%s'", result)
	}
}

func TestInterpolateNoMatchStays(t *testing.T) {
	result := interpolate("{{MISSING}}", map[string]string{"NAME": "hello"}, nil)
	if result != "{{MISSING}}" {
		t.Fatalf("expected '{{MISSING}}', got '%s'", result)
	}
}

func TestInterpolateOverlappingNames(t *testing.T) {
	vars := map[string]string{
		"BIN":     "run",
		"BINPATH": "/usr/bin",
	}
	result := interpolate("path={{BINPATH}} bin={{BIN}}", vars, nil)
	expected := "path=/usr/bin bin=run"
	if result != expected {
		t.Fatalf("expected '%s', got '%s'", expected, result)
	}
}

func TestInterpolateFallbackPrimaryPresent(t *testing.T) {
	vars := map[string]string{
		"--entry": "./cmd/app",
		"ENTRY":   "./cmd/run",
	}
	result := interpolate("{{--entry||ENTRY}}", vars, nil)
	if result != "./cmd/app" {
		t.Fatalf("expected './cmd/app', got '%s'", result)
	}
}

func TestInterpolateFallbackPrimaryMissing(t *testing.T) {
	vars := map[string]string{
		"ENTRY": "./cmd/run",
	}
	result := interpolate("{{--entry||ENTRY}}", vars, nil)
	if result != "./cmd/run" {
		t.Fatalf("expected './cmd/run', got '%s'", result)
	}
}

func TestInterpolateFallbackLiteral(t *testing.T) {
	result := interpolate("{{--entry||./default/path}}", map[string]string{}, nil)
	if result != "./default/path" {
		t.Fatalf("expected './default/path', got '%s'", result)
	}
}

func TestInterpolateFallbackPrimaryEmpty(t *testing.T) {
	vars := map[string]string{
		"--entry": "",
		"ENTRY":   "./cmd/run",
	}
	result := interpolate("{{--entry||ENTRY}}", vars, nil)
	if result != "./cmd/run" {
		t.Fatalf("expected './cmd/run' (primary is empty string), got '%s'", result)
	}
}

func TestInterpolateFallbackNoPipe(t *testing.T) {
	vars := map[string]string{
		"VAR": "value",
	}
	result := interpolate("{{VAR}}", vars, nil)
	if result != "value" {
		t.Fatalf("expected 'value', got '%s'", result)
	}
}

func TestInterpolateMultipleFallbacks(t *testing.T) {
	vars := map[string]string{
		"A": "a_val",
		"B": "b_val",
	}
	result := interpolate("{{X||A}}-{{Y||B}}", vars, nil)
	if result != "a_val-b_val" {
		t.Fatalf("expected 'a_val-b_val', got '%s'", result)
	}
}

func TestInterpolateMixed(t *testing.T) {
	vars := map[string]string{
		"BIN":   "run",
		"ENTRY": "./cmd/run",
	}
	result := interpolate("go build -o {{BIN}} {{--entry||ENTRY}}", vars, nil)
	if result != "go build -o run ./cmd/run" {
		t.Fatalf("expected 'go build -o run ./cmd/run', got '%s'", result)
	}
}

func TestInterpolateAtAll(t *testing.T) {
	result := interpolate("echo {{@}}", map[string]string{}, []string{"Hello", "World"})
	if result != "echo Hello World" {
		t.Fatalf("expected 'echo Hello World', got '%s'", result)
	}
}

func TestInterpolateAtAllEmpty(t *testing.T) {
	result := interpolate("echo {{@}}", map[string]string{}, nil)
	if result != "echo {{@}}" {
		t.Fatalf("expected 'echo {{@}}', got '%s'", result)
	}
}

func TestInterpolateAtAllFallback(t *testing.T) {
	result := interpolate("echo {{@||no args}}", map[string]string{}, nil)
	if result != "echo no args" {
		t.Fatalf("expected 'echo no args', got '%s'", result)
	}
}

func TestInterpolateAtFirstN(t *testing.T) {
	result := interpolate("{{@3}}", map[string]string{}, []string{"a", "b", "c", "d", "e"})
	if result != "a b c" {
		t.Fatalf("expected 'a b c', got '%s'", result)
	}
}

func TestInterpolateAtFirstNMoreThanLen(t *testing.T) {
	result := interpolate("{{@10}}", map[string]string{}, []string{"a", "b"})
	if result != "a b" {
		t.Fatalf("expected 'a b', got '%s'", result)
	}
}

func TestInterpolateAtFirstNFallback(t *testing.T) {
	result := interpolate("{{@3||fallback}}", map[string]string{}, nil)
	if result != "fallback" {
		t.Fatalf("expected 'fallback', got '%s'", result)
	}
}

func TestInterpolateAtLastN(t *testing.T) {
	result := interpolate("{{2@}}", map[string]string{}, []string{"a", "b", "c", "d", "e"})
	if result != "d e" {
		t.Fatalf("expected 'd e', got '%s'", result)
	}
}

func TestInterpolateAtLastNMoreThanLen(t *testing.T) {
	result := interpolate("{{5@}}", map[string]string{}, []string{"a", "b"})
	if result != "a b" {
		t.Fatalf("expected 'a b', got '%s'", result)
	}
}

func TestInterpolateAtLastNFallback(t *testing.T) {
	result := interpolate("{{2@||none}}", map[string]string{}, nil)
	if result != "none" {
		t.Fatalf("expected 'none', got '%s'", result)
	}
}

func TestInterpolateAtRange(t *testing.T) {
	result := interpolate("{{2@4}}", map[string]string{}, []string{"a", "b", "c", "d", "e"})
	if result != "b c d" {
		t.Fatalf("expected 'b c d', got '%s'", result)
	}
}

func TestInterpolateAtRangeClamped(t *testing.T) {
	result := interpolate("{{2@10}}", map[string]string{}, []string{"a", "b", "c"})
	if result != "b c" {
		t.Fatalf("expected 'b c', got '%s'", result)
	}
}

func TestInterpolateAtRangeFullSlice(t *testing.T) {
	result := interpolate("{{1@3}}", map[string]string{}, []string{"a", "b", "c"})
	if result != "a b c" {
		t.Fatalf("expected 'a b c', got '%s'", result)
	}
}

func TestInterpolateAtRangeMgtN(t *testing.T) {
	result := interpolate("{{4@2}}", map[string]string{}, []string{"a", "b", "c", "d"})
	if result != "{{4@2}}" {
		t.Fatalf("expected '{{4@2}}', got '%s'", result)
	}
}

func TestInterpolateAtRangeFallback(t *testing.T) {
	result := interpolate("{{4@2||invalid range}}", map[string]string{}, []string{"a", "b", "c", "d"})
	if result != "invalid range" {
		t.Fatalf("expected 'invalid range', got '%s'", result)
	}
}

func TestInterpolateAtSingleArg(t *testing.T) {
	result := interpolate("echo {{@}}", map[string]string{}, []string{"Hello"})
	if result != "echo Hello" {
		t.Fatalf("expected 'echo Hello', got '%s'", result)
	}
}

func TestInterpolateAtWithNamedArgsStripped(t *testing.T) {
	vars := map[string]string{
		"--verbose": "true",
		"-p":        "conf",
	}
	positional := []string{"Hello", "World"}
	result := interpolate("a_command {{-p}} print {{@}} --verbose={{--verbose}}", vars, positional)
	if result != "a_command conf print Hello World --verbose=true" {
		t.Fatalf("expected 'a_command conf print Hello World --verbose=true', got '%s'", result)
	}
}

func TestInterpolateAtZero(t *testing.T) {
	result := interpolate("{{@0}}", map[string]string{}, []string{"a", "b"})
	if result != "{{@0}}" {
		t.Fatalf("expected '{{@0}}', got '%s'", result)
	}
}

func TestInterpolateAtNegative(t *testing.T) {
	result := interpolate("{{@-1}}", map[string]string{}, []string{"a", "b"})
	if result != "{{@-1}}" {
		t.Fatalf("expected '{{@-1}}', got '%s'", result)
	}
}

func TestInterpolateAtRangeFallbackLiteral(t *testing.T) {
	result := interpolate("{{5@2||default text}}", map[string]string{}, nil)
	if result != "default text" {
		t.Fatalf("expected 'default text', got '%s'", result)
	}
}
