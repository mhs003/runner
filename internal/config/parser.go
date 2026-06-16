package config

import (
	"fmt"
	"strings"
)

type parser struct {
	f                  *File
	lines              []Line
	current            *Task
	pendingAnnotations []string
}

func Parse(lines []Line) (*File, error) {
	p := &parser{
		f:     &File{Vars: map[string]string{}, Tasks: map[string]*Task{}},
		lines: lines,
	}
	if err := p.parse(); err != nil {
		return nil, err
	}
	return p.f, nil
}

func (p *parser) parse() error {
	for i := 0; i < len(p.lines); {
		l := p.lines[i]
		if l.Text == "" {
			i++
			continue
		}

		if strings.HasSuffix(l.Text, ":") && l.Indent == 0 {
			next, err := p.dispatchHeader(l.Text, i)
			if err != nil {
				return err
			}
			i = next
			continue
		}

		if l.Indent == 0 && strings.HasPrefix(l.Text, "[") && strings.HasSuffix(l.Text, "]") {
			rest := l.Text[1 : len(l.Text)-1]
			switch rest {
			case "exit-on-error":
				p.pendingAnnotations = append(p.pendingAnnotations, "exit-on-error")
			default:
				return &ParseError{
					Line: l.No,
					Msg:  fmt.Sprintf("Unknown annotation '%s' at line %d", l.Text, l.No),
				}
			}
			i++
			continue
		}

		if l.Indent == 0 && p.current == nil {
			return &ParseError{
				Line: l.No,
				Msg:  fmt.Sprintf("Syntax error: unknown keyword '%s' at line '%d'", l.Text, l.No),
			}
		}

		if l.Indent > 0 {
			if p.current == nil {
				return &ParseError{
					Line: l.No,
					Msg:  fmt.Sprintf("Syntax error: command found outside of a task at line '%d' in '%s'", l.No, l.Text),
				}
			}
			if strings.HasPrefix(l.Text, "@") {
				rest := strings.TrimSpace(l.Text[1:])
				if rest != "" {
					parts := strings.Fields(rest)
					d := Dep{Name: parts[0]}
					if len(parts) > 1 {
						d.Args = parts[1:]
					}
					p.current.Deps = append(p.current.Deps, d)
				}
			} else {
				p.current.Commands = append(p.current.Commands, l.Text)
			}
			i++
			continue
		}
		i++
	}
	return nil
}

func (p *parser) dispatchHeader(text string, i int) (int, error) {
	name := strings.TrimSuffix(text, ":")

	if name == "@vars" {
		return p.parseVars(i)
	}

	return p.parseTaskHeader(name, i)
}

func (p *parser) parseVars(i int) (int, error) {
	j := i + 1
	for ; j < len(p.lines); j++ {
		if p.lines[j].Indent == 0 {
			break
		}
		if p.lines[j].Text == "" {
			continue
		}
		parts := strings.SplitN(p.lines[j].Text, "=", 2)
		if len(parts) == 2 {
			p.f.Vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	p.current = nil
	return j, nil
}

func (p *parser) parseTaskHeader(name string, i int) (int, error) {
	taskName := name
	deps := []Dep{}

	if strings.Contains(name, " ") {
		parts := strings.SplitN(name, " ", 2)
		taskName = strings.TrimSuffix(parts[0], ":")
		depNames := strings.Fields(parts[1])
		for _, dn := range depNames {
			deps = append(deps, Dep{Name: dn})
		}
	}

	if _, ok := p.f.Tasks[taskName]; ok {
		return 0, &ParseError{
			Line: p.lines[i].No,
			Msg:  fmt.Sprintf("Duplicate task '%s' at line %d", taskName, p.lines[i].No),
		}
	}

	p.current = &Task{Name: taskName, Deps: deps}
	for _, a := range p.pendingAnnotations {
		if a == "exit-on-error" {
			p.current.ExitOnError = true
		}
	}
	p.pendingAnnotations = nil
	p.f.Tasks[taskName] = p.current
	return i + 1, nil
}
