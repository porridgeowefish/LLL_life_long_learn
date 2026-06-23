// Package imageconfig loads configuration for infographic image generation.
package imageconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/paths"
)

// Config holds the image generation settings from config.local.json.
type Config struct {
	ImageAPIKey      string `json:"imageApiKey"`
	ImageBaseURL     string `json:"imageBaseURL"`     // e.g. http://69.5.20.196:8080
	ImageModel       string `json:"imageModel"`       // gpt-image-2
	ImagePromptModel string `json:"imagePromptModel"` // opus (Stage A high-level model)
	PythonBin        string `json:"pythonBin"`        // python | python3

	// Backup image provider — tried only when the primary request fails or
	// returns a non-image (e.g. an error page). Optional; omit to disable.
	ImageBackupAPIKey  string `json:"imageBackupApiKey"`
	ImageBackupBaseURL string `json:"imageBackupBaseURL"` // e.g. https://api.ohmygpt.com/v1
	ImageBackupModel   string `json:"imageBackupModel"`   // gpt-image-2
}

// HasImageProvider reports whether at least one image provider (primary or
// backup) is configured with both an API key and a base URL.
func (c *Config) HasImageProvider() bool {
	return (c.ImageAPIKey != "" && c.ImageBaseURL != "") ||
		(c.ImageBackupAPIKey != "" && c.ImageBackupBaseURL != "")
}

// Load reads the config file from <workspace>/config.local.json.
// Returns (nil, nil) when the file is absent (feature disabled).
// Returns an error only when the file exists but cannot be parsed.
func Load() (*Config, error) {
	configPath := paths.WORKSPACE
	if configPath == "" {
		configPath = paths.PROJECT_ROOT
	}
	configFile := filepath.Join(configPath, "config.local.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // feature disabled
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Trim whitespace from string fields
	cfg.ImageAPIKey = strings.TrimSpace(cfg.ImageAPIKey)
	cfg.ImageBaseURL = strings.TrimSpace(cfg.ImageBaseURL)
	cfg.ImageModel = strings.TrimSpace(cfg.ImageModel)
	cfg.ImagePromptModel = strings.TrimSpace(cfg.ImagePromptModel)
	cfg.PythonBin = strings.TrimSpace(cfg.PythonBin)
	cfg.ImageBackupAPIKey = strings.TrimSpace(cfg.ImageBackupAPIKey)
	cfg.ImageBackupBaseURL = strings.TrimSpace(cfg.ImageBackupBaseURL)
	cfg.ImageBackupModel = strings.TrimSpace(cfg.ImageBackupModel)
	return &cfg, nil
}
