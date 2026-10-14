package main

import "fmt"

// Basic callback: a function passed in and called inside another function
func greetUser(name string, callback func(string)) {
	callback(name)
}

// Callback with a return value processed after
func calculate(a int, b int, operation func(int, int) int) int {
	return operation(a, b)
}

// Callback used to process each item in a slice
func processNames(names []string, callback func(string)) {
	for _, name := range names {
		callback(name)
	}
}

func main() {
	// Example 1: basic callback
	greetUser("MTB", func(name string) {
		fmt.Println("Hi, my name is", name)
	})

	// Example 2: callback that computes and returns a value
	sum := calculate(10, 20, func(a, b int) int {
		return a + b
	})
	fmt.Println("Sum:", sum)

	product := calculate(10, 20, func(a, b int) int {
		return a * b
	})
	fmt.Println("Product:", product)

	// Example 3: callback applied to each item in a slice
	names := []string{"MTB", "Go", "Rust"}
	processNames(names, func(name string) {
		fmt.Println("Processing:", name)
	})
}
