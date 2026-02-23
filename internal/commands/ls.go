package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/neitanod/navisql/internal/db"
	"github.com/neitanod/navisql/internal/output"
)

// Ls lists records from a table with pagination
func Ls(args []string) error {
	// Parse arguments
	opts := &lsOptions{
		page:    1,
		perPage: 30,
	}

	positional := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--where=") {
			opts.where = strings.TrimPrefix(arg, "--where=")
		} else if strings.HasPrefix(arg, "--per-page=") {
			val := strings.TrimPrefix(arg, "--per-page=")
			if n, err := strconv.Atoi(val); err == nil {
				opts.perPage = n
			}
		} else if arg == "--json" {
			opts.jsonOutput = true
		} else if arg == "--skip-ssl" {
			opts.skipSSL = true
		} else if arg == "--help" || arg == "-h" {
			printLsHelp()
			return nil
		} else if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
		} else {
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	if len(positional) < 3 {
		printLsHelp()
		return fmt.Errorf("missing required arguments")
	}

	opts.connection = positional[0]
	opts.database = positional[1]
	opts.table = positional[2]

	if len(positional) > 3 {
		if n, err := strconv.Atoi(positional[3]); err == nil {
			opts.page = n
		}
	}

	return runLs(opts)
}

type lsOptions struct {
	connection string
	database   string
	table      string
	page       int
	perPage    int
	where      string
	jsonOutput bool
	skipSSL    bool
}

func runLs(opts *lsOptions) error {
	conn, err := db.Connect(opts.connection, opts.database, opts.skipSSL)
	if err != nil {
		return err
	}
	defer conn.Close()

	offset := (opts.page - 1) * opts.perPage

	query := fmt.Sprintf("SELECT * FROM `%s`", opts.table)
	if opts.where != "" {
		query += " WHERE " + opts.where
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.perPage, offset)

	result, err := db.Query(conn, query)
	if err != nil {
		return err
	}

	if opts.jsonOutput {
		return output.PrintJSON(result)
	}

	output.PrintTabular(result)
	fmt.Printf("\nPage %d (%d rows)\n", opts.page, len(result.Rows))
	return nil
}

func printLsHelp() {
	fmt.Println(`Usage: navisql ls <connection> <database> <table> [page] [options]

Options:
  --where=CONDITION   WHERE clause for filtering
  --per-page=N        Number of rows per page (default: 30)
  --json              Output as JSON
  --skip-ssl          Skip SSL verification

Examples:
  navisql ls local mydb users
  navisql ls local mydb users 2
  navisql ls local mydb users --where="status = 1" --per-page=50
  navisql ls local mydb users --json`)
}
