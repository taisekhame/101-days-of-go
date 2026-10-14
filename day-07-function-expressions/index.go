package main

import "fmt"

func main() {
	// Function Expression: a function stored in a variable
	add := func(a int, b int) int {
		return a + b
	}

	result := add(10, 20)
	fmt.Println("MTB's Function Expression Example")
	fmt.Println("10 + 20 =", result)

	// Anonymous Function: a function with no name, called immediately
	func() {
		fmt.Println("This is an Anonymous Function, running right away")
	}()

	// Anonymous Function assigned to a variable, then called later
	greet := func(name string) {
		fmt.Println("Hello,", name, "welcome to Day 7 of #101DaysOfCode")
	}
	greet("MTB")
}
