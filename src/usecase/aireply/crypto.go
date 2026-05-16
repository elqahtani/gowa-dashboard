package aireply

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
)

// ErrNoEncryptionKey is returned when AI_ENCRYPTION_KEY is not configured.
var ErrNoEncryptionKey = errors.New("AI_ENCRYPTION_KEY not configured; set 32-byte hex key")

// loadKey decodes the configured hex key, validating it is exactly 32 bytes
// (AES-256).
func loadKey() ([]byte, error) {
	if config.AIEncryptionKey == "" {
		return nil, ErrNoEncryptionKey
	}
	key, err := hex.DecodeString(config.AIEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("AI_ENCRYPTION_KEY must be hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("AI_ENCRYPTION_KEY must decode to 32 bytes (got %d)", len(key))
	}
	return key, nil
}

// Encrypt seals plaintext with AES-256-GCM. Output layout: nonce || ciphertext.
func Encrypt(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	key, err := loadKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt reverses Encrypt.
func Decrypt(blob []byte) (string, error) {
	if len(blob) == 0 {
		return "", nil
	}
	key, err := loadKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(blob) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// MaskKey returns a masked representation safe for display in API responses.
func MaskKey(key string) string {
	if len(key) <= 8 {
		if key == "" {
			return ""
		}
		return "********"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
