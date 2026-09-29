package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shreyaabaranwal/pr-herder/internal/authz"
	"github.com/shreyaabaranwal/pr-herder/internal/config"
	"github.com/shreyaabaranwal/pr-herder/internal/githubmcp"
	"github.com/shreyaabaranwal/pr-herder/internal/httpmw"
	"github.com/shreyaabaranwal/pr-herder/internal/ingest"
	"github.com/shreyaabaranwal/pr-herder/internal/llm"
	"github.com/shreyaabaranwal/pr-herder/internal/scheduler"
	"github.com/shreyaabaranwal/pr-herder/internal/slackui"
	"github.com/shreyaabaranwal/pr-herder/internal/store"
	"github.com/shreyaabaranwal/pr-herder/internal/triage"
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


	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready: " + err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.Handle("/metrics", promhttp.Handler())

	webhookHandler := ingest.NewHandler(cfg.GitHubWebhookSecret, db, log)
	mux.Handle("/github/webhook", webhookHandler)

	
	authorizer := authz.NewAuthorizer(db, cfg.GitHubReadToken)
	mcpClient := githubmcp.NewClient(cfg.GitHubMCPURL, cfg.GitHubReadToken)
	executor := githubmcp.NewExecutor(mcpClient)
	interactionHandler := slackui.NewInteractionHandler(cfg.SlackSigningSecret, authorizer, executor, log)
	mux.Handle("/slack/interact", interactionHandler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpmw.Recover(log, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("prherder listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()


	triageEngine := triage.NewEngine()
	publisher := slackui.NewPublisher(cfg.SlackBotToken, cfg.SlackDefaultChan)


	ollamaClient := llm.NewOllamaClient(
		cfg.OllamaURL,
		cfg.OllamaModel,
	)
	summarizer := llm.NewSummarizer(ollamaClient)

	worker := ingest.NewWorker(
		db,
		triageEngine,
		publisher,
		mcpClient,
		summarizer,
		log,
	)
	go worker.Run(ctx, 5*time.Second)

	sched := scheduler.NewScheduler(
		db,
		mcpClient,
		publisher,
		log,
		cfg.DigestHourLocal,
		cfg.StaleDaysThreshold,
		cfg.QuietHoursStart,
		cfg.QuietHoursEnd,
	)
	go sched.Run(ctx)

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
	}
}
