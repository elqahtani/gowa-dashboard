package aireply

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
)

func withTestKey(t *testing.T) {
	t.Helper()
	prev := config.AIEncryptionKey
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	config.AIEncryptionKey = hex.EncodeToString(key)
	t.Cleanup(func() { config.AIEncryptionKey = prev })
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	withTestKey(t)
	plain := "sk-very-secret-api-key-12345"
	enc, err := Encrypt(plain)
	assert.NoError(t, err)
	assert.NotEmpty(t, enc)
	dec, err := Decrypt(enc)
	assert.NoError(t, err)
	assert.Equal(t, plain, dec)
}

func TestEncrypt_Empty(t *testing.T) {
	withTestKey(t)
	enc, err := Encrypt("")
	assert.NoError(t, err)
	assert.Nil(t, enc)
}

func TestEncrypt_NoKey(t *testing.T) {
	prev := config.AIEncryptionKey
	config.AIEncryptionKey = ""
	t.Cleanup(func() { config.AIEncryptionKey = prev })
	_, err := Encrypt("x")
	assert.ErrorIs(t, err, ErrNoEncryptionKey)
}

func TestMaskKey(t *testing.T) {
	assert.Equal(t, "", MaskKey(""))
	assert.Equal(t, "********", MaskKey("short"))
	assert.Equal(t, "sk-1****cdef", MaskKey("sk-1abcdef"))
}
