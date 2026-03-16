package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pass-pivot/internal/config"
	authbootstrap "pass-pivot/internal/server/auth/bootstrap"
)

func main() {
	if os.Getenv("PPVT_HTTP_ADDR") == "" {
		_ = os.Setenv("PPVT_HTTP_ADDR", "0.0.0.0:8091")
	}
	cfg := config.Load()
	application, err := authbootstrap.NewApp(cfg)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           application.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		application.Logger.Info("ppvt-auth listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.Logger.Error("server crashed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		application.Logger.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}
