package config

import (
	"strings"
	"testing"
)

func TestResolveVarRefsSimple(t *testing.T) {
	vars := map[string]string{
		"NAME":     "world",
		"GREETING": "hello {{NAME}}",
	}
	if err := resolveVarRefs(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["GREETING"] != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", vars["GREETING"])
	}
}

func TestResolveVarRefsForwardRefs(t *testing.T) {
	vars := map[string]string{
		"A": "hello {{B}}",
		"B": "world {{C}}",
		"C": "!",
	}
	if err := resolveVarRefs(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["A"] != "hello world !" {
		t.Fatalf("expected 'hello world !', got '%s'", vars["A"])
	}
}

func TestResolveVarRefsMultiLevel(t *testing.T) {
	vars := map[string]string{
		"V1": "a",
		"V2": "{{V1}}b",
		"V3": "{{V2}}c",
	}
	if err := resolveVarRefs(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["V3"] != "abc" {
		t.Fatalf("expected 'abc', got '%s'", vars["V3"])
	}
}

func TestResolveVarRefsCircular(t *testing.T) {
	vars := map[string]string{
		"A": "{{B}}",
		"B": "{{A}}",
	}
	err := resolveVarRefs(vars)
	if err == nil {
		t.Fatal("expected error for circular reference, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Fatalf("expected 'circular' in error, got '%s'", err.Error())
	}
}

func TestResolveVarRefsSelfRef(t *testing.T) {
	vars := map[string]string{
		"A": "{{A}}",
	}
	err := resolveVarRefs(vars)
	if err == nil {
		t.Fatal("expected error for self-reference, got nil")
	}
}

func TestResolveVarRefsMissing(t *testing.T) {
	vars := map[string]string{
		"A": "hello {{MISSING}}",
	}
	if err := resolveVarRefs(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["A"] != "hello {{MISSING}}" {
		t.Fatalf("expected 'hello {{MISSING}}' (unchanged), got '%s'", vars["A"])
	}
}

func TestResolveVarRefsNoVars(t *testing.T) {
	vars := map[string]string{
		"A": "plain text",
	}
	if err := resolveVarRefs(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["A"] != "plain text" {
		t.Fatalf("expected 'plain text', got '%s'", vars["A"])
	}
}

func TestResolveVarRefsSkipsShellVar(t *testing.T) {
	vars := map[string]string{
		"IP":   "$(hostname -I)",
		"REF":  "{{IP}} echo",
		"DESC": "{{REF}} done",
	}
	if err := resolveVarRefs(vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["REF"] != "{{IP}} echo" {
		t.Fatalf("expected REF to keep '{{IP}} echo' (shell var should not be substituted), got '%s'", vars["REF"])
	}
	if vars["DESC"] != "{{IP}} echo done" {
		t.Fatalf("expected DESC to be '{{IP}} echo done' (REF expanded, IP kept), got '%s'", vars["DESC"])
	}
}
