package main

import (
	"fmt"
	"strings"
)

//?========================================================+
//?   This file is responsible for parsing and executing   |
//?    commands from the user in the interactive CLI.      |
//?========================================================+

// Parse a command and respond accordingly.
// True return value indicates program should exit.
func CliParseCommand(tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	switch strings.ToLower(tokens[0]) {
	case "exit", "quit", "q":
		return true
	case "help", "?":
		CliHelpCommand(tokens[1:])
	case "info", "overview", "general", "status":
		CliOverview()
	case "find", "f":
		CliFindCommand(tokens[1:])
	case "ps", "process", "p":
		CliPsCommand(tokens[1:])
	case "clusters", "c":
		CliClustersCommand(tokens[1:])
	case "outliers":
		CliOutliersCommand(tokens[1:])
	case "bm":
		if BenchmarkRegistry == nil || len(BenchmarkRegistry) == 0 {
			fmt.Println("No benchmark samples collected.")
			return false
		}

		for _, b := range BenchmarkRegistry {
			b.mu.RLock()
			defer b.mu.RUnlock()
			if len(b.entries) == 0 {
				continue
			}
			b.PrintDistribution()
		}

	//case "export":
	default:
		fmt.Printf("Unknown command: %s\n", tokens[0])
		fmt.Println("Run \"help\" to view available commands.")
	}
	return false
}
