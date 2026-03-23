package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey string `json:"api_key"`
	Token  string `json:"token"`
}

func Load() (Config, error) {
	// Try environment variables first
	key := os.Getenv("TRELLO_API_KEY")
	token := os.Getenv("TRELLO_TOKEN")
	if key != "" && token != "" {
		return Config{APIKey: key, Token: token}, nil
	}

	// Fall back to config file
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf("could not determine config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "trello-tui", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("no credentials found.\n\n"+
			"Set environment variables:\n"+
			"  export TRELLO_API_KEY=your_key\n"+
			"  export TRELLO_TOKEN=your_token\n\n"+
			"Or create a config file at %s:\n"+
			"  {\"api_key\": \"your_key\", \"token\": \"your_token\"}\n\n"+
			"Get your API key at: https://trello.com/power-ups/admin\n"+
			"Generate a token at: https://trello.com/1/authorize?expiration=never&scope=read,write&response_type=token&key=YOUR_API_KEY",
			configPath)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config file at %s: %w", configPath, err)
	}

	if cfg.APIKey == "" || cfg.Token == "" {
		return Config{}, fmt.Errorf("config file at %s must contain both api_key and token", configPath)
	}

	return cfg, nil
}
