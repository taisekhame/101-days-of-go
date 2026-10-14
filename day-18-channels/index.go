package main

import (
	"fmt"
	"time"
)

type Service struct {
	name  string
	data  string
	delay time.Duration
}

type Result struct {
	Service string
	Data    string
	Took    time.Duration
}

// Fetches from a slow service and sends the result back through a channel
func fetch(s Service, out chan<- Result) {
	time.Sleep(s.delay)
	out <- Result{Service: s.name, Data: s.data, Took: s.delay}
}

// Reads emails off a queue until the channel is closed
func emailWorker(jobs <-chan string, done chan<- bool) {
	for email := range jobs {
		fmt.Println("Sending email to", email)
		time.Sleep(100 * time.Millisecond)
	}
	done <- true
}

func main() {
	fmt.Println("MTB's Dashboard Loader")

	// Part 1: unbuffered channel collecting results from goroutines
	services := []Service{
		{"profile", "user: MTB", 300 * time.Millisecond},
		{"orders", "4 open orders", 500 * time.Millisecond},
		{"notifications", "2 unread", 200 * time.Millisecond},
	}

	results := make(chan Result)
	start := time.Now()

	for _, s := range services {
		go fetch(s, results)
	}

	// No time.Sleep needed: receiving blocks until each result arrives
	for range services {
		r := <-results
		fmt.Printf("%s -> %s (%v)\n", r.Service, r.Data, r.Took)
	}
	fmt.Println("Dashboard ready in", time.Since(start).Round(100*time.Millisecond))

	// Part 2: buffered channel used as a job queue
	fmt.Println("\nEmail queue")
	emails := []string{"mtb@example.com", "alex@example.com", "sam@example.com"}

	jobs := make(chan string, 3)
	done := make(chan bool)

	go emailWorker(jobs, done)

	for _, e := range emails {
		jobs <- e
	}
	close(jobs) // tells the worker no more jobs are coming

	<-done // wait until the worker says it's finished
	fmt.Println("All emails sent")
}
