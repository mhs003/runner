package main

import (
	"flag"
	"fmt"
	"maps"
	"mhs003/runner/internal/config"
	"mhs003/runner/internal/engine"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

func main() {
	showList := flag.Bool("list", false, "Show list of all tasks")
	dry := flag.Bool("dry", false, "dry run")
	flag.Parse()

	taskName := "main" // default task
	args := flag.Args()
	if flag.NArg() >= 1 {
		taskName = flag.Arg(0)
		args = args[1:]
	}

	data, _, err := config.Load()
	if err != nil {
		panic(err)
	}

	lines := config.Lex(string(data))
	file, err := config.Parse(lines)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *showList {
		fmt.Println("Available tasks:\n")

		taskNames := make([]string, 0, len(file.Tasks))
		for name := range file.Tasks {
			taskNames = append(taskNames, name)
		}
		sort.Strings(taskNames)

		for i, name := range taskNames {
			task := file.Tasks[name]

			fmt.Printf("Task: %s\n", task.Name)

			if len(task.Deps) > 0 {
				fmt.Printf("  Dependencies: %s\n", strings.Join(task.Deps, ", "))
			} else {
				fmt.Println("  Dependencies: none")
			}

			if len(task.Commands) > 0 {
				fmt.Println("  Commands:")
				for i, cmd := range task.Commands {
					fmt.Printf("    %d. %s\n", i+1, cmd)
				}
			} else {
				fmt.Println("  Commands: none")
			}

			if task.Condition != nil {
				fmt.Printf("  Condition: %v\n", task.Condition)
			}

			if i < len(taskNames)-1 {
				fmt.Println("---")
			}
		}
		fmt.Println()
		os.Exit(0)
	}

	if _, ok := file.Tasks[taskName]; !ok {
		if flag.NArg() == 0 {
			fmt.Println("Please specify a task name.")
		} else {
			fmt.Printf("Task '%s' not found\n", taskName)
		}
		os.Exit(1)
	}

	vars := map[string]string{}
	maps.Copy(vars, file.Vars)

	cats := map[string]*config.Cat{}
	maps.Copy(cats, file.Cats)

	vars["CWD"], _ = os.Getwd()
	vars["OS"] = runtime.GOOS
	vars["ARCH"] = runtime.GOARCH

	// inject args
	ra := config.ParseArgs(args)

	vars["ARGS"] = strings.Join(ra.Positional, " ")
	// positional args
	for i, v := range ra.Positional {
		vars[strconv.Itoa(i+1)] = v
	}

	// named args
	maps.Copy(vars, ra.Named)

	// flags
	for k, v := range ra.Flags {
		vars[k] = strconv.FormatBool(v)
	}

	seen := map[string]bool{}
	stack := map[string]bool{}
	order := []*config.Task{}
	if err := engine.Resolve(file, taskName, seen, stack, &order); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := engine.Execute(order, vars, cats, *dry); err != nil {
		os.Exit(1)
	}
}
