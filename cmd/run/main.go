package main

import (
	"errors"
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
	fileExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "file" {
			fileExplicit = true
		}
	})

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

	file, err := loadConfig(*filePath, fileExplicit)
	if err != nil {
		if errors.Is(err, errNoConfig) {
			fmt.Fprintln(os.Stderr, "No config found")
			os.Exit(1)
		}
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

	if err := engine.Execute(file, order, vars, ra.Positional, ra.All, *dry); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var errNoConfig = errors.New("no config found")

func loadConfig(path string, explicit bool) (*config.File, error) {
	if explicit {
		file, err := loadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil, errNoConfig
		}
		return file, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	local, localErr := loadFile(path)
	if localErr != nil && !errors.Is(localErr, os.ErrNotExist) {
		return nil, localErr
	}

	global, globalErr := loadFile(home + "/.runner.global")
	if globalErr != nil && !errors.Is(globalErr, os.ErrNotExist) {
		return nil, globalErr
	}

	if local == nil && global == nil {
		return nil, errNoConfig
	}
	return config.Merge(global, local), nil
}

func loadFile(path string) (*config.File, error) {
	data, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return config.Parse(config.Lex(string(data)))
}
