package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"
)

// Key is the 32-byte AES key (same as legacy C# implementation)
var Key = []byte("12345678901234561234567890123456")

// Encrypt encrypts plainText using AES-256-CBC with a random IV prepended
func Encrypt(plainText string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Pad plaintext to block size
	plainBytes := pkcs7Pad([]byte(plainText), aes.BlockSize)

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	// Encrypt
	cipherText := make([]byte, len(plainBytes))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, plainBytes)

	// Prepend IV to ciphertext (same format as C# implementation)
	combined := append(iv, cipherText...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts base64 cipherText with IV prepended
func Decrypt(cipherText string, key []byte) (string, error) {
	combined, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(combined) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract IV (first 16 bytes)
	iv := combined[:aes.BlockSize]
	encryptedBytes := combined[aes.BlockSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Decrypt
	mode := cipher.NewCBCDecrypter(block, iv)
	plainBytes := make([]byte, len(encryptedBytes))
	mode.CryptBlocks(plainBytes, encryptedBytes)

	// Remove padding
	plainBytes, err = pkcs7Unpad(plainBytes)
	if err != nil {
		return "", fmt.Errorf("failed to unpad: %w", err)
	}

	return string(plainBytes), nil
}

// GerarSenha generates the daily password using the same algorithm as legacy C#
// Formula: ((day * 100 + month) * day) % 10000, padded to 4 digits
func GerarSenha() string {
	now := time.Now()
	day := now.Day()
	month := int(now.Month())
	result := ((day*100 + month) * day) % 10000
	return fmt.Sprintf("%04d", result)
}

// pkcs7Pad pads data to blockSize using PKCS7
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := make([]byte, padding)
	for i := range padText {
		padText[i] = byte(padding)
	}
	return append(data, padText...)
}

// pkcs7Unpad removes PKCS7 padding
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding == 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, fmt.Errorf("invalid padding bytes")
		}
	}
	return data[:len(data)-padding], nil
}
