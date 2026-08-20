package engine

import (
	"testing"
)

func interp(s string, vars map[string]string, positional []string, shellCache map[string]string) string {
	result, err := interpolate(s, vars, positional, shellCache)
	if err != nil {
		return ""
	}
	return result
}

func interpWithAll(s string, vars map[string]string, positional, allArgs []string) string {
	result, err := interpolate(s, vars, positional, nil, allArgs)
	if err != nil {
		return ""
	}
	return result
}

func TestInterpolateNoMatch(t *testing.T) {
	result := interp("hello world", map[string]string{}, nil, nil)
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", result)
	}
}

func TestInterpolateSimple(t *testing.T) {
	result := interp("{{NAME}}", map[string]string{"NAME": "world"}, nil, nil)
	if result != "world" {
		t.Fatalf("expected 'world', got '%s'", result)
	}
}

func TestInterpolateMultiple(t *testing.T) {
	result := interp("{{A}}-{{B}}", map[string]string{"A": "foo", "B": "bar"}, nil, nil)
	if result != "foo-bar" {
		t.Fatalf("expected 'foo-bar', got '%s'", result)
	}
}

func TestInterpolateNoMatchStays(t *testing.T) {
	result := interp("{{MISSING}}", map[string]string{"NAME": "hello"}, nil, nil)
	if result != "" {
		t.Fatalf("expected empty result, got '%s'", result)
	}
}

func TestInterpolateMissingNamedArgumentIsEmpty(t *testing.T) {
	result := interp("echo {{--path}}", map[string]string{}, nil, nil)
	if result != "echo " {
		t.Fatalf("expected missing named argument to be empty, got '%s'", result)
	}
}

func TestInterpolateMissingFlagIsEmpty(t *testing.T) {
	result := interp("echo {{-v}}", map[string]string{}, nil, nil)
	if result != "echo " {
		t.Fatalf("expected missing flag to be empty, got '%s'", result)
	}
}

func TestInterpolateNestedMissingTokenIsEmpty(t *testing.T) {
	result := interp("echo {{MESSAGE}}", map[string]string{"MESSAGE": "hello {{NAME}}"}, nil, nil)
	if result != "echo hello " {
		t.Fatalf("expected nested missing token to be empty, got '%s'", result)
	}
}

func TestInterpolateOverlappingNames(t *testing.T) {
	vars := map[string]string{
		"BIN":     "run",
		"BINPATH": "/usr/bin",
	}
	result := interp("path={{BINPATH}} bin={{BIN}}", vars, nil, nil)
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
	result := interp("{{--entry||ENTRY}}", vars, nil, nil)
	if result != "./cmd/app" {
		t.Fatalf("expected './cmd/app', got '%s'", result)
	}
}

func TestInterpolateFallbackPrimaryMissing(t *testing.T) {
	vars := map[string]string{
		"ENTRY": "./cmd/run",
	}
	result := interp("{{--entry||ENTRY}}", vars, nil, nil)
	if result != "./cmd/run" {
		t.Fatalf("expected './cmd/run', got '%s'", result)
	}
}

func TestInterpolateFallbackLiteral(t *testing.T) {
	result := interp("{{--entry||./default/path}}", map[string]string{}, nil, nil)
	if result != "./default/path" {
		t.Fatalf("expected './default/path', got '%s'", result)
	}
}

func TestInterpolateFallbackPrimaryEmpty(t *testing.T) {
	vars := map[string]string{
		"--entry": "",
		"ENTRY":   "./cmd/run",
	}
	result := interp("{{--entry||ENTRY}}", vars, nil, nil)
	if result != "./cmd/run" {
		t.Fatalf("expected './cmd/run' (primary is empty string), got '%s'", result)
	}
}

func TestInterpolateFallbackNoPipe(t *testing.T) {
	vars := map[string]string{
		"VAR": "value",
	}
	result := interp("{{VAR}}", vars, nil, nil)
	if result != "value" {
		t.Fatalf("expected 'value', got '%s'", result)
	}
}

func TestInterpolateMultipleFallbacks(t *testing.T) {
	vars := map[string]string{
		"A": "a_val",
		"B": "b_val",
	}
	result := interp("{{X||A}}-{{Y||B}}", vars, nil, nil)
	if result != "a_val-b_val" {
		t.Fatalf("expected 'a_val-b_val', got '%s'", result)
	}
}

func TestInterpolateMixed(t *testing.T) {
	vars := map[string]string{
		"BIN":   "run",
		"ENTRY": "./cmd/run",
	}
	result := interp("go build -o {{BIN}} {{--entry||ENTRY}}", vars, nil, nil)
	if result != "go build -o run ./cmd/run" {
		t.Fatalf("expected 'go build -o run ./cmd/run', got '%s'", result)
	}
}

func TestInterpolateAtAll(t *testing.T) {
	result := interp("echo {{@}}", map[string]string{}, []string{"Hello", "World"}, nil)
	if result != "echo Hello World" {
		t.Fatalf("expected 'echo Hello World', got '%s'", result)
	}
}

func TestInterpolateAtAllEmpty(t *testing.T) {
	result := interp("echo {{@}}", map[string]string{}, nil, nil)
	if result != "echo " {
		t.Fatalf("expected 'echo ', got '%s'", result)
	}
}

func TestInterpolateAtAllIncludesNamedArgsAndFlags(t *testing.T) {
	args := []string{"artisan", "migrate", "--path=path/to/migration-file.php", "-v"}
	result := interpWithAll("{{@}}", map[string]string{}, []string{"artisan", "migrate"}, args)
	if result != "artisan migrate --path=path/to/migration-file.php -v" {
		t.Fatalf("expected all original args, got '%s'", result)
	}
}

func TestInterpolateAtAllFallback(t *testing.T) {
	result := interp("echo {{@||no args}}", map[string]string{}, nil, nil)
	if result != "echo no args" {
		t.Fatalf("expected 'echo no args', got '%s'", result)
	}
}

func TestInterpolateAtFirstN(t *testing.T) {
	result := interp("{{@3}}", map[string]string{}, []string{"a", "b", "c", "d", "e"}, nil)
	if result != "a b c" {
		t.Fatalf("expected 'a b c', got '%s'", result)
	}
}

func TestInterpolateAtFirstNMoreThanLen(t *testing.T) {
	result := interp("{{@10}}", map[string]string{}, []string{"a", "b"}, nil)
	if result != "a b" {
		t.Fatalf("expected 'a b', got '%s'", result)
	}
}

func TestInterpolateAtFirstNFallback(t *testing.T) {
	result := interp("{{@3||fallback}}", map[string]string{}, nil, nil)
	if result != "fallback" {
		t.Fatalf("expected 'fallback', got '%s'", result)
	}
}

func TestInterpolateAtLastN(t *testing.T) {
	result := interp("{{2@}}", map[string]string{}, []string{"a", "b", "c", "d", "e"}, nil)
	if result != "d e" {
		t.Fatalf("expected 'd e', got '%s'", result)
	}
}

func TestInterpolateAtLastNMoreThanLen(t *testing.T) {
	result := interp("{{5@}}", map[string]string{}, []string{"a", "b"}, nil)
	if result != "a b" {
		t.Fatalf("expected 'a b', got '%s'", result)
	}
}

func TestInterpolateAtLastNFallback(t *testing.T) {
	result := interp("{{2@||none}}", map[string]string{}, nil, nil)
	if result != "none" {
		t.Fatalf("expected 'none', got '%s'", result)
	}
}

func TestInterpolateAtRange(t *testing.T) {
	result := interp("{{2@4}}", map[string]string{}, []string{"a", "b", "c", "d", "e"}, nil)
	if result != "b c d" {
		t.Fatalf("expected 'b c d', got '%s'", result)
	}
}

func TestInterpolateAtRangeClamped(t *testing.T) {
	result := interp("{{2@10}}", map[string]string{}, []string{"a", "b", "c"}, nil)
	if result != "b c" {
		t.Fatalf("expected 'b c', got '%s'", result)
	}
}

func TestInterpolateAtRangeFullSlice(t *testing.T) {
	result := interp("{{1@3}}", map[string]string{}, []string{"a", "b", "c"}, nil)
	if result != "a b c" {
		t.Fatalf("expected 'a b c', got '%s'", result)
	}
}

func TestInterpolateAtRangeMgtN(t *testing.T) {
	result := interp("{{4@2}}", map[string]string{}, []string{"a", "b", "c", "d"}, nil)
	if result != "" {
		t.Fatalf("expected empty result, got '%s'", result)
	}
}

func TestInterpolateAtRangeFallback(t *testing.T) {
	result := interp("{{4@2||invalid range}}", map[string]string{}, []string{"a", "b", "c", "d"}, nil)
	if result != "invalid range" {
		t.Fatalf("expected 'invalid range', got '%s'", result)
	}
}

func TestInterpolateAtSingleArg(t *testing.T) {
	result := interp("echo {{@}}", map[string]string{}, []string{"Hello"}, nil)
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
	result := interp("a_command {{-p}} print {{@}} --verbose={{--verbose}}", vars, positional, nil)
	if result != "a_command conf print Hello World --verbose=true" {
		t.Fatalf("expected 'a_command conf print Hello World --verbose=true', got '%s'", result)
	}
}

func TestInterpolateAtZero(t *testing.T) {
	result := interp("{{@0}}", map[string]string{}, []string{"a", "b"}, nil)
	if result != "" {
		t.Fatalf("expected empty result, got '%s'", result)
	}
}

func TestInterpolateAtNegative(t *testing.T) {
	result := interp("{{@-1}}", map[string]string{}, []string{"a", "b"}, nil)
	if result != "" {
		t.Fatalf("expected empty result, got '%s'", result)
	}
}

func TestInterpolateAtRangeFallbackLiteral(t *testing.T) {
	result := interp("{{5@2||default text}}", map[string]string{}, nil, nil)
	if result != "default text" {
		t.Fatalf("expected 'default text', got '%s'", result)
	}
}

func TestResolveLazyShellBasic(t *testing.T) {
	vars := map[string]string{
		"IP": "$(echo 1.2.3.4)",
	}
	v, ok, err := resolveLazyValue("IP", vars, nil, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected key to be resolved")
	}
	if v != "1.2.3.4" {
		t.Fatalf("expected '1.2.3.4', got '%s'", v)
	}
}

func TestResolveLazyShellCached(t *testing.T) {
	vars := map[string]string{
		"IP": "$(echo 1.2.3.4)",
	}
	shellCache := map[string]string{}

	v, ok, err := resolveLazyValue("IP", vars, nil, shellCache, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected key to be resolved")
	}
	if v != "1.2.3.4" {
		t.Fatalf("expected '1.2.3.4', got '%s'", v)
	}
	if shellCache["IP"] != "1.2.3.4" {
		t.Fatal("expected shellCache to have cached value")
	}

	v2, ok2, err2 := resolveLazyValue("IP", vars, nil, shellCache, 0)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if !ok2 {
		t.Fatal("expected cached key to be resolved")
	}
	if v2 != "1.2.3.4" {
		t.Fatalf("expected cached '1.2.3.4', got '%s'", v2)
	}
}

func TestResolveLazyShellWithVars(t *testing.T) {
	vars := map[string]string{
		"TOOL":  "go",
		"WHERE": "$(echo $TOOL)",
	}
	v, ok, err := resolveLazyValue("WHERE", vars, nil, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected key to be resolved")
	}
	if v != "go" {
		t.Fatalf("expected 'go', got '%s'", v)
	}
}

func TestResolveLazyChain(t *testing.T) {
	vars := map[string]string{
		"IP":   "$(echo 1.2.3.4)",
		"REF":  "{{IP}} echo",
		"DESC": "{{REF}} done",
	}
	v, ok, err := resolveLazyValue("DESC", vars, nil, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected key to be resolved")
	}
	if v != "1.2.3.4 echo done" {
		t.Fatalf("expected '1.2.3.4 echo done', got '%s'", v)
	}
}

func TestResolveLazyPlainValue(t *testing.T) {
	vars := map[string]string{
		"NAME": "world",
	}
	v, ok, err := resolveLazyValue("NAME", vars, nil, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected key to be resolved")
	}
	if v != "world" {
		t.Fatalf("expected 'world', got '%s'", v)
	}
}

func TestResolveLazyMissingKey(t *testing.T) {
	_, ok, err := resolveLazyValue("MISSING", map[string]string{}, nil, nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected missing key to not resolve")
	}
}

func TestResolveLazyShellFailed(t *testing.T) {
	vars := map[string]string{
		"BAD": "$(sh -c 'exit 1')",
	}
	_, _, err := resolveLazyValue("BAD", vars, nil, nil, 0)
	if err == nil {
		t.Fatal("expected error for failed shell command, got nil")
	}
}

func TestInterpolateShellFailed(t *testing.T) {
	vars := map[string]string{
		"BAD": "$(sh -c 'exit 1')",
	}
	_, err := interpolate("{{BAD}}", vars, nil, nil)
	if err == nil {
		t.Fatal("expected error from interpolate for failed shell, got nil")
	}
}

func TestInterpolateShellLazyChain(t *testing.T) {
	vars := map[string]string{
		"IP":   "$(echo 1.2.3.4)",
		"DESC": "IP is {{IP}}",
	}
	result := interp("echo {{DESC}}", vars, nil, nil)
	if result != "echo IP is 1.2.3.4" {
		t.Fatalf("expected 'echo IP is 1.2.3.4', got '%s'", result)
	}
}

func TestInterpolateShellCachedAcrossCalls(t *testing.T) {
	vars := map[string]string{
		"IP": "$(echo once)",
	}
	shellCache := map[string]string{}

	r1, err1 := interpolate("{{IP}}", vars, nil, shellCache)
	if err1 != nil {
		t.Fatalf("unexpected error: %v", err1)
	}
	if r1 != "once" {
		t.Fatalf("expected 'once', got '%s'", r1)
	}

	r2, err2 := interpolate("{{IP}}", vars, nil, shellCache)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if r2 != "once" {
		t.Fatalf("expected cached 'once', got '%s'", r2)
	}
	if shellCache["IP"] != "once" {
		t.Fatal("expected shellCache to have cached value")
	}
}
