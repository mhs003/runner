package config

import (
	"strings"
)

type Line struct {
	No     int
	Indent int
	Text   string
}

func Lex(input string) []Line {
	lines := strings.Split(input, "\n")
	out := make([]Line, 0, len(lines))

	for i, raw := range lines {
		// skip commented lines
		// if strings.HasPrefix(raw, "#") {
		// 	continue
		// }
		indent := 0
		for _, ch := range raw {
			if ch == ' ' {
				indent++
			} else {
				break
			}
		}

		content := strings.TrimSpace(raw)
		// skip commented lines
		if strings.HasPrefix(content, "#") {
			continue
		}
		// skip inline comments from task name and bultin block labels
		if hashPos := strings.Index(content, "#"); hashPos >= 0 && indent == 0 {
			content = content[:hashPos]
		}
		out = append(out, Line{
			No:     i + 1,
			Indent: indent,
			Text:   content,
		})
	}

	return out
}
