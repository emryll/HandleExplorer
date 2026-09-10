package app

import (
	"HandleExplorer/cli"
	_ "HandleExplorer/handles"
	ps "HandleExplorer/process"
	"strings"

	"context"
	"os"
	"sync"
)

func Run() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	config := ParseCmdLine(os.Args)
	if config.Debug {
		DebugInit()
	}

	wg.Add(2)
	go ps.ProcessScanner(&wg, ctx)
	go HandleTable.Init()

	cli.PrintBanner()
	cli.CommandParsingLoop(&wg, cancel)

	wg.Wait()
}

type Config struct {
	Debug bool
	// more fields added in the future
}

// Parse process commandline args (os.Args)
func ParseCmdLine(args []string) Config {
	var config Config
	// this will be updated in the future...
	for _, arg := range args {
		arg = strings.ToLower(arg)
		if arg == "debug" || arg == "dbg" {
			config.Debug = true
		}
	}
	return config
}
