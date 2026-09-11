package profiler

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

//?===========================================================+
//?   This is a utility file for conveniently benchmarking    |
//?    functions and viewing collected benchmarking data.     |
//?===========================================================+

// Benchmark collection for a specific thing.
// Collect many samples of performance.
type Benchmarker struct {
	sync.RWMutex
	Entries  []time.Duration
	Name     string
	capacity int
}

func NewBenchmarker(name string, capacity int) *Benchmarker {
	return &Benchmarker{
		Entries:  make([]time.Duration, 0, capacity),
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
	r.Lock()
	defer r.Unlock()

	//TODO: make this remove like 10% at a time
	if len(r.Entries) >= r.capacity {
		// Remove oldest entry.
		copy(r.Entries, r.Entries[1:])
		r.Entries = r.Entries[:len(r.Entries)-1]
	}

	r.Entries = append(r.Entries, duration)
}

func (r *Benchmarker) Print() {
	r.RLock()
	defer r.RUnlock()
	fmt.Printf("[*] %d benchmarks labeled \"%s\"\n", len(r.Entries), r.Name)
	for i, entry := range r.Entries {
		fmt.Printf("\t#%d %dms\n", i, entry.Milliseconds())
	}
	fmt.Printf("\n\t[ avg: %.1fms     median: %.1fms ]\n", r.GetAvg(), r.GetMedian())
}

// Returns benchmark average in milliseconds
func (r *Benchmarker) GetAvg() float32 {
	var sum time.Duration
	for _, entry := range r.Entries {
		sum += entry
	}
	return float32(sum.Milliseconds()) / float32(len(r.Entries))
}

// Returns benchmark median in milliseconds
func (r *Benchmarker) GetMedian() float32 {
	// avoid out of bounds error
	if len(r.Entries) == 0 {
		return 0
	} else if len(r.Entries) == 1 {
		return float32(r.Entries[0].Milliseconds())
	}

	if len(r.Entries)%2 == 0 {
		upperMidIndex := len(r.Entries) / 2
		totalMiddle := r.Entries[upperMidIndex-1].Milliseconds()
		totalMiddle += r.Entries[upperMidIndex].Milliseconds()
		return float32(totalMiddle) / 2
	} else {
		midIndex := len(r.Entries) / 2
		return float32(r.Entries[midIndex].Milliseconds())
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
	b := NewBenchmarker(name, 50000)
	BenchmarkRegistry[name] = b
	return b
}

// This is a little bit broken but dont worry about that
func (b *Benchmarker) PrintDistribution() {
	if len(b.Entries) == 0 {
		fmt.Printf("%s: no samples\n", b.Name)
		return
	}

	samples := append([]time.Duration(nil), b.Entries...)
	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})

	var (
		min  = samples[0]
		max  = samples[len(samples)-1]
		line = "───────────────────────────────────────────────────────────────"
	)
	const (
		bucketCount = 10
		barWidth    = 40
	)

	// avoid zero range when every sample is the same
	if min == max {
		fmt.Printf("\t%s\n%s\n", b.Name, line)
		fmt.Printf("%s │ %s\n", formatDuration(min), "████████████████████████████████████████")
		fmt.Println(line)
		return
	}

	bucketSize := (max - min) / bucketCount
	if bucketSize == 0 {
		bucketSize = 1
	}

	buckets := make([]int, bucketCount)

	for _, sample := range samples {
		index := int((sample - min) / bucketSize)

		// The maximum value can land exactly on bucketCount.
		if index >= bucketCount {
			index = bucketCount - 1
		}

		buckets[index]++
	}

	maxBucket := 0
	for _, count := range buckets {
		if count > maxBucket {
			maxBucket = count
		}
	}

	fmt.Printf("\n\t%s\n", b.Name)
	fmt.Println(line)

	for i, count := range buckets {
		start := min + time.Duration(i)*bucketSize
		end := start + bucketSize

		barLength := count * barWidth / maxBucket
		bar := ""

		for j := 0; j < barLength; j++ {
			bar += "█"
		}

		fmt.Printf(
			"%8s - %-8s │ %-40s %d\n",
			formatDuration(start),
			formatDuration(end),
			bar,
			count,
		)
	}

	fmt.Println(line)
	b.printStats(samples)
}

func (b *Benchmarker) printStats(samples []time.Duration) {
	fmt.Printf("%16s: %s\n", "min", formatDuration(samples[0]))
	fmt.Printf("%16s: %s\n", "median", formatDuration(percentile(samples, 50)))
	fmt.Printf("%16s: %s\n", "p95", formatDuration(percentile(samples, 95)))
	fmt.Printf("%16s: %s\n", "p99", formatDuration(percentile(samples, 99)))
	fmt.Printf("%16s: %s\n", "max", formatDuration(samples[len(samples)-1]))
}

func percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 1 {
		return samples[0]
	}

	index := p / 100 * float64(len(samples)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(samples) {
		return samples[lower]
	}

	fraction := index - float64(lower)

	return time.Duration(
		float64(samples[lower]) +
			fraction*float64(samples[upper]-samples[lower]),
	)
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%.2fns", float64(d.Nanoseconds()))
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fµs", float64(d)/float64(time.Microsecond))
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
