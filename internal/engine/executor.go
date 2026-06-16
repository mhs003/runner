package engine

import (
	"fmt"
	"mhs003/runner/internal/config"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var tokenRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

func Execute(f *config.File, order []*config.Task, vars map[string]string, dry bool) error {
	for _, t := range order {
		if err := runTask(f, t, vars, dry, map[string]bool{}); err != nil {
			return err
		}
	}
	return nil
}

func runTask(f *config.File, t *config.Task, vars map[string]string, dry bool, stack map[string]bool) error {
	cmds, err := collectCommands(f, t, vars, stack)
	if err != nil {
		return err
	}
	if len(cmds) == 0 {
		return nil
	}

	if dry {
		for _, cmd := range cmds {
			fmt.Println(cmd)
		}
		return nil
	}

	var script strings.Builder
	if t.ExitOnError {
		script.WriteString("set -e\n")
	}
	for _, cmd := range cmds {
		script.WriteString(cmd)
		script.WriteByte('\n')
	}

	ec := exec.Command("/bin/sh", "-c", script.String())
	ec.Stdout = os.Stdout
	ec.Stderr = os.Stderr
	ec.Stdin = os.Stdin
	return ec.Run()
}

func collectCommands(f *config.File, t *config.Task, vars map[string]string, stack map[string]bool) ([]string, error) {
	if stack[t.Name] {
		return nil, fmt.Errorf("circular dependency detected at '%s'", t.Name)
	}
	stack[t.Name] = true
	defer delete(stack, t.Name)

	var cmds []string
	for _, line := range t.BodyLines {
		switch line.Type {
		case "cmd":
			shouldVerbose := false
			c := line.Text
			if strings.HasPrefix(c, "!") {
				shouldVerbose = true
				c = c[1:]
			}
			cmd := interpolate(c, vars)
			if shouldVerbose {
				escaped := strings.ReplaceAll(cmd, "'", "'\\''")
				cmds = append(cmds, "echo '> "+escaped+"'")
			}
			cmds = append(cmds, cmd)

		case "dep":
			var callCleanup []string
			if len(line.Args) > 0 {
				ra := config.ParseArgs(line.Args)
				for k, v := range ra.Named {
					if _, exists := vars[k]; !exists {
						callCleanup = append(callCleanup, k)
					}
					vars[k] = v
				}
				for k, v := range ra.Flags {
					if _, exists := vars[k]; !exists {
						callCleanup = append(callCleanup, k)
					}
					vars[k] = strconv.FormatBool(v)
				}
			}

			dep := f.Tasks[line.Text]
			if dep == nil {
				return nil, fmt.Errorf("unknown dependency task '%s'", line.Text)
			}

			sub, err := collectCommands(f, dep, vars, stack)
			if err != nil {
				return nil, err
			}
			cmds = append(cmds, sub...)

			for _, k := range callCleanup {
				delete(vars, k)
			}
		}
	}

	return cmds, nil
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
