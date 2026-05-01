package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sgerhart/aegisflux/websocket-gateway/internal/types"
)

type registerInitResponse struct {
	RegistrationID string `json:"registration_id"`
	Nonce          string `json:"nonce"`
	ServerTime     string `json:"server_time"`
}

type registerCompleteResponse struct {
	AgentUID string `json:"agent_uid"`
}

type sendMessageResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

func main() {
	actionsBaseURL := strings.TrimRight(getenv("ACTIONS_API_BASE_URL", "http://localhost:8083"), "/")
	gatewayBaseURL := strings.TrimRight(getenv("GATEWAY_BASE_URL", "http://localhost:8080"), "/")
	hostID := fmt.Sprintf("ws-smoke-%d", time.Now().UnixNano())

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatal(fmt.Errorf("generate agent key: %w", err))
	}
	publicKey := base64.StdEncoding.EncodeToString(pub)

	agentUID, err := registerAgent(actionsBaseURL, hostID, publicKey, priv)
	if err != nil {
		fatal(err)
	}

	conn, err := connectAndAuthenticate(gatewayBaseURL, agentUID, publicKey, priv)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()

	sendResp, err := sendAgentMessage(actionsBaseURL, agentUID)
	if err != nil {
		fatal(err)
	}
	if sendResp.Status != "sent" {
		fatal(fmt.Errorf("expected send status sent, got %q: %s", sendResp.Status, sendResp.Error))
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		fatal(fmt.Errorf("set read deadline: %w", err))
	}
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		fatal(fmt.Errorf("read delivered WebSocket message: %w", err))
	}
	if messageType != websocket.TextMessage {
		fatal(fmt.Errorf("expected WebSocket text message, got type %d", messageType))
	}

	var delivered types.SecureMessage
	if err := json.Unmarshal(data, &delivered); err != nil {
		fatal(fmt.Errorf("decode delivered SecureMessage: %w", err))
	}
	if delivered.ID != sendResp.MessageID {
		fatal(fmt.Errorf("message id mismatch: send=%q delivered=%q", sendResp.MessageID, delivered.ID))
	}
	if delivered.Type != types.MessageTypeRequest {
		fatal(fmt.Errorf("unexpected delivered type: %q", delivered.Type))
	}
	expectedChannel := "agent." + agentUID + ".policies"
	if delivered.Channel != expectedChannel {
		fatal(fmt.Errorf("unexpected delivered channel: got %q want %q", delivered.Channel, expectedChannel))
	}
	if delivered.Payload == "" || delivered.Nonce == "" {
		fatal(fmt.Errorf("delivered message missing encrypted payload or nonce: %+v", delivered))
	}

	printJSON(map[string]string{
		"ok":         "true",
		"agent_uid":  agentUID,
		"message_id": delivered.ID,
		"channel":    delivered.Channel,
		"validated":  "websocket_secure_message",
	})
}

func registerAgent(baseURL, hostID, publicKey string, privateKey ed25519.PrivateKey) (string, error) {
	initReq := map[string]interface{}{
		"org_id":        "smoke-org",
		"host_id":       hostID,
		"agent_pubkey":  publicKey,
		"agent_version": "smoke",
	}

	var initResp registerInitResponse
	if err := postJSON(baseURL+"/agents/register/init", initReq, &initResp); err != nil {
		return "", err
	}

	nonce, err := base64.StdEncoding.DecodeString(initResp.Nonce)
	if err != nil {
		return "", fmt.Errorf("decode registration nonce: %w", err)
	}
	signingPayload := append(nonce, []byte(initResp.ServerTime+hostID)...)
	signature := ed25519.Sign(privateKey, signingPayload)

	completeReq := map[string]interface{}{
		"registration_id": initResp.RegistrationID,
		"host_id":         hostID,
		"signature":       base64.StdEncoding.EncodeToString(signature),
	}

	var completeResp registerCompleteResponse
	if err := postJSON(baseURL+"/agents/register/complete", completeReq, &completeResp); err != nil {
		return "", err
	}
	if completeResp.AgentUID == "" {
		return "", fmt.Errorf("registration complete response missing agent_uid")
	}
	return completeResp.AgentUID, nil
}

func connectAndAuthenticate(baseURL, agentUID, publicKey string, privateKey ed25519.PrivateKey) (*websocket.Conn, error) {
	wsURL, err := websocketURL(baseURL, "/ws/agent")
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set("X-Agent-ID", agentUID)
	headers.Set("X-Agent-Public-Key", publicKey)
	headers.Set("User-Agent", "Aegis-Agent/1.0")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("connect to gateway WebSocket: %w", err)
	}

	timestamp := time.Now().Unix()
	nonce := "smoke-auth-nonce"
	authReq := types.AuthenticationRequest{
		AgentID:   agentUID,
		PublicKey: publicKey,
		Timestamp: timestamp,
		Nonce:     nonce,
	}
	signaturePayload := fmt.Sprintf("%s:%s:%d:%s", authReq.AgentID, authReq.PublicKey, authReq.Timestamp, authReq.Nonce)
	authReq.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(signaturePayload)))

	authReqData, err := json.Marshal(authReq)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("marshal auth request: %w", err)
	}

	authMessage := types.SecureMessage{
		ID:        fmt.Sprintf("auth_%d", time.Now().UnixNano()),
		Type:      types.MessageTypeRequest,
		Channel:   "auth",
		Payload:   base64.StdEncoding.EncodeToString(authReqData),
		Timestamp: time.Now().Unix(),
		Headers:   map[string]string{},
	}
	if err := conn.WriteJSON(authMessage); err != nil {
		conn.Close()
		return nil, fmt.Errorf("write auth message: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set auth read deadline: %w", err)
	}
	var authResp types.SecureMessage
	if err := conn.ReadJSON(&authResp); err != nil {
		conn.Close()
		return nil, fmt.Errorf("read auth response: %w", err)
	}
	if authResp.Type != types.MessageTypeResponse || authResp.Channel != "auth" {
		conn.Close()
		return nil, fmt.Errorf("unexpected auth response: %+v", authResp)
	}

	return conn, nil
}

func sendAgentMessage(baseURL, agentUID string) (sendMessageResponse, error) {
	req := map[string]interface{}{
		"channel":      "agent." + agentUID + ".policies",
		"message_type": "request",
		"message": map[string]string{
			"policy": "observe",
		},
	}

	var resp sendMessageResponse
	if err := postJSON(baseURL+"/agents/"+agentUID+"/send", req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func postJSON(url string, req interface{}, resp interface{}) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpResp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read response from %s: %w", url, err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("POST %s returned %d: %s", url, httpResp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if err := json.Unmarshal(responseBody, resp); err != nil {
		return fmt.Errorf("decode response from %s: %w", url, err)
	}
	return nil
}

func websocketURL(baseURL, path string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse gateway base URL: %w", err)
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported gateway URL scheme %q", parsed.Scheme)
	}
	parsed.Path = path
	return parsed.String(), nil
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
