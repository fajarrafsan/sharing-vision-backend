package main

import (
	"log"
	"os"

	"sharing-vision-backend/internal/config"
	"sharing-vision-backend/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("perintah yang tersedia: up | down")
	}

	cfg := config.Load()

	switch os.Args[1] {
	case "up":
		if err := database.CreateDatabase(cfg.ServerDSN(), cfg.DBName); err != nil {
			log.Fatal("gagal membuat database: ", err)
		}
		if err := database.MigrateUp(cfg.DSN(), cfg.DBName); err != nil {
			log.Fatal("migrasi up gagal: ", err)
		}
		log.Println("migrasi up selesai")

	case "down":
		if err := database.MigrateDown(cfg.DSN(), cfg.DBName); err != nil {
			log.Fatal("migrasi down gagal: ", err)
		}
		log.Println("migrasi down selesai")

	default:
		log.Fatal("perintah yang tersedia: up | down")
	}
}
