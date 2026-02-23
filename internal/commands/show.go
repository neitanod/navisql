package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/neitanod/navisql/internal/config"
	"github.com/neitanod/navisql/internal/db"
	"github.com/neitanod/navisql/internal/output"
)

// Show displays a single record with FK references and navi links
func Show(args []string) error {
	opts := &showOptions{}

	positional := []string{}
	for _, arg := range args {
		if arg == "--skip-ssl" {
			opts.skipSSL = true
		} else if arg == "--help" || arg == "-h" {
			printShowHelp()
			return nil
		} else if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
		} else {
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	if len(positional) < 4 {
		printShowHelp()
		return fmt.Errorf("missing required arguments")
	}

	opts.connection = positional[0]
	opts.database = positional[1]
	opts.table = positional[2]
	opts.id = positional[3]
	opts.idField = "id"

	if len(positional) > 4 {
		opts.idField = positional[4]
	}

	return runShow(opts)
}

type showOptions struct {
	connection string
	database   string
	table      string
	id         string
	idField    string
	skipSSL    bool
}

func runShow(opts *showOptions) error {
	conn, err := db.Connect(opts.connection, opts.database, opts.skipSSL)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Query the record
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` = '%s' LIMIT 1",
		opts.table, opts.idField, opts.id)

	result, err := db.Query(conn, query)
	if err != nil {
		return err
	}

	if len(result.Rows) == 0 {
		return fmt.Errorf("record not found")
	}

	// Load foreign keys for this table
	fks, err := LoadForeignKeysForTable(opts.connection, opts.database, opts.table)
	if err != nil {
		fks = nil // Continue without FKs
	}

	// Build FK lookup map
	fkMap := make(map[string]ForeignKey)
	for _, fk := range fks {
		fkMap[fk.Field] = fk
	}

	// Initialize link generator
	linkGen, err := newLinkGenerator()
	if err != nil {
		return err
	}

	// Print the record
	row := result.Rows[0]
	for _, col := range result.Columns {
		val := row[col]
		valStr := "NULL"
		if val != nil {
			valStr = fmt.Sprintf("%v", val)
		}

		// Check if this field has a FK reference
		decorator := ""
		if fk, ok := fkMap[col]; ok && val != nil {
			// Generate navi link
			cmd := fmt.Sprintf("navisql show %s %s %s %v %s",
				opts.connection, fk.RefDatabase, fk.RefTable, val, fk.RefField)
			linkKey := linkGen.AddLink(cmd)
			decorator = fmt.Sprintf(" %s[navi %s]%s", output.Yellow, linkKey, output.Normal)
		}

		fmt.Printf("- %s%s%s: %s%s\n", output.Green, col, output.Normal, valStr, decorator)
	}

	// Add web edit link if configured
	cfg, err := config.Load()
	if err == nil && cfg.WebEdit != "" {
		// Get connection details for the URL
		connConfig, err := config.GetConnection(opts.connection)
		if err == nil {
			webURL := cfg.WebEdit
			webURL = strings.ReplaceAll(webURL, "{{SERVER}}", connConfig.Host)
			webURL = strings.ReplaceAll(webURL, "{{USER}}", connConfig.User)
			webURL = strings.ReplaceAll(webURL, "{{DB}}", opts.database)
			webURL = strings.ReplaceAll(webURL, "{{TABLE}}", opts.table)
			webURL = strings.ReplaceAll(webURL, "{{ID}}", opts.id)

			linkKey := linkGen.AddLink(webURL)
			fmt.Printf("  [ Web edit: %s[navi %s]%s %s ]\n",
				output.Yellow, linkKey, output.Normal, webURL)
		}
	}

	// Save links
	if err := linkGen.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save navi links: %v\n", err)
	}

	return nil
}

// linkGenerator manages navi link creation
type linkGenerator struct {
	links    []string
	keyIndex int
}

func newLinkGenerator() (*linkGenerator, error) {
	// Ensure navi directory exists
	if err := config.EnsureNaviDir(); err != nil {
		return nil, err
	}

	return &linkGenerator{
		links:    make([]string, 0),
		keyIndex: 0,
	}, nil
}

func (lg *linkGenerator) AddLink(cmdOrURL string) string {
	key := lg.nextKey()
	lg.links = append(lg.links, fmt.Sprintf("navi %s|%s", key, cmdOrURL))
	return key
}

func (lg *linkGenerator) nextKey() string {
	lg.keyIndex++
	if lg.keyIndex <= 9 {
		return fmt.Sprintf("%d", lg.keyIndex)
	}
	// After 9, use letters a-z
	letterIndex := lg.keyIndex - 10
	if letterIndex < 26 {
		return string(rune('a' + letterIndex))
	}
	// Fallback to numbers again
	return fmt.Sprintf("%d", lg.keyIndex)
}

func (lg *linkGenerator) Save() error {
	if len(lg.links) == 0 {
		return nil
	}

	linksPath, err := config.GetNaviLinksPath()
	if err != nil {
		return err
	}

	content := strings.Join(lg.links, "\n") + "\n"
	return os.WriteFile(linksPath, []byte(content), 0644)
}

func printShowHelp() {
	fmt.Println(`Usage: navisql show <connection> <database> <table> <id> [id_field] [options]

Arguments:
  connection   Connection name
  database     Database name
  table        Table name
  id           Value to search for
  id_field     Field to search by (default: id)

Options:
  --skip-ssl   Skip SSL verification

Description:
  Displays a single record with all its fields.
  If foreign keys are configured, shows [navi N] links to referenced records.
  If web_edit is configured, shows a link to edit the record in a web interface.

Examples:
  navisql show local mydb users 1
  navisql show local mydb users john username`)
}
