// Package crypto provides AES-256-CBC encryption with HMAC-SHA256 authentication
// for the .intunewin package format.
//
// The encryption format follows the authenticated encryption scheme used by
// Microsoft's Win32 Content Prep Tool:
// - AES-256-CBC for encryption
// - HMAC-SHA256 for message authentication
// - Separate encryption and MAC keys (both 256-bit)
//
// File structure after encryption:
// [HMAC (32 bytes)][IV (16 bytes)][Encrypted Data]
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

const (
	// AES256KeySize is the key size for AES-256 encryption (32 bytes)
	AES256KeySize = 32
	// IVSize is the initialization vector size for AES-CBC (16 bytes)
	IVSize = 16
	// HMACSize is the size of HMAC-SHA256 output (32 bytes)
	HMACSize = 32
)

// EncryptionInfo contains all the cryptographic parameters needed
// for encrypting and decrypting the inner IntuneWin package
type EncryptionInfo struct {
	// EncryptionKey is the AES-256 encryption key (32 bytes)
	EncryptionKey []byte
	// MacKey is the HMAC-SHA256 key (32 bytes)
	MacKey []byte
	// IV is the initialization vector for AES-CBC (16 bytes)
	IV []byte
	// MAC is the HMAC-SHA256 of the encrypted content (32 bytes)
	MAC []byte
	// FileDigest is the SHA256 hash of the unencrypted content
	FileDigest []byte
	// UnencryptedSize is the size of the original unencrypted content
	UnencryptedSize int64
}

// EncryptionInfoBase64 returns the encryption info with values encoded as base64
type EncryptionInfoBase64 struct {
	EncryptionKey   string
	MacKey          string
	IV              string
	MAC             string
	FileDigest      string
	UnencryptedSize int64
}

// GenerateKey generates a cryptographically secure random key of the specified size
func GenerateKey(size int) ([]byte, error) {
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return key, nil
}

// GenerateIV generates a cryptographically secure random initialization vector
func GenerateIV() ([]byte, error) {
	return GenerateKey(IVSize)
}

// ComputeSHA256 computes the SHA256 hash of the provided data
func ComputeSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// ComputeHMACSHA256 computes the HMAC-SHA256 of the provided data using the given key
func ComputeHMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// pkcs7Pad pads the data to the specified block size using PKCS#7 padding
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padBytes := make([]byte, padding)
	for i := range padBytes {
		padBytes[i] = byte(padding)
	}
	return append(data, padBytes...)
}

// EncryptAES256CBC encrypts data using AES-256-CBC with PKCS#7 padding
func EncryptAES256CBC(key, iv, plaintext []byte) ([]byte, error) {
	if len(key) != AES256KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", AES256KeySize, len(key))
	}
	if len(iv) != IVSize {
		return nil, fmt.Errorf("invalid IV size: expected %d, got %d", IVSize, len(iv))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Apply PKCS#7 padding
	paddedPlaintext := pkcs7Pad(plaintext, aes.BlockSize)

	// Create ciphertext buffer
	ciphertext := make([]byte, len(paddedPlaintext))

	// Encrypt using CBC mode
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	return ciphertext, nil
}

// Encrypt performs authenticated encryption on the provided data
// Returns the encrypted data with HMAC and IV prepended, along with encryption info
func Encrypt(plaintext []byte) (*EncryptionInfo, []byte, error) {
	// Generate random keys and IV
	encryptionKey, err := GenerateKey(AES256KeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	macKey, err := GenerateKey(AES256KeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate MAC key: %w", err)
	}

	iv, err := GenerateIV()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Compute SHA256 of original content
	fileDigest := ComputeSHA256(plaintext)

	// Encrypt the content
	ciphertext, err := EncryptAES256CBC(encryptionKey, iv, plaintext)
	if err != nil {
		return nil, nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Create the data that will be HMAC'd (IV + ciphertext)
	// Note: The HMAC covers the IV and ciphertext together
	dataToMAC := append(iv, ciphertext...)
	mac := ComputeHMACSHA256(macKey, dataToMAC)

	// Construct final output: [HMAC][IV][Ciphertext]
	output := make([]byte, 0, HMACSize+IVSize+len(ciphertext))
	output = append(output, mac...)
	output = append(output, iv...)
	output = append(output, ciphertext...)

	info := &EncryptionInfo{
		EncryptionKey:   encryptionKey,
		MacKey:          macKey,
		IV:              iv,
		MAC:             mac,
		FileDigest:      fileDigest,
		UnencryptedSize: int64(len(plaintext)),
	}

	return info, output, nil
}

// EncryptReader performs authenticated encryption on data from a reader
// This is useful for large files to avoid loading everything into memory at once
func EncryptReader(r io.Reader) (*EncryptionInfo, []byte, error) {
	// Read all data (for now - could be optimized for streaming)
	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read input: %w", err)
	}
	return Encrypt(plaintext)
}

// ToBase64 converts the encryption info to base64-encoded strings
func (e *EncryptionInfo) ToBase64() EncryptionInfoBase64 {
	return EncryptionInfoBase64{
		EncryptionKey:   base64.StdEncoding.EncodeToString(e.EncryptionKey),
		MacKey:          base64.StdEncoding.EncodeToString(e.MacKey),
		IV:              base64.StdEncoding.EncodeToString(e.IV),
		MAC:             base64.StdEncoding.EncodeToString(e.MAC),
		FileDigest:      base64.StdEncoding.EncodeToString(e.FileDigest),
		UnencryptedSize: e.UnencryptedSize,
	}
}
