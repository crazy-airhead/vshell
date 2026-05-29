package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	once     sync.Once
	instance *CryptoService
)

type CryptoService struct {
	gcm cipher.AEAD
}

func New() *CryptoService {
	once.Do(func() {
		key := deriveKey()
		block, err := aes.NewCipher(key)
		if err != nil {
			panic("crypto: failed to create AES cipher: " + err.Error())
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			panic("crypto: failed to create GCM: " + err.Error())
		}
		instance = &CryptoService{gcm: gcm}
	})
	return instance
}

func deriveKey() []byte {
	if key := os.Getenv("VSHELL_ENCRYPTION_KEY"); key != "" {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err == nil && len(decoded) == 32 {
			return decoded
		}
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		panic("crypto: failed to get config dir: " + err.Error())
	}
	keyPath := configDir + "/vshell/.enc_key"

	// Try to load existing key
	if data, err := os.ReadFile(keyPath); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil && len(decoded) == 32 {
			return decoded
		}
	}

	// Generate new key and persist it
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic("crypto: failed to generate key: " + err.Error())
	}
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		panic("crypto: failed to create config dir: " + err.Error())
	}
	if err := os.WriteFile(keyPath, []byte(encoded), 0600); err != nil {
		panic("crypto: failed to save key: " + err.Error())
	}
	return key
}

func (c *CryptoService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (c *CryptoService) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	nonceSize := c.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
