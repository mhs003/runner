package main

import (
	"flag"
	"fmt"
	"mhs003/runner/internal/config"
	"mhs003/runner/internal/engine"
	"os"
	"runtime"
)

func main() {
    dry := flag.Bool("dry", false, "dry run")
    flag.Parse()

    if flag.NArg() < 1 {
        fmt.Println("task name required")
        os.Exit(1)
    }

    taskName := flag.Arg(0)

    data, _, err := config.Load()
    if err != nil {
        panic(err)
    }

    lines := config.Lex(string(data))
    file := config.Parse(lines)

    vars := map[string]string{}
    for k, v := range file.Vars {
        vars[k] = v
    }

    vars["PWD"], _ = os.Getwd()
    vars["OS"] = runtime.GOOS
    vars["ARCH"] = runtime.GOARCH

    seen := map[string]bool{}
    order := []*config.Task{}
    engine.Resolve(file, taskName, seen, &order)

    if err := engine.Execute(order, vars, *dry); err != nil {
        os.Exit(1)
    }
}
