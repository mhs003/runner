package config

type File struct {
    Vars  map[string]string
    Tasks map[string]*Task
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