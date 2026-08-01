package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("secret mpgs credentials")

	encrypted, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	require.NotEqual(t, plaintext, encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestEncryptDecryptBase64(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte(`{"merchant_id":"TEST123","api_password":"secret"}`)

	encrypted, err := EncryptToBase64(plaintext, key)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	decrypted, err := DecryptFromBase64(encrypted, key)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestInvalidKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("test"), []byte("short"))
	require.ErrorIs(t, err, ErrInvalidKey)

	_, err = Decrypt([]byte("test"), []byte("short"))
	require.ErrorIs(t, err, ErrInvalidKey)
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	// Too short ciphertext
	_, err = Decrypt([]byte("short"), key)
	require.Error(t, err)
}

func TestDecryptFromBase64InvalidBase64(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	_, err = DecryptFromBase64("not-valid-base64!!!", key)
	require.Error(t, err)
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("same plaintext")

	encrypted1, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	encrypted2, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Due to random nonce, ciphertexts should be different
	require.NotEqual(t, encrypted1, encrypted2)
}
