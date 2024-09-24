package main

import (
	"context"
	"fmt"
	"github.com/JMURv/media-server/internal/cleaner"
	mongoCleaner "github.com/JMURv/media-server/internal/cleaner/mongo"
	pgCleaner "github.com/JMURv/media-server/internal/cleaner/pg"
	handler "github.com/JMURv/media-server/internal/handlers/http"
	cfg "github.com/JMURv/media-server/pkg/config"
	"github.com/JMURv/media-server/pkg/consts"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const configPath = "local.config.yaml"

func handleGracefulShutdown(ctx context.Context, cancel context.CancelFunc, h *handler.Handler) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	log.Println("Shutting down gracefully...")

	cancel()
	if err := h.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down server: %s\n", err)
	}

	os.Exit(0)
}

func startCleanerScheduler(ctx context.Context, c cleaner.Cleaner, interval time.Duration) {
	c.Clean(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Running scheduled cleaner...")
			c.Clean(ctx)
		case <-ctx.Done():
			log.Println("Cleaner scheduler stopped.")
			return
		}
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic occurred: %v", err)
			os.Exit(1)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	conf := cfg.MustLoad(configPath)
	if _, err := os.Stat(consts.SavePath); os.IsNotExist(err) {
		err = os.MkdirAll(consts.SavePath, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating save path: %s\n", err)
		}
	}

	var c cleaner.Cleaner
	switch conf.Db {
	case "pg":
		c = pgCleaner.New(conf)
	case "mongo":
		c = mongoCleaner.New(conf)
	default:
		log.Fatalf("Invalid database type: %s\n", conf.Db)
	}

	h := handler.New(fmt.Sprintf(":%v", conf.Port))
	go handleGracefulShutdown(ctx, cancel, h)
	go startCleanerScheduler(ctx, c, 24*time.Hour)
	h.Start()
}
