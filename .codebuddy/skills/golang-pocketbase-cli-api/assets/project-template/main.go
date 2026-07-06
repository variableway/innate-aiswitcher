package main

import (
	"log"

	"myapp/internal/app"

	_ "myapp/migrations"
)

func main() {
	if err := app.NewCLI().Execute(); err != nil {
		log.Fatal(err)
	}
}
