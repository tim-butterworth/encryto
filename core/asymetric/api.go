package asymetric

type RemoteRSAContainer interface {
	Encrypt([]byte) ([]byte, error)
}

type DecryptHandler interface {
	Success(string)
	Failure(error)
}

type LocalRSAContainer interface {
	PublicKeyBytes() []byte
	Decrypt(DecryptHandler, []byte)
}
