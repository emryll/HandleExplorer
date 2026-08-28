package main

import (
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
	pids := parsePsTargetString(tokens)
	filter.Pids = pids

	if len(pids) == 1 {
		PrintProcess(pids[0])
		return
	} else if len(pids) == 0 {
		fmt.Printf("No processes found matching \"%v\"", tokens)
		return
	}

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

//*=================================[ Search filters ]===================================

// Search wrapper for CLI find commands.
func (f SearchFilter) Search() []*AccessEntry {
	if !HandleTable.Valid() {
		HandleTable.Init()
	}

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
	defer g_ProcessTable.mu.RLock()
	var results []*Process

	//* quick lookup, used internally when only path is provided
	if len(f.Pids) > 0 {
		for _, pid := range f.Pids {
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
	//* Path / Name
	if f.Path != "" && ps.Path != f.Path &&
		f.Path != filepath.Base(ps.Path) &&
		filepath.Base(f.Path) != ps.Path {
		return false
	}
	// does not qualify if it has no dir listed
	// while the directory filter has been set
	if f.Path == filepath.Base(f.Path) && len(f.DirFilter) > 0 {
		return false
	}

	//* Directory
	var dirFound bool
	for _, dir := range f.DirFilter {
		f.Path = NormalizePath(f.Path)
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
	for _, parent := range f.Parent {
		//TODO: normalize path / pid
	}
	if len(f.Parent) > 0 && !parentFound {
		return false
	}

	//* Process elevation
	if f.Elevated && !ps.Elevated {
		return false
	}
	//* Signature status
	if f.SigStatus != 0 && f.SigStatus != ps.SigStatus {
		return false
	}

	//* Accessed objects
	accessed := GetObjectTypesAccessed(ps.ProcessId)
	for objType := range f.ObjTypes {
		if _, exists := accessed[GetTypeIdentifier(objType)]; !exists {
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
