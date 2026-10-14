package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Task struct {
	id    int
	title string
	done  bool
}

var tasks []Task
var nextID int = 1

func addTask(title string) {
	task := Task{id: nextID, title: title, done: false}
	tasks = append(tasks, task)
	nextID++
	fmt.Println("Added:", title)
}

func listTasks() {
	if len(tasks) == 0 {
		fmt.Println("No tasks yet")
		return
	}
	for _, t := range tasks {
		status := " "
		if t.done {
			status = "x"
		}
		fmt.Printf("[%s] %d. %s\n", status, t.id, t.title)
	}
}

func completeTask(id int) {
	for i := range tasks {
		if tasks[i].id == id {
			tasks[i].done = true
			fmt.Println("Marked done:", tasks[i].title)
			return
		}
	}
	fmt.Println("Task not found")
}

func deleteTask(id int) {
	for i, t := range tasks {
		if t.id == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			fmt.Println("Deleted:", t.title)
			return
		}
	}
	fmt.Println("Task not found")
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("MTB's To-Do List Manager")
	fmt.Println("Commands: add <task>, list, done <id>, delete <id>, exit")

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil && input == "" {
			fmt.Println("\nGoodbye")
			return
		}
		input = strings.TrimSpace(input)
		parts := strings.SplitN(input, " ", 2)
		command := parts[0]

		switch command {
		case "add":
			if len(parts) < 2 {
				fmt.Println("Usage: add <task title>")
				continue
			}
			addTask(parts[1])

		case "list":
			listTasks()

		case "done":
			if len(parts) < 2 {
				fmt.Println("Usage: done <id>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid id")
				continue
			}
			completeTask(id)

		case "delete":
			if len(parts) < 2 {
				fmt.Println("Usage: delete <id>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid id")
				continue
			}
			deleteTask(id)

		case "exit":
			fmt.Println("Goodbye")
			return

		default:
			fmt.Println("Unknown command:", command)
		}
	}
}
