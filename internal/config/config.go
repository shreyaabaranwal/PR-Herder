
// Package config centralizes environment loading. Nothing else in the
// codebase should call os.Getenv directly — that pattern scatters
// "what env vars does this service need?" across every package and makes
// missing-config failures show up at runtime, deep in some unrelated code
// path, instead of at startup where they're cheap to diagnose.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	PublicURL string

	DatabaseURL string

	SlackBotToken      string
	SlackSigningSecret string
	SlackDefaultChan   string

	GitHubAppID          string
	GitHubPrivateKeyPath string
	GitHubWebhookSecret  string
    GitHubReadToken      string
	GitHubMCPURL string

	AnthropicAPIKey string
	GeminiAPIKey    string
    
	OllamaURL   string
    OllamaModel string



	StaleDaysThreshold int
	DigestHourLocal    int
	QuietHoursStart    int
	QuietHoursEnd      int
}

// Load reads .env (if present — fine if it's missing in real prod, where
// secrets come from the environment/secret-manager instead) and then
// requires every field below to be set. Returns an error rather than
// panicking so cmd/prherder can log a clean fatal message.
func Load() (*Config, error) {
	_ = godotenv.Load() // ignore error: .env is optional (e.g. absent in prod containers)

	cfg := &Config{
		Port:      getOr("PORT", "8080"),
		PublicURL: os.Getenv("PUBLIC_URL"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		SlackBotToken:      os.Getenv("SLACK_BOT_TOKEN"),
		SlackSigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		SlackDefaultChan:   os.Getenv("SLACK_DEFAULT_CHANNEL"),

		GitHubAppID:          os.Getenv("GITHUB_APP_ID"),
		GitHubPrivateKeyPath: os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH"),
		GitHubWebhookSecret:  os.Getenv("GITHUB_WEBHOOK_SECRET"),
        GitHubReadToken:      os.Getenv("GITHUB_READ_TOKEN"),
		GitHubMCPURL: os.Getenv("GITHUB_MCP_URL"),

		
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		OllamaURL:   getOr("OLLAMA_URL", "http://localhost:11434"),
        OllamaModel: getOr("OLLAMA_MODEL", "llama3.2:3b"),
	}
	var err error
	cfg.StaleDaysThreshold, err = getInt("STALE_PR_DAYS", 7)
	if err != nil {
		return nil, err
	}
	cfg.DigestHourLocal, err = getInt("DIGEST_HOUR_LOCAL", 9)
	if err != nil {
		return nil, err
	}
	if err := cfg.parseQuietHours(getOr("QUIET_HOURS", "22-08")); err != nil {
		return nil, err
	}

	// Layer 1 only truly needs these two to boot. As later layers land,
	// add their required vars here — fail at startup, not at first use.
	required := map[string]string{
		"DATABASE_URL":          cfg.DatabaseURL,
		"GITHUB_WEBHOOK_SECRET": cfg.GitHubWebhookSecret,
	}
	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid int for %s: %w", key, err)
	}
	return n, nil
}

func (c *Config) parseQuietHours(raw string) error {
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return fmt.Errorf("QUIET_HOURS must be HH-HH, got %q", raw)
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid QUIET_HOURS start: %w", err)
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid QUIET_HOURS end: %w", err)
	}
	c.QuietHoursStart = start
	c.QuietHoursEnd = end
	return nil
}
