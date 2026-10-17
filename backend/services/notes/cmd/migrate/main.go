package main

import (
	"os"

	"github.com/lcnascimento/boilerplate/backend/services/notes/bundle"
)

func main() {
	bundle.RunMigrations(os.Args[1:])
}
