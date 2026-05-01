package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const websocketMessagesSubject = "websocket.messages"

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

type websocketGatewayMessage struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Channel     string            `json:"channel"`
	Payload     string            `json:"payload"`
	Timestamp   int64             `json:"timestamp"`
	Headers     map[string]string `json:"headers"`
	TargetAgent string            `json:"target_agent"`
}

func main() {
	actionsBaseURL := strings.TrimRight(getenv("ACTIONS_API_BASE_URL", "http://localhost:8083"), "/")
	natsURL := getenv("NATS_URL", "nats://localhost:14222")
	hostID := fmt.Sprintf("send-smoke-%d", time.Now().UnixNano())

	agentUID, err := registerAgent(actionsBaseURL, hostID)
	if err != nil {
		fatal(err)
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fatal(fmt.Errorf("connect to NATS: %w", err))
	}
	defer nc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgCh := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe(websocketMessagesSubject, msgCh)
	if err != nil {
		fatal(fmt.Errorf("subscribe to %s: %w", websocketMessagesSubject, err))
	}
	defer sub.Unsubscribe()
	if err := nc.Flush(); err != nil {
		fatal(fmt.Errorf("flush NATS subscription: %w", err))
	}

	sendResponse, err := sendAgentMessage(actionsBaseURL, agentUID)
	if err != nil {
		fatal(err)
	}
	if sendResponse.Status != "sent" {
		fatal(fmt.Errorf("expected send status sent, got %q: %s", sendResponse.Status, sendResponse.Error))
	}

	var natsMsg *nats.Msg
	select {
	case natsMsg = <-msgCh:
	case <-ctx.Done():
		fatal(fmt.Errorf("timed out waiting for %s", websocketMessagesSubject))
	}

	var gatewayMsg websocketGatewayMessage
	if err := json.Unmarshal(natsMsg.Data, &gatewayMsg); err != nil {
		fatal(fmt.Errorf("decode gateway message: %w", err))
	}

	if gatewayMsg.ID != sendResponse.MessageID {
		fatal(fmt.Errorf("message id mismatch: send=%q nats=%q", sendResponse.MessageID, gatewayMsg.ID))
	}
	if gatewayMsg.TargetAgent != agentUID {
		fatal(fmt.Errorf("target_agent mismatch: got %q want %q", gatewayMsg.TargetAgent, agentUID))
	}
	if gatewayMsg.Channel != "agent."+agentUID+".policies" {
		fatal(fmt.Errorf("unexpected channel: %q", gatewayMsg.Channel))
	}
	if gatewayMsg.Type != "request" {
		fatal(fmt.Errorf("unexpected message type: %q", gatewayMsg.Type))
	}

	payload, err := base64.StdEncoding.DecodeString(gatewayMsg.Payload)
	if err != nil {
		fatal(fmt.Errorf("payload is not base64: %w", err))
	}
	var decodedPayload map[string]string
	if err := json.Unmarshal(payload, &decodedPayload); err != nil {
		fatal(fmt.Errorf("payload is not JSON: %w", err))
	}
	if decodedPayload["policy"] != "observe" {
		fatal(fmt.Errorf("unexpected payload: %v", decodedPayload))
	}

	printJSON(map[string]string{
		"ok":              "true",
		"agent_uid":       agentUID,
		"message_id":      sendResponse.MessageID,
		"nats_subject":    natsMsg.Subject,
		"validated_field": "target_agent/channel/payload",
	})
}

func registerAgent(baseURL, hostID string) (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate agent key: %w", err)
	}

	initReq := map[string]interface{}{
		"org_id":        "smoke-org",
		"host_id":       hostID,
		"agent_pubkey":  base64.StdEncoding.EncodeToString(pub),
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
	signature := ed25519.Sign(priv, signingPayload)

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
