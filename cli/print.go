package cli

import (
	"HandleExplorer/handles"
	"HandleExplorer/handles/registry"
	"HandleExplorer/nt"
	"HandleExplorer/process"
	"HandleExplorer/store"
	_ "HandleExplorer/store"
	tlist "HandleExplorer/tui/list"
	"HandleExplorer/utils"
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

func PrintItem[T tlist.ListItem](w io.Writer, item T) {
	switch item := any(item).(type) {
	case *process.Process:
		PrintProcess(w, item)
	case process.Process:
		PrintProcess(w, &item)
	case *handles.HandleEntry:
		PrintHandleEntry(w, item)
	case handles.HandleEntry:
		PrintHandleEntry(w, &item)
	case *registry.AccessEntry:
		PrintAccessEntry(w, item)
	case registry.AccessEntry:
		PrintAccessEntry(w, &item)
	case *registry.Cluster:
		PrintCluster(w, item)
	case registry.Cluster:
		PrintCluster(w, &item)
	}
}

func PrintProcess(w io.Writer, ps *process.Process) {
	if w == nil {
		w = os.Stdout
	}

	var (
		yellow = color.New(color.FgHiYellow)
		grey   = color.New(color.FgWhite)
	)

	yellow.Fprintf(w, "\nprocess id %d\n", ps.ProcessId)
	yellow.Fprintf(w, "path: ")
	fmt.Fprintf(w, "%s\n", ps.Path)

	yellow.Fprintf(w, "parent: ")
	fmt.Fprintf(w, "PID %d ", ps.ParentPid)
	grey.Fprintf(w, "(%s)", utils.OrUnknown(ps.ParentPath))
	fmt.Fprintln(w)

	yellow.Fprintf(w, "elevated: ")
	if ps.Elevated {
		fmt.Fprintln(w, "true")
	} else {
		fmt.Fprintln(w, "false")
	}

	yellow.Fprintf(w, "signature: ")
	fmt.Fprintf(w, "%s\n", process.GetSigStatusAsString(ps.SigStatus))

	//* handles
	var totalHandles int
	handlesByType := store.AccessTracker.HandleTable.GetPsHandleCountsByType(ps.ProcessId)
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
	overlapping, clusters := store.AccessTracker.AccessRegistry.FindOverlappingWithPs(ps.ProcessId)
	fmt.Fprintf(w, "\nprocess %d is a part of ", ps.ProcessId)
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
		p := store.PsTable.LookupProcess(entries[i].pid)
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
		if p := store.PsTable.LookupProcess(pid); p != nil {
			path = p.Path
		}
		frequencyTable[path]++
	}
	fmt.Fprintln(w, "overlapping processes:")
	PrintPathDistribution(w, frequencyTable)
}

func PrintProcessByPid(w io.Writer, pid uint32) {
	ps := store.PsTable.LookupProcess(pid)
	if ps == nil { //TODO: change this, allow with ps as nil
		fmt.Fprintf(w, "No process found with PID %d in lookup table.\n", pid)
		return
	}
	PrintProcess(w, ps)
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

	yellow := color.New(color.FgHiYellow)

	yellow.Fprintf(w, "%s", nt.GetTypeName(objType))
	fmt.Fprintf(w, " %s\n\n", utils.OrAnon(name))

	pids := store.AccessTracker.GetObjectAccessPids(objType, name)
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

func PrintAccessEntry(w io.Writer, e *registry.AccessEntry) {
	if w == nil {
		w = os.Stdout
	}

	path := store.PsTable.LookupProcessPath(e.Pid)
	fmt.Fprintf(w, "* Access by process %d (%s)\n",
		e.Pid, utils.OrDash(filepath.Base(path)))
	fmt.Fprintf(w, "\tObject type: %s\n", nt.GetTypeName(e.Object))
	fmt.Fprintf(w, "\tObject name: %s\n", utils.OrAnon(e.Name))
	fmt.Fprintf(w, "\tAccess level: %v\n", e.GetAccessAsString())

	fmt.Fprintln(w)
	PrintObject(w, e.Object, e.Name)
	fmt.Fprintln(w)
}

func PrintHandleEntry(w io.Writer, h *handles.HandleEntry) {
	if w == nil {
		w = os.Stdout
	}

	path := store.PsTable.LookupProcessPath(h.Pid)
	fmt.Fprintf(w, "* Access by process %d (%s)\n",
		h.Pid, utils.OrUnknown(filepath.Base(path)))
	fmt.Fprintf(w, "\tObject type: %s\n", nt.GetTypeName(h.Type))
	nameParam := h.GetParameter("Name")
	var name string
	if !nameParam.Empty() {
		name = nameParam.GetValue().(string)
	}
	fmt.Fprintf(w, "\tObject name: %s\n", utils.OrAnon(name))
	fmt.Fprintf(w, "\tAccess level: %v\n", h.GetAccessAsString())
}

//TODO make all these functions instead of methods (imported type)

func PrintCluster(w io.Writer, c *registry.Cluster) {
	if w == nil {
		w = os.Stdout
	}

	fmt.Printf("cluster size: %d\n", len(c.Members))

	frequencyTable := make(map[string]int)
	for _, pid := range c.Members {
		var path string
		ps := store.PsTable.LookupProcess(pid)
		if ps != nil {
			path = ps.Path
		}
		frequencyTable[path]++
	}
	PrintPathDistribution(w, frequencyTable)
	fmt.Println()
	PrintObject(w, c.ObjType, c.ObjName)
}

func PrintClusterStats(w io.Writer, s *registry.ClusterStats) {
	if s == nil {
		return
	}

	if w == nil {
		w = os.Stdout
	}

	yellow := color.New(color.FgHiYellow)

	fmt.Fprintf(w, "avg cluster size: ")
	yellow.Fprintf(w, "%.1f\n", s.AvgSize)
	fmt.Fprintf(w, "median cluster size: ")
	yellow.Fprintf(w, "%.1f\n", s.MedianSize)

	var (
		appDataCount int
		windirCount  int
		pfCount      int
		total        int
	)

	var (
		appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData")
		windir  = os.Getenv("WINDIR")
		pf      = os.Getenv("ProgramFiles")
	)

	appData = strings.ToUpper(appData)
	windir = strings.ToUpper(windir)
	pf = strings.ToUpper(pf)

	for dir, count := range s.DirFrequency {
		dir = strings.ToUpper(dir) // case-insensitive search
		if strings.HasPrefix(dir, appData) {
			appDataCount += count
		}
		if strings.HasPrefix(dir, windir) {
			windirCount += count
		}
		if strings.HasPrefix(dir, pf) {
			pfCount += count
		}
		total += count
	}

	remaining := total - windirCount - pfCount
	pfPercentage := float64(pfCount) / float64(total) * 100
	windirPercentage := float64(windirCount) / float64(total) * 100
	appDataPercentage := float64(appDataCount) / float64(total) * 100
	remainingPercentage := float64(remaining) / float64(total) * 100

	yellow.Fprintf(w, "\n%.1f%%", pfPercentage)
	fmt.Fprintln(w, " in Program Files")

	yellow.Fprintf(w, "%.1f%%", windirPercentage)
	fmt.Fprintf(w, " in Windows directory\n")

	if appDataCount > 0 {
		yellow.Fprintf(w, "%.1f%%", appDataPercentage)
		fmt.Fprintf(w, " in AppData directory\n")
	}
	yellow.Fprintf(w, "%.1f%%", remainingPercentage)
	fmt.Fprintf(w, " in ")
	yellow.Fprintf(w, "%d", total-appDataCount-windirCount-pfCount)
	fmt.Fprintf(w, " other directories\n")

	fmt.Fprintln(w)
	//PrintDirDistribution(w, s.DirFrequency)
	//fmt.Fprintln(w)
}

//*========================[ Distribution Charts ]=================================

func PrintDirDistribution(w io.Writer, frequencies map[string]int) {
	if w == nil {
		w = os.Stdout
	}

	if len(frequencies) == 0 {
		return
	}

	var entries []dataEntry
	for dir, count := range frequencies {
		entries = append(entries, dataEntry{name: filepath.Dir(dir), value: count})
	}

	PrintHistogram(w, entries)
}

func PrintPathDistribution(w io.Writer, frequencies map[string]int) {
	if w == nil {
		w = os.Stdout
	}

	if len(frequencies) == 0 {
		return
	}

	var entries []dataEntry
	for path, count := range frequencies {
		entries = append(entries, dataEntry{name: filepath.Base(path), value: count})
	}

	PrintHistogram(w, entries)
	if frequencies[""] > 0 {
		fmt.Fprintf(w, "\t(%d processes with unknown path)\n", frequencies[""])
	}
}

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

	store.AccessTracker.AccessRegistry.RLock()
	if len(store.AccessTracker.AccessRegistry.ObjectLookup[objType]) == 0 {
		store.AccessTracker.AccessRegistry.RUnlock()
		return
	}

	accessLevels := make(map[string]int) // key: access flag, value: count
	for key, entries := range store.AccessTracker.AccessRegistry.ObjectLookup[objType] {
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
	store.AccessTracker.AccessRegistry.RUnlock()

	entries := make([]dataEntry, len(accessLevels))
	for flag, count := range accessLevels {
		entries = append(entries, dataEntry{name: flag, value: count})
	}
	PrintHistogram(w, entries)
}

// Interface for prints with width limit
type WidthWriter interface {
	Width() int
}

// Helper type for rendering diagrams
type dataEntry struct {
	name  string
	value int
}

const DEFAULT_DIAGRAM_WIDTH = 70

// TODO: truncate too long names with "..." cut-off
// Print a histogram visualizing distribution of data.
// If w implements the WidthWriter interface, it will control width.
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
		if utils.IsEmptyName(entry.name) {
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
		if entry.value == 0 || utils.IsEmptyName(entry.name) {
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
