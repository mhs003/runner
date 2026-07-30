package config

import (
	"fmt"
	"regexp"
	"strings"
)

var varRefRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

func ResolveVars(vars map[string]string) error {
	return resolveVarRefs(vars)
}

func resolveVarRefs(vars map[string]string) error {
	const maxIter = 10
	for iter := 0; iter < maxIter; iter++ {
		changed := false
		for k, v := range vars {
			resolved := varRefRe.ReplaceAllStringFunc(v, func(match string) string {
				key := match[2 : len(match)-2]
				if val, ok := vars[key]; ok && !strings.Contains(val, "$(") {
					return val
				}
				return match
			})
			if resolved != v {
				vars[k] = resolved
				changed = true
			}
		}
		if !changed {
			for k, v := range vars {
				matches := varRefRe.FindAllStringSubmatch(v, -1)
				for _, m := range matches {
					if _, ok := vars[m[1]]; ok {
						if strings.Contains(vars[m[1]], "$(") {
							continue
						}
						return fmt.Errorf("circular variable reference at '%s'", k)
					}
				}
			}
			return nil
		}
	}
	return fmt.Errorf("circular variable reference detected in @vars")
}
