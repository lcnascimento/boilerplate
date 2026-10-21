package main

import "github.com/lcnascimento/boilerplate/backend/services/audit/bundle"

func main() {
	bundle.Run(bundle.NewConsumer)
}
