package main

import "fmt"

var appName = "MTB's Go Journey" // package scope, visible to the whole file

func main() {
	x := 10 // function scope, only visible inside main
	fmt.Println(appName)
	fmt.Println(x)

	if true {
		y := 20 // block scope, only visible inside this if block
		fmt.Println(y)
	}

	// y is not accessible here, it stayed inside the if block
	showAppName()
}

func showAppName() {
	fmt.Println(appName) // works, appName has package scope
}
