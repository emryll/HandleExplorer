package cli

import (
	"HandleExplorer/handles"
	"HandleExplorer/nt"
	"HandleExplorer/process"
	"HandleExplorer/profiler"
	"HandleExplorer/stats"
	"HandleExplorer/store"
	tlist "HandleExplorer/tui/list"
	tmenu "HandleExplorer/tui/menu"
	"HandleExplorer/utils"
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			store.AccessTracker.HandleTable.CleanupIfNeeded()
		}(wg)

		g.Printf(" $ ")
		command := utils.GetInput(reader)
		if command == "" {
			continue
		}

		tokens := strings.Fields(command)
		exit := CliParseCommand(tokens)
		if exit {
			cancel() // shutdown command
			return
		}

		wg.Add(1)
		// start refresh just in-case
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if !store.AccessTracker.HandleTable.Valid() {
				store.AccessTracker.HandleTable.Init()
			}
		}(wg)
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
	case "overview", "info":
		CliOverviewCommand()
	/*case "outliers":
	CliOutliersCommand(tokens[1:])*/
	case "bm":
		if len(profiler.BenchmarkRegistry) == 0 {
			fmt.Println("No benchmark samples collected.")
			return false
		}

		for _, b := range profiler.BenchmarkRegistry {
			b.RLock()
			defer b.RUnlock()
			if len(b.Entries) == 0 {
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
	var filter = &process.ProcessFilter{}
	if len(tokens) == 0 {
		filter = tmenu.PsFilterSelectionMenu()
	} else {
		filter.Pids = make(map[uint32]bool)
		pids := parsePsTargetString(tokens[0])
		for _, pid := range pids {
			filter.Pids[pid] = true
		}
	}

	// PsFilterSelectionMenu may return nil...
	if filter == nil { // avoid nil pointer panic
		filter = &process.ProcessFilter{}
	}

	if len(filter.Pids) == 1 {
		for ps := range filter.Pids {
			PrintProcessByPid(nil, ps)
		}
		return
	} else if len(tokens) > 0 && len(filter.Pids) == 0 {
		fmt.Printf("No processes found matching \"%v\"", tokens)
		return
	}

	results := store.PsTable.Search(*filter)
	if len(results) == 0 {
		fmt.Println("Search returned no results.")
	}
	if selected := tlist.RenderList(results); selected != nil {
		PrintItem(nil, selected)
	}
}

// Main routine for parsing and executing the
// "find" command (find object access with filters)
func CliFindCommand(flags []string) {
	var filter *handles.HandleFilter
	if len(flags) == 0 {
		filter = tmenu.ObjFilterSelectionMenu()
	} else {
		filter = parseFindFlags(flags)
	}

	// ObjFilterSelectionMenu may return nil...
	if filter == nil { // avoid nil pointer panic
		filter = &handles.HandleFilter{}
	}

	if filter.Empty() {
		fmt.Println("You must enter atleast one search filter.")
		return
	}

	entries := store.AccessTracker.Search(*filter)
	if len(entries) == 0 {
		fmt.Println("Search returned no results.")
		return
	}

	if selected := tlist.RenderList(entries); selected != nil {
		PrintItem(nil, selected)
	}
}

// Main routine for parsing and executing the
// "clusters" command (find overlapping object access)
func CliClustersCommand(flags []string) {
	store.AccessTracker.HandleTable.WaitReady()

	filter := parseClustersFlags(flags)
	clusters, stats := store.AccessTracker.AccessRegistry.
		FindOverlapping(&filter, store.PsTable.LookupProcessPath)

	if selected := tlist.RenderList(clusters); selected != nil {
		PrintItem(nil, selected)
		PrintClusterStats(nil, &stats)
	}
}

// Main routine for "overview" command.
func CliOverviewCommand() {
	store.AccessTracker.HandleTable.WaitReady()

	yellow := color.New(color.FgHiYellow)

	fmt.Printf("handle count: ")
	yellow.Printf("%d\n", stats.GetTotalHandleCount(store.RuntimeStats))
	PrintGlobalObjTypeDistribution()

	fmt.Printf("process count: ")
	yellow.Printf("%d\n", store.PsTable.GetTotalProcessCount())
	rankedProcesses := store.PsTable.RankProcessHandleCount()
	for i := 0; i < 5; i++ {
		if i >= len(rankedProcesses) {
			break
		}
		ps := rankedProcesses[i]
		name := filepath.Base(ps.Path)
		fmt.Printf("\t- [%d handles] ", ps.GetHandleCount())
		yellow.Printf("PID %d", ps.ProcessId)
		fmt.Printf("(%s)\n", utils.OrUnknown(name))
	}

	//TODO most wide-reaching

	clusters, clusterStats := store.AccessTracker.AccessRegistry.
		FindOverlapping(nil, store.PsTable.LookupProcessPath)
	fmt.Println("\nobjects with overlapping access:")
	if len(clusters) == 0 {
		fmt.Printf("\tNone.\n")
	}
	// clusters list is already sorted by size
	for i := 0; i < 5; i++ {
		if i >= len(clusters) {
			break
		}
		fmt.Printf("\t- [%d] %s : %s\n", len(clusters[i].Members),
			nt.GetTypeName(clusters[i].ObjType), clusters[i].ObjName)
	}
	if len(clusters) > 5 {
		fmt.Printf("\t(and %d others)\n", len(clusters)-5)
	}
	fmt.Println()
	PrintClusterStats(nil, &clusterStats)
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
	fmt.Println("help [command]              Get help on available commands.")
	fmt.Println("overview | info             View the current status of data and generic analysis.")
	fmt.Println("ps  [pid | path]            View data about a process. No args opens up filter selection.")
	fmt.Println("find  [flags]               Search for handles with filters. No args opens up a menu.")
	fmt.Println("clusters [flags]            Find processes with overlapping object access.")
	//fmt.Println("outliers                    Find statistical outliers.")
	fmt.Println("exit | quit | q             Exit the program.")
}
