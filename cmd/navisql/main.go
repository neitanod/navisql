package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/neitanod/navisql/internal/commands"
	"github.com/neitanod/navisql/internal/config"
	"github.com/neitanod/navisql/internal/output"
)

// version is set via ldflags at build time
// Build with: ./build
var version = "not-built-correctly"

func main() {
	// Check version was injected at build time
	if version == "not-built-correctly" {
		fmt.Fprintln(os.Stderr, "Error: This binary was not built correctly.")
		fmt.Fprintln(os.Stderr, "Please build with: ./build")
		os.Exit(1)
	}

	// Ensure config directory exists
	if err := config.EnsureDir(); err != nil {
		output.PrintError("failed to initialize config: %v", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	// Handle command routing
	cmd := os.Args[1]
	args := os.Args[2:]

	// Normalize command (support both "connection-add" and "connection add")
	if len(args) > 0 {
		combined := cmd + "-" + args[0]
		if isValidCommand(combined) {
			cmd = combined
			args = args[1:]
		}
	}

	var err error
	saveHistory := false

	switch cmd {
	case "ls":
		err = commands.Ls(args)
		saveHistory = true
	case "show":
		err = commands.Show(args)
		saveHistory = true
	case "query":
		err = commands.Query(args)
		saveHistory = true
	case "run":
		err = commands.Run(args)
		saveHistory = true
	case "config":
		if len(args) > 0 {
			switch args[0] {
			case "add":
				err = commands.ConfigAdd(args[1:])
			case "remove":
				err = commands.ConfigRemove(args[1:])
			default:
				err = fmt.Errorf("unknown config subcommand: %s", args[0])
			}
		} else {
			err = fmt.Errorf("config requires a subcommand: add or remove")
		}
	case "connection", "connection-add", "connection-list", "connection-remove":
		err = handleConnectionCommand(cmd, args)
	case "cache-build", "cache":
		err = handleCacheCommand(cmd, args)
	case "fk-add", "fk-export", "fk":
		err = handleFKCommand(cmd, args)
	case "history":
		err = commands.History(args)
	case "version", "--version", "-v":
		fmt.Printf("navisql version %s\n", version)
	case "help", "--help", "-h":
		printHelp()
	default:
		output.PrintError("unknown command: %s", cmd)
		printHelp()
		os.Exit(1)
	}

	if err != nil {
		output.PrintError("%v", err)
		os.Exit(1)
	}

	// Save command to history
	if saveHistory && err == nil {
		historyEntry := "navisql " + strings.Join(os.Args[1:], " ")
		_ = commands.SaveHistory(historyEntry)
	}
}

func handleConnectionCommand(cmd string, args []string) error {
	// Handle both "connection-add" and "connection add" styles
	if cmd == "connection" && len(args) > 0 {
		switch args[0] {
		case "add":
			return commands.ConnectionAdd(args[1:])
		case "list":
			return commands.ConnectionList(args[1:])
		case "remove":
			return commands.ConnectionRemove(args[1:])
		default:
			return fmt.Errorf("unknown connection subcommand: %s", args[0])
		}
	}

	switch cmd {
	case "connection-add":
		return commands.ConnectionAdd(args)
	case "connection-list":
		return commands.ConnectionList(args)
	case "connection-remove":
		return commands.ConnectionRemove(args)
	}
	return fmt.Errorf("connection requires a subcommand: add, list, or remove")
}

func handleCacheCommand(cmd string, args []string) error {
	if cmd == "cache" && len(args) > 0 {
		switch args[0] {
		case "build":
			return commands.CacheBuild(args[1:])
		default:
			return fmt.Errorf("unknown cache subcommand: %s", args[0])
		}
	}
	if cmd == "cache-build" {
		return commands.CacheBuild(args)
	}
	return fmt.Errorf("cache requires a subcommand: build")
}

func handleFKCommand(cmd string, args []string) error {
	if cmd == "fk" && len(args) > 0 {
		switch args[0] {
		case "add":
			return commands.FKAdd(args[1:])
		case "export":
			return commands.FKExport(args[1:])
		default:
			return fmt.Errorf("unknown fk subcommand: %s", args[0])
		}
	}

	switch cmd {
	case "fk-add":
		return commands.FKAdd(args)
	case "fk-export":
		return commands.FKExport(args)
	}
	return fmt.Errorf("fk requires a subcommand: add or export")
}

func isValidCommand(cmd string) bool {
	validCommands := []string{
		"connection-add", "connection-list", "connection-remove",
		"cache-build", "fk-add", "fk-export",
	}
	for _, valid := range validCommands {
		if cmd == valid {
			return true
		}
	}
	return false
}

func printHelp() {
	green := output.Green
	yellow := output.Yellow
	normal := output.Normal

	help := fmt.Sprintf(`Usage: navisql <command> [arguments]

Commands:
  %sls%s                %sPrints a page of records from a table%s
  %sshow%s              %sShow a record by id%s
  %squery%s             %sRun a custom query%s
  %srun%s               %sExecute queries from a SQL file%s
  %sconnection add%s    %sAdd a new connection%s
  %sconnection list%s   %sList all connections%s
  %sconnection remove%s %sRemove a connection%s
  %scache build%s       %sBuild cache for databases and tables%s
  %sfk add%s            %sAdd a foreign key reference%s
  %sfk export%s         %sExport FK configuration as commands%s
  %sconfig add%s        %sAdd a configuration value%s
  %sconfig remove%s     %sRemove a configuration value%s
  %shistory%s           %sShow/replay command history%s

Use "navisql <command> --help" for more information about a command.
`,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
		green, normal, yellow, normal,
	)

	// Also support hyphenated versions
	help += fmt.Sprintf(`
Aliases (for compatibility):
  connection-add, connection-list, connection-remove
  cache-build, fk-add, fk-export
`)

	fmt.Print(strings.TrimSpace(help) + "\n")
}
