package gateway

import (
	"encoding/json"
	"testing"

	"github.com/sgerhart/aegisflux/websocket-gateway/internal/types"
)

func TestParseWebSocketGatewayMessage(t *testing.T) {
	data := []byte(`{
		"id":"msg-1",
		"type":"request",
		"channel":"agent.agent-1.policies",
		"payload":"eyJwb2xpY3kiOiJvYnNlcnZlIn0=",
		"timestamp":1777414711,
		"headers":{"x-test":"true"},
		"target_agent":"agent-1"
	}`)

	msg, err := parseWebSocketGatewayMessage(data)
	if err != nil {
		t.Fatalf("parseWebSocketGatewayMessage returned error: %v", err)
	}

	if msg.ID != "msg-1" {
		t.Fatalf("unexpected id: %q", msg.ID)
	}
	if msg.TargetAgent != "agent-1" {
		t.Fatalf("unexpected target agent: %q", msg.TargetAgent)
	}
	if msg.Channel != "agent.agent-1.policies" {
		t.Fatalf("unexpected channel: %q", msg.Channel)
	}
	if msg.Type != "request" {
		t.Fatalf("unexpected type: %q", msg.Type)
	}
	if msg.Headers["x-test"] != "true" {
		t.Fatalf("unexpected headers: %v", msg.Headers)
	}
}

func TestParseWebSocketGatewayMessageDefaultsTypeAndHeaders(t *testing.T) {
	data := []byte(`{
		"id":"msg-1",
		"channel":"agent.agent-1.status",
		"payload":"e30=",
		"target_agent":"agent-1"
	}`)

	msg, err := parseWebSocketGatewayMessage(data)
	if err != nil {
		t.Fatalf("parseWebSocketGatewayMessage returned error: %v", err)
	}
	if msg.Type != "event" {
		t.Fatalf("expected default type event, got %q", msg.Type)
	}
	if msg.Headers == nil {
		t.Fatalf("expected headers to default to empty map")
	}
}

func TestParseWebSocketGatewayMessageRequiresRoutingFields(t *testing.T) {
	base := map[string]interface{}{
		"id":           "msg-1",
		"channel":      "agent.agent-1.status",
		"payload":      "e30=",
		"target_agent": "agent-1",
	}

	for _, field := range []string{"id", "channel", "payload", "target_agent"} {
		t.Run(field, func(t *testing.T) {
			msg := map[string]interface{}{}
			for k, v := range base {
				msg[k] = v
			}
			delete(msg, field)
			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			if _, err := parseWebSocketGatewayMessage(data); err == nil {
				t.Fatalf("expected validation error for missing %s", field)
			}
		})
	}
}

func TestActionsAPIEndpointUsesConfiguredBaseURL(t *testing.T) {
	wsg := &WebSocketGateway{
		config: &types.Configuration{ActionsAPIURL: "http://actions-api.test:8083/"},
	}

	got := wsg.actionsAPIEndpoint("/agents/heartbeat")
	want := "http://actions-api.test:8083/agents/heartbeat"
	if got != want {
		t.Fatalf("unexpected Actions API endpoint: got %q, want %q", got, want)
	}
}

func TestActionsAPIEndpointDefaultsToComposeService(t *testing.T) {
	wsg := &WebSocketGateway{
		config: &types.Configuration{},
	}

	got := wsg.actionsAPIEndpoint("/agents/register/init")
	want := "http://actions-api:8083/agents/register/init"
	if got != want {
		t.Fatalf("unexpected default Actions API endpoint: got %q, want %q", got, want)
	}
}
