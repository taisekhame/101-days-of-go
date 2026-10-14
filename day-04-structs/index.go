package main

import "fmt"

type Person struct {
	name   string
	age    int
	job    string
	salary int
}

func main() {
	// Named struct, fields set one by one
	var userOne Person
	userOne.name = "MTB"
	userOne.age = 24
	userOne.job = "Programmer"
	userOne.salary = 40000
	fmt.Println(userOne)
	fmt.Println("My name is", userOne.name, "and I'm", userOne.age, "years old. I work as a", userOne.job)

	// Named struct, fields set inline
	userTwo := Person{name: "Alex", age: 30, job: "Designer", salary: 35000}
	fmt.Println(userTwo)

	// Anonymous struct: no named type, defined and used on the spot
	h := struct {
		firstName string
		lastName  string
	}{
		firstName: "MTB",
		lastName:  "Codes",
	}
	fmt.Println(h.firstName, h.lastName)
}
