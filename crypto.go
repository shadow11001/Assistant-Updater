package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
)

// In a real build, we will replace these with the actual hex strings using a build script or by hand.
// We use variables instead of constants so they can be injected dynamically via ldflags if preferred.
var (
	masterKeyHex   = "replace_with_32_byte_hex_key"
	masterNonceHex = "replace_with_12_byte_hex_nonce"
	masterTokenHex = "replace_with_hex_ciphertext"
)

// decryptToken takes the hardcoded hex strings and decrypts the original token
func decryptToken() (string, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode key: %v", err)
	}

	nonce, err := hex.DecodeString(masterNonceHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %v", err)
	}

	ciphertext, err := hex.DecodeString(masterTokenHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %v", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to wrap cipher: %v", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %v", err)
	}

	return string(plaintext), nil
}
