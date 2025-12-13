package engine

import (
	"fmt"
	"mhs003/runner/internal/config"
	"os"
	"os/exec"
	"strings"
)

func Execute(tasks []*config.Task, vars map[string]string, dry bool) error {
    for _, t := range tasks {
		// if t.Name == "vars" {
		// 	continue
		// }
        for _, c := range t.Commands {
            cmd := interpolate(c, vars)
            if dry {
                fmt.Println(cmd)
                continue
            }
            ec := exec.Command("/bin/sh", "-c", cmd)
            ec.Stdout = os.Stdout
            ec.Stderr = os.Stderr
            if err := ec.Run(); err != nil {
                return err
            }
        }
    }
    return nil
}

func interpolate(s string, vars map[string]string) string {
    for k, v := range vars {
        s = strings.ReplaceAll(s, "{{"+k+"}}", v)
    }
    return s
}