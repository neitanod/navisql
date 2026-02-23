package parser

import (
	"regexp"
	"strings"
)

// SplitQueries splits SQL content by semicolons, respecting strings and comments
func SplitQueries(content string) []string {
	var queries []string
	var current strings.Builder

	i := 0
	n := len(content)

	for i < n {
		ch := content[i]

		// Check for single-line comment
		if ch == '-' && i+1 < n && content[i+1] == '-' {
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
					// Check for escaped quote (doubled)
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

// IsSelectQuery checks if a query is a SELECT-type query (returns rows)
func IsSelectQuery(query string) bool {
	q := strings.TrimSpace(query)

	// Strip leading comments
	for {
		if strings.HasPrefix(q, "--") {
			if idx := strings.Index(q, "\n"); idx != -1 {
				q = strings.TrimSpace(q[idx+1:])
				continue
			}
			return false
		}
		if strings.HasPrefix(q, "/*") {
			if idx := strings.Index(q, "*/"); idx != -1 {
				q = strings.TrimSpace(q[idx+2:])
				continue
			}
			return false
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

// SubstituteVariables replaces {{var}} placeholders with values
func SubstituteVariables(content string, vars map[string]string, allowMissing bool, emptyIfMissing bool) (string, []string, error) {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	missing := []string{}
	seen := make(map[string]bool)

	for _, match := range matches {
		varName := match[1]
		if seen[varName] {
			continue
		}
		seen[varName] = true

		if _, ok := vars[varName]; !ok {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		if emptyIfMissing {
			for _, v := range missing {
				vars[v] = ""
			}
		} else if !allowMissing {
			return "", missing, nil
		}
	}

	// Replace variables that we have
	result := content
	for varName, value := range vars {
		result = strings.ReplaceAll(result, "{{"+varName+"}}", value)
	}

	return result, missing, nil
}
