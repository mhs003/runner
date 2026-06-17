package display

import (
	"encoding/json"
	"fmt"
	"mhs003/runner/internal/config"
	"sort"
	"strings"
)

func PrintTasks(f *config.File, jsonOutput bool) {
	taskNames := make([]string, 0, len(f.Tasks))
	for name := range f.Tasks {
		taskNames = append(taskNames, name)
	}
	sort.Strings(taskNames)

	if jsonOutput {
		printJSON(f, taskNames)
		return
	}

	printPretty(f, taskNames)
}

func printJSON(f *config.File, taskNames []string) {
	type bodyItem struct {
		Type string   `json:"type"`
		Text string   `json:"text"`
		Args []string `json:"args,omitempty"`
	}
	taskList := make([]map[string]interface{}, 0, len(taskNames))
	for _, name := range taskNames {
		task := f.Tasks[name]
		deps := make([]string, len(task.HeaderDeps))
		for i, d := range task.HeaderDeps {
			deps[i] = d.Name
		}
		body := make([]bodyItem, len(task.BodyLines))
		for j, bl := range task.BodyLines {
			body[j] = bodyItem{Type: bl.Type, Text: bl.Text, Args: bl.Args}
		}
		taskList = append(taskList, map[string]interface{}{
			"name":        task.Name,
			"deps":        deps,
			"body":        body,
			"exitOnError": task.ExitOnError,
		})
	}
	out := map[string]interface{}{
		"tasks": taskList,
		"vars":  f.Vars,
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

func printPretty(f *config.File, taskNames []string) {
	maxLen := 0
	for _, name := range taskNames {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	pad := func(s string) string {
		return s + strings.Repeat(" ", maxLen-len(s))
	}

	fmt.Printf("Tasks (%d):\n\n", len(taskNames))
	for _, name := range taskNames {
		task := f.Tasks[name]
		first := true
		for _, bl := range task.BodyLines {
			line := ""
			switch bl.Type {
			case "cmd":
				line = bl.Text
			case "dep":
				line = "@ " + bl.Text
				for _, a := range bl.Args {
					line += " " + a
				}
			}
			if first {
				fmt.Printf("  %s  %s\n", pad(name), line)
				first = false
			} else {
				fmt.Printf("  %s  %s\n", pad(""), line)
			}
		}
		if first {
			fmt.Printf("  %s  (no commands)\n", pad(name))
		}
	}

	hasDeps := false
	for _, name := range taskNames {
		if len(f.Tasks[name].HeaderDeps) > 0 {
			hasDeps = true
			break
		}
	}
	if hasDeps {
		fmt.Println()
		fmt.Println("Dependencies:")
		for _, name := range taskNames {
			task := f.Tasks[name]
			if len(task.HeaderDeps) > 0 {
				depNames := make([]string, len(task.HeaderDeps))
				for i, d := range task.HeaderDeps {
					depNames[i] = d.Name
				}
				fmt.Printf("  %s  ←  %s\n", pad(name), strings.Join(depNames, ", "))
			}
		}
	}

	if len(f.Vars) > 0 {
		fmt.Println()
		fmt.Println("Vars:")
		varNames := make([]string, 0, len(f.Vars))
		for k := range f.Vars {
			varNames = append(varNames, k)
		}
		sort.Strings(varNames)
		maxKeyLen := 0
		for _, k := range varNames {
			if len(k) > maxKeyLen {
				maxKeyLen = len(k)
			}
		}
		for _, k := range varNames {
			fmt.Printf("  %s  %s\n", k+strings.Repeat(" ", maxKeyLen-len(k)), f.Vars[k])
		}
	}
	fmt.Println()
}
