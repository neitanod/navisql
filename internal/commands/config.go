package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/neitanod/navisql/internal/config"
)

// ConfigAdd adds a configuration value
func ConfigAdd(args []string) error {
	if len(args) < 2 {
		printConfigAddHelp()
		return fmt.Errorf("missing required arguments")
	}

	path := args[0]
	value := args[1]

	paths, err := config.GetPaths()
	if err != nil {
		return err
	}

	// Read current config as raw JSON
	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		return err
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Set value at path (supports dot notation)
	if err := setNestedValue(cfg, path, value); err != nil {
		return err
	}

	// Write back
	newData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(paths.ConfigFile, newData, 0644); err != nil {
		return err
	}

	fmt.Printf("Configuration '%s' set to '%s'\n", path, value)
	return nil
}

// ConfigRemove removes a configuration value
func ConfigRemove(args []string) error {
	if len(args) < 1 {
		printConfigRemoveHelp()
		return fmt.Errorf("missing path argument")
	}

	path := args[0]

	paths, err := config.GetPaths()
	if err != nil {
		return err
	}

	// Read current config as raw JSON
	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		return err
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Remove value at path
	if err := removeNestedValue(cfg, path); err != nil {
		return err
	}

	// Write back
	newData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(paths.ConfigFile, newData, 0644); err != nil {
		return err
	}

	fmt.Printf("Configuration '%s' removed\n", path)
	return nil
}

func setNestedValue(obj map[string]interface{}, path, value string) error {
	parts := strings.Split(path, ".")

	// Navigate to parent
	current := obj
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		if next, ok := current[key]; ok {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// Overwrite non-map value with a map
				newMap := make(map[string]interface{})
				current[key] = newMap
				current = newMap
			}
		} else {
			// Create new nested map
			newMap := make(map[string]interface{})
			current[key] = newMap
			current = newMap
		}
	}

	// Set the final value
	finalKey := parts[len(parts)-1]
	current[finalKey] = value

	return nil
}

func removeNestedValue(obj map[string]interface{}, path string) error {
	parts := strings.Split(path, ".")

	// Navigate to parent
	current := obj
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		if next, ok := current[key]; ok {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return fmt.Errorf("path '%s' does not exist", path)
			}
		} else {
			return fmt.Errorf("path '%s' does not exist", path)
		}
	}

	// Remove the final key
	finalKey := parts[len(parts)-1]
	if _, ok := current[finalKey]; !ok {
		return fmt.Errorf("path '%s' does not exist", path)
	}
	delete(current, finalKey)

	return nil
}

func printConfigAddHelp() {
	fmt.Println(`Usage: navisql config add <path> <value>

Arguments:
  path    Dot-separated path to configuration key
  value   Value to set

Examples:
  navisql config add web_edit "http://adminer.local/?db={{DB}}&edit={{TABLE}}&where[id]={{ID}}"
  navisql config add defaults.per_page 50`)
}

func printConfigRemoveHelp() {
	fmt.Println(`Usage: navisql config remove <path>

Arguments:
  path    Dot-separated path to configuration key

Example:
  navisql config remove web_edit`)
}
