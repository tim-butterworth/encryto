package communication

import (
	"fmt"

	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/communication/handshake"
	"util.tim/encrypto/core/subscribable"
)

type hub struct {
	localEncryptionProvider          LocalEncryptionProvider
	remoteEncryptionProvider         RemoteEncryptionProvider
	verificationCodeGenerator        VerificationCodeGenerator
	handshakeWorkflowHandlerProvider HandshakeWorkflowProvider
	exchange                         AcceptConnections
}

type Greeting struct {
	Message   string
	PublicKey string
}

type subscription struct {
	messageChannel    chan<- subscribable.Message
	disconnectChannel chan<- bool
}

func (sub *subscription) ConnectionDropped() {
	sub.disconnectChannel <- true
}
func (sub *subscription) ReceivedMessage(message subscribable.Message) {
	sub.messageChannel <- message
}

func newSubscription(
	messageChannel chan<- subscribable.Message,
	disconnectChannel chan<- bool,
) subscribable.Subscription {
	return &subscription{
		messageChannel:    messageChannel,
		disconnectChannel: disconnectChannel,
	}
}

type keyData struct {
	PublicKey string
}

func toIntArray(bytes []byte) []int8 {
	result := make([]int8, len(bytes))
	for i, b := range bytes {
		result[i] = int8(b)
	}

	return result
}

func toBytes(ints []int16) []byte {
	result := make([]byte, len(ints))
	for i := range ints {
		result[i] = byte(ints[i])
	}

	return result
}

type VerificationRequest struct {
	Message []int16 `json:"message"`
}

type DecryptResult interface {
	ifSuccess(func(string))
	ifFailure(func(error))
}

type NoOpDecryptResult struct{}

func (result NoOpDecryptResult) ifSuccess(wasSuccess func(string)) {}
func (result NoOpDecryptResult) ifFailure(wasError func(error))    {}

type FailureDecryptResult struct {
	err error
}

func (result FailureDecryptResult) ifSuccess(wasSuccess func(string)) {}
func (result FailureDecryptResult) ifFailure(wasError func(error)) {
	wasError(result.err)
}

type SuccessDecryptResult struct {
	decrypted string
}

func (result SuccessDecryptResult) ifSuccess(wasSuccess func(string)) {
	wasSuccess(result.decrypted)
}
func (result SuccessDecryptResult) ifFailure(wasError func(error)) {}

type handShakeWorkflowHandler struct {
	conn                  subscribable.Connection
	addVerifiedConnection chan<- subscribable.Connection
	hadError              bool
}

type ServerKey struct {
	PublicKey []int8 `json:"publicKey"`
}

func (handler *handShakeWorkflowHandler) PublicKeyResponse(publicKey []byte) {
	handler.conn.WriteMessage(subscribable.OutgoingMessage{
		Variant: "ServerKey",
		Body: ServerKey{
			PublicKey: toIntArray(publicKey),
		},
	})
}

type VerificationResponse struct {
	Message []int8 `json:"message"`
}

func (handler *handShakeWorkflowHandler) VerificationResponse(encrypted []byte) {
	handler.conn.WriteMessage(subscribable.OutgoingMessage{
		Variant: "Verification",
		Body: VerificationResponse{
			Message: toIntArray(encrypted),
		},
	})
}

func (handler *handShakeWorkflowHandler) Verified(
	conn subscribable.Connection,
	remoteEncryption asymetric.RemoteRSAContainer,
	localEncryption asymetric.LocalRSAContainer,
) {
	handler.addVerifiedConnection <- &encryptedConnection{
		underlyingConnection: conn,
		localEncryption:      localEncryption,
		remoteEncryption:     remoteEncryption,
	}
}

func (handler *handShakeWorkflowHandler) KeyReceived() {
	handler.conn.WriteMessage(subscribable.OutgoingMessage{
		Variant: "KeyReceived",
	})
}

func (handler *handShakeWorkflowHandler) ErrorResponse(message string) {
	fmt.Printf("Had Error \n\n[%s]\n\n", message)
	handler.hadError = true
	handler.conn.WriteMessage(subscribable.OutgoingMessage{
		Variant: "Error",
		Body:    fmt.Sprintf("Error -> [%s] connection will be dropped", message),
	})
}

func (handler *handShakeWorkflowHandler) SignalReady() {
	handler.conn.WriteMessage(subscribable.OutgoingMessage{
		Variant: "Ready",
	})
}

func newWorkflowResponder(conn subscribable.Connection, addConnection chan<- subscribable.Connection) *handShakeWorkflowHandler {
	return &handShakeWorkflowHandler{
		conn:                  conn,
		addVerifiedConnection: addConnection,
		hadError:              false,
	}
}

type handshakeWorkflowProvider struct {
	conn                     subscribable.Connection
	localEncryption          asymetric.LocalRSAContainer
	remoteEncryptionProvider RemoteEncryptionProvider
}

func (provider *handshakeWorkflowProvider) GetServerKeyContainer() asymetric.LocalRSAContainer {
	return provider.localEncryption
}

func (provider *handshakeWorkflowProvider) GetClientKeyContainer(keyBytes []byte) (asymetric.RemoteRSAContainer, error) {
	return provider.remoteEncryptionProvider.NewRSAContainer(string(keyBytes))
}

func (provider *handshakeWorkflowProvider) GetConnection() subscribable.Connection {
	return provider.conn
}

func newWorkflowProvider(
	conn subscribable.Connection,
	localEncryption asymetric.LocalRSAContainer,
	remoteEncryption RemoteEncryptionProvider,
) handshake.HandshakeWorkflowDependenciesProvider {
	return &handshakeWorkflowProvider{
		conn:                     conn,
		localEncryption:          localEncryption,
		remoteEncryptionProvider: remoteEncryption,
	}
}

func (hub *hub) AddConnection(connection subscribable.Connection) {
	keyExchangeSuccess := make(chan subscribable.Connection)

	go keyExchange(
		connection,
		keyExchangeSuccess,
		hub.localEncryptionProvider,
		hub.remoteEncryptionProvider,
		hub.verificationCodeGenerator,
		hub.handshakeWorkflowHandlerProvider,
	)
	go registerConnection(
		keyExchangeSuccess,
		hub.exchange,
	)
}

func newHub(
	idGenerator IdGenerator,
	verificationCodeGenerator VerificationCodeGenerator,
	localEncryptionProvider LocalEncryptionProvider,
	remoteEncryptionProvider RemoteEncryptionProvider,
	handshakeWorkflowHandlerProvider HandshakeWorkflowProvider,
	exchange AcceptConnections,
) Hub {
	return &hub{
		localEncryptionProvider:          localEncryptionProvider,
		remoteEncryptionProvider:         remoteEncryptionProvider,
		verificationCodeGenerator:        verificationCodeGenerator,
		handshakeWorkflowHandlerProvider: handshakeWorkflowHandlerProvider,
		exchange:                         exchange,
	}
}
