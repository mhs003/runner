package engine

import (
	"fmt"
	"mhs003/runner/internal/config"
)

func Resolve(f *config.File, name string, seen map[string]bool, stack map[string]bool, out *[]*config.Task) error {
	if stack[name] {
		return fmt.Errorf("Circular dependency detected at '%s'", name)
	}

	if seen[name] {
		return nil
	}

	if f.Tasks[name].Commands == nil {
		fmt.Printf("Warning: task '%s' has no command\n", name)
	}

	t, ok := f.Tasks[name]
	if !ok {
		return fmt.Errorf("Unknown dependency task '%s'", name)
	}

	stack[name] = true

	for _, d := range t.Deps {
		if err := Resolve(f, d, seen, stack, out); err != nil {
			return err
		}
	}

	stack[name] = false
	seen[name] = true

	*out = append(*out, t)

	return nil
}
