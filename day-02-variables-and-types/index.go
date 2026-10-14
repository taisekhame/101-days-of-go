package main

import "fmt"

// Package-level constant and variable
const appName = "MTB's Go Journey"

var language = "Go"

func main() {
	fmt.Println(appName)

	// Two ways to declare a variable
	var fruit = "Apple" // var keyword
	count := 5          // := shorthand (only inside functions)
	fmt.Println(language, fruit, count)

	// Explicit type vs inferred type
	var topic string = "Variables" // type written out
	year := 2026                   // type inferred as int
	fmt.Println(topic, year)

	// Multiple variables at once
	var one, two, three int = 1, 2, 3
	fmt.Println(one, two, three)

	// Declaring in a block
	var (
		num       int    // zero value is 0
		number    int    = 1
		greetings string = "hello"
	)
	fmt.Println(num, number, greetings)

	// Constants cannot be changed
	const maxRetries = 3
	fmt.Println("Max retries:", maxRetries)

	// Booleans
	isGolang := true
	isHTML := false
	fmt.Println(isGolang, isHTML)

	// Arithmetic operators
	x := 10
	fmt.Println(x+2, x-2, x*2, x/2, x%3)
	x++ // increment
	x-- // decrement

	// Assignment operators
	x += 5
	x -= 3
	x *= 2
	x /= 4
	x %= 3
	fmt.Println("x is now", x)

	// Strings
	name := "MTB"
	greeting := "Hello, " + name
	fmt.Println(greeting, len(name))
}
