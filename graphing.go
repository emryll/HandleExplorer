package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/fatih/color"
)

func printPsHandleDistribution(pid uint32) {
	handlesByType := HandleTable.getPsHandleCountsByType(pid)
	if handlesByType == nil || len(handlesByType) == 0 {
		return
	}
	var (
		maxValue    int
		longestName int
		maxWidth    = 35

		divider = strings.Repeat("─", longestName+base)
		yellow  = color.New(color.FgHiYellow)
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

	// sort object types in descending order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	fmt.Println(divider)
	for _, entry := range entries {
		yellow.Printf("%s", entry.objType)
		fmt.Printf("%s  │  ", strings.Repeat(" ", longestName-len(objType)))
		fmt.Printf("%s %d\n",
			GetHorizontalBar(entry.count, maxValue, maxWidth), entry.count)
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
