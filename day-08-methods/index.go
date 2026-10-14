package main

import "fmt"

type Person struct {
	firstName  string
	lastName   string
	age        int
	profession string
}

// Value receiver method: reads data, doesn't need to modify the struct
func (p Person) fullName() string {
	return p.firstName + " " + p.lastName
}

func (p Person) printInfo() {
	fmt.Println("Name:", p.fullName())
	fmt.Println("Age:", p.age)
	fmt.Println("Profession:", p.profession)
}

// Pointer receiver method: modifies the actual struct, not a copy
func (p *Person) haveBirthday() {
	p.age++
}

func (p *Person) changeProfession(newProfession string) {
	p.profession = newProfession
}

func main() {
	mtb := Person{
		firstName:  "MTB",
		lastName:   "Codes",
		age:        24,
		profession: "Student",
	}

	fmt.Println("Before:")
	mtb.printInfo()

	mtb.haveBirthday()
	mtb.changeProfession("Go Developer")

	fmt.Println("\nAfter:")
	mtb.printInfo()
}
