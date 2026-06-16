package engine

import (
	"mhs003/runner/internal/config"
	"strings"
	"testing"
)

func makeTasks(tasks ...config.Task) *config.File {
	f := &config.File{
		Vars:  map[string]string{},
		Tasks: map[string]*config.Task{},
	}
	for i := range tasks {
		t := &tasks[i]
		f.Tasks[t.Name] = t
	}
	return f
}

func task(name string, depNames ...string) config.Task {
	deps := make([]config.Dep, len(depNames))
	for i, dn := range depNames {
		deps[i] = config.Dep{Name: dn}
	}
	return config.Task{Name: name, HeaderDeps: deps}
}

func taskWithCmd(name string, cmds ...string) config.Task {
	bls := make([]config.BodyLine, len(cmds))
	for i, c := range cmds {
		bls[i] = config.BodyLine{Type: "cmd", Text: c}
	}
	return config.Task{Name: name, BodyLines: bls}
}

func resolve(f *config.File, name string) ([]*config.Task, error) {
	seen := map[string]bool{}
	stack := map[string]bool{}
	order := []*config.Task{}
	err := Resolve(f, name, seen, stack, &order)
	return order, err
}

func names(tasks []*config.Task) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.Name
	}
	return out
}

func TestResolveSingleTask(t *testing.T) {
	f := makeTasks(task("build"))
	order, err := resolve(f, "build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 1 {
		t.Fatalf("expected 1 task, got %d", len(order))
	}
	if order[0].Name != "build" {
		t.Fatalf("expected 'build', got '%s'", order[0].Name)
	}
}

func TestResolveLinearDeps(t *testing.T) {
	f := makeTasks(
		task("a", "b"),
		task("b", "c"),
		task("c"),
	)
	order, err := resolve(f, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := names(order)
	// c before b before a
	if len(got) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(got))
	}
	// c must be before b, b must be before a
	pos := make(map[string]int)
	for i, n := range got {
		pos[n] = i
	}
	if pos["a"] < pos["b"] || pos["b"] < pos["c"] {
		t.Fatalf("expected order c -> b -> a, got %v", got)
	}
}

func TestResolveDiamondDeps(t *testing.T) {
	f := makeTasks(
		task("a", "b", "c"),
		task("b", "d"),
		task("c", "d"),
		task("d"),
	)
	order, err := resolve(f, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := names(order)
	if len(got) != 4 {
		t.Fatalf("expected 4 tasks, got %d: %v", len(got), got)
	}
	// d must be first, a must be last
	if got[0] != "d" {
		t.Fatalf("expected 'd' first, got '%s'", got[0])
	}
	if got[3] != "a" {
		t.Fatalf("expected 'a' last, got '%s'", got[3])
	}
}

func TestResolveCycleDetection(t *testing.T) {
	f := makeTasks(
		task("a", "b"),
		task("b", "a"),
	)
	_, err := resolve(f, "a")
	if err == nil {
		t.Fatal("expected error for circular dependency, got nil")
	}
	if !strings.Contains(err.Error(), "Circular dependency") {
		t.Fatalf("expected 'Circular dependency' error, got '%s'", err.Error())
	}
}

func TestResolveSelfReference(t *testing.T) {
	f := makeTasks(task("a", "a"))
	_, err := resolve(f, "a")
	if err == nil {
		t.Fatal("expected error for self-referencing task, got nil")
	}
}

func TestResolveMissingDependency(t *testing.T) {
	f := makeTasks(task("a", "nonexistent"))
	_, err := resolve(f, "a")
	if err == nil {
		t.Fatal("expected error for missing dependency, got nil")
	}
	if !strings.Contains(err.Error(), "Unknown dependency") {
		t.Fatalf("expected 'Unknown dependency' error, got '%s'", err.Error())
	}
}

func TestResolveNoDeps(t *testing.T) {
	f := makeTasks(
		task("a"),
		task("b"),
		task("c"),
	)
	order, err := resolve(f, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := names(order)
	if len(got) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got))
	}
	if got[0] != "a" {
		t.Fatalf("expected 'a', got '%s'", got[0])
	}
}

func TestResolveMultipleDeps(t *testing.T) {
	f := makeTasks(
		task("a", "b", "c"),
		task("b"),
		task("c"),
	)
	order, err := resolve(f, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := names(order)
	if len(got) != 3 {
		t.Fatalf("expected 3 tasks, got %d: %v", len(got), got)
	}
	// both b and c must be before a
	pos := make(map[string]int)
	for i, n := range got {
		pos[n] = i
	}
	if pos["a"] < pos["b"] || pos["a"] < pos["c"] {
		t.Fatalf("expected b and c before a, got %v", got)
	}
}

func TestResolveEmptyTaskWarning(t *testing.T) {
	f := makeTasks(
		taskWithCmd("a"),
		task("b", "a"),
	)
	// should not error, but prints warning via fmt.Printf
	order, err := resolve(f, "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(order))
	}
}

func TestResolveDeepGraph(t *testing.T) {
	f := makeTasks(
		task("a", "b"),
		task("b", "c"),
		task("c", "d"),
		task("d", "e"),
		task("e"),
	)
	order, err := resolve(f, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := names(order)
	if len(got) != 5 {
		t.Fatalf("expected 5 tasks, got %d: %v", len(got), got)
	}
	// order must be e, d, c, b, a
	expected := []string{"e", "d", "c", "b", "a"}
	for i, n := range expected {
		if got[i] != n {
			t.Fatalf("expected order %v, got %v", expected, got)
		}
	}
}
