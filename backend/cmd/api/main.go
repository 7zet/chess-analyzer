package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ahror/royal-move/internal/config"
	"github.com/ahror/royal-move/internal/httpapi"
	"github.com/ahror/royal-move/internal/jobs"
	"github.com/ahror/royal-move/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.DB.Close()
	if err := db.EnsureAdmin(ctx, cfg.AdminEmail); err != nil {
		log.Fatal(err)
	}
	queue := jobs.New(cfg, db)
	queue.Start(ctx)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.New(cfg, db, queue),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	go func() {
		log.Printf("Royal Move API :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
