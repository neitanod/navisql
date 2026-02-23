package main

import (
	"bufio"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Config structures
type ConnectionConfig struct {
	User string      `json:"user"`
	Pass string      `json:"pass"`
	Host string      `json:"host"`
	Port interface{} `json:"port"`
}

type Config struct {
	Connection map[string]ConnectionConfig `json:"connection"`
}

// Result structures for JSON output
type QueryResult struct {
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

type Summary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type Output struct {
	Queries []QueryResult `json:"queries"`
	Summary Summary       `json:"summary"`
}

// Command line options
type Options struct {
	Connection           string
	Database             string
	File                 string
	Mode                 string
	JSONOutput           bool
	OutputFile           string
	Full                 bool
	Variables            map[string]string
	AllowMissingVars     bool
	MissingVarsAsEmpty   bool
	SkipSSL              bool
	MaxRows              int
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		printUsage()
		os.Exit(1)
	}

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `
Usage: navisql run <connection> <database> <file.sql> [options]

Options:
  --mode=MODE                   Execution mode: sequential (default), transaction, interactive
  --json                        Output results as JSON
  --output=FILE                 Write results to file
  --full                        Show all rows (default: max 20 per query)
  --var KEY=VALUE               Set variable (can be repeated)
  --allow-missing-vars          Continue with warning if variables are missing
  --missing-vars-as-empty-string  Replace missing variables with empty string
  --skip-ssl                    Skip SSL verification

Examples:
  navisql run local mydb script.sql
  navisql run local mydb report.sql --mode=transaction --json --output=result.json
  navisql run local mydb template.sql --var table=users --var status=active
`)
}

func parseArgs(args []string) (*Options, error) {
	opts := &Options{
		Mode:      "sequential",
		Variables: make(map[string]string),
		MaxRows:   20,
	}

	positional := []string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "--mode=") {
			opts.Mode = strings.TrimPrefix(arg, "--mode=")
			if opts.Mode != "sequential" && opts.Mode != "transaction" && opts.Mode != "interactive" {
				return nil, fmt.Errorf("invalid mode: %s (must be sequential, transaction, or interactive)", opts.Mode)
			}
		} else if arg == "--json" {
			opts.JSONOutput = true
		} else if strings.HasPrefix(arg, "--output=") {
			opts.OutputFile = strings.TrimPrefix(arg, "--output=")
		} else if arg == "--full" {
			opts.Full = true
			opts.MaxRows = -1
		} else if strings.HasPrefix(arg, "--var") {
			var varDef string
			if strings.HasPrefix(arg, "--var=") {
				varDef = strings.TrimPrefix(arg, "--var=")
			} else if arg == "--var" && i+1 < len(args) {
				i++
				varDef = args[i]
			} else {
				return nil, fmt.Errorf("--var requires a value")
			}
			eqIdx := strings.Index(varDef, "=")
			if eqIdx == -1 {
				return nil, fmt.Errorf("invalid variable format: %s (expected KEY=VALUE)", varDef)
			}
			key := varDef[:eqIdx]
			value := varDef[eqIdx+1:]
			opts.Variables[key] = value
		} else if arg == "--allow-missing-vars" {
			opts.AllowMissingVars = true
		} else if arg == "--missing-vars-as-empty-string" {
			opts.MissingVarsAsEmpty = true
		} else if arg == "--skip-ssl" {
			opts.SkipSSL = true
		} else if strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unknown option: %s", arg)
		} else {
			positional = append(positional, arg)
		}
	}

	if len(positional) < 3 {
		return nil, fmt.Errorf("missing required arguments: connection, database, file")
	}

	opts.Connection = positional[0]
	opts.Database = positional[1]
	opts.File = positional[2]

	return opts, nil
}

func run(opts *Options) error {
	// Load config
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	connConfig, ok := config.Connection[opts.Connection]
	if !ok {
		return fmt.Errorf("connection '%s' not found in config", opts.Connection)
	}

	// Read SQL file
	content, err := ioutil.ReadFile(opts.File)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	sqlContent := string(content)

	// Substitute variables
	sqlContent, err = substituteVariables(sqlContent, opts)
	if err != nil {
		return err
	}

	// Parse queries
	queries := parseQueries(sqlContent)
	if len(queries) == 0 {
		return fmt.Errorf("no queries found in file")
	}

	// Connect to database
	db, err := connectDB(connConfig, opts.Database, opts.SkipSSL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Execute queries
	output, err := executeQueries(db, queries, opts)
	if err != nil {
		return err
	}

	// Format and output results
	return outputResults(output, opts)
}

func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".navisql", "navisql.json")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func connectDB(conn ConnectionConfig, database string, skipSSL bool) (*sql.DB, error) {
	port := "3306"
	switch p := conn.Port.(type) {
	case string:
		port = p
	case float64:
		port = fmt.Sprintf("%.0f", p)
	}

	tlsConfig := ""
	if skipSSL {
		tlsConfig = "?tls=false"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s%s",
		conn.User, conn.Pass, conn.Host, port, database, tlsConfig)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Configure TLS if needed
	if skipSSL {
		_ = tls.Config{InsecureSkipVerify: true}
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func substituteVariables(content string, opts *Options) (string, error) {
	// Find all {{var}} patterns
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	missingVars := []string{}
	seen := make(map[string]bool)

	for _, match := range matches {
		varName := match[1]
		if seen[varName] {
			continue
		}
		seen[varName] = true

		if _, ok := opts.Variables[varName]; !ok {
			missingVars = append(missingVars, varName)
		}
	}

	if len(missingVars) > 0 {
		if opts.MissingVarsAsEmpty {
			// Replace missing vars with empty string
			for _, v := range missingVars {
				opts.Variables[v] = ""
			}
		} else if opts.AllowMissingVars {
			// Warn but continue, leaving placeholders
			fmt.Fprintf(os.Stderr, "Warning: missing variables: %s\n", strings.Join(missingVars, ", "))
			// Don't replace these variables
			result := content
			for varName, value := range opts.Variables {
				result = strings.ReplaceAll(result, "{{"+varName+"}}", value)
			}
			return result, nil
		} else {
			// Error and abort
			return "", fmt.Errorf("missing variables: %s", strings.Join(missingVars, ", "))
		}
	}

	// Replace all variables
	result := content
	for varName, value := range opts.Variables {
		result = strings.ReplaceAll(result, "{{"+varName+"}}", value)
	}

	return result, nil
}

// parseQueries splits SQL content by semicolons, respecting strings and comments
func parseQueries(content string) []string {
	var queries []string
	var current strings.Builder

	i := 0
	n := len(content)

	for i < n {
		ch := content[i]

		// Check for single-line comment
		if ch == '-' && i+1 < n && content[i+1] == '-' {
			// Include comment in current query
			current.WriteByte(ch)
			i++
			current.WriteByte(content[i])
			i++
			// Read until end of line
			for i < n && content[i] != '\n' {
				current.WriteByte(content[i])
				i++
			}
			continue
		}

		// Check for multi-line comment
		if ch == '/' && i+1 < n && content[i+1] == '*' {
			current.WriteByte(ch)
			i++
			current.WriteByte(content[i])
			i++
			// Read until */
			for i < n {
				if content[i] == '*' && i+1 < n && content[i+1] == '/' {
					current.WriteByte(content[i])
					i++
					current.WriteByte(content[i])
					i++
					break
				}
				current.WriteByte(content[i])
				i++
			}
			continue
		}

		// Check for strings (single or double quotes)
		if ch == '\'' || ch == '"' {
			quote := ch
			current.WriteByte(ch)
			i++
			// Read until closing quote (handle escaped quotes)
			for i < n {
				c := content[i]
				current.WriteByte(c)
				i++
				if c == quote {
					// Check for escaped quote
					if i < n && content[i] == quote {
						current.WriteByte(content[i])
						i++
						continue
					}
					break
				}
				if c == '\\' && i < n {
					current.WriteByte(content[i])
					i++
				}
			}
			continue
		}

		// Check for backtick identifiers
		if ch == '`' {
			current.WriteByte(ch)
			i++
			for i < n && content[i] != '`' {
				current.WriteByte(content[i])
				i++
			}
			if i < n {
				current.WriteByte(content[i])
				i++
			}
			continue
		}

		// Check for semicolon (query separator)
		if ch == ';' {
			query := strings.TrimSpace(current.String())
			if query != "" {
				queries = append(queries, query)
			}
			current.Reset()
			i++
			continue
		}

		current.WriteByte(ch)
		i++
	}

	// Don't forget the last query if no trailing semicolon
	query := strings.TrimSpace(current.String())
	if query != "" {
		queries = append(queries, query)
	}

	return queries
}

func executeQueries(db *sql.DB, queries []string, opts *Options) (*Output, error) {
	output := &Output{
		Queries: make([]QueryResult, 0, len(queries)),
		Summary: Summary{Total: len(queries)},
	}

	var tx *sql.Tx
	var err error

	if opts.Mode == "transaction" {
		tx, err = db.Begin()
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
	}

	for i, query := range queries {
		result := QueryResult{
			Index: i + 1,
			SQL:   query,
		}

		// Interactive mode: ask before each query
		if opts.Mode == "interactive" {
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

		// Determine if it's a SELECT query
		isSelect := isSelectQuery(query)

		if isSelect {
			var rows *sql.Rows
			if tx != nil {
				rows, execErr = tx.Query(query)
			} else {
				rows, execErr = db.Query(query)
			}

			if execErr != nil {
				result.Success = false
				result.Error = execErr.Error()
			} else {
				result.Rows, result.TotalRows, result.Truncated, execErr = fetchRows(rows, opts.MaxRows)
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
				res, execErr = db.Exec(query)
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

			// In transaction mode, rollback on first error
			if opts.Mode == "transaction" && tx != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: rollback failed: %s\n", rbErr)
				}
				return output, fmt.Errorf("query %d failed: %s (transaction rolled back)", i+1, result.Error)
			}
		}
	}

	// Commit transaction if all successful
	if opts.Mode == "transaction" && tx != nil {
		if err := tx.Commit(); err != nil {
			return output, fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return output, nil
}

func isSelectQuery(query string) bool {
	// Strip leading comments and whitespace to find the actual statement
	q := strings.TrimSpace(query)

	for {
		// Skip single-line comments
		if strings.HasPrefix(q, "--") {
			if idx := strings.Index(q, "\n"); idx != -1 {
				q = strings.TrimSpace(q[idx+1:])
				continue
			}
			return false // Only comment, no actual query
		}
		// Skip multi-line comments
		if strings.HasPrefix(q, "/*") {
			if idx := strings.Index(q, "*/"); idx != -1 {
				q = strings.TrimSpace(q[idx+2:])
				continue
			}
			return false // Unclosed comment
		}
		break
	}

	q = strings.ToUpper(q)
	return strings.HasPrefix(q, "SELECT") ||
		strings.HasPrefix(q, "SHOW") ||
		strings.HasPrefix(q, "DESCRIBE") ||
		strings.HasPrefix(q, "DESC") ||
		strings.HasPrefix(q, "EXPLAIN")
}

func fetchRows(rows *sql.Rows, maxRows int) ([]map[string]interface{}, int, bool, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, false, err
	}

	var results []map[string]interface{}
	totalRows := 0
	truncated := false

	for rows.Next() {
		totalRows++

		// If we've hit the limit, just count remaining rows
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

func outputResults(output *Output, opts *Options) error {
	var out strings.Builder

	if opts.JSONOutput {
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		out.Write(data)
		out.WriteString("\n")
	} else {
		for _, result := range output.Queries {
			fmt.Fprintf(&out, "\n=== Query %d ===\n", result.Index)
			fmt.Fprintf(&out, "%s\n\n", result.SQL)

			if !result.Success {
				fmt.Fprintf(&out, "Error: %s\n", result.Error)
			} else if result.Rows != nil {
				// Print tabular output
				if len(result.Rows) > 0 {
					// Get column order from first row
					columns := getColumnOrder(result.Rows[0])

					// Print header
					fmt.Fprintln(&out, strings.Join(columns, "\t"))

					// Print rows
					for _, row := range result.Rows {
						values := make([]string, len(columns))
						for i, col := range columns {
							if val, ok := row[col]; ok {
								if val == nil {
									values[i] = "NULL"
								} else {
									values[i] = fmt.Sprintf("%v", val)
								}
							}
						}
						fmt.Fprintln(&out, strings.Join(values, "\t"))
					}

					if result.Truncated {
						fmt.Fprintf(&out, "... (%d more rows, use --full to see all)\n", result.TotalRows-len(result.Rows))
					}
				}
				fmt.Fprintf(&out, "\nRows: %d | Time: %dms\n", result.TotalRows, result.TimeMs)
			} else {
				fmt.Fprintf(&out, "Affected: %d | Time: %dms\n", result.AffectedRows, result.TimeMs)
			}
		}

		fmt.Fprintf(&out, "\n=== Summary ===\n")
		fmt.Fprintf(&out, "%d queries executed, %d failed\n", output.Summary.Total, output.Summary.Failed)
	}

	// Output to file or stdout
	content := out.String()

	if opts.OutputFile != "" {
		if err := ioutil.WriteFile(opts.OutputFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Results written to %s\n", opts.OutputFile)
	} else {
		fmt.Print(content)
	}

	return nil
}

func getColumnOrder(row map[string]interface{}) []string {
	// Try to put "id" first if it exists
	columns := make([]string, 0, len(row))
	hasID := false

	for col := range row {
		if col == "id" {
			hasID = true
		} else {
			columns = append(columns, col)
		}
	}

	if hasID {
		columns = append([]string{"id"}, columns...)
	}

	return columns
}
