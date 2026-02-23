package commands

import (
	"fmt"
	"strings"

	"github.com/neitanod/navisql/internal/config"
	"github.com/neitanod/navisql/internal/db"
)

// CacheBuild builds the autocompletion cache for a connection
func CacheBuild(args []string) error {
	skipSSL := false
	var connName string

	for _, arg := range args {
		if arg == "--skip-ssl" {
			skipSSL = true
		} else if arg == "--help" || arg == "-h" {
			printCacheBuildHelp()
			return nil
		} else if !strings.HasPrefix(arg, "-") {
			connName = arg
		}
	}

	if connName == "" {
		printCacheBuildHelp()
		return fmt.Errorf("missing connection name")
	}

	fmt.Printf("Building cache for connection '%s'...\n", connName)

	// Connect without selecting a database
	conn, err := db.ConnectNoDatabase(connName, skipSSL)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Get all databases
	databases, err := db.GetDatabases(conn)
	if err != nil {
		return fmt.Errorf("failed to get databases: %w", err)
	}

	// Load existing cache
	cache, err := config.LoadCache()
	if err != nil {
		cache = make(config.Cache)
	}

	// Initialize connection in cache
	cache[connName] = make(map[string][]string)

	// Get tables for each database
	for _, database := range databases {
		// Skip system databases
		if isSystemDatabase(database) {
			continue
		}

		fmt.Printf("  Scanning database: %s\n", database)

		tables, err := db.GetTables(conn, database)
		if err != nil {
			fmt.Printf("    Warning: failed to get tables for %s: %v\n", database, err)
			continue
		}

		cache[connName][database] = tables
	}

	// Save cache
	if err := config.SaveCache(cache); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	fmt.Printf("Cache built successfully for '%s' (%d databases)\n", connName, len(cache[connName]))
	return nil
}

func isSystemDatabase(name string) bool {
	systemDBs := []string{
		"information_schema",
		"mysql",
		"performance_schema",
		"sys",
	}

	for _, sysDB := range systemDBs {
		if name == sysDB {
			return true
		}
	}
	return false
}

func printCacheBuildHelp() {
	fmt.Println(`Usage: navisql cache build <connection> [options]

Arguments:
  connection    Name of the connection to build cache for

Options:
  --skip-ssl    Skip SSL verification

Description:
  Builds a cache of database and table names for the specified connection.
  This cache is used for tab completion in the shell.

Example:
  navisql cache build local
  navisql cache build production --skip-ssl`)
}
