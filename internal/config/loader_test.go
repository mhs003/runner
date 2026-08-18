package config

import "testing"

func TestMergeLocalOverridesGlobal(t *testing.T) {
	globalTask := &Task{Name: "push", BodyLines: []BodyLine{{Type: "cmd", Text: "global push"}}}
	localTask := &Task{Name: "push", BodyLines: []BodyLine{{Type: "cmd", Text: "local push"}}}
	global := &File{
		Vars: map[string]string{
			"SHARED":      "global",
			"GLOBAL_ONLY": "available",
		},
		Tasks: map[string]*Task{
			"push":        globalTask,
			"global-only": {Name: "global-only"},
		},
	}
	local := &File{
		Vars: map[string]string{
			"SHARED":     "local",
			"LOCAL_ONLY": "available",
		},
		Tasks: map[string]*Task{
			"push":       localTask,
			"local-only": {Name: "local-only"},
		},
	}

	merged := Merge(global, local)

	if merged.Vars["SHARED"] != "local" {
		t.Fatalf("expected local var to win, got %q", merged.Vars["SHARED"])
	}
	if merged.Vars["GLOBAL_ONLY"] != "available" || merged.Vars["LOCAL_ONLY"] != "available" {
		t.Fatal("expected vars unique to both files to be retained")
	}
	if merged.Tasks["push"] != localTask {
		t.Fatal("expected local task to win")
	}
	if merged.Tasks["global-only"] == nil || merged.Tasks["local-only"] == nil {
		t.Fatal("expected tasks unique to both files to be retained")
	}
}

func TestMergeVarsResolveAcrossFiles(t *testing.T) {
	global := &File{
		Vars:  map[string]string{"GLOBAL": "global {{LOCAL}}"},
		Tasks: map[string]*Task{},
	}
	local := &File{
		Vars:  map[string]string{"LOCAL": "local", "LOCAL_REF": "{{GLOBAL}} value"},
		Tasks: map[string]*Task{},
	}

	merged := Merge(global, local)
	if err := ResolveVars(merged.Vars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged.Vars["GLOBAL"] != "global local" {
		t.Fatalf("expected global var to reference local var, got %q", merged.Vars["GLOBAL"])
	}
	if merged.Vars["LOCAL_REF"] != "global local value" {
		t.Fatalf("expected local var to reference global var, got %q", merged.Vars["LOCAL_REF"])
	}
}

func TestMergeDoesNotMutateInputs(t *testing.T) {
	global := &File{Vars: map[string]string{"VALUE": "global"}, Tasks: map[string]*Task{}}
	local := &File{Vars: map[string]string{"VALUE": "local"}, Tasks: map[string]*Task{}}

	merged := Merge(global, local)
	merged.Vars["VALUE"] = "changed"

	if global.Vars["VALUE"] != "global" || local.Vars["VALUE"] != "local" {
		t.Fatal("merge mutated an input vars map")
	}
}
