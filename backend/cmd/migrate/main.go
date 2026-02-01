package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/zeshan-weel/backend/internal/db"
)

func main() {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	if len(os.Args) > 1 && os.Args[1] == "down" {
		if err := db.RunMigrationsDown(); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		log.Println("migrate: down ok")
		return
	}

	if err := db.RunMigrations(); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrate: up ok")
}
