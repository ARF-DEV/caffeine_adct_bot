package config

import (
	"encoding/json"
	"io"
	"os"
)

type Config struct {
	DiscordAppKey string `json:"discord_app_key"`
	RedisUrl      string `json:"redis_url"`
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}

	jsonB, err := io.ReadAll(file)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{}
	if err := json.Unmarshal(jsonB, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
