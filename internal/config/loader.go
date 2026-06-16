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