package config

type File struct {
	Vars  map[string]string
	Tasks map[string]*Task
}

type Dep struct {
	Name string
	Args []string
}

type Task struct {
	Name        string
	Deps        []Dep
	Commands    []string
	ExitOnError bool
}

type RunArgs struct {
	Positional []string
	Named      map[string]string
	Flags      map[string]bool
}

type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return e.Msg
}
