package commands

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/neitanod/navisql/internal/db"
	"github.com/neitanod/navisql/internal/parser"
)

// Run executes queries from a SQL file
func Run(args []string) error {
	opts := &runOptions{
		mode:    "sequential",
		maxRows: 20,
		vars:    make(map[string]string),
	}

	positional := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--mode=") {
			opts.mode = strings.TrimPrefix(arg, "--mode=")
			if opts.mode != "sequential" && opts.mode != "transaction" && opts.mode != "interactive" {
				return fmt.Errorf("invalid mode: %s", opts.mode)
			}
		} else if arg == "--json" {
			opts.jsonOutput = true
		} else if strings.HasPrefix(arg, "--output=") {
			opts.outputFile = strings.TrimPrefix(arg, "--output=")
		} else if arg == "--full" {
			opts.maxRows = -1
		} else if strings.HasPrefix(arg, "--var") {
			var varDef string
			if strings.HasPrefix(arg, "--var=") {
				varDef = strings.TrimPrefix(arg, "--var=")
			} else if arg == "--var" && i+1 < len(args) {
				i++
				varDef = args[i]
			} else {
				return fmt.Errorf("--var requires a value")
			}
			eqIdx := strings.Index(varDef, "=")
			if eqIdx == -1 {
				return fmt.Errorf("invalid variable format: %s", varDef)
			}
			opts.vars[varDef[:eqIdx]] = varDef[eqIdx+1:]
		} else if arg == "--allow-missing-vars" {
			opts.allowMissingVars = true
		} else if arg == "--missing-vars-as-empty-string" {
			opts.missingVarsAsEmpty = true
		} else if arg == "--skip-ssl" {
			opts.skipSSL = true
		} else if arg == "--help" || arg == "-h" {
			printRunHelp()
			return nil
		} else if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
		} else {
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	if len(positional) < 3 {
		printRunHelp()
		return fmt.Errorf("missing required arguments")
	}

	opts.connection = positional[0]
	opts.database = positional[1]
	opts.file = positional[2]

	return runRun(opts)
}

type runOptions struct {
	connection         string
	database           string
	file               string
	mode               string
	jsonOutput         bool
	outputFile         string
	maxRows            int
	vars               map[string]string
	allowMissingVars   bool
	missingVarsAsEmpty bool
	skipSSL            bool
}

type queryResult struct {
	Index        int                      `json:"index"`
	SQL          string                   `json:"sql"`
	Success      bool                     `json:"success"`
	Error        string                   `json:"error,omitempty"`
	Rows         []map[string]interface{} `json:"rows,omitempty"`
	TotalRows    int                      `json:"total_rows,omitempty"`
	Truncated    bool                     `json:"truncated,omitempty"`
	AffectedRows int64                    `json:"affected_rows,omitempty"`
	TimeMs       int64                    `json:"time_ms"`
}

type runOutput struct {
	Queries []queryResult `json:"queries"`
	Summary struct {
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	} `json:"summary"`
}

func runRun(opts *runOptions) error {
	// Read file
	content, err := os.ReadFile(opts.file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	sqlContent := string(content)

	// Substitute variables
	sqlContent, missing, err := parser.SubstituteVariables(
		sqlContent, opts.vars, opts.allowMissingVars, opts.missingVarsAsEmpty)
	if err != nil {
		return err
	}
	if len(missing) > 0 && !opts.allowMissingVars && !opts.missingVarsAsEmpty {
		return fmt.Errorf("missing variables: %s", strings.Join(missing, ", "))
	}
	if len(missing) > 0 && opts.allowMissingVars {
		fmt.Fprintf(os.Stderr, "Warning: missing variables: %s\n", strings.Join(missing, ", "))
	}

	// Parse queries
	queries := parser.SplitQueries(sqlContent)
	if len(queries) == 0 {
		return fmt.Errorf("no queries found in file")
	}

	// Connect
	conn, err := db.Connect(opts.connection, opts.database, opts.skipSSL)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Execute
	output, err := executeQueries(conn, queries, opts)
	if err != nil {
		return err
	}

	// Output results
	return outputRunResults(output, opts)
}

func executeQueries(conn *sql.DB, queries []string, opts *runOptions) (*runOutput, error) {
	output := &runOutput{
		Queries: make([]queryResult, 0, len(queries)),
	}
	output.Summary.Total = len(queries)

	var tx *sql.Tx
	var err error

	if opts.mode == "transaction" {
		tx, err = conn.Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
	}

	for i, query := range queries {
		result := queryResult{
			Index: i + 1,
			SQL:   query,
		}

		// Interactive mode
		if opts.mode == "interactive" {
			fmt.Printf("\n=== Query %d ===\n%s\n\n", i+1, query)
			fmt.Print("[E]xecute / [S]kip / [A]bort? ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "a" || response == "abort" {
				fmt.Println("Aborted by user.")
				output.Summary.Failed = len(queries) - i
				return output, nil
			}
			if response == "s" || response == "skip" {
				result.Success = false
				result.Error = "skipped by user"
				output.Queries = append(output.Queries, result)
				continue
			}
		}

		start := time.Now()
		var execErr error

		if parser.IsSelectQuery(query) {
			var rows *sql.Rows
			if tx != nil {
				rows, execErr = tx.Query(query)
			} else {
				rows, execErr = conn.Query(query)
			}

			if execErr != nil {
				result.Success = false
				result.Error = execErr.Error()
			} else {
				result.Rows, result.TotalRows, result.Truncated, execErr = fetchRowsLimited(rows, opts.maxRows)
				rows.Close()
				if execErr != nil {
					result.Success = false
					result.Error = execErr.Error()
				} else {
					result.Success = true
				}
			}
		} else {
			var res sql.Result
			if tx != nil {
				res, execErr = tx.Exec(query)
			} else {
				res, execErr = conn.Exec(query)
			}

			if execErr != nil {
				result.Success = false
				result.Error = execErr.Error()
			} else {
				result.Success = true
				result.AffectedRows, _ = res.RowsAffected()
			}
		}

		result.TimeMs = time.Since(start).Milliseconds()
		output.Queries = append(output.Queries, result)

		if result.Success {
			output.Summary.Success++
		} else {
			output.Summary.Failed++

			if opts.mode == "transaction" && tx != nil {
				tx.Rollback()
				return output, fmt.Errorf("query %d failed: %s (transaction rolled back)", i+1, result.Error)
			}
		}
	}

	if opts.mode == "transaction" && tx != nil {
		if err := tx.Commit(); err != nil {
			return output, fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return output, nil
}

func fetchRowsLimited(rows *sql.Rows, maxRows int) ([]map[string]interface{}, int, bool, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, false, err
	}

	var results []map[string]interface{}
	totalRows := 0
	truncated := false

	for rows.Next() {
		totalRows++

		if maxRows > 0 && len(results) >= maxRows {
			truncated = true
			continue
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, 0, false, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return results, totalRows, truncated, rows.Err()
}

func outputRunResults(out *runOutput, opts *runOptions) error {
	var result strings.Builder

	if opts.jsonOutput {
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		result.Write(data)
		result.WriteString("\n")
	} else {
		for _, qr := range out.Queries {
			fmt.Fprintf(&result, "\n=== Query %d ===\n%s\n\n", qr.Index, qr.SQL)

			if !qr.Success {
				fmt.Fprintf(&result, "Error: %s\n", qr.Error)
			} else if qr.Rows != nil {
				if len(qr.Rows) > 0 {
					// Get columns from first row
					var columns []string
					for col := range qr.Rows[0] {
						columns = append(columns, col)
					}

					// Print header
					fmt.Fprintln(&result, strings.Join(columns, "\t"))

					// Print rows
					for _, row := range qr.Rows {
						var values []string
						for _, col := range columns {
							if val, ok := row[col]; ok {
								if val == nil {
									values = append(values, "NULL")
								} else {
									values = append(values, fmt.Sprintf("%v", val))
								}
							}
						}
						fmt.Fprintln(&result, strings.Join(values, "\t"))
					}

					if qr.Truncated {
						fmt.Fprintf(&result, "... (%d more rows, use --full to see all)\n", qr.TotalRows-len(qr.Rows))
					}
				}
				fmt.Fprintf(&result, "\nRows: %d | Time: %dms\n", qr.TotalRows, qr.TimeMs)
			} else {
				fmt.Fprintf(&result, "Affected: %d | Time: %dms\n", qr.AffectedRows, qr.TimeMs)
			}
		}

		fmt.Fprintf(&result, "\n=== Summary ===\n")
		fmt.Fprintf(&result, "%d queries executed, %d failed\n", out.Summary.Total, out.Summary.Failed)
	}

	content := result.String()

	if opts.outputFile != "" {
		if err := os.WriteFile(opts.outputFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Results written to %s\n", opts.outputFile)
	} else {
		fmt.Print(content)
	}

	return nil
}

func printRunHelp() {
	fmt.Println(`Usage: navisql run <connection> <database> <file.sql> [options]

Options:
  --mode=MODE                      Execution mode: sequential (default), transaction, interactive
  --json                           Output results as JSON
  --output=FILE                    Write results to file
  --full                           Show all rows (default: max 20 per query)
  --var KEY=VALUE                  Set variable (can be repeated)
  --allow-missing-vars             Continue with warning if variables are missing
  --missing-vars-as-empty-string   Replace missing variables with empty string
  --skip-ssl                       Skip SSL verification

Examples:
  navisql run local mydb script.sql
  navisql run local mydb report.sql --mode=transaction --json --output=result.json
  navisql run local mydb template.sql --var table=users --var status=active`)
}
