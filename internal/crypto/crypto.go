package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encryptor handles AES-GCM encryption and decryption
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates a new encryptor with the given key (32 bytes for AES-256)
func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		return nil, errors.New("encryption key cannot be empty")
	}

	// Ensure key is 32 bytes (AES-256)
	// If shorter/longer, we might hash it or pad it, but for now we expect a proper key or at least use what's given if it fits 16/24/32
	// A robust way is to hash the key to get 32 bytes if it's a passphrase.
	// For simplicity, let's assume the env var is a 32-byte key or we pad/truncate?
	// Better: Use SHA-256 of the key to ensure 32 bytes always.
	// BUT, if the user provides a hex key, we should probably decode it.
	// Let's just use the key bytes directly if length is valid, otherwise error?
	// No, let's just error if it's too short.
	
	keyBytes := []byte(key)
	if len(keyBytes) != 16 && len(keyBytes) != 24 && len(keyBytes) != 32 {
		return nil, errors.New("encryption key must be 16, 24, or 32 bytes long")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plain text and returns a base64 encoded string
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded string
func (e *Encryptor) Decrypt(encodedCiphertext string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", err
	}

	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
