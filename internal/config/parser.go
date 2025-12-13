package config

import (
	"strings"
)

func Parse(lines []Line) *File {
    f := &File{
        Vars:  map[string]string{},
        Tasks: map[string]*Task{},
    }

    var current *Task

    for i := 0; i < len(lines); i++ {
        l := lines[i]
        if l.Text == "" {
            continue
        }

        if strings.HasSuffix(l.Text, ":") && l.Indent == 0 {
            name := strings.TrimSuffix(l.Text, ":")

            if name == "vars" {
                for j := i + 1; j < len(lines) && lines[j].Indent > 0; j++ {
                    parts := strings.SplitN(lines[j].Text, "=", 2)
                    if len(parts) == 2 {
                        f.Vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
                    }
                }
                continue
            }

            parts := strings.Split(name, " ")
            taskName := parts[0]
            deps := []string{}
            if len(parts) > 1 {
                deps = parts[1:]
            }

            current = &Task{Name: taskName, Deps: deps}
            f.Tasks[taskName] = current
            continue
        }

        if current != nil && l.Indent > 0 {
            current.Commands = append(current.Commands, l.Text)
        }
    }

    return f
}