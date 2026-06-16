package engine

import (
	"testing"
)

func TestInterpolateNoMatch(t *testing.T) {
	result := interpolate("hello world", map[string]string{})
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", result)
	}
}

func TestInterpolateSimple(t *testing.T) {
	result := interpolate("{{NAME}}", map[string]string{"NAME": "world"})
	if result != "world" {
		t.Fatalf("expected 'world', got '%s'", result)
	}
}

func TestInterpolateMultiple(t *testing.T) {
	result := interpolate("{{A}}-{{B}}", map[string]string{"A": "foo", "B": "bar"})
	if result != "foo-bar" {
		t.Fatalf("expected 'foo-bar', got '%s'", result)
	}
}

func TestInterpolateNoMatchStays(t *testing.T) {
	result := interpolate("{{MISSING}}", map[string]string{"NAME": "hello"})
	if result != "{{MISSING}}" {
		t.Fatalf("expected '{{MISSING}}', got '%s'", result)
	}
}

func TestInterpolateOverlappingNames(t *testing.T) {
	vars := map[string]string{
		"BIN":     "run",
		"BINPATH": "/usr/bin",
	}
	result := interpolate("path={{BINPATH}} bin={{BIN}}", vars)
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
	result := interpolate("{{--entry||ENTRY}}", vars)
	if result != "./cmd/app" {
		t.Fatalf("expected './cmd/app', got '%s'", result)
	}
}

func TestInterpolateFallbackPrimaryMissing(t *testing.T) {
	vars := map[string]string{
		"ENTRY": "./cmd/run",
	}
	result := interpolate("{{--entry||ENTRY}}", vars)
	if result != "./cmd/run" {
		t.Fatalf("expected './cmd/run', got '%s'", result)
	}
}

func TestInterpolateFallbackLiteral(t *testing.T) {
	result := interpolate("{{--entry||./default/path}}", map[string]string{})
	if result != "./default/path" {
		t.Fatalf("expected './default/path', got '%s'", result)
	}
}

func TestInterpolateFallbackPrimaryEmpty(t *testing.T) {
	vars := map[string]string{
		"--entry": "",
		"ENTRY":   "./cmd/run",
	}
	result := interpolate("{{--entry||ENTRY}}", vars)
	if result != "./cmd/run" {
		t.Fatalf("expected './cmd/run' (primary is empty string), got '%s'", result)
	}
}

func TestInterpolateFallbackNoPipe(t *testing.T) {
	vars := map[string]string{
		"VAR": "value",
	}
	result := interpolate("{{VAR}}", vars)
	if result != "value" {
		t.Fatalf("expected 'value', got '%s'", result)
	}
}

func TestInterpolateMultipleFallbacks(t *testing.T) {
	vars := map[string]string{
		"A": "a_val",
		"B": "b_val",
	}
	result := interpolate("{{X||A}}-{{Y||B}}", vars)
	if result != "a_val-b_val" {
		t.Fatalf("expected 'a_val-b_val', got '%s'", result)
	}
}

func TestInterpolateMixed(t *testing.T) {
	vars := map[string]string{
		"BIN":   "run",
		"ENTRY": "./cmd/run",
	}
	result := interpolate("go build -o {{BIN}} {{--entry||ENTRY}}", vars)
	if result != "go build -o run ./cmd/run" {
		t.Fatalf("expected 'go build -o run ./cmd/run', got '%s'", result)
	}
}
