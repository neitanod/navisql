package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Port int

func (p *Port) UnmarshalJSON(data []byte) error {
	// Si es un número
	if data[0] >= '0' && data[0] <= '9' {
		var num int
		err := json.Unmarshal(data, &num)
		if err != nil {
			return err
		}
		*p = Port(num)
		return nil
	}

	// Si es un string
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	var num int
	_, err = fmt.Sscanf(str, "%d", &num)
	if err != nil {
		return err
	}
	*p = Port(num)
	return nil
}

type Config struct {
	Connection map[string]struct {
		User string `json:"user"`
		Pass string `json:"pass"`
		Host string `json:"host"`
		Port Port   `json:"port"`
	} `json:"connection"`
}

func getConfig(connection string) (string, string, string, int, error) {
	usr, _ := user.Current()
	configPath := filepath.Join(usr.HomeDir, ".navisql", "navisql.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", "", 0, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return "", "", "", 0, err
	}

	conn, ok := cfg.Connection[connection]
	if !ok {
		return "", "", "", 0, fmt.Errorf("connection %s not found in config", connection)
	}

	return conn.User, conn.Pass, conn.Host, int(conn.Port), nil
}

func printTabular(rows *sql.Rows) error {
	cols, _ := rows.Columns()
	values := make([]sql.RawBytes, len(cols))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}
	rowNum := 1
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			return err
		}
		fmt.Printf("********** Row %d **********\n", rowNum)
		for i, col := range cols {
			fmt.Printf("%s: %s\n", col, values[i])
		}
		rowNum++
	}
	return nil
}

func printJSON(rows *sql.Rows) error {
	cols, _ := rows.Columns()
	colLen := len(cols)
	values := make([]sql.RawBytes, colLen)
	scanArgs := make([]interface{}, colLen)
	for i := range values {
		scanArgs[i] = &values[i]
	}
	var results []map[string]string
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			return err
		}
		rowMap := make(map[string]string)
		for i, col := range cols {
			rowMap[col] = string(values[i])
		}
		results = append(results, rowMap)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func main() {
	// Flags
	where := flag.String("where", "", "WHERE condition")
	perPage := flag.Int("per-page", 30, "Items per page")
	asJSON := flag.Bool("json", false, "Output as JSON")
	skipSSL := flag.Bool("skip-ssl", false, "Skip SSL certificate verification")
	flag.Parse()

	// Positional args
	args := flag.Args()
	if len(args) < 3 {
		fmt.Println("Usage: navisql_ls <connection> <db> <table> [<page>] [--where=...] [--per-page=N] [--json]")
		os.Exit(1)
	}
	connection := args[0]
	dbName := args[1]
	table := args[2]
	page := 1
	if len(args) >= 4 {
		fmt.Sscanf(args[3], "%d", &page)
	}
	if page < 1 {
		page = 1
	}
	offset := (*perPage) * (page - 1)

	// Config
	user, pass, host, port, err := getConfig(connection)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Config error:", err)
		os.Exit(1)
	}

	// DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, dbName)

	// Add TLS skip verification only if requested
	if *skipSSL {
		dsn += "&tls=skip-verify"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "DB open error:", err)
		os.Exit(1)
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT * FROM %s", table)
	if strings.TrimSpace(*where) != "" {
		query += fmt.Sprintf(" WHERE %s", *where)
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", *perPage, offset)

	rows, err := db.Query(query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Query error:", err)
		os.Exit(1)
	}
	defer rows.Close()

	if *asJSON {
		fmt.Fprintln(os.Stderr, "As JSON!")
		if err := printJSON(rows); err != nil {
			fmt.Fprintln(os.Stderr, "JSON output error:", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "As Tabular!")
		if err := printTabular(rows); err != nil {
			fmt.Fprintln(os.Stderr, "Tabular output error:", err)
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Output error:", err)
		os.Exit(1)
	}
}
