package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"stock-market/internal/api"
	"stock-market/internal/store"
	"stock-market/internal/store/memory"
	"stock-market/internal/store/postgres"
	"strings"
)

func main() {
	ctx := context.Background()

	st, err := buildStore(ctx)
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}
	defer st.Close()

	setupLogger()

	h := &api.Handlers{Store: st}
	r := api.NewRouter(h, api.RouterConfig{
		LogHTTP:      getEnvBool("LOG_HTTP", true),
		LogRequestID: getEnvBool("LOG_REQUEST_ID", true),
	})

	host := getenvDefault("HOST", "localhost")
	port := getenvDefault("PORT", "8080")
	addr := fmt.Sprintf("%s:%s", host, port)

	slog.Info("server_start", "addr", addr, "env", getEnvString("ENV", "dev"))

	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func buildStore(ctx context.Context) (store.Store, error) {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		pg, err := postgres.New(ctx, postgres.Config{
			DatabaseURL:   dbURL,
			MigrationsDir: os.Getenv("MIGRATIONS_DIR"),
		})
		if err != nil {
			return nil, err
		}
		return pg, nil
	}

	return memory.New(), nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func setupLogger() {
	if !getEnvBool("LOG_ENABLED", false) {
		// No-op logger: zero output, minimal overhead
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return
	}

	level := parseLogLevel(getEnvString("LOG_LEVEL", "info"))
	format := strings.ToLower(getEnvString("LOG_FORMAT", "json"))

	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(h))
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnvString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return def
	}
}
