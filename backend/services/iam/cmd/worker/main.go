package main

import "github.com/lcnascimento/boilerplate/backend/services/iam/bundle"

func main() {
	bundle.Run(bundle.NewWorker)
}
