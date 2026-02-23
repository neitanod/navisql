package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

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

	// Print column headers
	for i, col := range cols {
		if i > 0 {
			fmt.Print("\t")
		}
		fmt.Print(col)
	}
	fmt.Println()

	// Print rows
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			return err
		}
		for i, val := range values {
			if i > 0 {
				fmt.Print("\t")
			}
			if val == nil {
				fmt.Print("NULL")
			} else {
				fmt.Print(string(val))
			}
		}
		fmt.Println()
	}
	return nil
}

// OrderedMap mantiene el orden de las columnas con "id" primero
type OrderedMap struct {
	keys   []string
	values map[string]string
}

func (om *OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	for i, key := range om.keys {
		if i > 0 {
			buf.WriteString(",")
		}
		// Marshal key
		keyBytes, _ := json.Marshal(key)
		buf.Write(keyBytes)
		buf.WriteString(":")
		// Marshal value
		valBytes, _ := json.Marshal(om.values[key])
		buf.Write(valBytes)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

func printJSON(rows *sql.Rows) error {
	cols, _ := rows.Columns()
	colLen := len(cols)
	values := make([]sql.RawBytes, colLen)
	scanArgs := make([]interface{}, colLen)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Detectar si existe columna "id" y su índice
	idIndex := -1
	for i, col := range cols {
		if col == "id" {
			idIndex = i
			break
		}
	}

	var results []*OrderedMap
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			return err
		}

		om := &OrderedMap{
			keys:   make([]string, 0, colLen),
			values: make(map[string]string),
		}

		// Si existe "id", agregarlo primero
		if idIndex >= 0 {
			om.keys = append(om.keys, "id")
			if values[idIndex] == nil {
				om.values["id"] = "NULL"
			} else {
				om.values["id"] = string(values[idIndex])
			}
		}

		// Agregar las demás columnas en orden
		for i, col := range cols {
			if i == idIndex {
				continue // Ya agregamos "id"
			}
			om.keys = append(om.keys, col)
			if values[i] == nil {
				om.values[col] = "NULL"
			} else {
				om.values[col] = string(values[i])
			}
		}

		results = append(results, om)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}

func main() {
	// Process arguments manually to support --json and --skip-ssl at any position
	var asJSON bool
	var skipSSL bool
	var positionalArgs []string

	for _, arg := range os.Args[1:] {
		if arg == "--json" {
			asJSON = true
		} else if arg == "--skip-ssl" {
			skipSSL = true
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if len(positionalArgs) < 3 {
		fmt.Println("Usage: navisql_query <connection> <db> <query> [--json] [--skip-ssl]")
		os.Exit(1)
	}
	connection := positionalArgs[0]
	dbName := positionalArgs[1]
	query := positionalArgs[2]

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
	if skipSSL {
		dsn += "&tls=skip-verify"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "DB open error:", err)
		os.Exit(1)
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Query error:", err)
		os.Exit(1)
	}
	defer rows.Close()

	if asJSON {
		if err := printJSON(rows); err != nil {
			fmt.Fprintln(os.Stderr, "JSON output error:", err)
			os.Exit(1)
		}
	} else {
		if err := printTabular(rows); err != nil {
			fmt.Fprintln(os.Stderr, "Tabular output error:", err)
			os.Exit(1)
		}
	}
}
