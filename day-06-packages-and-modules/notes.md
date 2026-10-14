# Day 6: Packages and modules

A research day, going deeper into how Go organizes code.

## Packages

- Every Go file starts with a `package` declaration.
- `package main` is special. It tells Go the file belongs to an executable program, and the `main` function is where it starts.
- Packages are Go's way of grouping code and managing dependencies. Other packages are brought in with `import`.

## Modules

- A module is a collection of packages that are versioned together.
- A module is defined by a `go.mod` file. Create one with `go mod init <module-path>`.
- External dependencies are tracked in `go.mod`.

## How this repo uses them

This repo is one module (the `go.mod` at the root). Each day's folder is its own `package main`, which is why every day has its own folder and its own `index.go`.
