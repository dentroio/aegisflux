package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"aegisflux/backend/segmenter/internal/api"
	"aegisflux/backend/segmenter/internal/segmenter"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Get configuration from environment
	addr := os.Getenv("SEG_HTTP_ADDR")
	if addr == "" {
		addr = ":8086"
	}

	logger.Info("Starting segmenter service", "addr", addr)

	// Create segmenter and handler
	seg := segmenter.NewSegmenter(logger)
	handler := api.NewHandler(seg, logger)

	// Set up routes
	http.HandleFunc("/healthz", handler.HealthCheck)
	http.HandleFunc("/segment/propose", handler.ProposeSegmentation)
	http.HandleFunc("/segment/plan", handler.CreateSegmentationPlan)
	http.HandleFunc("/segment/strategies", handler.GetSegmentationStrategies)
	http.HandleFunc("/segment/goals", handler.GetSegmentationGoals)

	logger.Info("Segmenter service started", "addr", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
