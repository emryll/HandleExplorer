package main

import (
	"sync"
	"time"
)

type Benchmark struct {
	Name string
	Time time.Duration
}

type BenchmarkRegistry struct {
	mu       sync.RWMutex
	entries  []Benchmark
	capacity int
}

func NewBenchmarkRegistry(capacity int) *BenchmarkRegistry {
	return &BenchmarkRegistry{
		entries:  make([]Benchmark, 0, capacity),
		capacity: capacity,
	}
}

// Timer for benchmarking a function.
// Call the returned function to stop the timer.
func (r *BenchmarkRegistry) Benchmark(name string) func() {
	start := time.Now()
	return func() {
		r.Add(name, time.Since(start))
	}
}

func (r *BenchmarkRegistry) Add(name string, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	//TODO: make this remove like 10% at a time
	if len(r.entries) >= r.capacity {
		// Remove oldest entry.
		copy(r.entries, r.entries[1:])
		r.entries = r.entries[:len(r.entries)-1]
	}

	r.entries = append(r.entries, Benchmark{
		Name: name,
		Time: duration,
	})
}
