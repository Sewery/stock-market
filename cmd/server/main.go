package main

import (
	"fmt"
	"os"

	"stock-market/internal/api"
	"stock-market/internal/store/memory"
)

func main() {
	// Etap 2: na razie tylko MemoryStore (bez Postgresa)
	st := memory.New()
	defer st.Close()

	h := &api.Handlers{Store: st}
	r := api.NewRouter(h)

	host := getenvDefault("HOST", "localhost")
	port := getenvDefault("PORT", "8080")

	_ = r.Run(fmt.Sprintf("%s:%s", host, port))
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
