package config

import (
	"os"
	"path/filepath"
)

func Load() ([]byte, string, error) {
    cwd, err := os.Getwd()
    if err != nil {
        return nil, "", err
    }

    path := filepath.Join(cwd, ".runner")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, "", err
    }

    return data, path, nil
}