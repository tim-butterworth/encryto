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
	outFilePath := os.Getenv("OUT_FILE")
	if outFilePath == "" {
		fmt.Println("environment variable 'OUT_FILE' is required")
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

	encrypted := gcm.Seal(nonce, nonce, fileBytes, nil)
	fmt.Println(encrypted)
	fmt.Println(fmt.Sprintf("\n[%s]", string(encrypted)))

	ioutil.WriteFile(outFilePath, encrypted, 0644)
}
