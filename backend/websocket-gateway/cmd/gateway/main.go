package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sgerhart/aegisflux/websocket-gateway/internal/gateway"
	"github.com/sgerhart/aegisflux/websocket-gateway/internal/types"
)

var (
	port              = flag.Int("port", 8080, "WebSocket gateway port")
	readBufferSize    = flag.Int("read-buffer", 1024, "WebSocket read buffer size")
	writeBufferSize   = flag.Int("write-buffer", 1024, "WebSocket write buffer size")
	maxConnections    = flag.Int("max-connections", 1000, "Maximum concurrent connections")
	heartbeatInterval = flag.Duration("heartbeat", 30*time.Second, "Heartbeat interval")
	connectionTimeout = flag.Duration("connection-timeout", 60*time.Second, "Connection timeout")
	sessionTimeout    = flag.Duration("session-timeout", 24*time.Hour, "Session timeout")
	privateKeyPath    = flag.String("private-key", "", "Path to Ed25519 private key")
	publicKeyPath     = flag.String("public-key", "", "Path to Ed25519 public key")
	databaseURL       = flag.String("database", "", "Database connection URL")
	logLevel          = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	// Create configuration
	config := &types.Configuration{
		Port:              *port,
		ReadBufferSize:    *readBufferSize,
		WriteBufferSize:   *writeBufferSize,
		MaxConnections:    *maxConnections,
		HeartbeatInterval: *heartbeatInterval,
		ConnectionTimeout: *connectionTimeout,
		SessionTimeout:    *sessionTimeout,
		PrivateKeyPath:    *privateKeyPath,
		PublicKeyPath:     *publicKeyPath,
		DatabaseURL:       *databaseURL,
		LogLevel:          *logLevel,
	}

	// Create WebSocket gateway
	wsGateway, err := gateway.NewWebSocketGateway(config)
	if err != nil {
		log.Fatalf("Failed to create WebSocket gateway: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start gateway in a goroutine
	go func() {
		log.Printf("Starting WebSocket Gateway on port %d", config.Port)
		if err := wsGateway.Start(); err != nil {
			log.Fatalf("Failed to start WebSocket gateway: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, gracefully stopping...")

	// Stop gateway
	if err := wsGateway.Stop(); err != nil {
		log.Printf("Error stopping gateway: %v", err)
	}

	log.Println("WebSocket Gateway stopped successfully")
}

// printUsage prints usage information
func printUsage() {
	fmt.Println("AegisFlux WebSocket Gateway")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  websocket-gateway [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -port int")
	fmt.Println("        WebSocket gateway port (default 8080)")
	fmt.Println("  -read-buffer int")
	fmt.Println("        WebSocket read buffer size (default 1024)")
	fmt.Println("  -write-buffer int")
	fmt.Println("        WebSocket write buffer size (default 1024)")
	fmt.Println("  -max-connections int")
	fmt.Println("        Maximum concurrent connections (default 1000)")
	fmt.Println("  -heartbeat duration")
	fmt.Println("        Heartbeat interval (default 30s)")
	fmt.Println("  -connection-timeout duration")
	fmt.Println("        Connection timeout (default 60s)")
	fmt.Println("  -session-timeout duration")
	fmt.Println("        Session timeout (default 24h)")
	fmt.Println("  -private-key string")
	fmt.Println("        Path to Ed25519 private key")
	fmt.Println("  -public-key string")
	fmt.Println("        Path to Ed25519 public key")
	fmt.Println("  -database string")
	fmt.Println("        Database connection URL")
	fmt.Println("  -log-level string")
	fmt.Println("        Log level: debug, info, warn, error (default info)")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  WEBSOCKET_PORT              WebSocket gateway port")
	fmt.Println("  WEBSOCKET_MAX_CONNECTIONS   Maximum concurrent connections")
	fmt.Println("  WEBSOCKET_HEARTBEAT         Heartbeat interval")
	fmt.Println("  WEBSOCKET_CONNECTION_TIMEOUT Connection timeout")
	fmt.Println("  WEBSOCKET_SESSION_TIMEOUT   Session timeout")
	fmt.Println("  WEBSOCKET_PRIVATE_KEY_PATH  Path to Ed25519 private key")
	fmt.Println("  WEBSOCKET_PUBLIC_KEY_PATH   Path to Ed25519 public key")
	fmt.Println("  WEBSOCKET_DATABASE_URL      Database connection URL")
	fmt.Println("  WEBSOCKET_LOG_LEVEL         Log level")
}


