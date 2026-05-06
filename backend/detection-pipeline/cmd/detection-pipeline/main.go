package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aegisflux/backend/detection-pipeline/internal/api"
	"aegisflux/backend/detection-pipeline/internal/store"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	httpAddr := getenv("DETECTION_PIPELINE_HTTP_ADDR", ":8089")
	ingestURL := getenv("INGEST_URL", "http://127.0.0.1:9090")
	dataPath := getenv("DETECTION_PIPELINE_DATA_PATH", "")

	st := store.New(dataPath)
	if err := st.Load(); err != nil {
		logger.Printf("store load: %v", err)
		os.Exit(1)
	}

	srv, err := api.NewServer(st, ingestURL)
	if err != nil {
		logger.Printf("api init: %v", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	srv.Register(mux)

	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	go func() {
		logger.Printf("detection-pipeline listening on %s (ingest=%s)", httpAddr, ingestURL)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Printf("http: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	logger.Printf("shutdown complete")
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
