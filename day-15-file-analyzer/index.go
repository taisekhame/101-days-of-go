package main

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// Interface from Day 14: anything that can count something in a text
type Counter interface {
	Label() string
	Count(text string) int
}

type WordCounter struct{}

func (w WordCounter) Label() string { return "Words" }
func (w WordCounter) Count(text string) int {
	return len(strings.Fields(text))
}

type LineCounter struct{}

func (l LineCounter) Label() string { return "Lines" }
func (l LineCounter) Count(text string) int {
	if text == "" {
		return 0
	}
	return len(strings.Split(strings.TrimRight(text, "\n"), "\n"))
}

type CharCounter struct{}

func (c CharCounter) Label() string { return "Characters" }
func (c CharCounter) Count(text string) int {
	return utf8.RuneCountInString(text)
}

// Uses a map to track words it has already seen
type UniqueWordCounter struct{}

func (u UniqueWordCounter) Label() string { return "Unique words" }
func (u UniqueWordCounter) Count(text string) int {
	seen := map[string]bool{}
	for _, word := range strings.Fields(strings.ToLower(text)) {
		word = strings.Trim(word, ".,!?;:\"'")
		if word != "" {
			seen[word] = true
		}
	}
	return len(seen)
}

// Creates a sample file the first time you run the program
func ensureSampleFile(name string) {
	if _, err := os.Stat(name); err == nil {
		return
	}
	content := "MTB is learning Go.\nDay 15 of the challenge.\nBuilding a word counter with interfaces.\n"
	if err := os.WriteFile(name, []byte(content), 0644); err != nil {
		fmt.Println("Could not create sample file:", err)
	}
}

func main() {
	filename := "sample.txt"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	ensureSampleFile(filename)

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Could not read file:", err)
		return
	}
	text := string(data)

	counters := []Counter{
		WordCounter{},
		LineCounter{},
		CharCounter{},
		UniqueWordCounter{},
	}

	fmt.Println("MTB's File Analyzer")
	fmt.Println("Reading:", filename)
	for _, c := range counters {
		fmt.Printf("%s: %d\n", c.Label(), c.Count(text))
	}
}
