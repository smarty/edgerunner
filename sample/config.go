package main

import (
	"encoding/json"
	"os"
)

type ConfigFile struct {
	Message string `json:"message"`
}

func LoadConfig(path string) (config ConfigFile, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(raw, &config)
	return config, err
}
