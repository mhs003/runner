package config

type File struct {
	Vars  map[string]string
	Tasks map[string]*Task
	Cats  map[string]*Cat
}

type Cat struct {
	Name     string
	FilePath string
	Content  string
}

type Task struct {
	Name      string
	Deps      []string
	Commands  []string
	Condition *Condition
}

type Condition struct {
	EnvEquals map[string]string
}

type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return e.Msg
}
