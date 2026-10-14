package main

import "fmt"

// Takes a pointer, so it can change the original value
func rename(name *string) {
	*name = "MTB the Developer"
}

func main() {
	x := 42
	var pointerToX *int = &x

	fmt.Println("Value of x:", x)
	fmt.Println("Memory address of x:", &x)
	fmt.Println("Value pointed to by pointerToX:", *pointerToX)
	fmt.Println("Address stored in pointerToX:", pointerToX)

	// %T prints the type
	fmt.Printf("%T\n", x)
	fmt.Printf("%T\n", &x)

	// Changing a value through a pointer
	mtb := "MTB"
	ptr := &mtb
	*ptr = "Updated MTB"
	fmt.Println(mtb)

	// Passing a pointer into a function
	rename(&mtb)
	fmt.Println(mtb)
}
