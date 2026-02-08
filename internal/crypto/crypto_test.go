package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey(AES256KeySize)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if len(key) != AES256KeySize {
		t.Errorf("Expected key length %d, got %d", AES256KeySize, len(key))
	}

	// Verify keys are random (generate another and compare)
	key2, err := GenerateKey(AES256KeySize)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if bytes.Equal(key, key2) {
		t.Error("Generated keys should be different")
	}
}

func TestGenerateIV(t *testing.T) {
	iv, err := GenerateIV()
	if err != nil {
		t.Fatalf("GenerateIV failed: %v", err)
	}
	if len(iv) != IVSize {
		t.Errorf("Expected IV length %d, got %d", IVSize, len(iv))
	}
}

func TestComputeSHA256(t *testing.T) {
	data := []byte("test data")
	hash := ComputeSHA256(data)
	if len(hash) != 32 {
		t.Errorf("Expected SHA256 hash length 32, got %d", len(hash))
	}

	// Same data should produce same hash
	hash2 := ComputeSHA256(data)
	if !bytes.Equal(hash, hash2) {
		t.Error("SHA256 of same data should be equal")
	}

	// Different data should produce different hash
	hash3 := ComputeSHA256([]byte("different data"))
	if bytes.Equal(hash, hash3) {
		t.Error("SHA256 of different data should be different")
	}
}

func TestComputeHMACSHA256(t *testing.T) {
	key := []byte("test key 32 bytes long here!!!!!")
	data := []byte("test data")
	mac := ComputeHMACSHA256(key, data)
	if len(mac) != HMACSize {
		t.Errorf("Expected HMAC length %d, got %d", HMACSize, len(mac))
	}

	// Same key and data should produce same MAC
	mac2 := ComputeHMACSHA256(key, data)
	if !bytes.Equal(mac, mac2) {
		t.Error("HMAC of same key and data should be equal")
	}

	// Different key should produce different MAC
	mac3 := ComputeHMACSHA256([]byte("different key 32 bytes here!!!!"), data)
	if bytes.Equal(mac, mac3) {
		t.Error("HMAC with different key should be different")
	}
}

func TestPKCS7Pad(t *testing.T) {
	tests := []struct {
		input    []byte
		expected int // expected length after padding
	}{
		{make([]byte, 0), 16},  // empty -> 16 bytes padding
		{make([]byte, 1), 16},  // 1 byte -> 15 bytes padding
		{make([]byte, 15), 16}, // 15 bytes -> 1 byte padding
		{make([]byte, 16), 32}, // 16 bytes -> 16 bytes padding (full block)
		{make([]byte, 17), 32}, // 17 bytes -> 15 bytes padding
	}

	for _, tc := range tests {
		padded := pkcs7Pad(tc.input, aes.BlockSize)
		if len(padded) != tc.expected {
			t.Errorf("pkcs7Pad(%d bytes): expected length %d, got %d", len(tc.input), tc.expected, len(padded))
		}
		// Verify padding value
		paddingByte := padded[len(padded)-1]
		for i := len(padded) - int(paddingByte); i < len(padded); i++ {
			if padded[i] != paddingByte {
				t.Errorf("Invalid padding at position %d: expected %d, got %d", i, paddingByte, padded[i])
			}
		}
	}
}

func TestEncryptAES256CBC(t *testing.T) {
	key, _ := GenerateKey(AES256KeySize)
	iv, _ := GenerateIV()
	plaintext := []byte("This is a test message for encryption")

	ciphertext, err := EncryptAES256CBC(key, iv, plaintext)
	if err != nil {
		t.Fatalf("EncryptAES256CBC failed: %v", err)
	}

	// Ciphertext should be longer than plaintext due to padding
	if len(ciphertext) <= len(plaintext) {
		t.Error("Ciphertext should be longer than plaintext")
	}

	// Ciphertext length should be multiple of block size
	if len(ciphertext)%aes.BlockSize != 0 {
		t.Errorf("Ciphertext length %d is not multiple of block size %d", len(ciphertext), aes.BlockSize)
	}

	// Verify we can decrypt it
	block, _ := aes.NewCipher(key)
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	// Remove PKCS7 padding
	paddingLen := int(decrypted[len(decrypted)-1])
	decrypted = decrypted[:len(decrypted)-paddingLen]

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted content does not match original")
	}
}

func TestEncrypt(t *testing.T) {
	plaintext := []byte("Test content for full encryption workflow")

	info, encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Verify encryption info is populated
	if len(info.EncryptionKey) != AES256KeySize {
		t.Errorf("Invalid encryption key size: %d", len(info.EncryptionKey))
	}
	if len(info.MacKey) != AES256KeySize {
		t.Errorf("Invalid MAC key size: %d", len(info.MacKey))
	}
	if len(info.IV) != IVSize {
		t.Errorf("Invalid IV size: %d", len(info.IV))
	}
	if len(info.MAC) != HMACSize {
		t.Errorf("Invalid MAC size: %d", len(info.MAC))
	}
	if len(info.FileDigest) != 32 {
		t.Errorf("Invalid file digest size: %d", len(info.FileDigest))
	}
	if info.UnencryptedSize != int64(len(plaintext)) {
		t.Errorf("UnencryptedSize mismatch: expected %d, got %d", len(plaintext), info.UnencryptedSize)
	}

	// Verify output structure: [HMAC 32][IV 16][Ciphertext]
	expectedMinLen := HMACSize + IVSize + aes.BlockSize
	if len(encrypted) < expectedMinLen {
		t.Errorf("Encrypted output too short: %d bytes", len(encrypted))
	}

	// Extract and verify MAC
	storedMAC := encrypted[:HMACSize]
	if !bytes.Equal(storedMAC, info.MAC) {
		t.Error("Stored MAC does not match info.MAC")
	}

	// Extract and verify IV
	storedIV := encrypted[HMACSize : HMACSize+IVSize]
	if !bytes.Equal(storedIV, info.IV) {
		t.Error("Stored IV does not match info.IV")
	}

	// Verify HMAC is correct
	dataToMAC := encrypted[HMACSize:] // IV + ciphertext
	computedMAC := ComputeHMACSHA256(info.MacKey, dataToMAC)
	if !bytes.Equal(computedMAC, storedMAC) {
		t.Error("HMAC verification failed")
	}
}

func TestEncryptionInfoToBase64(t *testing.T) {
	info := &EncryptionInfo{
		EncryptionKey:   make([]byte, 32),
		MacKey:          make([]byte, 32),
		IV:              make([]byte, 16),
		MAC:             make([]byte, 32),
		FileDigest:      make([]byte, 32),
		UnencryptedSize: 12345,
	}

	b64 := info.ToBase64()

	// Base64 of 32 zero bytes
	expected32 := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	expected16 := "AAAAAAAAAAAAAAAAAAAAAA=="

	if b64.EncryptionKey != expected32 {
		t.Errorf("EncryptionKey base64 mismatch")
	}
	if b64.MacKey != expected32 {
		t.Errorf("MacKey base64 mismatch")
	}
	if b64.IV != expected16 {
		t.Errorf("IV base64 mismatch")
	}
	if b64.MAC != expected32 {
		t.Errorf("MAC base64 mismatch")
	}
	if b64.FileDigest != expected32 {
		t.Errorf("FileDigest base64 mismatch")
	}
	if b64.UnencryptedSize != 12345 {
		t.Errorf("UnencryptedSize mismatch: expected 12345, got %d", b64.UnencryptedSize)
	}
}
