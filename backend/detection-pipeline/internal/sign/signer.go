package sign

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

const keyEnv = "DETECTION_PIPELINE_ED25519_SEED_HEX"

// Signer produces ed25519 detached signatures over pack payloads.
type Signer struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	keyID string
}

func NewSigner(keyID string) (*Signer, error) {
	if keyID == "" {
		keyID = "detection-pipeline-dev"
	}
	seedHex := os.Getenv(keyEnv)
	var seed []byte
	var err error
	if seedHex != "" {
		seed, err = hex.DecodeString(seedHex)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", keyEnv, err)
		}
		if len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("%s must decode to %d bytes", keyEnv, ed25519.SeedSize)
		}
	} else {
		seed = make([]byte, ed25519.SeedSize)
		if _, err := rand.Read(seed); err != nil {
			return nil, err
		}
	}
	priv := ed25519.NewKeyFromSeed(seed)
	return &Signer{priv: priv, pub: priv.Public().(ed25519.PublicKey), keyID: keyID}, nil
}

// PublicKeyBase64 returns URL-safe base64 of raw public key (for operators).
func (s *Signer) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(s.pub)
}

// SignPackMessage signs sha256(domain || canonicalJSON) for stable message size.
func (s *Signer) SignPackMessage(canonicalJSON []byte) (string, error) {
	h := sha256.Sum256(canonicalJSON)
	msg := append([]byte("aegis.detection_pack.v1\x00"), h[:]...)
	sig := ed25519.Sign(s.priv, msg)
	return base64.StdEncoding.EncodeToString(sig), nil
}

func (s *Signer) KeyID() string { return s.keyID }

// AttachSignature sets signature on pack map and returns canonical JSON bytes of the signed pack.
func AttachSignature(pack map[string]any, sigB64 string, keyID string) ([]byte, error) {
	pack["signature"] = map[string]any{
		"algorithm":  "ed25519",
		"key_id":     keyID,
		"value_b64":  sigB64,
	}
	return json.Marshal(pack)
}
