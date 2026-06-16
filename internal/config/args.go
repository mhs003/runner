package config

import (
	"strings"
)

func ParseArgs(args []string) RunArgs {
	ra := RunArgs{
		Named: make(map[string]string),
		Flags: make(map[string]bool),
	}

	for i := 0; i < len(args); i++ {
		a := args[i]

		if after, ok := strings.CutPrefix(a, "--"); ok {
			if key, val, ok := strings.Cut(after, "="); ok {
				ra.Named["--"+key] = val
				continue
			}

			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				ra.Named["--"+after] = args[i+1]
				i++
			} else {
				ra.Flags["--"+after] = true
			}
			continue
		}

		if strings.HasPrefix(a, "-") && len(a) > 1 {
			if len(a) > 2 {
				for _, ch := range a[1:] {
					ra.Flags["-"+string(ch)] = true
				}
				continue
			}

			key := strings.TrimPrefix(a, "-")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				ra.Named["-"+key] = args[i+1]
				i++
			} else {
				ra.Flags["-"+key] = true
			}
			continue
		}

		ra.Positional = append(ra.Positional, a)
	}

	return ra
}
