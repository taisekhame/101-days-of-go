package main

import (
	"errors"
	"fmt"
)

// Multiple return values
func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("cannot divide by zero")
	}
	return a / b, nil
}

// Variadic function: takes any number of ints
func sum(numbers ...int) int {
	total := 0
	for _, n := range numbers {
		total += n
	}
	return total
}

func main() {
	fmt.Println("MTB's Day 3")

	// Comparison and logical operators
	fmt.Println(2 > 2, 2 >= 2, 2 == 2, 2 != 2)
	fmt.Println(true && false, true || false, !true)

	// if / else if / else
	password := "12345678"
	if len(password) > 7 {
		fmt.Println("Valid password")
	} else {
		fmt.Println("Invalid password")
	}

	// for loop with continue and break
	for i := 0; i < 5; i++ {
		if i == 1 {
			continue // skip 1
		}
		if i == 4 {
			break // stop before 4
		}
		fmt.Println("i =", i)
	}

	// Go has no while keyword, for does the job
	number := 0
	for number <= 3 {
		fmt.Println("number =", number)
		number++
	}

	// switch
	day := 3
	switch day {
	case 1:
		fmt.Println("Monday")
	case 2:
		fmt.Println("Tuesday")
	case 3:
		fmt.Println("Wednesday")
	default:
		fmt.Println("Another day")
	}

	// Arrays have a fixed size
	arr := [3]int{1, 2, 3}
	fmt.Println(arr, len(arr))

	// Slices can grow
	slice := []string{"Go", "Slices", "Are", "Powerful"}
	slice = append(slice, "Flexible")
	fmt.Println(slice, len(slice), cap(slice))

	made := make([]string, 3, 5) // type, length, capacity
	fmt.Println(made, len(made), cap(made))

	// Maps store key:value pairs
	userInfo := map[string]int{
		"mtb":  24,
		"alex": 18,
	}
	userInfo["sam"] = 27
	fmt.Println(userInfo["mtb"], userInfo)

	// Functions
	result, err := divide(10, 2)
	fmt.Println(result, err)
	_, err = divide(1, 0)
	fmt.Println("Error:", err)

	fmt.Println("Sum:", sum(1, 2, 3, 4, 5))
}
