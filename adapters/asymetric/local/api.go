package local

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"util.tim/encrypto/core/asymetric"
)

type rsaContainer struct {
	publicKey       rsa.PublicKey
	privateKey      *rsa.PrivateKey
	publicKeyString string
	publicKeyBytes  []byte
}

func (container *rsaContainer) populatePublicKey() {
	bytes, _ := x509.MarshalPKIXPublicKey(&container.publicKey)

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: bytes,
	})

	container.publicKeyBytes = bytes
	container.publicKeyString = string(pemBytes)
}

func (container *rsaContainer) PublicKey() string {
	if container.publicKeyString == "" {
		container.populatePublicKey()
	}

	return container.publicKeyString
}

func (container *rsaContainer) PublicKeyBytes() []byte {
	if len(container.publicKeyBytes) == 0 {
		container.populatePublicKey()
	}

	return container.publicKeyBytes
}

func (container *rsaContainer) Decrypt(handler asymetric.DecryptHandler, message []byte) {
	decrypted, err := container.privateKey.Decrypt(
		nil,
		message,
		&rsa.OAEPOptions{Hash: crypto.SHA256},
	)

	if err != nil {
		handler.Failure(err)
		return
	}

	handler.Success(string(decrypted))
}

func NewRSAContainer() (asymetric.LocalRSAContainer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.PublicKey

	return &rsaContainer{
		publicKey:  publicKey,
		privateKey: privateKey,
	}, nil
}
