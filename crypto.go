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
	masterKeyHex   = "e99a7835516b73c0be7aadf06c0ee20d8f6cc6e95170940f79dc8e47df5424b0"
	masterNonceHex = "fabee448f25ffc9291aac3ef"
	masterTokenHex = "58f67b8dcb2509c3dd84678ca5f1eb98ddb1b95707332454b290dece137260d14144e87ffaea4610d8c3ca9dcc328561cab339e4d98b3f7a1292d261454c8aeadc6c839c7e495b73d8fa4911062e9149c9158621cd6ac2d7163303878cf0c4e03161de132fa1f2a3a8d400238e"
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
