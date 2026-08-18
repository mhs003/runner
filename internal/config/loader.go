package config

import (
	"os"
)

func Load(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func Merge(global, local *File) *File {
	merged := &File{
		Vars:  map[string]string{},
		Tasks: map[string]*Task{},
	}

	if global != nil {
		for name, value := range global.Vars {
			merged.Vars[name] = value
		}
		for name, task := range global.Tasks {
			merged.Tasks[name] = task
		}
	}

	if local != nil {
		for name, value := range local.Vars {
			merged.Vars[name] = value
		}
		for name, task := range local.Tasks {
			merged.Tasks[name] = task
		}
	}

	return merged
}
