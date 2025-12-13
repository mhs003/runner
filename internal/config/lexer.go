package config

import "strings"

type Line struct {
    No     int
    Indent int
    Text   string
}

func Lex(input string) []Line {
    lines := strings.Split(input, "\n")
    out := make([]Line, 0, len(lines))

    for i, raw := range lines {
        indent := 0
        for _, ch := range raw {
            if ch == ' ' {
                indent++
            } else {
                break
            }
        }
        out = append(out, Line{
            No:     i + 1,
            Indent: indent,
            Text:   strings.TrimSpace(raw),
        })
    }

    return out
}