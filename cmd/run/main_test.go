package main

import (
	"errors"
	"mhs003/runner/internal/config"
	"mhs003/runner/internal/engine"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMergesGlobalAndLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workingDir := t.TempDir()
	localPath := filepath.Join(workingDir, ".runner")
	writeConfig(t, filepath.Join(home, ".runner.global"), "@vars:\n  GLOBAL = global\n  SHARED = global\n\npush:\n  echo global\n")
	writeConfig(t, localPath, "@vars:\n  LOCAL = local\n  SHARED = local\n\nbuild: push:\n  echo {{GLOBAL}}\n")

	file, err := loadConfig(localPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.Tasks["push"] == nil || file.Tasks["build"] == nil {
		t.Fatal("expected tasks from global and local files")
	}
	if file.Vars["SHARED"] != "local" {
		t.Fatalf("expected local var to win, got %q", file.Vars["SHARED"])
	}
	seen := map[string]bool{}
	stack := map[string]bool{}
	var order []*config.Task
	if err := engine.Resolve(file, "build", seen, stack, &order); err != nil {
		t.Fatalf("expected merged task graph to resolve: %v", err)
	}
	if len(order) != 2 || order[0].Name != "push" || order[1].Name != "build" {
		t.Fatalf("expected global dependency before local task, got %+v", order)
	}
}

func TestLoadConfigUsesGlobalWithoutLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, filepath.Join(home, ".runner.global"), "push:\n  echo global\n")

	file, err := loadConfig(filepath.Join(t.TempDir(), ".runner"), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.Tasks["push"] == nil {
		t.Fatal("expected global task")
	}
}

func TestLoadConfigExplicitFileDoesNotMergeGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, filepath.Join(home, ".runner.global"), "push:\n  echo global\n")
	explicitPath := filepath.Join(t.TempDir(), "tasks.runner")
	writeConfig(t, explicitPath, "build:\n  echo local\n")

	file, err := loadConfig(explicitPath, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if file.Tasks["push"] != nil {
		t.Fatal("explicit file unexpectedly included global task")
	}
	if file.Tasks["build"] == nil {
		t.Fatal("expected task from explicit file")
	}
}

func TestLoadConfigNoFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := loadConfig(filepath.Join(t.TempDir(), ".runner"), false)
	if !errors.Is(err, errNoConfig) {
		t.Fatalf("expected errNoConfig, got %v", err)
	}
}

func TestLoadConfigRejectsInvalidGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, filepath.Join(home, ".runner.global"), "invalid syntax\n")
	localPath := filepath.Join(t.TempDir(), ".runner")
	writeConfig(t, localPath, "build:\n  echo local\n")

	if _, err := loadConfig(localPath, false); err == nil {
		t.Fatal("expected invalid global config to fail")
	}
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
