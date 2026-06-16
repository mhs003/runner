package engine

import (
	"fmt"
	"mhs003/runner/internal/config"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var tokenRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

func Execute(tasks []*config.Task, vars map[string]string, dry bool) error {
	for _, t := range tasks {
		var script strings.Builder
		if t.ExitOnError {
			script.WriteString("set -e\n")
		}
		for _, c := range t.Commands {
			shouldVerbose := false
			if strings.HasPrefix(c, "!") {
				shouldVerbose = true
				c = c[1:]
			}
			cmd := interpolate(c, vars)
			if dry {
				fmt.Println(cmd)
				continue
			}
			if shouldVerbose {
				fmt.Printf("> %s\n", cmd)
			}
			script.WriteString(cmd)
			script.WriteByte('\n')
		}
		if script.Len() > 0 && !dry {
			ec := exec.Command("/bin/sh", "-c", script.String())
			ec.Stdout = os.Stdout
			ec.Stderr = os.Stderr
			ec.Stdin = os.Stdin
			if err := ec.Run(); err != nil {
				return err
			}
		}
	}
	return nil
}

func interpolate(s string, vars map[string]string) string {
	return tokenRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := match[2 : len(match)-2]

		var primary, fallback string
		if idx := strings.Index(inner, "||"); idx >= 0 {
			primary = inner[:idx]
			fallback = inner[idx+2:]
		} else {
			primary = inner
		}

		if v, ok := vars[primary]; ok && v != "" {
			return v
		}

		if fallback != "" {
			if v, ok := vars[fallback]; ok {
				return v
			}
			return fallback
		}

		return match
	})
}
