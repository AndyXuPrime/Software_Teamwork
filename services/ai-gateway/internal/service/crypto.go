package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const StorageModeEncryptedColumn = "encrypted_column"

type CredentialEncryptor struct {
	aead       cipher.AEAD
	keyVersion string
}

func NewCredentialEncryptor(key []byte, keyVersion string) (*CredentialEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("credential encryption key must be 32 bytes")
	}
	if strings.TrimSpace(keyVersion) == "" {
		return nil, fmt.Errorf("credential encryption key version is required")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &CredentialEncryptor{aead: aead, keyVersion: strings.TrimSpace(keyVersion)}, nil
}

func (e *CredentialEncryptor) Encrypt(apiKey string) (ProviderCredential, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return ProviderCredential{}, fmt.Errorf("api key is required")
	}
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return ProviderCredential{}, err
	}
	ciphertext := e.aead.Seal(nil, nonce, []byte(apiKey), nil)
	fingerprint := sha256.Sum256([]byte(apiKey))
	return ProviderCredential{
		StorageMode:          StorageModeEncryptedColumn,
		Ciphertext:           ciphertext,
		Nonce:                nonce,
		EncryptionKeyVersion: e.keyVersion,
		FingerprintSHA256:    hex.EncodeToString(fingerprint[:]),
		KeyLast4:             last4(apiKey),
		Status:               CredentialActive,
	}, nil
}

func (e *CredentialEncryptor) Decrypt(credential ProviderCredential) (string, error) {
	if credential.StorageMode != StorageModeEncryptedColumn {
		return "", fmt.Errorf("unsupported credential storage mode")
	}
	plain, err := e.aead.Open(nil, credential.Nonce, credential.Ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func last4(value string) string {
	if len(value) <= 4 {
		return value
	}
	return value[len(value)-4:]
}
