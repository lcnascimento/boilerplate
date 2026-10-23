package main

import "github.com/lcnascimento/boilerplate/backend/services/notes/bundle"

func main() {
	bundle.Run(bundle.NewWorkerForwarder)
}
