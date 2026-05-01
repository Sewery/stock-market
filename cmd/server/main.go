package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"stock-market/internal/api"
	"stock-market/internal/store"
	"stock-market/internal/store/memory"
	"stock-market/internal/store/postgres"
)

func main() {
	ctx := context.Background()

	st, err := buildStore(ctx)
	if err != nil {
		log.Fatalf("store init failed: %v", err)
	}
	defer st.Close()

	h := &api.Handlers{Store: st}
	r := api.NewRouter(h)

	host := getenvDefault("HOST", "localhost")
	port := getenvDefault("PORT", "8080")
	addr := fmt.Sprintf("%s:%s", host, port)

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
