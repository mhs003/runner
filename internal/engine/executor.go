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
var atRe = regexp.MustCompile(`^(\d*)(@)(\d*)$`)

func Execute(f *config.File, order []*config.Task, vars map[string]string, positional, allArgs []string, dry bool) error {
	shellCache := map[string]string{}
	for _, t := range order {
		if err := runTask(f, t, vars, positional, allArgs, dry, map[string]bool{}, shellCache); err != nil {
			return err
		}
	}
	return nil
}

func runTask(f *config.File, t *config.Task, vars map[string]string, positional, allArgs []string, dry bool, stack map[string]bool, shellCache map[string]string) error {
	cmds, err := collectCommands(f, t, vars, positional, allArgs, stack, shellCache)
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

func collectCommands(f *config.File, t *config.Task, vars map[string]string, positional, allArgs []string, stack map[string]bool, shellCache map[string]string) ([]string, error) {
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
			cmd, err := interpolate(c, vars, positional, shellCache, allArgs)
			if err != nil {
				return nil, err
			}
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

			sub, err := collectCommands(f, dep, vars, positional, allArgs, stack, shellCache)
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

func interpolate(s string, vars map[string]string, positional []string, shellCache map[string]string, allArgs ...[]string) (string, error) {
	all := positional
	if len(allArgs) > 0 {
		all = allArgs[0]
	}
	var firstErr error
	result := tokenRe.ReplaceAllStringFunc(s, func(match string) string {
		if firstErr != nil {
			return match
		}
		inner := match[2 : len(match)-2]

		var primary, fallback string
		if idx := strings.Index(inner, "||"); idx >= 0 {
			primary = inner[:idx]
			fallback = inner[idx+2:]
		} else {
			primary = inner
		}

		if v, ok := resolveAt(primary, positional, all); ok {
			if v != "" {
				return v
			}
			if fallback != "" {
				if v2, resolved, err := resolveLazyValue(fallback, vars, positional, shellCache, 0); resolved && v2 != "" {
					return v2
				} else if err != nil {
					firstErr = err
				}
				return fallback
			}
		}

		if v, resolved, err := resolveLazyValue(primary, vars, positional, shellCache, 0); resolved {
			if err != nil {
				firstErr = err
				return match
			}
			if v != "" {
				return v
			}
			if fallback != "" {
				if v2, resolved2, err2 := resolveLazyValue(fallback, vars, positional, shellCache, 0); resolved2 && v2 != "" {
					return v2
				} else if err2 != nil {
					firstErr = err2
				}
				return fallback
			}
		}

		if fallback != "" {
			if v, resolved, err := resolveLazyValue(fallback, vars, positional, shellCache, 0); resolved && v != "" {
				return v
			} else if err != nil {
				firstErr = err
			}
			return fallback
		}

		return ""
	})
	return result, firstErr
}

var varRefRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

func resolveLazyValue(key string, vars map[string]string, positional []string, shellCache map[string]string, depth int) (string, bool, error) {
	if depth > 10 {
		return "", false, nil
	}

	raw, ok := vars[key]
	if !ok {
		return "", false, nil
	}

	if cached, ok := shellCache[key]; ok {
		return cached, true, nil
	}

	hasShell := strings.Contains(raw, "$(")
	hasRefs := varRefRe.MatchString(raw)
	if !hasShell && !hasRefs {
		return raw, true, nil
	}

	if hasRefs {
		var refErr error
		raw = varRefRe.ReplaceAllStringFunc(raw, func(match string) string {
			if refErr != nil {
				return match
			}
			refKey := match[2 : len(match)-2]
			v, resolved, err := resolveLazyValue(refKey, vars, positional, shellCache, depth+1)
			if err != nil {
				refErr = err
				return match
			}
			if resolved {
				return v
			}
			return ""
		})
		if refErr != nil {
			return "", true, refErr
		}
	}

	if strings.Contains(raw, "$(") {
		resolved, err := resolveShellValue(raw, vars)
		if err != nil {
			return "", true, fmt.Errorf("var '%s': %w", key, err)
		}
		if shellCache != nil {
			shellCache[key] = resolved
		}
		return resolved, true, nil
	}

	return raw, true, nil
}

func resolveAt(primary string, positional, allArgs []string) (string, bool) {
	if primary == "@" {
		if len(allArgs) == 0 {
			return "", true
		}
		return strings.Join(allArgs, " "), true
	}

	m := atRe.FindStringSubmatch(primary)
	if m == nil {
		return "", false
	}

	prefix := m[1]
	suffix := m[3]

	if prefix == "" && suffix == "" {
		if len(positional) == 0 {
			return "", true
		}
		return strings.Join(positional, " "), true
	}

	if prefix == "" && suffix != "" {
		n, err := strconv.Atoi(suffix)
		if err != nil || n <= 0 {
			return "", true
		}
		if n > len(positional) {
			n = len(positional)
		}
		return strings.Join(positional[:n], " "), true
	}

	if prefix != "" && suffix == "" {
		n, err := strconv.Atoi(prefix)
		if err != nil || n <= 0 {
			return "", true
		}
		if n > len(positional) {
			return strings.Join(positional, " "), true
		}
		return strings.Join(positional[len(positional)-n:], " "), true
	}

	mv, err := strconv.Atoi(prefix)
	if err != nil || mv < 1 {
		mv = 1
	}
	nv, err := strconv.Atoi(suffix)
	if err != nil || nv < 1 {
		return "", true
	}
	mv--
	if mv >= nv {
		return "", true
	}
	if mv > len(positional) {
		return "", true
	}
	if nv > len(positional) {
		nv = len(positional)
	}
	return strings.Join(positional[mv:nv], " "), true
}

func resolveShellValue(value string, vars map[string]string) (string, error) {
	var result strings.Builder
	i := 0
	for i < len(value) {
		if i+1 < len(value) && value[i] == '$' && value[i+1] == '(' {
			j := i + 2
			depth := 1
			for j < len(value) && depth > 0 {
				switch value[j] {
				case '(':
					depth++
				case ')':
					depth--
				}
				j++
			}
			if depth != 0 {
				return "", fmt.Errorf("unclosed $( in value")
			}
			cmdStr := value[i+2 : j-1]
			out, err := runShellCmd(cmdStr, vars)
			if err != nil {
				return "", err
			}
			result.WriteString(strings.TrimRight(out, "\n\r"))
			i = j
		} else {
			result.WriteByte(value[i])
			i++
		}
	}
	return result.String(), nil
}

func runShellCmd(cmdStr string, vars map[string]string) (string, error) {
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	cmd.Env = os.Environ()
	for k, v := range vars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		stderr := string(out)
		if stderr == "" {
			stderr = err.Error()
		}
		return "", fmt.Errorf("$(%s) failed: exit code %s\n%s", cmdStr, err.Error(), strings.TrimRight(stderr, "\n\r"))
	}
	return string(out), nil
}
