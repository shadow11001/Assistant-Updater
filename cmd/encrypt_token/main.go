package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: encrypt_token <actual-github-token>")
		os.Exit(1)
	}
	token := os.Args[1]

	// Generate a 32-byte key for AES-256
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, []byte(token), nil)

	fmt.Println("Save the following values in your crypto.go:")
	fmt.Printf("Key: %s\n", hex.EncodeToString(key))
	fmt.Printf("Nonce: %s\n", hex.EncodeToString(nonce))
	fmt.Printf("Ciphertext: %s\n", hex.EncodeToString(ciphertext))
}
