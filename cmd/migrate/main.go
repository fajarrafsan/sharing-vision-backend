package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"warta/internal/config"
	"warta/internal/database"
)

const usage = `perintah yang tersedia:
  up          jalankan semua migrasi yang belum jalan
  down        batalkan semua migrasi
  goto <N>    naik atau turun sampai versi N
  version     tampilkan versi migrasi saat ini`

func main() {
	if len(os.Args) < 2 {
		log.Fatal(usage)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dsn := cfg.MigrationDSN()

	switch os.Args[1] {
	case "up":
		if err := database.CreateDatabase(context.Background(), cfg.ServerDSN(), cfg.DBName); err != nil {
			log.Fatal("gagal membuat database: ", err)
		}
		if err := database.MigrateUp(dsn, cfg.DBName); err != nil {
			log.Fatal("migrasi up gagal: ", err)
		}
		log.Println("migrasi up selesai")

	case "down":
		if err := database.MigrateDown(dsn, cfg.DBName); err != nil {
			log.Fatal("migrasi down gagal: ", err)
		}
		log.Println("migrasi down selesai")

	case "goto":
		if len(os.Args) < 3 {
			log.Fatal("pakai: goto <versi>")
		}
		version, err := strconv.ParseUint(os.Args[2], 10, 32)
		if err != nil {
			log.Fatal("versi harus berupa angka")
		}
		if err := database.MigrateTo(dsn, cfg.DBName, uint(version)); err != nil {
			log.Fatal("migrasi gagal: ", err)
		}
		log.Printf("database sekarang di versi %d", version)

	case "version":
		version, dirty, err := database.Version(dsn, cfg.DBName)
		if err != nil {
			log.Fatal("gagal membaca versi: ", err)
		}
		fmt.Printf("versi %d", version)
		if dirty {
			fmt.Print(" (dirty: migrasi terakhir gagal di tengah jalan)")
		}
		fmt.Println()

	default:
		log.Fatal(usage)
	}
}
