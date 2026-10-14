package main

import (
	"fmt"
	"sync"
	"time"
)

type Order struct {
	ID       int
	Customer string
	Amount   float64
}

type Receipt struct {
	OrderID  int
	Customer string
	Status   string
}

// Simulates charging a payment. Bigger order IDs take a bit longer.
func processOrder(o Order, wg *sync.WaitGroup, out chan<- Receipt) {
	defer wg.Done() // tell the WaitGroup this order is finished

	time.Sleep(time.Duration(o.ID*100) * time.Millisecond)

	status := "paid"
	if o.Amount > 1000 {
		status = "needs review"
	}
	out <- Receipt{OrderID: o.ID, Customer: o.Customer, Status: status}
}

func main() {
	fmt.Println("MTB's Order Processor")

	orders := []Order{
		{1, "MTB", 49.99},
		{2, "Alex", 1250.00},
		{3, "Sam", 15.50},
		{4, "MTB", 320.00},
		{5, "Alex", 89.00},
	}

	var wg sync.WaitGroup
	// Buffered so every worker can drop its receipt without waiting for main
	receipts := make(chan Receipt, len(orders))

	start := time.Now()
	var oneByOne time.Duration

	for _, o := range orders {
		wg.Add(1) // one more goroutine to wait for
		go processOrder(o, &wg, receipts)
		oneByOne += time.Duration(o.ID*100) * time.Millisecond
	}

	wg.Wait()       // block until every Done() has been called
	close(receipts) // safe now, nobody is sending anymore

	paid, review := 0, 0
	for r := range receipts {
		fmt.Printf("Order %d (%s): %s\n", r.OrderID, r.Customer, r.Status)
		if r.Status == "paid" {
			paid++
		} else {
			review++
		}
	}

	fmt.Printf("\n%d paid, %d flagged for review\n", paid, review)
	fmt.Printf("Finished in %v (one at a time: %v)\n",
		time.Since(start).Round(100*time.Millisecond), oneByOne)
}
