package main

import (
	"context"
	"fmt"
	handler "github.com/JMURv/media-server/internal/hdl/http"
	cfg "github.com/JMURv/media-server/pkg/config"
	"log"
	"os"
	"os/signal"
	"syscall"
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

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic occurred: %v", err)
			os.Exit(1)
		}
	}()

	conf := cfg.MustLoad(configPath)
	ctx, cancel := context.WithCancel(context.Background())

	if _, err := os.Stat(conf.SavePath); os.IsNotExist(err) {
		err = os.MkdirAll(conf.SavePath, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating save path: %s\n", err)
		}
	}

	h := handler.New(fmt.Sprintf(":%v", conf.Port), conf.SavePath, conf.HTTP)
	go handleGracefulShutdown(ctx, cancel, h)
	h.Start()
}
