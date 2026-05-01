package api

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestBuildWebSocketGatewayMessage(t *testing.T) {
	now := time.Unix(1777414711, 0)
	data, err := buildWebSocketGatewayMessage(
		"agent-1",
		"agent.agent-1.policies",
		map[string]interface{}{"policy": "observe"},
		"request",
		"msg-1",
		now,
	)
	if err != nil {
		t.Fatalf("buildWebSocketGatewayMessage returned error: %v", err)
	}

	var msg websocketGatewayMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("failed to unmarshal gateway message: %v", err)
	}

	if msg.ID != "msg-1" {
		t.Fatalf("expected id msg-1, got %q", msg.ID)
	}
	if msg.TargetAgent != "agent-1" {
		t.Fatalf("expected target agent agent-1, got %q", msg.TargetAgent)
	}
	if msg.Channel != "agent.agent-1.policies" {
		t.Fatalf("unexpected channel: %q", msg.Channel)
	}
	if msg.Type != "request" {
		t.Fatalf("unexpected type: %q", msg.Type)
	}
	if msg.Timestamp != now.Unix() {
		t.Fatalf("unexpected timestamp: %d", msg.Timestamp)
	}

	payload, err := base64.StdEncoding.DecodeString(msg.Payload)
	if err != nil {
		t.Fatalf("payload is not valid base64: %v", err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
	if decoded["policy"] != "observe" {
		t.Fatalf("unexpected payload: %v", decoded)
	}
}

func TestBuildWebSocketGatewayMessageDefaultsType(t *testing.T) {
	data, err := buildWebSocketGatewayMessage("agent-1", "agent.agent-1.status", nil, "", "msg-1", time.Unix(1, 0))
	if err != nil {
		t.Fatalf("buildWebSocketGatewayMessage returned error: %v", err)
	}

	var msg websocketGatewayMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("failed to unmarshal gateway message: %v", err)
	}
	if msg.Type != "event" {
		t.Fatalf("expected default type event, got %q", msg.Type)
	}
}

func TestBuildWebSocketGatewayMessageRequiresRoutingFields(t *testing.T) {
	tests := []struct {
		name      string
		agentUID  string
		channel   string
		messageID string
	}{
		{name: "agent", channel: "agent.agent-1.status", messageID: "msg-1"},
		{name: "channel", agentUID: "agent-1", messageID: "msg-1"},
		{name: "message ID", agentUID: "agent-1", channel: "agent.agent-1.status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildWebSocketGatewayMessage(tt.agentUID, tt.channel, nil, "event", tt.messageID, time.Unix(1, 0))
			if err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}
