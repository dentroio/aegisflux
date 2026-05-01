package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const segMapsSubject = "actions.seg.maps"

type segMapsResponse struct {
	Accepted    bool     `json:"accepted"`
	ServiceID   uint32   `json:"service_id"`
	TargetHosts []string `json:"target_hosts"`
}

type segMapsEvent struct {
	Snapshot struct {
		Version    int `json:"version"`
		ServiceID  int `json:"service_id"`
		TTLSeconds int `json:"ttl_seconds"`
	} `json:"snapshot"`
	TargetHosts []string `json:"target_hosts"`
	Timestamp   string   `json:"timestamp"`
}

func main() {
	orchestratorBaseURL := strings.TrimRight(getenv("ORCHESTRATOR_BASE_URL", "http://localhost:18084"), "/")
	natsURL := getenv("NATS_URL", "nats://localhost:14222")
	targetHost := fmt.Sprintf("seg-smoke-%d", time.Now().UnixNano())
	serviceID := 4242

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fatal(fmt.Errorf("connect to NATS: %w", err))
	}
	defer nc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(segMapsSubject, msgCh)
	if err != nil {
		fatal(fmt.Errorf("subscribe to %s: %w", segMapsSubject, err))
	}
	defer sub.Unsubscribe()
	if err := nc.Flush(); err != nil {
		fatal(fmt.Errorf("flush NATS subscription: %w", err))
	}

	response, err := postMapSnapshot(orchestratorBaseURL, targetHost, serviceID)
	if err != nil {
		fatal(err)
	}
	if !response.Accepted {
		fatal(fmt.Errorf("expected accepted response, got %+v", response))
	}
	if response.ServiceID != uint32(serviceID) {
		fatal(fmt.Errorf("response service_id mismatch: got %d want %d", response.ServiceID, serviceID))
	}
	if len(response.TargetHosts) != 1 || response.TargetHosts[0] != targetHost {
		fatal(fmt.Errorf("response target_hosts mismatch: %+v", response.TargetHosts))
	}

	var natsMsg *nats.Msg
	select {
	case natsMsg = <-msgCh:
	case <-ctx.Done():
		fatal(fmt.Errorf("timed out waiting for %s", segMapsSubject))
	}

	var event segMapsEvent
	if err := json.Unmarshal(natsMsg.Data, &event); err != nil {
		fatal(fmt.Errorf("decode %s message: %w", segMapsSubject, err))
	}
	if event.Snapshot.ServiceID != serviceID {
		fatal(fmt.Errorf("event service_id mismatch: got %d want %d", event.Snapshot.ServiceID, serviceID))
	}
	if event.Snapshot.Version != 1 || event.Snapshot.TTLSeconds != 300 {
		fatal(fmt.Errorf("unexpected snapshot in event: %+v", event.Snapshot))
	}
	if len(event.TargetHosts) != 1 || event.TargetHosts[0] != targetHost {
		fatal(fmt.Errorf("event target_hosts mismatch: %+v", event.TargetHosts))
	}

	printJSON(map[string]interface{}{
		"ok":           true,
		"service_id":   serviceID,
		"target_host":  targetHost,
		"nats_subject": natsMsg.Subject,
	})
}

func postMapSnapshot(baseURL, targetHost string, serviceID int) (segMapsResponse, error) {
	snapshot := map[string]interface{}{
		"version":     1,
		"service_id":  serviceID,
		"ttl_seconds": 300,
		"edges": []map[string]interface{}{
			{"dst_cidr": "192.0.2.0/24", "proto": "tcp", "port": 443},
		},
		"allow_cidrs": []map[string]interface{}{
			{"cidr": "10.0.0.0/8", "proto": "any", "port": 0},
		},
		"meta": map[string]string{"source": "smoke"},
	}

	body, err := json.Marshal(snapshot)
	if err != nil {
		return segMapsResponse{}, fmt.Errorf("marshal snapshot: %w", err)
	}

	url := fmt.Sprintf("%s/seg/maps?target_host=%s", baseURL, targetHost)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return segMapsResponse{}, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return segMapsResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return segMapsResponse{}, fmt.Errorf("POST %s returned %d: %s", url, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var response segMapsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return segMapsResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return response, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func printJSON(value interface{}) {
	data, err := json.Marshal(value)
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(data))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
