package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/fatih/color"
)

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
			flags := GetStringFlagsFromValue(entry.Access)
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
