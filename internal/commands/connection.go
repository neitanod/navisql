package commands

import (
	"fmt"

	"github.com/neitanod/navisql/internal/config"
	"github.com/neitanod/navisql/internal/output"
)

// ConnectionAdd adds a new database connection
func ConnectionAdd(args []string) error {
	if len(args) < 3 {
		printConnectionAddHelp()
		return fmt.Errorf("missing required arguments")
	}

	name := args[0]
	user := args[1]
	pass := args[2]
	host := "127.0.0.1"
	port := "3306"

	if len(args) > 3 {
		host = args[3]
	}
	if len(args) > 4 {
		port = args[4]
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.Connection[name] = config.Connection{
		User: user,
		Pass: pass,
		Host: host,
		Port: port,
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Connection '%s' added successfully\n", name)
	return nil
}

// ConnectionList lists all configured connections
func ConnectionList(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Connection) == 0 {
		fmt.Println("No connections configured")
		return nil
	}

	for name, conn := range cfg.Connection {
		fmt.Printf("%s%s%s: %s@%s:%s\n",
			output.Green, name, output.Normal,
			conn.User, conn.Host, conn.GetPort())
	}

	return nil
}

// ConnectionRemove removes a database connection
func ConnectionRemove(args []string) error {
	if len(args) < 1 {
		printConnectionRemoveHelp()
		return fmt.Errorf("missing connection name")
	}

	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Connection[name]; !ok {
		return fmt.Errorf("connection '%s' not found", name)
	}

	delete(cfg.Connection, name)

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Connection '%s' removed successfully\n", name)
	return nil
}

func printConnectionAddHelp() {
	fmt.Println(`Usage: navisql connection add <name> <user> <password> [host] [port]

Arguments:
  name       Connection name (used to reference this connection)
  user       MySQL username
  password   MySQL password
  host       MySQL host (default: 127.0.0.1)
  port       MySQL port (default: 3306)

Examples:
  navisql connection add local root password
  navisql connection add production dbuser dbpass db.example.com 3306`)
}

func printConnectionRemoveHelp() {
	fmt.Println(`Usage: navisql connection remove <name>

Arguments:
  name       Connection name to remove

Example:
  navisql connection remove old_connection`)
}
