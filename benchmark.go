package main

import (
	"fmt"
	"sync"
	"time"
)

type Benchmarker struct {
	mu       sync.RWMutex
	entries  []time.Duration
	Name     string
	capacity int
}

func NewBenchmarker(name string, capacity int) *Benchmarker {
	return &Benchmarker{
		entries:  make([]time.Duration, 0, capacity),
		capacity: capacity,
		Name:     name,
	}
}

// Timer for benchmarking a function.
// Call the returned function to stop the timer.
func (r *Benchmarker) Benchmark() func() {
	start := time.Now()
	return func() {
		r.Add(time.Since(start))
	}
}

func (r *Benchmarker) Add(duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	//TODO: make this remove like 10% at a time
	if len(r.entries) >= r.capacity {
		// Remove oldest entry.
		copy(r.entries, r.entries[1:])
		r.entries = r.entries[:len(r.entries)-1]
	}

	r.entries = append(r.entries, duration)
}

func (r *Benchmarker) Print() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fmt.Printf("[*] %d benchmarks labeled \"%s\"\n", len(r.entries), r.Name)
	for i, entry := range r.entries {
		fmt.Printf("\t#%d %dms\n", i, entry.Milliseconds())
	}
	fmt.Printf("\n\t[ avg: %.1fms     median: %.1fms ]\n", r.GetAvg(), r.GetMedian())
}

// Returns benchmark average in milliseconds
func (r *Benchmarker) GetAvg() float32 {
	var sum time.Duration
	for _, entry := range r.entries {
		sum += entry
	}
	return float32(sum.Milliseconds()) / float32(len(r.entries))
}

// Returns benchmark median in milliseconds
func (r *Benchmarker) GetMedian() float32 {
	// avoid out of bounds error
	if len(r.entries) == 0 {
		return 0
	} else if len(r.entries) == 1 {
		return float32(r.entries[0].Milliseconds())
	}

	if len(r.entries)%2 == 0 {
		upperMidIndex := len(r.entries) / 2
		totalMiddle := r.entries[upperMidIndex-1].Milliseconds()
		totalMiddle += r.entries[upperMidIndex].Milliseconds()
		return float32(totalMiddle) / 2
	} else {
		midIndex := len(r.entries) / 2
		return float32(r.entries[midIndex].Milliseconds())
	}
}

var BenchmarkRegistry map[string]*Benchmarker

func GetBenchmarker(name string) *Benchmarker {
	if BenchmarkRegistry == nil {
		BenchmarkRegistry = make(map[string]*Benchmarker)
	}
	if b, exists := BenchmarkRegistry[name]; exists {
		return b
	}
	fmt.Printf("[dbg] create new benchmarker \"%s\"\n", name)
	b := NewBenchmarker(name, 50000)
	BenchmarkRegistry[name] = b
	return b
}
