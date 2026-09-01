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
	go HandleTable.Init()

	PrintBanner()
	CommandParsingLoop(&wg, cancel)

	wg.Wait()
}
