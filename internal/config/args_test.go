package config

import (
	"reflect"
	"testing"
)

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

func TestParseArgsNamedDashDash(t *testing.T) {
	ra := ParseArgs([]string{"--entry", "./cmd/app"})
	if ra.Named["--entry"] != "./cmd/app" {
		t.Fatalf("expected --entry=./cmd/app, got --entry=%s", ra.Named["--entry"])
	}
}

func TestParseArgsNamedShort(t *testing.T) {
	ra := ParseArgs([]string{"-e", "./cmd/app"})
	if ra.Named["-e"] != "./cmd/app" {
		t.Fatalf("expected -e=./cmd/app, got -e=%s", ra.Named["-e"])
	}
}

func TestParseArgsFlagDashDash(t *testing.T) {
	ra := ParseArgs([]string{"--verbose"})
	if !ra.Flags["--verbose"] {
		t.Fatal("expected --verbose flag to be true")
	}
}

func TestParseArgsFlagShort(t *testing.T) {
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

func TestParseArgsKeyEqualsValue(t *testing.T) {
	ra := ParseArgs([]string{"--entry=./cmd/app"})
	if ra.Named["--entry"] != "./cmd/app" {
		t.Fatalf("expected --entry=./cmd/app, got --entry=%s", ra.Named["--entry"])
	}
}

func TestParseArgsKeyEqualsValueWithMixed(t *testing.T) {
	ra := ParseArgs([]string{"--entry=./cmd/app", "build", "--dry"})
	if ra.Named["--entry"] != "./cmd/app" {
		t.Fatalf("expected --entry=./cmd/app, got --entry=%s", ra.Named["--entry"])
	}
	if !ra.Flags["--dry"] {
		t.Fatal("expected --dry flag")
	}
	expected := []string{"build"}
	if !reflect.DeepEqual(ra.Positional, expected) {
		t.Fatalf("expected positional %v, got %v", expected, ra.Positional)
	}
}

func TestParseArgsCombinedShortFlags(t *testing.T) {
	ra := ParseArgs([]string{"-abc"})
	if !ra.Flags["-a"] || !ra.Flags["-b"] || !ra.Flags["-c"] {
		t.Fatal("expected -a, -b, -c flags to all be true")
	}
}

func TestParseArgsCombinedFlagsWithPositional(t *testing.T) {
	ra := ParseArgs([]string{"-abc", "build"})
	if !ra.Flags["-a"] || !ra.Flags["-b"] || !ra.Flags["-c"] {
		t.Fatal("expected -a, -b, -c flags")
	}
	expected := []string{"build"}
	if !reflect.DeepEqual(ra.Positional, expected) {
		t.Fatalf("expected positional %v, got %v", expected, ra.Positional)
	}
}

func TestParseArgsEqualsInValue(t *testing.T) {
	ra := ParseArgs([]string{"--var=a=b=c"})
	if ra.Named["--var"] != "a=b=c" {
		t.Fatalf("expected --var=a=b=c, got --var=%s", ra.Named["--var"])
	}
}
