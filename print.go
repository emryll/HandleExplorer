package main

import (
	"fmt"
	"io"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
)

//?===================================================================================+
//?  This file is responsible for non-TUI prints for the user, such as histograms.    |
//?===================================================================================+

func PrintProcess(pid uint32) {
	ps := g_ProcessTable.LookupProcess(pid)
	fmt.Printf("process %d\n", pid)
	fmt.Printf("path: %s\n", ps.Path)

	if ps.ParentPath != "" {
		fmt.Printf("parent: %s (PID %d)\n", ps.ParentPath, ps.ParentPid)
	} else {
		fmt.Printf("parent: %d\n", ps.ParentPath)
	}

	//* handles
	var totalHandles int
	handlesByType := HandleTable.getPsHandleCountsByType(pid)
	for _, count := range handlesByType {
		totalHandles += count
	}
	fmt.Printf("\nhandles: %d\n", totalHandles)
	PrintHandleDistribution(handlesByType)

	/*
		//* most similar
		processes := FindMostSimilar(pid)
		for pid, process := range processes {
			fmt.Printf("\t- %s (%d)\n")
		}
	*/

	//* overlapping
	fmt.Println("\naccess overlaps with:")
	overlapping := g_ObjectAccessRegistry.FindOverlappingWithPs(pid)
	if len(overlapping) == 0 {
		fmt.Printf("\tNone.\n\n")
		return
	}

	// flatten the map to sort it
	// and show top results only
	type entry struct {
		count int
		pid   uint32
	}
	var entries []entry
	for pid, count := range overlapping {
		entries = append(entries, entry{pid: pid, count: count})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	for i := 0; i < 5; i++ {
		if i >= len(overlapping) {
			break
		}
		p := g_ProcessTable.LookupProcess(entries[i].pid)
		if p != nil {
			fmt.Printf("\t- %s (PID %d)", p.Path, p.ProcessId)
		} else {
			fmt.Printf("\t- PID %d (unknown)", entries[i].pid)
		}
		if entries[i].count > 1 {
			fmt.Printf(" [x%d]", entries[i].count)
		}
		fmt.Println()
	}
	if len(overlapping) > 5 {
		fmt.Printf("\t(and %d others)\n", len(overlapping)-5)
	}
	fmt.Println()
}

func PrintObject(objType uint32, name string) {
	if name == "" {
		fmt.Println("Anonymous objects can't be tracked in the current version. :-/")
		fmt.Println("Sorry about that... Better object tracking will be added in the future.")
		return
	}
	fmt.Printf("%s %s\n", GetTypeName(objType), name)

	pids := GetObjectAccessPids(objType, name)
	if len(pids) == 0 {
		fmt.Println("object not accessible by any processes")
		return
	}

	fmt.Printf("accessible by %d processes:\n\t", len(pids))
	for i := 0; i < len(pids); i++ {
		if i != 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%d", pids[i])
		if i%16 == 0 {
			fmt.Printf("\n\t")
		}
	}

	fmt.Println("access distribution:")
	PrintAccessDistribution(objType, name)
}

func (e *AccessEntry) Print(w io.Writer) {
	fmt.Fprintf(w, "* Access by process %d (%s)\n", e.Pid, filepath.Base(LookupProcessPath(e.Pid)))
	fmt.Fprintf(w, "\tObject type: %s\n", GetTypeName(e.Object))
	if e.Name != "" {
		fmt.Fprintf(w, "\tObject name: %s\n", e.Name)
	}
	fmt.Fprintf(w, "\tAccess level: %v\n", e.GetAccessAsString())
}

//TODO: handle.Print()

//TODO: cluster.Print()

//*========================[ Distribution Charts ]=================================

func PrintHandleDistribution(handlesByType map[string]int) {
	if handlesByType == nil || len(handlesByType) == 0 {

	}
	if len(handlesByType) == 0 {
		return
	}

	var (
		maxValue    int
		longestName int
		maxWidth    = 35

		yellow = color.New(color.FgHiYellow)
	)

	type entry struct {
		objType string
		count   int
	}
	entries := make([]entry, len(handlesByType))
	for objType, count := range handlesByType {
		entries = append(entries, entry{objType: objType, count: count})
		if len(objType) > longestName {
			longestName = len(objType)
		}
		if count > maxValue {
			maxValue = count
		}
	}
	divider := strings.Repeat("─", longestName+maxWidth)

	// sort object types in descending order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	fmt.Println(divider)
	for _, entry := range entries {
		yellow.Printf("%s", entry.objType)
		fmt.Printf("%s  │  ", strings.Repeat(" ", longestName-len(entry.objType)))
		fmt.Printf("%s %d\n",
			GetHorizontalBar(entry.count, maxValue, maxWidth), entry.count)
	}
	fmt.Println(divider)
}

// Print the access distribution chart of a named object.
func PrintAccessDistribution(objType uint32, name string) {
	if name == "" {
		return
	}

	var (
		maxValue    int
		longestName int
		maxWidth    = 30

		yellow = color.New(color.FgHiYellow)
	)

	g_ObjectAccessRegistry.mu.RLock()
	if len(g_ObjectAccessRegistry.ObjectLookup[objType]) == 0 {
		return
	}

	accessLevels := make(map[string]int) // key: access flag, value: count
	for key, entries := range g_ObjectAccessRegistry.ObjectLookup[objType] {
		if key.Name != name {
			continue
		}
		for _, entry := range entries {
			flags := entry.GetAccessFlagsAsString().([]string)
			for _, flag := range flags {
				if len(flag) > longestName {
					longestName = len(flag)
				}
				accessLevels[flag]++
			}
		}
	}
	// unlock manually instead of defer for shorter lock
	g_ObjectAccessRegistry.mu.RUnlock()
	divider := strings.Repeat("─", longestName+maxWidth)

	// Convert to list for sorting
	type entry struct {
		flag  string
		count int
	}
	entries := make([]entry, len(accessLevels))
	for flag, count := range accessLevels {
		entries = append(entries, entry{flag: flag, count: count})
		if count > maxValue {
			maxValue = count
		}
	}
	// sort entries in descending order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	fmt.Println(divider)
	for flag, count := range accessLevels {
		yellow.Printf("%s", flag)
		fmt.Printf("%s  |  ", strings.Repeat(" ", longestName-len(flag)))
		fmt.Printf("%s %d\n",
			GetHorizontalBar(count, maxValue, maxWidth))
	}
	fmt.Println(divider)
}

// Print a horizontal bar for a chart. The axis scale is 0->maxValue.
// The bar is returned with a maxValue at maxWidth, with padding if needed.
func GetHorizontalBar(value int, maxValue int, maxWidth int) string {
	relativeVal := float64(value) / float64(maxValue)
	count := int(math.Round(float64(maxWidth) * relativeVal))
	return strings.Repeat("█", count) + strings.Repeat(" ", maxWidth-count)
}
