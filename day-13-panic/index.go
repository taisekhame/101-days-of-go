package main

import "fmt"

func checkAge(name string, age int) {
	if age > 65 {
		panic(name + " is past retirement age")
	}
	fmt.Println(name, "cleared the age check")
}

func safeCheckAge(name string, age int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	checkAge(name, age)
}

func main() {
	fmt.Println("Starting checks for MTB's company")
	safeCheckAge("MTB", 30)
	safeCheckAge("John", 75)
	fmt.Println("Program kept running after the panic")
}
