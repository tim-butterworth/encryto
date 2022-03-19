package remote

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"util.tim/encrypto/core/asymetric"
)

type remoteContainer struct {
	publicKey *rsa.PublicKey
}

func (container *remoteContainer) Encrypt(message []byte) ([]byte, error) {
	encrypted, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		container.publicKey,
		[]byte(message),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

func attemptPKCS1(pemBytes []byte) (*rsa.PublicKey, error) {
	return x509.ParsePKCS1PublicKey(pemBytes)
}

func attemptPKIX(pemBytes []byte) (*rsa.PublicKey, error) {
	rawPub, err := x509.ParsePKIXPublicKey(pemBytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := rawPub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("could not cast raw key to RSA Public Key")
	}

	return publicKey, nil
}

func NewRSARemoteContainer(pemString string) (asymetric.RemoteRSAContainer, error) {
	block, _ := pem.Decode([]byte(pemString))

	publicKey, err := attemptPKCS1(block.Bytes)
	if err == nil {
		return &remoteContainer{
			publicKey: publicKey,
		}, nil
	}

	publicKey, err = attemptPKIX(block.Bytes)
	if err == nil {
		return &remoteContainer{
			publicKey: publicKey,
		}, nil
	}

	return nil, err
}
