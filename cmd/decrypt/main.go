package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func main() {
	filePath := os.Getenv("FILE")
	if filePath == "" {
		fmt.Println("environment variable 'FILE' is required")
		return
	}
	key := os.Getenv("KEY")
	if key == "" {
		fmt.Println("environment variable 'KEY' is required")
		return
	}

	if len(key) < 32 {
		fmt.Println("key is too short, must be at least 32 characters")
		fmt.Printf("Has length in bytes: [%d]\n", len([]byte(key)))
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error opening file -> %s", filePath), err)
		return
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file", err)
		return
	}

	fmt.Println(string(fileBytes))
	// key := []byte("this is a fancy thing going for the keyit is going to begreat!")
	correctSizeKey := make([]byte, 32)

	for i := range correctSizeKey {
		if i < len(key) {
			correctSizeKey[i] = key[i]
		} else {
			correctSizeKey[i] = byte(111)
		}
	}

	block, err := aes.NewCipher(correctSizeKey)

	if err != nil {
		fmt.Println("error", err)
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		fmt.Println("error", err)
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println(err)
	}

	readNonce, text := fileBytes[:len(nonce)], fileBytes[len(nonce):]

	decrypted, err := gcm.Open(nil, readNonce, text, nil)

	if err != nil {
		fmt.Println("error", err)
	}

	fmt.Println(string(decrypted))
}
