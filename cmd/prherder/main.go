package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shreyaabaranwal/pr-herder/internal/authz"
	"github.com/shreyaabaranwal/pr-herder/internal/config"
	"github.com/shreyaabaranwal/pr-herder/internal/ingest"
	"github.com/shreyaabaranwal/pr-herder/internal/slackui"
	"github.com/shreyaabaranwal/pr-herder/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("store init failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	webhookHandler := ingest.NewHandler(cfg.GitHubWebhookSecret, db, log)
	mux.Handle("/github/webhook", webhookHandler)

	// Layer 4: Slack interactivity. The Authorizer needs a GitHub token
	// with read access to check collaborator permissions — for now this
	// reuses GITHUB_WEBHOOK_SECRET's absence as a signal to skip wiring
	// cleanly; a real deployment needs a separate token with repo read
	// scope, tracked as a Layer 5 config addition.
	authorizer := authz.NewAuthorizer(db, cfg.GitHubWebhookSecret)
	executor := slackui.NewStubActionExecutor(log)
	interactionHandler := slackui.NewInteractionHandler(cfg.SlackSigningSecret, authorizer, executor, log)
	mux.Handle("/slack/interact", interactionHandler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("prherder listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
}