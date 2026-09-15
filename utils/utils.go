package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
)

//*========================[ Access Masks ]============================

// Returns string interpretation of all contained flags,
// or if it couldn't find corresponding enums, it returns raw value
func InterpretBitmaskValue(mask Bitmask, domain uint8, array ...bool) any {
	fillOnce.Do(fillReverseEnumLookup)

	var flags []string
	for _, entry := range valToEnum[domain] {
		if mask&entry.Value == entry.Value {
			//* strip flag and save it
			mask &^= entry.Value
			flags = append(flags, entry.Name)
		}
	}
	//* check the generic domain
	for _, entry := range valToEnum[DOMAIN_GLOBAL] {
		if mask&entry.Value != 0 {
			//* strip flag and save it
			mask &^= entry.Value
			flags = append(flags, entry.Name)
		}
	}

	if len(flags) == 0 {
		if len(array) > 0 && array[0] {
			return []string{fmt.Sprintf("0x%X", mask)}
		}
		return mask
	}

	if len(array) == 0 || !array[0] {
		var result string
		for i := range flags {
			result += flags[i]
			if i+1 < len(flags) {
				result += " | "
			}
		}
		return result
	}
	return flags
}

// Convert an access mask from a human readable
// string-based declaration to an actual bitmask value.
func ParseAccessString(accessList string) Bitmask {
	var mask Bitmask
	flags := strings.Split(accessList, "|")
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if enum, exists := enumToVal[flag]; exists {
			mask |= enum.Value
		}
	}
	return mask
}

// Display a bitmask value in human readable format,
// with a maximum width. Flag names are always shown in full,
// or not at all. No "..." truncation, since flags lose meaning.
// The result is "flag1 | flag2 + n flags", with as many flags as fit.
// The displayed flag order is determined by GetFlagPriority() scores.
func DisplayBitflags(mask Bitmask, domain uint8, width int) string {
	flags := InterpretBitmaskValue(mask, domain, true).([]string)
	sort.Slice(flags, func(i, j int) bool {
		return GetFlagPriority(flags[i]) > GetFlagPriority(flags[j])
	})

	var (
		result string
		added  = make(map[string]bool)
	)

	for {
		var flagAdded bool
		for _, flag := range flags {
			if added[flag] {
				continue
			}

			//* Does this flag fit?
			totalLen := len(result) + len(flag)
			remainingFlags := len(flags) - len(added) - 1
			if remainingFlags > 0 {
				totalLen += len(fmt.Sprintf(" + %d flags", remainingFlags))
			}
			if len(added) > 0 {
				totalLen += 3 // for " | "
			}

			if totalLen > width {
				continue
			}

			//* Flag fits so add it
			if len(added) > 0 {
				result += " | "
			}
			result += flag
			flagAdded = true
			added[flag] = true
			break
		}
		// there is no flag that fits anymore
		if !flagAdded {
			break
		}
	}

	if len(flags)-len(added) > 0 {
		if len(added) > 0 {
			result += " + "
		}
		result += fmt.Sprintf("%d flags", len(flags)-len(added))
	}
	return result
}

//*========================[ String helpers ]===========================

// Expand all environment variables %env%/file
// and normalize the usage of slashes.
func NormalizePath(path string) string {
	clean := filepath.Clean(path)

	// if none are seen, there are no env vars
	if !strings.HasPrefix(clean, "%") {
		return clean
	}

	var (
		inside bool
		start  int
		b      strings.Builder
	)

	for i, c := range clean {
		if c == '%' {
			if inside {
				env, ok := os.LookupEnv(clean[start+1 : i])
				if ok {
					b.WriteString(env)
				} else {
					PrintError(nil, "Unknown environment variable: %s\n", clean[start+1:i])
				}
			} else {
				start = i
			}
			inside = !inside
		} else if !inside {
			b.WriteRune(c)
		}
	}

	if inside {
		PrintError(nil, "Invalid use of environment variables, broken path: %v\n", path)
	}
	return b.String()
}

func IsEmptyName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return true
	}
	return false
}

// Print the string, or a dash if empty
func OrDash(s string) string {
	return DefaultOnEmpty(s, "-")
}

// Print the string, or "anonymous" if empty
func OrAnon(s string) string {
	return DefaultOnEmpty(s, "anonymous")
}

// Print the string, or "(anonymous)" if empty
func OrAnon2(s string) string {
	return DefaultOnEmpty(s, "(anonymous)")
}

// Print the string, or "unknown" if empty
func OrUnknown(s string) string {
	return DefaultOnEmpty(s, "unknown")
}

// Print the string, or "unknown" if empty
func OrUnknown2(s string) string {
	return DefaultOnEmpty(s, "(unknown)")
}

func DefaultOnEmpty(s, fallback string) string {
	if IsEmptyName(s) {
		return fallback
	}
	return s
}

//*======================[ Print Helpers ]========================

var (
	red    = color.New(color.FgRed)
	green  = color.New(color.FgGreen)
	yellow = color.New(color.FgHiYellow, color.Bold)
	grey   = color.New(color.FgWhite)
)

func PrintError(w io.Writer, format string, v ...any) {
	PrintWithRedLabel(w, "[*]", format, v...)
}

func PrintWithRedLabel(w io.Writer, label string, format string, v ...any) {
	red.Fprintf(w, "%s ", label)
	fmt.Fprintf(w, format, v...)
}

func PrintBanner(major, minor int) {
	grey := color.New(color.FgWhite)

	fmt.Println("\t   __ _____   _  _____  __   ____   ")
	fmt.Println("\t  / // / _ | / |/ / _ \\/ /  / __/   ")
	fmt.Println("\t / _  / __ |/    / // / /__/ _/     ")
	fmt.Println("\t/_//_/_/ |_/_/|_/____/____/___/     ")
	yellow.Printf("\t  / __/_ __ ___  / /__  _______ ____\n")
	yellow.Printf("\t / _/ \\ \\ // _ \\/ / _ \\/ __/ -_) __/\n")
	yellow.Printf("\t/___//_\\_\\/ .__/_/\\___/_/  \\__/_/   \n")
	yellow.Printf("\t         /_/                        \n")
	grey.Printf("\t\t\tv%d.%d by emryll\n\n", major, minor)

	PrintDescription()
	fmt.Println()
}

func PrintDescription() {
	fmt.Print("\tThis is a commandline-tool for searching\n")
	fmt.Print("\t& analyzing object access through handles.\n\n")
	fmt.Print("\tTo view available commands, run \"help\"\n")
}

//*=======================[ Generic utils ]==========================

func GetInput(reader *bufio.Reader, msg ...string) string {
	if len(msg) > 0 {
		fmt.Printf("%s: ", msg)
	}
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}

	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func RemoveSliceMember[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

// interpret raw c ansi string as a go string
func GetAnsiValue(data []byte) string {
	n := 0
	for ; n < len(data); n++ {
		if data[n] == 0 {
			break // null terminator
		}
	}
	return string(data[:n])
}
