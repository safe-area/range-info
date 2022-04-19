package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port    string `json:"port"`
	Storage struct {
		Host string `json:"host"`
	} `json:"storage"`
	Dev bool `json:"dev"`
}

func ParseConfig(configPath string) (*Config, error) {
	fileBody, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(fileBody, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
