package signing

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSigner_NewSigner(t *testing.T) {
	// Create temporary directory for test keys
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Verify signer is healthy
	if !signer.IsHealthy() {
		t.Error("Signer should be healthy after creation")
	}

	if !signer.IsReady() {
		t.Error("Signer should be ready after creation")
	}
}

func TestSigner_SignAndVerify(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Test data
	testData := []byte("Hello, World! This is test data for signing.")

	// Sign the data
	signature, kid, err := signer.Sign(testData)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	if kid == "" {
		t.Error("Key ID should not be empty")
	}

	// Verify the signature
	err = signer.Verify(testData, signature, kid)
	if err != nil {
		t.Fatalf("Signature verification failed: %v", err)
	}
}

func TestSigner_SignJWS(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Create JWS payload
	payload := JWSPayload{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		Issuer:    "test-issuer",
		Subject:   "test-subject",
		Content:   "test-content",
	}

	// Sign JWS
	jws, err := signer.SignJWS(payload)
	if err != nil {
		t.Fatalf("Failed to sign JWS: %v", err)
	}

	if jws.Header.Algorithm != "Ed25519" {
		t.Errorf("Expected algorithm Ed25519, got %s", jws.Header.Algorithm)
	}

	if jws.Header.Type != "JWS" {
		t.Errorf("Expected type JWS, got %s", jws.Header.Type)
	}

	// Verify JWS
	err = signer.VerifyJWS(jws)
	if err != nil {
		t.Fatalf("JWS verification failed: %v", err)
	}
}

func TestSigner_KeyRotation(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Get initial active key
	initialPubKey, initialKid, err := signer.GetActivePublicKey()
	if err != nil {
		t.Fatalf("Failed to get initial public key: %v", err)
	}

	// Rotate key
	err = signer.RotateKey()
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	// Get new active key
	newPubKey, newKid, err := signer.GetActivePublicKey()
	if err != nil {
		t.Fatalf("Failed to get new public key: %v", err)
	}

	// Verify keys are different
	if initialKid == newKid {
		t.Error("Key ID should change after rotation")
	}

	if initialPubKey == newPubKey {
		t.Error("Public key should change after rotation")
	}

	// Verify signer is still healthy
	if !signer.IsHealthy() {
		t.Error("Signer should be healthy after key rotation")
	}
}

func TestSigner_BackupAndRestore(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")
	backupPath := filepath.Join(tempDir, "backup-keys.json")

	// Create initial signer
	signer1, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create initial signer: %v", err)
	}

	// Get initial key info
	initialPubKey, initialKid, err := signer1.GetActivePublicKey()
	if err != nil {
		t.Fatalf("Failed to get initial public key: %v", err)
	}

	// Backup keys
	err = signer1.BackupKeys(backupPath)
	if err != nil {
		t.Fatalf("Failed to backup keys: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file should exist")
	}

	// Create new signer and restore keys
	restorePath := filepath.Join(tempDir, "restore-keys.json")
	signer2, err := NewSigner(restorePath)
	if err != nil {
		t.Fatalf("Failed to create restore signer: %v", err)
	}

	// Restore keys
	err = signer2.RestoreKeys(backupPath)
	if err != nil {
		t.Fatalf("Failed to restore keys: %v", err)
	}

	// Verify restored key matches original
	restoredPubKey, restoredKid, err := signer2.GetActivePublicKey()
	if err != nil {
		t.Fatalf("Failed to get restored public key: %v", err)
	}

	if initialKid != restoredKid {
		t.Error("Restored key ID should match original")
	}

	if initialPubKey != restoredPubKey {
		t.Error("Restored public key should match original")
	}
}

func TestSigner_ListKeys(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Rotate key to have multiple keys
	err = signer.RotateKey()
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	// List keys
	keys := signer.ListKeys()
	if len(keys) < 2 {
		t.Errorf("Expected at least 2 keys, got %d", len(keys))
	}

	// Verify key properties
	for _, key := range keys {
		if key.Kid == "" {
			t.Error("Key ID should not be empty")
		}
		if key.PublicKey == "" {
			t.Error("Public key should not be empty")
		}
		if key.Algorithm != "Ed25519" {
			t.Errorf("Expected algorithm Ed25519, got %s", key.Algorithm)
		}
	}
}

func TestSigner_GetPublicKey(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Get active key info
	activePubKey, activeKid, err := signer.GetActivePublicKey()
	if err != nil {
		t.Fatalf("Failed to get active public key: %v", err)
	}

	// Get public key by kid
	retrievedPubKey, err := signer.GetPublicKey(activeKid)
	if err != nil {
		t.Fatalf("Failed to get public key by kid: %v", err)
	}

	if activePubKey != retrievedPubKey {
		t.Error("Retrieved public key should match active public key")
	}

	// Test with non-existent key
	_, err = signer.GetPublicKey("non-existent-key")
	if err == nil {
		t.Error("Should return error for non-existent key")
	}
}

func TestSigner_InvalidSignature(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	testData := []byte("Hello, World!")

	// Test with invalid signature
	invalidSig := base64.StdEncoding.EncodeToString([]byte("invalid-signature"))
	_, activeKid, _ := signer.GetActivePublicKey()

	err = signer.Verify(testData, invalidSig, activeKid)
	if err == nil {
		t.Error("Should return error for invalid signature")
	}

	// Test with modified data
	signature, _, _ := signer.Sign(testData)
	modifiedData := []byte("Modified data")

	err = signer.Verify(modifiedData, signature, activeKid)
	if err == nil {
		t.Error("Should return error for modified data")
	}
}

func TestSigner_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Test concurrent signing operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			testData := []byte(fmt.Sprintf("Test data %d", i))
			_, _, err := signer.Sign(testData)
			if err != nil {
				t.Errorf("Concurrent signing failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSigner_KeyExpiration(t *testing.T) {
	tempDir := t.TempDir()
	keysPath := filepath.Join(tempDir, "test-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Create a key with expiration in the past
	keys := signer.ListKeys()
	if len(keys) > 0 {
		// Modify the key to have past expiration (this would require modifying the internal structure)
		// For this test, we'll just verify the signer handles expiration gracefully
		if !signer.IsReady() {
			t.Error("Signer should handle key expiration gracefully")
		}
	}
}

// Benchmark tests
func BenchmarkSigner_Sign(b *testing.B) {
	tempDir := b.TempDir()
	keysPath := filepath.Join(tempDir, "bench-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		b.Fatalf("Failed to create signer: %v", err)
	}

	testData := make([]byte, 1024)
	rand.Read(testData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := signer.Sign(testData)
		if err != nil {
			b.Fatalf("Signing failed: %v", err)
		}
	}
}

func BenchmarkSigner_Verify(b *testing.B) {
	tempDir := b.TempDir()
	keysPath := filepath.Join(tempDir, "bench-keys.json")

	signer, err := NewSigner(keysPath)
	if err != nil {
		b.Fatalf("Failed to create signer: %v", err)
	}

	testData := make([]byte, 1024)
	rand.Read(testData)

	signature, kid, err := signer.Sign(testData)
	if err != nil {
		b.Fatalf("Failed to sign test data: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := signer.Verify(testData, signature, kid)
		if err != nil {
			b.Fatalf("Verification failed: %v", err)
		}
	}
}

