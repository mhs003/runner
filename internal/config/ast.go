package config

type File struct {
	Vars  map[string]string
	Tasks map[string]*Task
}

type Dep struct {
	Name string
	Args []string
}

type BodyLine struct {
	Type string   // "cmd" or "dep"
	Text string   // command text (with ! prefix) or dep name
	Args []string // dep arguments (only for "dep" type)
}

type Task struct {
	Name        string
	HeaderDeps  []Dep
	BodyLines   []BodyLine
	ExitOnError bool
}

type RunArgs struct {
	Positional []string
	All        []string
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
