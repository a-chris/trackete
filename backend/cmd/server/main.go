package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-chris/trackete/backend/internal/api"
	"github.com/a-chris/trackete/backend/internal/config"
	"github.com/a-chris/trackete/backend/internal/store"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Version == "dev" && version != "dev" {
		cfg.Version = version
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer st.Close()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewRouter(cfg, st),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("trackete %s listening on %s (database: %s)", cfg.Version, server.Addr, cfg.DBPath)
		// Say where invitation mail goes: without an SMTP server the links land
		// in a file, and that file is the only way to find them.
		switch cfg.MailTransport {
		case "smtp":
			log.Printf("invitation mail: smtp via %s:%s (from %s)", cfg.SMTPHost, cfg.SMTPPort, cfg.MailFrom)
		case "file":
			log.Printf("invitation mail: file outbox at %s", cfg.MailFile)
		default:
			log.Printf("invitation mail: log transport (set SMTP_HOST to deliver by email)")
		}
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Print("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
