package commands

import (
	"fmt"
	"strings"

	"github.com/neitanod/navisql/internal/db"
	"github.com/neitanod/navisql/internal/output"
	"github.com/neitanod/navisql/internal/parser"
)

// Query executes a custom SQL query
func Query(args []string) error {
	opts := &queryOptions{}

	positional := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			opts.jsonOutput = true
		} else if arg == "--skip-ssl" {
			opts.skipSSL = true
		} else if arg == "--help" || arg == "-h" {
			printQueryHelp()
			return nil
		} else if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
		} else {
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	if len(positional) < 3 {
		printQueryHelp()
		return fmt.Errorf("missing required arguments")
	}

	opts.connection = positional[0]
	opts.database = positional[1]
	opts.query = positional[2]

	return runQuery(opts)
}

type queryOptions struct {
	connection string
	database   string
	query      string
	jsonOutput bool
	skipSSL    bool
}

func runQuery(opts *queryOptions) error {
	conn, err := db.Connect(opts.connection, opts.database, opts.skipSSL)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Check if it's a SELECT-type query
	if parser.IsSelectQuery(opts.query) {
		result, err := db.Query(conn, opts.query)
		if err != nil {
			return err
		}

		if opts.jsonOutput {
			return output.PrintJSON(result)
		}

		output.PrintTabular(result)
	} else {
		res, err := db.Exec(conn, opts.query)
		if err != nil {
			return err
		}

		affected, _ := res.RowsAffected()
		fmt.Printf("Affected rows: %d\n", affected)
	}

	return nil
}

func printQueryHelp() {
	fmt.Println(`Usage: navisql query <connection> <database> "<sql>" [options]

Options:
  --json       Output as JSON
  --skip-ssl   Skip SSL verification

Examples:
  navisql query local mydb "SELECT * FROM users LIMIT 10"
  navisql query local mydb "SELECT COUNT(*) FROM users" --json
  navisql query local mydb "UPDATE users SET status = 1 WHERE id = 5"`)
}
