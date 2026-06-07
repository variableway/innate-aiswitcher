package main

import (
	"log"

	"github.com/variableway/innate-aiswitcher/internal/app"

	_ "github.com/variableway/innate-aiswitcher/migrations"
)

func main() {
	if err := app.NewCLI().Execute(); err != nil {
		log.Fatal(err)
	}
}
