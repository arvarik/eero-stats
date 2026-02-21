package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all the required environment variables.
type Config struct {
	EeroLogin    string
	InfluxURL    string
	InfluxToken  string
	InfluxOrg    string
	InfluxBucket string
}

// Load reads config from environment variables and an optional .env file.
func Load() (*Config, error) {
	// Attempt to load .env file; it's okay if it doesn't exist 
	// (e.g., config passed via Docker env vars)
	_ = godotenv.Load()

	cfg := &Config{
		EeroLogin:    os.Getenv("EERO_LOGIN"),
		InfluxURL:    os.Getenv("INFLUX_URL"),
		InfluxToken:  os.Getenv("INFLUX_TOKEN"),
		InfluxOrg:    os.Getenv("INFLUX_ORG"),
		InfluxBucket: os.Getenv("INFLUX_BUCKET"),
	}

	if cfg.EeroLogin == "" {
		return nil, errors.New("EERO_LOGIN environment variable is required")
	}
	if cfg.InfluxURL == "" {
		return nil, errors.New("INFLUX_URL environment variable is required")
	}
	if cfg.InfluxToken == "" {
		return nil, errors.New("INFLUX_TOKEN environment variable is required")
	}
	if cfg.InfluxOrg == "" {
		return nil, errors.New("INFLUX_ORG environment variable is required")
	}
	if cfg.InfluxBucket == "" {
		return nil, errors.New("INFLUX_BUCKET environment variable is required")
	}

	return cfg, nil
}
