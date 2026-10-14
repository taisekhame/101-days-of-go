package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	target := rng.Intn(100) + 1
	reader := bufio.NewReader(os.Stdin)
	attempts := 0

	fmt.Println("MTB's Number Guessing Game")
	fmt.Println("Guess a number between 1 and 100")

	for {
		fmt.Print("Your guess: ")
		input, err := reader.ReadString('\n')
		if err != nil && input == "" {
			fmt.Println("\nNo more input, goodbye")
			return
		}
		input = strings.TrimSpace(input)

		guess, convErr := strconv.Atoi(input)
		if convErr != nil {
			fmt.Println("Please enter a valid number")
			continue
		}

		attempts++

		if guess < target {
			fmt.Println("Too low")
		} else if guess > target {
			fmt.Println("Too high")
		} else {
			fmt.Println("Correct! You got it in", attempts, "tries")
			return
		}
	}
}
