package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/neitanod/navisql/internal/config"
)

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	Connection    string
	Database      string
	Table         string
	Field         string
	RefDatabase   string
	RefTable      string
	RefField      string
}

// FKAdd adds a foreign key reference
func FKAdd(args []string) error {
	if len(args) < 6 {
		printFKAddHelp()
		return fmt.Errorf("missing required arguments")
	}

	fk := ForeignKey{
		Connection:  args[0],
		Database:    args[1],
		Table:       args[2],
		Field:       args[3],
		RefDatabase: args[4],
		RefTable:    args[5],
		RefField:    "id", // default
	}

	if len(args) > 6 {
		fk.RefField = args[6]
	}

	paths, err := config.GetPaths()
	if err != nil {
		return err
	}

	// Read existing FK file or create new
	existingFKs, err := loadForeignKeys(paths.FKFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check for duplicates
	for _, existing := range existingFKs {
		if existing.Connection == fk.Connection &&
			existing.Database == fk.Database &&
			existing.Table == fk.Table &&
			existing.Field == fk.Field {
			return fmt.Errorf("foreign key already exists for %s.%s.%s",
				fk.Database, fk.Table, fk.Field)
		}
	}

	// Append new FK
	existingFKs = append(existingFKs, fk)

	// Save
	if err := saveForeignKeys(paths.FKFile, existingFKs); err != nil {
		return err
	}

	fmt.Printf("Foreign key added: %s.%s.%s -> %s.%s.%s\n",
		fk.Database, fk.Table, fk.Field,
		fk.RefDatabase, fk.RefTable, fk.RefField)
	return nil
}

// FKExport exports foreign key configuration as shell commands
func FKExport(args []string) error {
	// Optional filters
	var filterConn, filterDB, filterTable string

	for i := 0; i < len(args); i++ {
		if args[i] == "--help" || args[i] == "-h" {
			printFKExportHelp()
			return nil
		}
		if !strings.HasPrefix(args[i], "-") {
			switch {
			case filterConn == "":
				filterConn = args[i]
			case filterDB == "":
				filterDB = args[i]
			case filterTable == "":
				filterTable = args[i]
			}
		}
	}

	paths, err := config.GetPaths()
	if err != nil {
		return err
	}

	fks, err := loadForeignKeys(paths.FKFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("# No foreign keys configured")
			return nil
		}
		return err
	}

	fmt.Println("# Foreign key configuration export")
	fmt.Println("# Run these commands to restore the configuration")
	fmt.Println()

	count := 0
	for _, fk := range fks {
		// Apply filters
		if filterConn != "" && fk.Connection != filterConn {
			continue
		}
		if filterDB != "" && fk.Database != filterDB {
			continue
		}
		if filterTable != "" && fk.Table != filterTable {
			continue
		}

		fmt.Printf("navisql fk add %s %s %s %s %s %s %s\n",
			fk.Connection, fk.Database, fk.Table, fk.Field,
			fk.RefDatabase, fk.RefTable, fk.RefField)
		count++
	}

	if count == 0 {
		fmt.Println("# No matching foreign keys found")
	}

	return nil
}

// LoadForeignKeysForTable returns all FKs for a specific table
func LoadForeignKeysForTable(connection, database, table string) ([]ForeignKey, error) {
	paths, err := config.GetPaths()
	if err != nil {
		return nil, err
	}

	allFKs, err := loadForeignKeys(paths.FKFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []ForeignKey
	for _, fk := range allFKs {
		if fk.Connection == connection && fk.Database == database && fk.Table == table {
			result = append(result, fk)
		}
	}

	return result, nil
}

func loadForeignKeys(path string) ([]ForeignKey, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fks []ForeignKey
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}

		fks = append(fks, ForeignKey{
			Connection:  parts[0],
			Database:    parts[1],
			Table:       parts[2],
			Field:       parts[3],
			RefDatabase: parts[4],
			RefTable:    parts[5],
			RefField:    parts[6],
		})
	}

	return fks, scanner.Err()
}

func saveForeignKeys(path string, fks []ForeignKey) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, fk := range fks {
		fmt.Fprintf(file, "%s,%s,%s,%s,%s,%s,%s\n",
			fk.Connection, fk.Database, fk.Table, fk.Field,
			fk.RefDatabase, fk.RefTable, fk.RefField)
	}

	return nil
}

func printFKAddHelp() {
	fmt.Println(`Usage: navisql fk add <conn> <db> <table> <field> <ref_db> <ref_table> [ref_field]

Arguments:
  conn        Connection name
  db          Database name
  table       Table name
  field       Field name that references another table
  ref_db      Referenced database name
  ref_table   Referenced table name
  ref_field   Referenced field name (default: id)

Example:
  navisql fk add local mydb users user_group_id mydb user_groups id
  navisql fk add local mydb orders customer_id mydb customers`)
}

func printFKExportHelp() {
	fmt.Println(`Usage: navisql fk export [connection] [database] [table]

Arguments (all optional, for filtering):
  connection   Filter by connection name
  database     Filter by database name
  table        Filter by table name

Examples:
  navisql fk export                    # Export all
  navisql fk export local              # Export for connection 'local'
  navisql fk export local mydb         # Export for 'local' connection, 'mydb' database
  navisql fk export local mydb users   # Export only for 'users' table`)
}
