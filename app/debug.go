package app

import (
	"log"
	"net/http"
)

// Currently very simple.
// Just launches pprof on localhost:6060
func DebugInit() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}
