package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sharing-vision-backend/internal/config"
	"sharing-vision-backend/internal/database"
	"sharing-vision-backend/internal/handler"
	"sharing-vision-backend/internal/repository"
	"sharing-vision-backend/internal/router"
	"sharing-vision-backend/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.AutoMigrate {
		if err := database.CreateDatabase(cfg.ServerDSN(), cfg.DBName); err != nil {
			log.Fatal("gagal membuat database: ", err)
		}
		if err := database.MigrateUp(cfg.DSN(), cfg.DBName); err != nil {
			log.Fatal("gagal menjalankan migrasi: ", err)
		}
		log.Println("migrasi database selesai")
	}

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatal("gagal terhubung ke database: ", err)
	}
	defer db.Close()

	articleRepository := repository.NewArticleRepository(db)
	articleService := service.NewArticleService(articleRepository, cfg.DefaultLimit, cfg.MaxLimit)
	articleHandler := handler.NewArticleHandler(articleService)
	healthHandler := handler.NewHealthHandler(db)

	server := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router.New(articleHandler, healthHandler),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("service berjalan di http://localhost:" + cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server berhenti: ", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("mematikan service...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("gagal mematikan service: ", err)
	}
	log.Println("service berhenti")
}
