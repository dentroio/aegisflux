package api

import (
	"log"
	"net/http"
	"os"

	"github.com/nats-io/nats.go"
)

type Server struct {
	mux      *http.ServeMux
	store    *Store
	nc       *nats.Conn
	platform *PlatformData
}

func NewServer() *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		store:    NewStore(),
		platform: newPlatformData(),
	}

	// Connect to NATS for WebSocket Gateway integration
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to NATS: %v", err)
	} else {
		s.nc = nc
		log.Printf("Connected to NATS at %s", natsURL)
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	s.registerPlatformRoutes()

	s.mux.HandleFunc("/console/summary/agents-workbench", s.getAgentsWorkbenchSummary)
	s.mux.HandleFunc("/console/summary/agent-readiness", s.handleAgentReadinessFleet)

	// Agent registration endpoints
	s.mux.HandleFunc("/agents/register/init", s.postRegisterInit)
	s.mux.HandleFunc("/agents/register/complete", s.postRegisterComplete)

	// Agent heartbeat endpoint
	s.mux.HandleFunc("/agents/heartbeat", s.handleHeartbeat)

	// Agents API endpoints - specific routes first, then catch-all
	s.mux.HandleFunc("/agents/broadcast", s.broadcastToAgents) // Broadcast endpoint (must come before /agents/)
	s.mux.HandleFunc("/agents", s.getAgents)                   // List agents

	s.mux.HandleFunc("/agents/", s.agentDispatch) // Subrouter emulation for /agents/{uid}/* paths
}
func (s *Server) Handler() http.Handler { return s.mux }
