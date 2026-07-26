package main

import (
	"log"

	"github.com/raithlin/gha/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		log.Fatal(err)
	}
}