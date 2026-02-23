package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/neitanod/navisql/internal/config"
	"github.com/neitanod/navisql/internal/output"
)

const maxHistoryLines = 70

// History shows or replays command history
func History(args []string) error {
	if len(args) > 0 && args[0] == "--help" {
		printHistoryHelp()
		return nil
	}

	paths, err := config.GetPaths()
	if err != nil {
		return err
	}

	// Read history file
	entries, err := readHistory(paths.HistoryFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No history yet")
			return nil
		}
		return err
	}

	// Take last N entries
	if len(entries) > maxHistoryLines {
		entries = entries[len(entries)-maxHistoryLines:]
	}

	// If a number is provided, replay that command
	if len(args) > 0 {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid history number: %s", args[0])
		}

		// Find the entry
		for i, entry := range entries {
			lineNum := i + 1
			if len(entries) > maxHistoryLines {
				lineNum = len(entries) - maxHistoryLines + i + 1
			}

			if lineNum == num {
				// Replay the command
				fmt.Printf("%s%s%s\n", output.Grey, entry, output.Normal)
				return replayCommand(entry)
			}
		}

		return fmt.Errorf("history entry %d not found", num)
	}

	// Show history with line numbers
	startNum := 1
	if len(entries) > maxHistoryLines {
		startNum = len(entries) - maxHistoryLines + 1
	}

	for i, entry := range entries {
		fmt.Printf("%s%5d%s  %s\n", output.Grey, startNum+i, output.Normal, entry)
	}

	return nil
}

// SaveHistory saves a command to history
func SaveHistory(command string) error {
	paths, err := config.GetPaths()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(paths.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintln(file, command)
	return err
}

func readHistory(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			entries = append(entries, line)
		}
	}

	return entries, scanner.Err()
}

func replayCommand(entry string) error {
	// Entry format is "navisql <command> <args...>"
	// We need to execute it

	parts := strings.Fields(entry)
	if len(parts) < 2 {
		return fmt.Errorf("invalid history entry")
	}

	// Skip "navisql" prefix if present
	if parts[0] == "navisql" {
		parts = parts[1:]
	}

	// Find the navisql binary
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to just "navisql"
		execPath = "navisql"
	}

	cmd := exec.Command(execPath, parts...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func printHistoryHelp() {
	fmt.Println(`Usage: navisql history [number]

Arguments:
  number    If provided, replay that history entry

Description:
  Without arguments, shows the last 70 commands.
  With a number, replays that command.

Examples:
  navisql history       # Show history
  navisql history 5     # Replay command #5`)
}
