# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NaviSQL is a MySQL navigation utility written in Go. It provides an interactive command-line experience for exploring MySQL databases with tab-completion for connections, databases, tables, and fields.

Key features:
- Single unified Go binary for all commands
- Auto-completion for connection names, database names, table names, and column names
- FK reference navigation using the "navi" command system (separate tool)
- Persistent command history
- Web edit link integration (e.g., Adminer)

## Requirements

- Go 1.21+ (for building)
- `bash` or `zsh` shell (for auto-completion)
- MySQL server (for database operations)

## Architecture

NaviSQL is a single Go binary with a monorepo structure:

```
navisql/
├── go.mod                    # Go module definition
├── cmd/navisql/main.go       # Entry point and command router
├── internal/
│   ├── commands/             # Command implementations
│   │   ├── ls.go
│   │   ├── show.go
│   │   ├── query.go
│   │   ├── run.go
│   │   ├── connection.go
│   │   ├── config.go
│   │   ├── cache.go
│   │   ├── fk.go
│   │   └── history.go
│   ├── config/config.go      # Configuration loading/saving
│   ├── db/db.go              # Database connection and queries
│   ├── output/formatter.go   # Output formatting (tabular, JSON)
│   └── parser/sql.go         # SQL parsing (query splitting)
├── navisql                   # Compiled binary
├── navisql_autocomplete      # Bash/zsh completion script
├── navi                      # Navigation script (separate tool)
└── legacy/                   # Old bash scripts (for reference)
```

### Configuration and Data Storage

All configuration is stored in `~/.navisql/`:
- `navisql.json` - Connection credentials and global config
- `navisql_cache.json` - Cached database and table names for auto-completion
- `fk.csv` - Foreign key relationships
- `history` - Command history

Navigation links are stored in `~/.navi/links` (used by the separate `navi` tool).

## Development

### Building

```bash
# Always use the build script (injects version timestamp)
./build

# Manual "go build" will produce a binary that refuses to run
```

### Adding a New Command

1. Create `internal/commands/<command>.go`
2. Implement the command function with signature `func CommandName(args []string) error`
3. Add the command to the router in `cmd/navisql/main.go`
4. Add auto-completion config in `navisql_autocomplete` if needed

### Auto-completion System

Defined in `navisql_autocomplete` using a configuration array:
- `_navi_cmd_config[<command>,<position>]="<function_to_get_options>"`
- Functions can reference previous arguments: `_navi_get_tables \$2 \$3`
- Must be sourced (not executed) to load into shell

### Testing

```bash
# Test commands directly
./navisql connection list
./navisql ls <connection> <database> <table> --skip-ssl
./navisql show <connection> <database> <table> <id> --skip-ssl
./navisql query <connection> <database> "SELECT * FROM users LIMIT 5" --skip-ssl

# View command history
./navisql history
```

### Installation

```bash
# Build the binary
./build

# Install shell integration (adds to PATH and loads autocomplete)
./install_bash   # or ./install_zsh
source ~/.bashrc # or source ~/.zshrc
```

## Commands

| Command | Description |
|---------|-------------|
| `ls <conn> <db> <table>` | List paginated records |
| `show <conn> <db> <table> <id>` | Show single record with FK links |
| `query <conn> <db> "<sql>"` | Execute arbitrary SQL |
| `run <conn> <db> <file.sql>` | Execute SQL file |
| `connection add/list/remove` | Manage connections |
| `config add/remove` | Manage configuration |
| `cache build <conn>` | Build auto-completion cache |
| `fk add/export` | Manage foreign key references |
| `history [N]` | Show/replay command history |

All data commands support `--skip-ssl` and `--json` flags.

## Code Conventions

- Go code uses standard formatting (gofmt)
- Commands return `error` for failures
- Output uses `internal/output` for consistent formatting
- Colors are only used when stdout is a terminal
- Configuration is accessed through `internal/config`
