package main

import (
	"fmt"
	"time"
)

type Service struct {
	name  string
	delay time.Duration
}

// Simulates calling a slow service (a database, an API, etc.)
func fetch(name string, delay time.Duration) {
	fmt.Printf("[%s] request started\n", name)
	time.Sleep(delay)
	fmt.Printf("[%s] finished after %v\n", name, delay)
}

func main() {
	fmt.Println("MTB's Dashboard Loader")

	services := []Service{
		{"profile", 300 * time.Millisecond},
		{"orders", 500 * time.Millisecond},
		{"notifications", 200 * time.Millisecond},
	}

	// One at a time: each call waits for the one before it
	fmt.Println("\n--- Sequential ---")
	start := time.Now()
	for _, s := range services {
		fetch(s.name, s.delay)
	}
	fmt.Println("Sequential total:", time.Since(start).Round(100*time.Millisecond))

	// With goroutines: every call starts at the same time
	fmt.Println("\n--- With goroutines ---")
	start = time.Now()
	for _, s := range services {
		go fetch(s.name, s.delay)
	}

	// main must wait, or the program exits before the goroutines finish
	// (WaitGroups do this properly, that's coming on Day 19)
	time.Sleep(600 * time.Millisecond)
	fmt.Println("Concurrent wait:", time.Since(start).Round(100*time.Millisecond))
}
