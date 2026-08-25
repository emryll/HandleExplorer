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

func CliHelpCommand(tokens []string) {
	if len(tokens) > 0 && tokens[0] == "help" {
		tokens = tokens[1:]
	}
	if len(tokens) == 0 {
		PrintBasicHelp()
		return
	}
	switch tokens[1] {
	case "find":
		fmt.Println("\tfind - Search for handles with filters.")
		fmt.Println("Usage: find [flags]")
		fmt.Println("\t-p <ps_1,ps_2,...,ps_n>          Set a filter of processes (pid or name).")
		fmt.Println("\t-o <obj_1,obj_2,...,obj_n>       Set a filter of object types.")
		fmt.Println("\t-n <name_1,name_2,...,name_n>    Set a filter of object names.")
		fmt.Println("\n\tAll of the flags allow comma-separated lists of entries.")
		fmt.Println("\tIt is suggested to simply run \"find\" as this will open")
		fmt.Println("\t a graphical (TUI) menu for selecting search filters.")
		fmt.Println()
	case "ps", "process", "p":
		fmt.Println("\tps - Search for processes with filters.")
		fmt.Println("Usage: ps [pid | path]")
		fmt.Println("\tSimply running \"ps\" will open up a graphical")
		fmt.Println("\t (TUI) menu for selecting process search filters.")
		fmt.Println()
	case "clusters", "c":
		fmt.Println("\tclusters - Find overlapping access to Windows objects.")
		fmt.Println("Usage: clusters [flags]")
		fmt.Println("\t-o <type>         Filter for a certain object type (NT names)")
		fmt.Println("\t-p <pid>          Find overlapping access with specific process.")
		fmt.Println("\t-m <min>          Minimum size of clusters.")
		fmt.Println()
	}
}

func PrintBasicHelp() {
	fmt.Println("\t\"obj\" is an universal alias for \"object\".")
	fmt.Println("\t\"ps\" is an universal alias for \"process\".\n")
	fmt.Println("help [command]              Get help on available commands.")
	fmt.Println("overview                    View the current status of data and generic analysis.")
	fmt.Println("ps  [pid | path]            View a process, or search for processes with filters.")
	fmt.Println("find  [flags]               Search for handles with filters. No args opens up a menu.")
	fmt.Println("clusters [flags]            Find processes with overlapping object access.")
	fmt.Println("outliers                    Find statistical outliers.")
	fmt.Println("exit | quit | q             Exit the program.")
}
