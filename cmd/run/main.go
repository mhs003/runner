package main

import (
	"flag"
	"fmt"
	"maps"
	"mhs003/runner/internal/config"
	"mhs003/runner/internal/display"
	"mhs003/runner/internal/engine"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	showList := flag.Bool("list", false, "Show list of all tasks")
	showJSON := flag.Bool("json", false, "Output in JSON format (use with --list)")
	dry := flag.Bool("dry", false, "dry run")
	filePath := flag.String("file", ".runner", "Path to runner config file")
	doInit := flag.Bool("init", false, "Initialize a .runner file in current directory")
	flag.Parse()

	if *doInit {
		if _, err := os.Stat(".runner"); err == nil {
			fmt.Fprintf(os.Stderr, "Error: .runner already exists in the current directory\n")
			os.Exit(1)
		}
		content := []byte("main:\n    echo \"I am running\"\n\n# Runner: https://github.com/mhs003/runner\n")
		if err := os.WriteFile(".runner", content, 0644); err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	taskName := "main" // default task
	args := flag.Args()
	if flag.NArg() >= 1 {
		taskName = flag.Arg(0)
		args = args[1:]
	}

	data, err := config.Load(*filePath)
	if err != nil {
		if *filePath == ".runner" {
			home, homeErr := os.UserHomeDir()
			if homeErr == nil {
				globalPath := home + "/.runner.global"
				data, err = config.Load(globalPath)
			}
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "No config found\n")
		os.Exit(1)
	}

	lines := config.Lex(string(data))
	file, err := config.Parse(lines)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := config.ResolveVars(file.Vars); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *showList {
		display.PrintTasks(file, *showJSON)
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

	if err := engine.Execute(file, order, vars, ra.Positional, *dry); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
