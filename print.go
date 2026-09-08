package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
)

//?===================================================================================+
//?  This file is responsible for non-TUI prints for the user, such as histograms.    |
//?===================================================================================+

func PrintProcess(w io.Writer, pid uint32) {
	if w == nil {
		w = os.Stdout
	}

	ps := g_ProcessTable.LookupProcess(pid)
	if ps == nil { //TODO: change this, allow with ps as nil
		fmt.Fprintf(w, "No process found with PID %d in lookup table.\n", pid)
		return
	}

	yellow.Fprintf(w, "\nprocess id %d\n", pid)
	yellow.Fprintf(w, "path: ")
	fmt.Fprintf(w, "%s\n", ps.Path)

	yellow.Fprintf(w, "parent: ")
	fmt.Fprintf(w, "PID %d", ps.ParentPid)
	if ps.ParentPath != "" {
		fmt.Fprintf(w, " (%s)", ps.ParentPath)
	}
	fmt.Fprintln(w)

	yellow.Fprintf(w, "elevated: ")
	if ps.Elevated {
		fmt.Fprintln(w, "true")
	} else {
		fmt.Fprintln(w, "false")
	}

	yellow.Fprintf(w, "signature: ")
	fmt.Fprintf(w, "%s\n", GetSigStatusAsString(ps.SigStatus))

	//* handles
	var totalHandles int
	handlesByType := HandleTable.getPsHandleCountsByType(pid)
	for _, count := range handlesByType {
		totalHandles += count
	}
	yellow.Fprintf(w, "\nhandles: ")
	fmt.Fprintf(w, "%d\n", totalHandles)
	PrintHandleDistribution(w, handlesByType)

	/*
		//* most similar
		processes := FindMostSimilar(pid)
		for pid, process := range processes {
			fmt.Printf("\t- %s (%d)\n")
		}
	*/

	//* overlapping
	overlapping, clusters := g_ObjectAccessRegistry.FindOverlappingWithPs(pid)
	fmt.Fprintf(w, "\nprocess %d is a part of ", pid)
	yellow.Fprintf(w, "%d", len(clusters))
	fmt.Fprintln(w, " clusters")

	yellow.Fprintln(w, "\naccess overlaps with:")
	if len(overlapping) == 0 {
		fmt.Fprintf(w, "\tNone.\n\n")
		return
	}

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
		fmt.Fprintf(w, "\t- ")
		if entries[i].count > 1 {
			fmt.Fprintf(w, "[x%d] ", entries[i].count)
		}
		fmt.Fprintf(w, "PID %d", entries[i].pid)

		if p != nil && p.Path != "" {
			fmt.Fprintf(w, " (%s)", filepath.Base(p.Path))
		} else {
			fmt.Fprintf(w, "(unknown)")
		}
		fmt.Fprintln(w)
	}
	if len(overlapping) > 5 {
		fmt.Fprintf(w, "\t(and %d others)\n", len(overlapping)-5)
	}
	fmt.Fprintln(w)

	//* Overlapping processes path frequencies
	frequencyTable := make(map[string]int, len(overlapping))
	for pid := range overlapping {
		var path string
		if ps := g_ProcessTable.LookupProcess(pid); ps != nil {
			path = ps.Path
		}
		frequencyTable[path]++
	}
	fmt.Fprintln(w, "overlapping processes:")
	PrintPathDistribution(w, frequencyTable)
}

func PrintObject(w io.Writer, objType uint32, name string) {
	if w == nil {
		w = os.Stdout
	}

	if name == "" {
		fmt.Fprintln(w, "Anonymous objects can't be tracked in the current version. :-/")
		fmt.Fprintln(w, "Sorry about that... Better object tracking will be added in the future.")
		return
	}
	yellow.Fprintf(w, "%s", GetTypeName(objType))
	fmt.Fprintf(w, " %s\n\n", orAnon(name))

	pids := GetObjectAccessPids(objType, name)
	if len(pids) == 0 {
		fmt.Fprintln(w, "object not accessible by any processes")
		return
	}

	fmt.Fprintf(w, "accessible by ")
	yellow.Fprintf(w, "%d", len(pids))
	fmt.Fprintln(w, " processes")

	fmt.Fprintln(w, "\naccess distribution:")
	PrintAccessDistribution(w, objType, name)
	fmt.Fprintln(w)
}

func (e *AccessEntry) Print(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "* Access by process %d (%s)\n",
		e.Pid, orDash(filepath.Base(LookupProcessPath(e.Pid))))
	fmt.Fprintf(w, "\tObject type: %s\n", GetTypeName(e.Object))
	fmt.Fprintf(w, "\tObject name: %s\n", orAnon(e.Name))
	fmt.Fprintf(w, "\tAccess level: %v\n", e.GetAccessAsString())

	fmt.Fprintln(w)
	PrintObject(w, e.Object, e.Name)
	fmt.Fprintln(w)
}

func (h *HandleEntry) Print(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "* Access by process %d (%s)\n",
		h.Pid, orDash(filepath.Base(LookupProcessPath(h.Pid))))
	fmt.Fprintf(w, "\tObject type: %s\n", GetTypeName(h.Type))
	nameParam := h.GetParameter("Name")
	var name string
	if !nameParam.Empty() {
		name = nameParam.GetValue().(string)
	}
	fmt.Fprintf(w, "\tObject name: %s\n", orAnon(name))
	fmt.Fprintf(w, "\tAccess level: %v\n", h.GetAccessAsString())
}

func (c *Cluster) Print(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Printf("cluster size: %d\n", len(c.Members))

	frequencyTable := make(map[string]int)
	for _, pid := range c.Members {
		var path string
		ps := g_ProcessTable.LookupProcess(pid)
		if ps != nil {
			path = ps.Path
		}
		frequencyTable[path]++
	}
	PrintPathDistribution(w, frequencyTable)
	fmt.Println()
	PrintObject(w, c.ObjType, c.ObjName)
}

func (s *ClusterStats) Print(w io.Writer) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "avg cluster size: ")
	yellow.Fprintf(w, "%.1f\n", s.AvgSize)
	fmt.Fprintf(w, "median cluster size: ")
	yellow.Fprintf(w, "%.1f\n", s.MedianSize)

}

func (p *Process) Print(w io.Writer) {
	PrintProcess(w, p.ProcessId)
}

//*========================[ Distribution Charts ]=================================

func PrintHandleDistribution(w io.Writer, handlesByType map[string]int) {
	if w == nil {
		w = os.Stdout
	}

	if len(handlesByType) == 0 {
		return
	}

	entries := make([]dataEntry, len(handlesByType))
	for objType, count := range handlesByType {
		entries = append(entries, dataEntry{name: objType, value: count})
	}

	PrintHistogram(w, entries)
}

// Print the access distribution chart of a named object.
func PrintAccessDistribution(w io.Writer, objType uint32, name string) {
	if name == "" {
		return
	}

	g_ObjectAccessRegistry.mu.RLock()
	if len(g_ObjectAccessRegistry.ObjectLookup[objType]) == 0 {
		g_ObjectAccessRegistry.mu.RUnlock()
		return
	}

	accessLevels := make(map[string]int) // key: access flag, value: count
	for key, entries := range g_ObjectAccessRegistry.ObjectLookup[objType] {
		if key.Name != name {
			continue
		}
		for _, entry := range entries {
			flags := entry.GetAccessFlagsAsString()
			for _, flag := range flags {
				accessLevels[flag]++
			}
		}
	}
	// unlock manually instead of defer for shorter lock
	g_ObjectAccessRegistry.mu.RUnlock()

	entries := make([]dataEntry, len(accessLevels))
	for flag, count := range accessLevels {
		entries = append(entries, dataEntry{name: flag, value: count})
	}
	PrintHistogram(w, entries)
}

// Helper type for rendering diagrams
type dataEntry struct {
	name  string
	value int
}

const DEFAULT_DIAGRAM_WIDTH = 70

// TODO: truncate too long names with "..." cut-off
func PrintHistogram(w io.Writer, entries []dataEntry) {
	if w == nil {
		w = os.Stdout
	}

	maxTotalWidth := DEFAULT_DIAGRAM_WIDTH
	if ww, ok := w.(WidthWriter); ok {
		maxTotalWidth = ww.Width()
	}

	if len(entries) == 0 {
		return
	}

	var (
		maxValue    int
		longestName int
		maxBarWidth int

		yellow  = color.New(color.FgHiYellow)
		divider = strings.Repeat("─", maxTotalWidth)
	)

	for _, entry := range entries {
		if isEmptyName(entry.name) {
			entry.name = "(unknown)"
		}

		if entry.value > maxValue {
			maxValue = entry.value
		}
		if len(entry.name) > longestName {
			longestName = len(entry.name)
		}
	}

	// the -5 is for "  |  " between name and the bar
	// the -1 is for a space at the end of the bar
	maxBarWidth = maxTotalWidth - longestName - 5 - 1
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].value > entries[j].value
	})

	fmt.Fprintln(w, divider)
	for _, entry := range entries {
		if entry.value == 0 || isEmptyName(entry.name) {
			continue
		}
		yellow.Fprintf(w, "%s", entry.name)
		fmt.Fprintf(w, "%s  |  ", strings.Repeat(" ", longestName-len(entry.name)))
		fmt.Fprintf(w, "%s ",
			GetHorizontalBar(entry.value, maxValue, maxBarWidth))
		yellow.Fprintf(w, "%d\n", entry.value)
	}
	fmt.Fprintln(w, divider)
}

func PrintGlobalObjTypeDistribution() {
	//TODO
}

// Print a horizontal bar for a chart. The axis scale is 0->maxValue.
// The bar is returned with a maxValue at maxWidth, with padding if needed.
func GetHorizontalBar(value int, maxValue int, maxWidth int) string {
	relativeVal := float64(value) / float64(maxValue)
	count := int(math.Round(float64(maxWidth) * relativeVal))

	if count < 0 {
		count = 0 // avoid panic
	}

	if count == 0 && value > 0 {
		count = 1
	}

	barWidth := maxWidth - count
	if barWidth < 0 {
		barWidth = 0 // avoid panic
	}

	return strings.Repeat("█", count) + strings.Repeat(" ", barWidth)
}
