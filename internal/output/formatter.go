package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/neitanod/navisql/internal/db"
)

// IsTerminal returns true if stdout is a terminal
func IsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Colors for terminal output
var (
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Grey   = "\033[90m"
	Normal = "\033[0m"
)

func init() {
	if !IsTerminal() {
		Green = ""
		Yellow = ""
		Grey = ""
		Normal = ""
	}
}

// PrintTabular prints query results in tabular format
func PrintTabular(result *db.QueryResult) {
	if len(result.Rows) == 0 {
		fmt.Println("No rows returned")
		return
	}

	// Print header
	fmt.Println(strings.Join(result.Columns, "\t"))

	// Print rows
	for _, row := range result.Rows {
		values := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			val := row[col]
			if val == nil {
				values[i] = "NULL"
			} else {
				values[i] = fmt.Sprintf("%v", val)
			}
		}
		fmt.Println(strings.Join(values, "\t"))
	}
}

// PrintTabularTruncated prints query results with truncation info
func PrintTabularTruncated(result *db.QueryResult, totalRows int, truncated bool) {
	PrintTabular(result)
	if truncated {
		fmt.Printf("%s... (%d more rows, use --full to see all)%s\n",
			Grey, totalRows-len(result.Rows), Normal)
	}
}

// PrintJSON prints query results as JSON
func PrintJSON(result *db.QueryResult) error {
	// Build ordered output preserving column order
	output := make([]map[string]interface{}, len(result.Rows))
	for i, row := range result.Rows {
		orderedRow := make(map[string]interface{})
		for _, col := range result.Columns {
			orderedRow[col] = row[col]
		}
		output[i] = orderedRow
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintVertical prints a single record in vertical format (like mysql \G)
func PrintVertical(result *db.QueryResult, decorators map[string]string) {
	if len(result.Rows) == 0 {
		fmt.Println("No rows returned")
		return
	}

	for _, row := range result.Rows {
		for _, col := range result.Columns {
			val := row[col]
			valStr := "NULL"
			if val != nil {
				valStr = fmt.Sprintf("%v", val)
			}

			decorator := ""
			if d, ok := decorators[col]; ok {
				decorator = " " + d
			}

			fmt.Printf("- %s%s%s: %s%s\n", Green, col, Normal, valStr, decorator)
		}
	}
}

// PrintKeyValue prints simple key-value pairs
func PrintKeyValue(data map[string]string) {
	for k, v := range data {
		fmt.Printf("%s%s%s: %s\n", Green, k, Normal, v)
	}
}

// PrintList prints a simple list
func PrintList(items []string) {
	for _, item := range items {
		fmt.Println(item)
	}
}

// PrintError prints an error message to stderr
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", Green, msg, Normal)
}
