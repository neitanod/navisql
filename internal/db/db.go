package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/neitanod/navisql/internal/config"
)

// Connect creates a database connection for the given connection name and database
func Connect(connName, database string, skipSSL bool) (*sql.DB, error) {
	conn, err := config.GetConnection(connName)
	if err != nil {
		return nil, err
	}

	return ConnectWithConfig(conn, database, skipSSL)
}

// ConnectWithConfig creates a database connection using provided config
func ConnectWithConfig(conn *config.Connection, database string, skipSSL bool) (*sql.DB, error) {
	tlsConfig := ""
	if skipSSL {
		tlsConfig = "?tls=false"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s%s",
		conn.User, conn.Pass, conn.Host, conn.GetPort(), database, tlsConfig)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return db, nil
}

// ConnectNoDatabase creates a connection without selecting a database
func ConnectNoDatabase(connName string, skipSSL bool) (*sql.DB, error) {
	conn, err := config.GetConnection(connName)
	if err != nil {
		return nil, err
	}

	tlsConfig := ""
	if skipSSL {
		tlsConfig = "?tls=false"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		conn.User, conn.Pass, conn.Host, conn.GetPort(), tlsConfig)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return db, nil
}

// QueryResult holds the result of a query
type QueryResult struct {
	Columns []string
	Rows    []map[string]interface{}
}

// Query executes a SELECT query and returns the results
func Query(db *sql.DB, query string) (*QueryResult, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
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
		result.Rows = append(result.Rows, row)
	}

	return result, rows.Err()
}

// QueryWithLimit executes a SELECT query with optional row limit (for display truncation)
func QueryWithLimit(db *sql.DB, query string, maxRows int) (*QueryResult, int, bool, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, false, err
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	totalRows := 0
	truncated := false

	for rows.Next() {
		totalRows++

		// If we've hit the limit, just count remaining rows
		if maxRows > 0 && len(result.Rows) >= maxRows {
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
		result.Rows = append(result.Rows, row)
	}

	return result, totalRows, truncated, rows.Err()
}

// Exec executes a non-SELECT query (INSERT, UPDATE, DELETE)
func Exec(db *sql.DB, query string) (sql.Result, error) {
	return db.Exec(query)
}

// GetDatabases returns list of databases for a connection
func GetDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		databases = append(databases, name)
	}
	return databases, rows.Err()
}

// GetTables returns list of tables for a database
func GetTables(db *sql.DB, database string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf("SHOW TABLES FROM `%s`", database))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// GetColumns returns list of column names for a table
func GetColumns(db *sql.DB, database, table string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf("SHOW COLUMNS FROM `%s`.`%s`", database, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var field, colType, null, key, extra string
		var defaultVal interface{}
		if err := rows.Scan(&field, &colType, &null, &key, &defaultVal, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, field)
	}
	return columns, rows.Err()
}
