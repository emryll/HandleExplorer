package main

import (
	tlist "HandleExplorer/tui-list"
	tmenu "HandleExplorer/tui-menu"
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			HandleTable.CleanupIfNeeded()
		}(wg)

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

		wg.Add(1)
		// start refresh just in-case
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if !HandleTable.Valid() {
				HandleTable.Init()
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
		if len(BenchmarkRegistry) == 0 {
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
		tfilter := tmenu.PsFilterSelectionMenu()
		// convert imported type to native
		filter = ConvertProcessFilter(*tfilter)
	} else {
		filter.Pids = make(map[uint32]bool)
		pids := parsePsTargetString(tokens[0])
		for _, pid := range pids {
			filter.Pids[pid] = true
		}
	}

	if len(filter.Pids) == 1 {
		for ps := range filter.Pids {
			PrintProcess(nil, ps)
		}
		return
	} else if len(tokens) > 0 && len(filter.Pids) == 0 {
		fmt.Printf("No processes found matching \"%v\"", tokens)
		return
	}

	results := filter.Search()
	if len(results) == 0 {
		fmt.Println("Search returned no results.")
	}
	if selected := tlist.RenderList(results); selected != nil {
		selected.Print(nil)
	}
}

// Main routine for parsing and executing the
// "find" command (find object access with filters)
func CliFindCommand(flags []string) {
	var filter SearchFilter
	if len(flags) == 0 {
		tfilter := tmenu.ObjFilterSelectionMenu()
		// convert imported type to native
		filter = ConvertHandleFilter(*tfilter)
	} else {
		filter = parseFindFlags(flags)
	}

	if filter.Empty() {
		fmt.Println("You must enter atleast one search filter.")
		return
	}

	entries := filter.Search()
	if len(entries) == 0 {
		fmt.Println("Search returned no results.")
		return
	}

	if selected := tlist.RenderList(entries); selected != nil {
		selected.Print(nil)
	}
}

// Main routine for parsing and executing the
// "clusters" command (find overlapping object access)
func CliClustersCommand(flags []string) {
	HandleTable.WaitReady()

	filter := parseClustersFlags(flags)
	clusters, stats := g_ObjectAccessRegistry.FindOverlapping(&filter)

	if selected := tlist.RenderList(clusters); selected != nil {
		selected.Print(nil)
		stats.Print(nil)
	}
}

// Main routine for "overview" command.
func CliOverviewCommand() {
	HandleTable.WaitReady()

	fmt.Printf("handle count: ")
	yellow.Printf("%d\n", GetTotalHandleCount())
	PrintGlobalObjTypeDistribution()

	fmt.Printf("process count: ")
	yellow.Printf("%d\n", GetTotalProcessCount())
	rankedProcesses := RankProcessHandleCount()
	for i := 0; i < 5; i++ {
		if i >= len(rankedProcesses) {
			break
		}
		ps := rankedProcesses[i]
		name := filepath.Base(ps.Path)
		fmt.Printf("\t- [%d handles] ", ps.GetHandleCount())
		yellow.Printf("PID %d", ps.ProcessId)
		fmt.Printf("(%s)\n", orUnknown(name))
	}

	//TODO most wide-reaching

	clusters, clusterStats := g_ObjectAccessRegistry.FindOverlapping(nil)
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
			GetTypeName(clusters[i].ObjType), clusters[i].ObjName)
	}
	if len(clusters) > 5 {
		fmt.Printf("\t(and %d others)\n", len(clusters)-5)
	}
	fmt.Println()
	clusterStats.Print(nil)
}

//*=================================[ Search filters ]===================================

// Search wrapper for CLI find commands.
func (f SearchFilter) Search() []*AccessEntry {
	if !HandleTable.Valid() {
		HandleTable.Init()
	}
	HandleTable.WaitReady()

	g_ObjectAccessRegistry.mu.RLock()
	defer g_ObjectAccessRegistry.mu.RUnlock()

	if len(f.Pids) != 0 {
		return g_ObjectAccessRegistry.FindByProcess(f.Pids, f.ObjType, f.Names, f.Access)
	} else if len(f.ObjType) != 0 {
		return g_ObjectAccessRegistry.FindByObject(f.ObjType, f.Access, f.Names...)
	} else {
		return nil
	}
}

func (f SearchFilter) Empty() bool {
	if len(f.ObjType) == 0 && len(f.Pids) == 0 && len(f.Names) == 0 {
		return true
	}
	return false
}

func (f ProcessFilter) Search() []*Process {
	g_ProcessTable.mu.RLock()
	defer g_ProcessTable.mu.RUnlock()
	var results []*Process

	//* quick lookup, used only when pid is provided
	if len(f.Pids) > 0 {
		for pid := range f.Pids {
			if ps, exists := g_ProcessTable.Table[pid]; exists && f.Passes(ps) {
				results = append(results, ps)
			}
		}
		return results
	}
	//* regular lookup
	for _, ps := range g_ProcessTable.Table {
		if f.Passes(ps) {
			results = append(results, ps)
		}
	}
	return results
}

func (f ProcessFilter) Passes(ps *Process) bool {
	//* Process Id
	if len(f.Pids) > 0 && !f.Pids[ps.ProcessId] {
		return false
	}

	//* Path / Name
	if f.Path != "" && ps.Path != f.Path &&
		f.Path != filepath.Base(ps.Path) &&
		filepath.Base(f.Path) != ps.Path {
		return false
	}

	//* Directory
	// does not qualify if it has no dir listed
	// while the directory filter has been set
	if f.Path == filepath.Base(f.Path) && len(f.DirFilter) > 0 {
		return false
	}

	var dirFound bool
	for _, dir := range f.DirFilter {
		if strings.HasPrefix(f.Path, dir) { // allow subdirs
			dirFound = true
			break
		}
	}
	if len(f.DirFilter) > 0 && !dirFound {
		return false
	}

	//* Parent
	var parentFound bool
	if f.Parent[ps.ParentPath] || f.Parent[filepath.Base(ps.ParentPath)] {
		parentFound = true
	}

	for parent := range f.Parent {
		if parent == "" {
			continue
		}
		pid, err := strconv.Atoi(parent)
		if err == nil && uint32(pid) == ps.ParentPid {
			parentFound = true
			break
		}
	}
	if len(f.Parent) > 0 && !parentFound {
		return false
	}

	//* Process elevation
	if f.Elevated && !ps.Elevated {
		return false
	}
	if f.NotElevated && ps.Elevated {
		return false
	}

	//* Signature status
	if len(f.SigStatus) > 0 && !f.SigStatus[ps.SigStatus] {
		return false
	}

	//* Accessed objects
	accessed := GetObjectTypesAccessed(ps.ProcessId)
	for objType := range f.ObjTypes {
		if !accessed[objType] {
			return false
		}
	}
	return true
}

//*===============================[ Flag parsing ]=====================================

// Parse command-line flags for the "find" command.
func parseFindFlags(flags []string) SearchFilter {
	var filter SearchFilter
	fs := flag.NewFlagSet("find", flag.ExitOnError)

	var (
		rawTargets string
		rawObjType string
		rawAccess  string
		rawNames   string
	)

	fs.StringVar(&rawTargets, "p", "", "process filter")
	fs.StringVar(&rawObjType, "o", "", "object type filter")
	fs.StringVar(&rawNames, "n", "", "object name filter")
	fs.StringVar(&rawAccess, "a", "", "handle access filter")
	fs.Parse(flags)

	filter.Pids = parsePsTargetString(rawTargets)
	filter.Access = parseAccessString(rawAccess)

	for _, objType := range strings.Split(rawObjType, ",") {
		objType = strings.TrimSpace(objType)
		if enum := GetTypeIdentifier(objType); enum != OBJ_TYPE_UNKNOWN {
			filter.ObjType = append(filter.ObjType, enum)
		}
	}

	for _, name := range strings.Split(rawNames, ",") {
		name = strings.TrimSpace(name)
		filter.Names = append(filter.Names, name)
	}

	return filter
}

func parsePsTargetString(targets string) []uint32 {
	var pids []uint32
	tokens := strings.Split(targets, ",")

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if pid, err := strconv.Atoi(token); err == nil {
			pids = append(pids, uint32(pid))
		} else {
			pids = append(pids, findProcesses(token)...)
		}
	}
	return pids
}

func parseClustersFlags(flags []string) ClusterFilter {
	var (
		filter      ClusterFilter
		objTypeName string
	)
	cf := flag.NewFlagSet("clusters", flag.ExitOnError)

	cf.IntVar(&filter.MinSize, "m", 0, "minimum cluster size")
	cf.StringVar(&filter.ObjName, "n", "", "object name")
	cf.StringVar(&objTypeName, "o", "", "object type")

	cf.Parse(flags)
	filter.ObjType = GetTypeIdentifier(objTypeName)
	return filter
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
