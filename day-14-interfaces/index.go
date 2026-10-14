package main

import "fmt"

// An interface is a contract: any type with these methods qualifies
type Speaker interface {
	Speak() string
	Name() string
}

type Dog struct {
	name string
}

func (d Dog) Speak() string { return "Woof!" }
func (d Dog) Name() string  { return d.name }

type Cat struct {
	name string
}

func (c Cat) Speak() string { return "Meow!" }
func (c Cat) Name() string  { return c.name }

type Developer struct {
	name string
}

func (d Developer) Speak() string { return "Hello, world!" }
func (d Developer) Name() string  { return d.name }

// Works with ANY type that satisfies Speaker
func announce(s Speaker) {
	fmt.Println(s.Name(), "says:", s.Speak())
}

func main() {
	fmt.Println("MTB's Interface Demo")

	speakers := []Speaker{
		Dog{name: "Buddy"},
		Cat{name: "Whiskers"},
		Developer{name: "MTB"},
	}

	for _, s := range speakers {
		announce(s)
	}
}
