package main

import (
	"context"
	"sync"
)

var (
	MAJOR_VERSION = 0
	MINOR_VERSION = 1
)

func main() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(2)
	go ProcessScanner(&wg, ctx)
	go HandleTable.CacheCleaner(&wg, ctx) // TODO: migrate to run on events
	PrintBanner()

	HandleTable.Init()
	CommandParsingLoop(&wg, cancel)

	wg.Wait()
}
