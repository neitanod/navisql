package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Connection holds MySQL connection credentials
type Connection struct {
	User string      `json:"user"`
	Pass string      `json:"pass"`
	Host string      `json:"host"`
	Port interface{} `json:"port"` // Can be string or number in JSON
}

// GetPort returns the port as a string
func (c Connection) GetPort() string {
	switch p := c.Port.(type) {
	case string:
		if p == "" {
			return "3306"
		}
		return p
	case float64:
		return fmt.Sprintf("%.0f", p)
	default:
		return "3306"
	}
}

// Config holds the main navisql configuration
type Config struct {
	Connection map[string]Connection `json:"connection"`
	WebEdit    string                `json:"web_edit,omitempty"`
}

// Cache holds the database/table cache for autocompletion
type Cache map[string]map[string][]string // connection -> database -> tables

// Paths returns the standard navisql paths
type Paths struct {
	Dir        string
	ConfigFile string
	CacheFile  string
	FKFile     string
	HistoryFile string
}

// GetPaths returns the standard navisql paths
func GetPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(homeDir, ".navisql")
	return &Paths{
		Dir:         dir,
		ConfigFile:  filepath.Join(dir, "navisql.json"),
		CacheFile:   filepath.Join(dir, "navisql_cache.json"),
		FKFile:      filepath.Join(dir, "fk.csv"),
		HistoryFile: filepath.Join(dir, "history"),
	}, nil
}

// GetNaviLinksPath returns the path to navi links file
func GetNaviLinksPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".navi", "links"), nil
}

// EnsureDir ensures the navisql directory exists
func EnsureDir() error {
	paths, err := GetPaths()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(paths.Dir, 0755); err != nil {
		return err
	}

	// Create default config if it doesn't exist
	if _, err := os.Stat(paths.ConfigFile); os.IsNotExist(err) {
		defaultConfig := Config{Connection: make(map[string]Connection)}
		data, _ := json.MarshalIndent(defaultConfig, "", "  ")
		if err := os.WriteFile(paths.ConfigFile, data, 0644); err != nil {
			return err
		}
	}

	// Create default cache if it doesn't exist
	if _, err := os.Stat(paths.CacheFile); os.IsNotExist(err) {
		if err := os.WriteFile(paths.CacheFile, []byte("{}"), 0644); err != nil {
			return err
		}
	}

	return nil
}

// EnsureNaviDir ensures the navi directory exists
func EnsureNaviDir() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	naviDir := filepath.Join(homeDir, ".navi")
	return os.MkdirAll(naviDir, 0755)
}

// Load reads the main config file
func Load() (*Config, error) {
	paths, err := GetPaths()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Connection == nil {
		config.Connection = make(map[string]Connection)
	}

	return &config, nil
}

// Save writes the main config file
func Save(config *Config) error {
	paths, err := GetPaths()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(paths.ConfigFile, data, 0644)
}

// GetConnection retrieves a specific connection by name
func GetConnection(name string) (*Connection, error) {
	config, err := Load()
	if err != nil {
		return nil, err
	}

	conn, ok := config.Connection[name]
	if !ok {
		return nil, fmt.Errorf("connection '%s' not found", name)
	}

	return &conn, nil
}

// LoadCache reads the cache file
func LoadCache() (Cache, error) {
	paths, err := GetPaths()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(paths.CacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	return cache, nil
}

// SaveCache writes the cache file
func SaveCache(cache Cache) error {
	paths, err := GetPaths()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(paths.CacheFile, data, 0644)
}
