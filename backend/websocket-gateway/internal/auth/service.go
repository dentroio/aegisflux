package auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sgerhart/aegisflux/websocket-gateway/internal/types"
)

// AuthService handles agent authentication and session management
type AuthService struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	config     *types.Configuration
}

// NewAuthService creates a new authentication service
func NewAuthService(config *types.Configuration) (*AuthService, error) {
	// TODO: Load keys from file paths in config
	// For now, generate new keys (in production, load from secure storage)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 keys: %w", err)
	}

	service := &AuthService{
		privateKey: privateKey,
		publicKey:  publicKey,
		config:     config,
	}

	return service, nil
}

// DecryptAuthenticationRequest decrypts an authentication request message
func (as *AuthService) DecryptAuthenticationRequest(message types.SecureMessage) (*types.AuthenticationRequest, error) {
	// TODO: Implement proper decryption with shared key
	// For now, assume payload is base64 encoded JSON
	payload, err := base64.StdEncoding.DecodeString(message.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var authReq types.AuthenticationRequest
	if err := json.Unmarshal(payload, &authReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal authentication request: %w", err)
	}

	return &authReq, nil
}

// AuthenticateAgent authenticates an agent using Ed25519 signature verification
func (as *AuthService) AuthenticateAgent(req *types.AuthenticationRequest, agentPublicKey ed25519.PublicKey) (*types.AuthenticationResponse, error) {
	// Decode agent's public key from request
	reqPublicKey, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		return &types.AuthenticationResponse{
			Success: false,
			Message: "Invalid public key format",
		}, nil
	}

	// Verify public key matches
	if !ed25519.PublicKey(reqPublicKey).Equal(agentPublicKey) {
		return &types.AuthenticationResponse{
			Success: false,
			Message: "Public key mismatch",
		}, nil
	}

	// Verify signature
	if !as.verifySignature(req, agentPublicKey) {
		return &types.AuthenticationResponse{
			Success: false,
			Message: "Invalid signature",
		}, nil
	}

	// Generate session token
	sessionToken, err := as.generateSessionToken(req.AgentID)
	if err != nil {
		return &types.AuthenticationResponse{
			Success: false,
			Message: "Failed to generate session token",
		}, nil
	}

	// Generate backend public key for shared key derivation
	backendKey := base64.StdEncoding.EncodeToString(as.publicKey)

	// Calculate expiration time
	expiresAt := time.Now().Add(as.config.SessionTimeout).Unix()

	return &types.AuthenticationResponse{
		Success:      true,
		BackendKey:   backendKey,
		SessionToken: sessionToken,
		ExpiresAt:    expiresAt,
		Message:      "Authentication successful",
	}, nil
}

// EncryptAuthenticationResponse encrypts an authentication response
func (as *AuthService) EncryptAuthenticationResponse(resp *types.AuthenticationResponse) (*types.SecureMessage, error) {
	// TODO: Implement proper encryption with shared key
	// For now, encode as base64 JSON
	respData, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	payload := base64.StdEncoding.EncodeToString(respData)

	// Generate nonce (in production, use crypto/rand)
	nonce := make([]byte, 12)
	nonceStr := base64.StdEncoding.EncodeToString(nonce)

	// Create secure message
	message := &types.SecureMessage{
		Payload:   payload,
		Nonce:     nonceStr,
		Timestamp: time.Now().Unix(),
	}

	// Sign message
	signature, err := as.signMessage(message)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}
	message.Signature = signature

	return message, nil
}

// verifySignature verifies an Ed25519 signature
func (as *AuthService) verifySignature(req *types.AuthenticationRequest, publicKey ed25519.PublicKey) bool {
	// Create data to verify: agent_id:public_key:timestamp:nonce
	data := fmt.Sprintf("%s:%s:%d:%s", req.AgentID, req.PublicKey, req.Timestamp, req.Nonce)

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		return false
	}

	// Verify signature
	return ed25519.Verify(publicKey, []byte(data), signature)
}

// signMessage signs a message with the backend's private key
func (as *AuthService) signMessage(message *types.SecureMessage) (string, error) {
	// Create data to sign: id:type:channel:timestamp:payload
	data := fmt.Sprintf("%s:%s:%s:%d:%s", 
		message.ID, message.Type, message.Channel, message.Timestamp, message.Payload)

	// Sign the data
	signature := ed25519.Sign(as.privateKey, []byte(data))

	return base64.StdEncoding.EncodeToString(signature), nil
}

// generateSessionToken generates a JWT session token
func (as *AuthService) generateSessionToken(agentID string) (string, error) {
	// Create JWT claims
	claims := jwt.MapClaims{
		"agent_id":   agentID,
		"issued_at":  time.Now().Unix(),
		"expires_at": time.Now().Add(as.config.SessionTimeout).Unix(),
		"type":       "session",
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with a secret (in production, use a proper secret)
	secret := []byte("aegisflux-websocket-secret") // TODO: Load from config
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateSessionToken validates a JWT session token
func (as *AuthService) ValidateSessionToken(tokenString string) (*jwt.MapClaims, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("aegisflux-websocket-secret"), nil // TODO: Load from config
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return &claims, nil
	}

	return nil, fmt.Errorf("failed to extract claims")
}

// DeriveSharedKey derives a shared encryption key using Ed25519 key agreement
func (as *AuthService) DeriveSharedKey(agentPublicKey ed25519.PublicKey) []byte {
	// Simple key derivation: SHA256(backend_private_key + agent_public_key)
	combined := append(as.privateKey, agentPublicKey...)
	hash := sha256.Sum256(combined)
	return hash[:]
}

// GetBackendPublicKey returns the backend's public key
func (as *AuthService) GetBackendPublicKey() ed25519.PublicKey {
	return as.publicKey
}

// GetBackendPrivateKey returns the backend's private key
func (as *AuthService) GetBackendPrivateKey() ed25519.PrivateKey {
	return as.privateKey
}

