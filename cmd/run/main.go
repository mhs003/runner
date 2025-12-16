package main

import (
	"flag"
	"fmt"
	"maps"
	"mhs003/runner/internal/config"
	"mhs003/runner/internal/engine"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func main() {
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
