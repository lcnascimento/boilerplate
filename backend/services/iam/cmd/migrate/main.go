package main

import (
	"os"

	"github.com/lcnascimento/boilerplate/backend/services/iam/bundle"
)

func main() {
	bundle.RunMigrations(os.Args[1:])
}
