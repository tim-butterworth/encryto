package handshake_test

import (
	"errors"
	"fmt"

	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/communication/handshake"
	"util.tim/encrypto/core/subscribable"
)

type testResponder struct {
	onSignalReady func()
	onPublicKey   func([]byte)
	onFailure     func(string)
	onKeyReceived func()
	onSendVerify  func([]byte)
	onVerified    func(subscribable.Connection, asymetric.RemoteRSAContainer, asymetric.LocalRSAContainer)
}

func (responder *testResponder) SignalReady() {
	responder.onSignalReady()
}

func (responder *testResponder) ErrorResponse(message string) {
	responder.onFailure(message)
}
func (responder *testResponder) PublicKeyResponse(key []byte) {
	responder.onPublicKey(key)
}
func (responder *testResponder) KeyReceived() {
	responder.onKeyReceived()
}
func (responder *testResponder) VerificationResponse(verification []byte) {
	responder.onSendVerify(verification)
}
func (responder *testResponder) Verified(connection subscribable.Connection, clientContainer asymetric.RemoteRSAContainer, serverContainer asymetric.LocalRSAContainer) {
	responder.onVerified(connection, clientContainer, serverContainer)
}

type encryptSuccessRemoteRSAContainer struct {
	prefix string
}

func (container encryptSuccessRemoteRSAContainer) Encrypt(messageBytes []byte) ([]byte, error) {
	encryptedMessage := fmt.Sprintf("%s: [%s]", container.prefix, string(messageBytes))
	return []byte(encryptedMessage), nil
}

func newSuccessRemoteRSAContainer(prefix string) asymetric.RemoteRSAContainer {
	return encryptSuccessRemoteRSAContainer{
		prefix: prefix,
	}
}

type encryptFailureRemoteRSAContainer struct {
	errorMessage string
}

func (container encryptFailureRemoteRSAContainer) Encrypt(message []byte) ([]byte, error) {
	return nil, errors.New(container.errorMessage)
}

func newFailureRemoteRSAContainer(errorMessage string) asymetric.RemoteRSAContainer {
	return encryptFailureRemoteRSAContainer{
		errorMessage: errorMessage,
	}
}

type TestLocalRSAProps struct {
	PublicKey     string
	DecryptResult func(asymetric.DecryptHandler, []byte)
}
type testLocalRSAContainer struct {
	publicKey     string
	decryptResult func(asymetric.DecryptHandler, []byte)
}

func (container *testLocalRSAContainer) Decrypt(handler asymetric.DecryptHandler, bytes []byte) {
	container.decryptResult(handler, bytes)
}

func (container *testLocalRSAContainer) PublicKeyBytes() []byte {
	return []byte(container.publicKey)
}

func newLocalRSAContainer(override func(*TestLocalRSAProps)) asymetric.LocalRSAContainer {
	props := TestLocalRSAProps{
		PublicKey:     "",
		DecryptResult: func(dh asymetric.DecryptHandler, b []byte) {},
	}

	override(&props)

	return &testLocalRSAContainer{
		publicKey:     props.PublicKey,
		decryptResult: props.DecryptResult,
	}
}

type testConnection struct{}

func (conn testConnection) WriteMessage(subscribable.OutgoingMessage) error {
	return nil
}
func (conn testConnection) Subscribe(subscribable.Subscription) subscribable.SubscriptionId {
	return subscribable.NewSubscriptionId(12)
}
func (conn testConnection) UnSubscribe(subscribable.SubscriptionId) {}

func newTestConnection() subscribable.Connection {
	return testConnection{}
}

type testProvider struct {
	getServerContainer func() asymetric.LocalRSAContainer
	getClientResult    func() (asymetric.RemoteRSAContainer, error)
}

func (provider testProvider) GetServerKeyContainer() asymetric.LocalRSAContainer {
	return provider.getServerContainer()
}

func (provider testProvider) GetClientKeyContainer(bytes []byte) (asymetric.RemoteRSAContainer, error) {
	return provider.getClientResult()
}

func (provider testProvider) GetConnection() subscribable.Connection {
	return newTestConnection()
}

type TestProviderProps struct {
	GetServerContainer       func() asymetric.LocalRSAContainer
	GetClientContainerResult func() (asymetric.RemoteRSAContainer, error)
}

func newTestProvider(applyOverrides func(*TestProviderProps)) handshake.HandshakeWorkflowDependenciesProvider {
	testProviderProps := TestProviderProps{
		GetServerContainer: func() asymetric.LocalRSAContainer {
			return newLocalRSAContainer(func(tlr *TestLocalRSAProps) {})
		},
		GetClientContainerResult: func() (asymetric.RemoteRSAContainer, error) {
			return newSuccessRemoteRSAContainer("Default Prefix"), nil
		},
	}

	applyOverrides(&testProviderProps)
	return &testProvider{
		getServerContainer: testProviderProps.GetServerContainer,
		getClientResult:    testProviderProps.GetClientContainerResult,
	}
}
