package codec

import (
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"github.com/aswathylr-builds/temporal-order-processing/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptionCodec(t *testing.T) {
	// Create a test key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	codec, err := NewEncryptionCodec(key)
	require.NoError(t, err)

	// Create a test payload (simulating what Temporal's default converter creates)
	originalPayload := &commonpb.Payload{
		Metadata: map[string][]byte{
			"encoding": []byte("json/plain"),
		},
		Data: []byte(`{"ID":"TEST-001","Amount":100.0}`),
	}

	// Encrypt
	encrypted, err := codec.Encode([]*commonpb.Payload{originalPayload})
	require.NoError(t, err)
	require.Len(t, encrypted, 1)

	// Verify it's encrypted
	assert.Equal(t, MetadataEncodingEncrypted, string(encrypted[0].Metadata["encoding"]))
	assert.NotEqual(t, originalPayload.Data, encrypted[0].Data)

	// Decrypt
	decrypted, err := codec.Decode(encrypted)
	require.NoError(t, err)
	require.Len(t, decrypted, 1)

	// Verify it matches original
	assert.Equal(t, originalPayload.Data, decrypted[0].Data)
	assert.Equal(t, "json/plain", string(decrypted[0].Metadata["encoding"]))
}

func TestEncryptionDataConverter(t *testing.T) {
	// Create a test key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	// Create encryption data converter
	encryptionDC, err := NewEncryptionDataConverter(key)
	require.NoError(t, err)

	// Create a test order
	order := models.Order{
		ID:        "TEST-001",
		Items:     []string{"item1", "item2"},
		Amount:    100.0,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Convert to payloads (this should encrypt)
	payloads, err := encryptionDC.ToPayloads(order)
	require.NoError(t, err)
	require.NotNil(t, payloads)
	require.Len(t, payloads.Payloads, 1)

	// Verify it's encrypted
	assert.Equal(t, MetadataEncodingEncrypted, string(payloads.Payloads[0].Metadata["encoding"]))

	// Convert back from payloads (this should decrypt)
	var decodedOrder models.Order
	err = encryptionDC.FromPayloads(payloads, &decodedOrder)
	require.NoError(t, err)

	// Verify it matches original
	assert.Equal(t, order.ID, decodedOrder.ID)
	assert.Equal(t, order.Items, decodedOrder.Items)
	assert.Equal(t, order.Amount, decodedOrder.Amount)
	assert.Equal(t, order.Status, decodedOrder.Status)
}
