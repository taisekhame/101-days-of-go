package main

import "fmt"

func greet() {
	defer fmt.Println("World")
	fmt.Println("Hello")
}

func processOrder() {
	fmt.Println("Processing order for MTB")
	defer fmt.Println("Order cleanup complete")
	defer fmt.Println("Closing connection")
	fmt.Println("Order placed")
}

func main() {
	greet()
	fmt.Println("---")
	processOrder()
}
