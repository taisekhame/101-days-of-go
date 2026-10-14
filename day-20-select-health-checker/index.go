package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

type Target struct {
	Name string
	URL  string
}

type CheckResult struct {
	Name   string
	Status string
}

func check(t Target, timeout time.Duration, wg *sync.WaitGroup, out chan<- CheckResult) {
	defer wg.Done()

	// Buffered so the helper goroutine never gets stuck if we time out first
	reply := make(chan string, 1)

	go func() {
		resp, err := http.Get(t.URL)
		if err != nil {
			reply <- "down"
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			reply <- fmt.Sprintf("unhealthy (HTTP %d)", resp.StatusCode)
			return
		}
		reply <- "healthy"
	}()

	// select waits on several channels and takes whichever is ready first
	select {
	case status := <-reply:
		out <- CheckResult{t.Name, status}
	case <-time.After(timeout):
		out <- CheckResult{t.Name, "timed out"}
	}
}

func main() {
	fmt.Println("MTB's Service Health Checker")

	// Fake services so this runs without internet.
	// To check real sites, put real URLs in the targets list below.
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	brokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer brokenServer.Close()

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	goneServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	goneServer.Close() // nothing is listening on this address anymore

	targets := []Target{
		{"Auth API", okServer.URL},
		{"Payments API", brokenServer.URL},
		{"Reports API", slowServer.URL},
		{"Legacy API", goneServer.URL},
	}

	timeout := time.Second
	var wg sync.WaitGroup
	results := make(chan CheckResult)
	start := time.Now()

	for _, t := range targets {
		wg.Add(1)
		go check(t, timeout, &wg, results)
	}

	// Close the channel once every check has finished
	go func() {
		wg.Wait()
		close(results)
	}()

	byName := map[string]string{}
	for r := range results {
		byName[r.Name] = r.Status
	}

	// Print in the original order so the output is easy to read
	healthyCount := 0
	for _, t := range targets {
		status := byName[t.Name]
		fmt.Printf("%-14s %s\n", t.Name, status)
		if status == "healthy" {
			healthyCount++
		}
	}

	fmt.Printf("\n%d healthy, %d need attention\n", healthyCount, len(targets)-healthyCount)
	fmt.Println("Checked everything in", time.Since(start).Round(100*time.Millisecond))
}
