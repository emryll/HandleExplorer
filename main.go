package main

import (
	"bufio"
	"context"
	"os"
	"strings"
	"sync"
)

var (
	VERSION = "0.0"
	stars   = strings.Repeat("*", 30)
)

func main() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	PrintBanner()
	go HandleScanner(&wg, ctx)

	reader := bufio.NewReader(os.Stdin)
	for {
		if ctx.Err() != nil {
			break
		}

		command := GetInput(" handles> ", reader)
		if command == "" {
			continue
		}

		tokens := strings.Fields(command)
		exit := CliParseCommand(tokens)
		if exit {
			break
		}
	}
	wg.Wait()
}
