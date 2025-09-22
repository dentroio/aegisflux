package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Signer handles Ed25519 signing operations with key rotation support
type Signer struct {
	keysPath string
	keys     *SigningKeys
	mu       sync.RWMutex
}

// SigningKeys represents the signing keys configuration
type SigningKeys struct {
	Active *SigningKey `json:"active"`
	Next   *SigningKey `json:"next,omitempty"`
	Keys   []SigningKey `json:"keys"`
}

// SigningKey represents a single signing key
type SigningKey struct {
	Kid        string    `json:"kid"`
	PublicKey  string    `json:"public_key"`  // Base64 encoded
	PrivateKey string    `json:"private_key"` // Base64 encoded
	Algorithm  string    `json:"algorithm"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}

// JWSHeader represents a JWS header
type JWSHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

// JWSPayload represents a JWS payload
type JWSPayload struct {
	IssuedAt   int64  `json:"iat"`
	ExpiresAt  int64  `json:"exp,omitempty"`
	NotBefore  int64  `json:"nbf,omitempty"`
	Issuer     string `json:"iss,omitempty"`
	Subject    string `json:"sub,omitempty"`
	Audience   string `json:"aud,omitempty"`
	Content    string `json:"content"`
}

// JWS represents a JSON Web Signature
type JWS struct {
	Header    JWSHeader  `json:"header"`
	Payload   JWSPayload `json:"payload"`
	Signature string     `json:"signature"`
}

// NewSigner creates a new signer instance
func NewSigner(keysPath string) (*Signer, error) {
	s := &Signer{
		keysPath: keysPath,
	}
	
	// Load existing keys or create new ones
	if err := s.loadKeys(); err != nil {
		if os.IsNotExist(err) {
			// Create new keys if file doesn't exist
			if err := s.generateNewKeys(); err != nil {
				return nil, fmt.Errorf("failed to generate new keys: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load keys: %w", err)
		}
	}
	
	return s, nil
}

// loadKeys loads signing keys from file
func (s *Signer) loadKeys() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	data, err := os.ReadFile(s.keysPath)
	if err != nil {
		return err
	}
	
	var keys SigningKeys
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("failed to unmarshal keys: %w", err)
	}
	
	s.keys = &keys
	return nil
}

// saveKeys saves signing keys to file
func (s *Signer) saveKeys() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, err := json.MarshalIndent(s.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(s.keysPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}
	
	// Write with restricted permissions
	if err := os.WriteFile(s.keysPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write keys file: %w", err)
	}
	
	return nil
}

// generateNewKeys generates a new set of signing keys
func (s *Signer) generateNewKeys() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	kid := generateKeyID()
	
	// Generate Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}
	
	now := time.Now()
	key := SigningKey{
		Kid:        kid,
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey),
		Algorithm:  "Ed25519",
		CreatedAt:  now,
		ExpiresAt:  now.AddDate(1, 0, 0), // 1 year from now
	}
	
	s.keys = &SigningKeys{
		Active: &key,
		Keys:   []SigningKey{key},
	}
	
	return s.saveKeys()
}

// generateKeyID generates a unique key ID
func generateKeyID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("key-%x", bytes)
}

// Sign signs data using the active key
func (s *Signer) Sign(data []byte) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.keys == nil || s.keys.Active == nil {
		return "", "", fmt.Errorf("no active signing key available")
	}
	
	// Decode private key
	privateKeyBytes, err := base64.StdEncoding.DecodeString(s.keys.Active.PrivateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode private key: %w", err)
	}
	
	// Create Ed25519 private key
	privateKey := ed25519.PrivateKey(privateKeyBytes)
	
	// Sign the data
	signature := ed25519.Sign(privateKey, data)
	signatureB64 := base64.StdEncoding.EncodeToString(signature)
	
	return signatureB64, s.keys.Active.Kid, nil
}

// SignJWS creates a JWS signature for the given payload
func (s *Signer) SignJWS(payload JWSPayload) (*JWS, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.keys == nil || s.keys.Active == nil {
		return nil, fmt.Errorf("no active signing key available")
	}
	
	// Set issued at if not set
	if payload.IssuedAt == 0 {
		payload.IssuedAt = time.Now().Unix()
	}
	
	// Create JWS header
	header := JWSHeader{
		Algorithm: "Ed25519",
		KeyID:     s.keys.Active.Kid,
		Type:      "JWS",
	}
	
	// Encode header and payload
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal header: %w", err)
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	
	// Create signing input (header.payload)
	signingInput := headerB64 + "." + payloadB64
	
	// Sign the input
	signature, _, err := s.Sign([]byte(signingInput))
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWS: %w", err)
	}
	
	// Decode signature to raw URL encoding for JWS
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}
	signatureB64 := base64.RawURLEncoding.EncodeToString(signatureBytes)
	
	return &JWS{
		Header:    header,
		Payload:   payload,
		Signature: signatureB64,
	}, nil
}

// Verify verifies a signature against data using the specified key ID
func (s *Signer) Verify(data []byte, signature, kid string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Find the key by kid
	var key *SigningKey
	for _, k := range s.keys.Keys {
		if k.Kid == kid {
			key = &k
			break
		}
	}
	
	if key == nil {
		return fmt.Errorf("key not found: %s", kid)
	}
	
	// Decode public key
	publicKeyBytes, err := base64.StdEncoding.DecodeString(key.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}
	
	// Decode signature
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	
	// Verify signature
	if !ed25519.Verify(ed25519.PublicKey(publicKeyBytes), data, signatureBytes) {
		return fmt.Errorf("signature verification failed")
	}
	
	return nil
}

// VerifyJWS verifies a JWS signature
func (s *Signer) VerifyJWS(jws *JWS) error {
	// Encode header and payload
	headerBytes, err := json.Marshal(jws.Header)
	if err != nil {
		return fmt.Errorf("failed to marshal header: %w", err)
	}
	
	payloadBytes, err := json.Marshal(jws.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	
	// Create signing input (header.payload)
	signingInput := headerB64 + "." + payloadB64
	
	// Decode signature
	signatureBytes, err := base64.RawURLEncoding.DecodeString(jws.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	signature := base64.StdEncoding.EncodeToString(signatureBytes)
	
	// Verify signature
	return s.Verify([]byte(signingInput), signature, jws.Header.KeyID)
}

// GetActivePublicKey returns the active public key
func (s *Signer) GetActivePublicKey() (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.keys == nil || s.keys.Active == nil {
		return "", "", fmt.Errorf("no active signing key available")
	}
	
	return s.keys.Active.PublicKey, s.keys.Active.Kid, nil
}

// GetPublicKey returns a public key by kid
func (s *Signer) GetPublicKey(kid string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, key := range s.keys.Keys {
		if key.Kid == kid {
			return key.PublicKey, nil
		}
	}
	
	return "", fmt.Errorf("key not found: %s", kid)
}

// RotateKey rotates the signing key
func (s *Signer) RotateKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.keys == nil {
		return fmt.Errorf("no keys loaded")
	}
	
	// Generate new key
	kid := generateKeyID()
	
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate new key pair: %w", err)
	}
	
	now := time.Now()
	newKey := SigningKey{
		Kid:        kid,
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey),
		Algorithm:  "Ed25519",
		CreatedAt:  now,
		ExpiresAt:  now.AddDate(1, 0, 0), // 1 year from now
	}
	
	// Set next key and add to keys list
	s.keys.Next = &newKey
	s.keys.Keys = append(s.keys.Keys, newKey)
	
	// Save keys
	if err := s.saveKeys(); err != nil {
		return fmt.Errorf("failed to save keys after rotation: %w", err)
	}
	
	return nil
}

// ActivateNextKey activates the next key as the active key
func (s *Signer) ActivateNextKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.keys == nil || s.keys.Next == nil {
		return fmt.Errorf("no next key available for activation")
	}
	
	// Move next key to active
	s.keys.Active = s.keys.Next
	s.keys.Next = nil
	
	// Save keys
	if err := s.saveKeys(); err != nil {
		return fmt.Errorf("failed to save keys after activation: %w", err)
	}
	
	return nil
}

// ListKeys returns all available keys
func (s *Signer) ListKeys() []SigningKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.keys == nil {
		return nil
	}
	
	return s.keys.Keys
}

// IsHealthy checks if the signer is healthy
func (s *Signer) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.keys != nil && s.keys.Active != nil
}

// IsReady checks if the signer is ready for operations
func (s *Signer) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.keys == nil || s.keys.Active == nil {
		return false
	}
	
	// Check if active key is not expired
	now := time.Now()
	return s.keys.Active.ExpiresAt.IsZero() || s.keys.Active.ExpiresAt.After(now)
}

// BackupKeys creates a backup of the signing keys
func (s *Signer) BackupKeys(backupPath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.keys == nil {
		return fmt.Errorf("no keys to backup")
	}
	
	data, err := json.MarshalIndent(s.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys for backup: %w", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// Write backup with restricted permissions
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	
	return nil
}

// RestoreKeys restores signing keys from a backup
func (s *Signer) RestoreKeys(backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	
	var keys SigningKeys
	if err := json.Unmarshal(data, &keys); err != nil {
		return fmt.Errorf("failed to unmarshal backup keys: %w", err)
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.keys = &keys
	
	// Save to the configured path
	if err := s.saveKeys(); err != nil {
		return fmt.Errorf("failed to save restored keys: %w", err)
	}
	
	return nil
}

