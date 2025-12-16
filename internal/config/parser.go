package config

import (
	"fmt"
	"os"
	"strings"
)

func Parse(lines []Line) (*File, error) {
	f := &File{
		Vars:  map[string]string{},
		Cats:  map[string]*Cat{},
		Tasks: map[string]*Task{},
	}

	var current *Task

	for i := 0; i < len(lines); {
		l := lines[i]
		if l.Text == "" {
			i++
			// current = nil
			continue
		}

		// consume task headers or meta blocks
		if strings.HasSuffix(l.Text, ":") && l.Indent == 0 {
			name := strings.TrimSuffix(l.Text, ":")

			if strings.HasPrefix(name, "@") {
				if name == "@vars" {
					j := i + 1
					for ; j < len(lines); j++ {
						if lines[j].Indent == 0 {
							break
						}
						if lines[j].Text == "" {
							continue
						}
						parts := strings.SplitN(lines[j].Text, "=", 2)
						if len(parts) == 2 {
							f.Vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
						}
					}
					current = nil
					i = j
					continue
				}

				if name == "@cat" {
					j := i + 1
					for ; j < len(lines); j++ {
						if lines[j].Indent == 0 {
							break
						}
						if lines[j].Text == "" {
							continue
						}
						parts := strings.SplitN(lines[j].Text, "=", 2)
						if len(parts) == 2 {
							catName := strings.TrimSpace(parts[0])
							path := strings.TrimSpace(parts[1])
							content := ""

							if data, err := os.ReadFile(path); err == nil {
								content = string(data)
							} else {
								fmt.Println(err)
							}

							f.Cats[catName] = &Cat{
								Name:     catName,
								FilePath: path,
								Content:  content,
							}
						}
					}
					current = nil
					i = j
					continue
				}

				// handle other ... meta blocks
				// ...
			}

			// comsume task headers
			taskName := name
			deps := []string{}

			if strings.Contains(name, " ") {
				parts := strings.SplitN(name, " ", 2)
				taskName = parts[0]
				deps = strings.Fields(parts[1])
			}

			current = &Task{Name: taskName, Deps: deps}
			f.Tasks[taskName] = current
			i++
			continue
		}

		if l.Indent == 0 && current == nil {
			return nil, &ParseError{
				Line: l.No,
				Msg:  fmt.Sprintf("Syntax error: unknown keyword '%s' at line '%d'", l.Text, l.No),
			}
		}

		// consume task commands
		if l.Indent > 0 {
			if current == nil {
				return nil, &ParseError{
					Line: l.No,
					Msg:  fmt.Sprintf("Syntax error: command found outside of a task at line '%d' in '%s'", l.No, l.Text),
				}
			}
			if strings.HasPrefix(l.Text, "@") {
				deps := strings.Fields(l.Text[1:])
				// push dependencies
				current.Deps = append(current.Deps, deps...)
			} else {
				current.Commands = append(current.Commands, l.Text)
			}
			i++
			continue
		}
		i++
	}

	return f, nil
}
