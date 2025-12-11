package codec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

const (
	// MetadataEncodingEncrypted is the encoding type for encrypted payloads
	MetadataEncodingEncrypted = "binary/encrypted"
)

// EncryptionCodec implements converter.PayloadCodec for encrypting/decrypting workflow data
type EncryptionCodec struct {
	key []byte
}

// NewEncryptionCodec creates a new encryption codec with the provided key
// The key should be 32 bytes for AES-256
func NewEncryptionCodec(key []byte) (*EncryptionCodec, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d bytes", len(key))
	}

	return &EncryptionCodec{
		key: key,
	}, nil
}

// Encode encrypts the provided payloads
func (e *EncryptionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, payload := range payloads {
		// Skip if already encrypted
		if payload.Metadata != nil && string(payload.Metadata["encoding"]) == MetadataEncodingEncrypted {
			result[i] = payload
			continue
		}

		// Marshal the entire payload (including metadata) to bytes
		origBytes, err := payload.Marshal()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}

		// Encrypt the marshaled payload
		encrypted, err := e.encrypt(origBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt payload: %w", err)
		}

		// Create new payload with encrypted data
		result[i] = &commonpb.Payload{
			Metadata: map[string][]byte{
				"encoding": []byte(MetadataEncodingEncrypted),
			},
			Data: encrypted,
		}
	}

	return result, nil
}

// Decode decrypts the provided payloads
func (e *EncryptionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))

	for i, payload := range payloads {
		// Skip if not encrypted
		if payload.Metadata == nil || string(payload.Metadata["encoding"]) != MetadataEncodingEncrypted {
			result[i] = payload
			continue
		}

		// Decrypt the data
		decrypted, err := e.decrypt(payload.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt payload: %w", err)
		}

		// Unmarshal the decrypted bytes back to a Payload
		result[i] = &commonpb.Payload{}
		if err := result[i].Unmarshal(decrypted); err != nil {
			return nil, fmt.Errorf("failed to unmarshal decrypted payload: %w", err)
		}
	}

	return result, nil
}

// encrypt encrypts data using AES-GCM
func (e *EncryptionCodec) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (e *EncryptionCodec) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// NewEncryptionDataConverter creates a data converter with encryption codec
func NewEncryptionDataConverter(key []byte) (converter.DataConverter, error) {
	codec, err := NewEncryptionCodec(key)
	if err != nil {
		return nil, err
	}

	return converter.NewCodecDataConverter(
		converter.GetDefaultDataConverter(),
		codec,
	), nil
}
