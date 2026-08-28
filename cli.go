package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
)

//?========================================================+
//?   This file is responsible for parsing and executing   |
//?    commands from the user in the interactive CLI.      |
//?========================================================+

//*===========================[ Command parsing ]===================================

// Main loop for simple command-line user interface.
// Returns when the user enters an exit command and call cancel.
func CommandParsingLoop(wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()

	reader := bufio.NewReader(os.Stdin)
	g := color.New(color.FgHiGreen, color.Bold)
	for {
		g.Printf(" $ ")
		command := GetInput(reader)
		if command == "" {
			continue
		}

		tokens := strings.Fields(command)
		exit := CliParseCommand(tokens)
		if exit {
			cancel() // shutdown command
			return
		}
	}
}

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
	case "find", "f":
		CliFindCommand(tokens[1:])
	case "ps", "process", "p":
		CliPsCommand(tokens[1:])
	case "clusters", "c":
		CliClustersCommand(tokens[1:])
	/*case "outliers":
	CliOutliersCommand(tokens[1:])*/
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

//*==================================[ Commands ]======================================

// Main routine for parsing and executing the
// "ps" command (search for processes / view process)
func CliPsCommand(tokens []string) {
	var filter ProcessFilter
	if len(tokens) == 0 {
		filter = PsFilterSelectionMenu()
	}
	//TODO: use parsePsTargetString
	pid, err := strconv.Atoi(tokens[0])
	if err == nil { // view process directly, single pid
		PrintProcess(uint32(pid))
		return
	}
	// get pids from path
	pids := findProcesses(tokens[0])
	if len(pids) == 0 {
		PrintError("Couldn't find any processes with \"%s\"", tokens[0])
		return
	}
	filter.Pids = pids

	results := filter.Search()
	if selected := RenderList(results); selected != nil {
		selected.Print()
	}
}

// Main routine for parsing and executing the
// "find" command (find object access with filters)
func CliFindCommand(flags []string) {
	var filter SearchFilter
	if len(flags) == 0 {
		filter = ObjFilterSelectionMenu()
	} else {
		filter = parseFindFlags(flags)
	}

	if filter.Empty() {
		fmt.Println("You must enter atleast one search filter.")
		return
	}
	entries := filter.Search()
	if selected := RenderList(entries); selected != nil {
		selected.Print()
	}
}

// Main routine for parsing and executing the
// "clusters" command (find overlapping object access)
func CliClustersCommand(flags []string) {
	filter := parseClustersFlags(flags)
	clusters := g_ObjectAccessRegistry.FindOverlapping(filter)

	if selected := RenderList(clusters); selected != nil {
		selected.Print()
	}
}

// *===============================[ Help messages ]===================================

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
