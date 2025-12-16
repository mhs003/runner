package engine

import (
	"fmt"
	"mhs003/runner/internal/config"
	"os"
	"os/exec"
	"strings"
)

func Execute(tasks []*config.Task, vars map[string]string, cats map[string]*config.Cat, dry bool) error {
	for _, t := range tasks {
		for _, c := range t.Commands {
			cmd := interpolate(c, vars, cats)
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

func interpolate(s string, vars map[string]string, cats map[string]*config.Cat) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}

	for name, cat := range cats {
		s = strings.ReplaceAll(s, "{{"+name+"}}", cat.Content)
	}
	return s
}
