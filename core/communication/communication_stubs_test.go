package communication_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/communication"
	"util.tim/encrypto/core/communication/handshake"
	"util.tim/encrypto/core/shared"
	"util.tim/encrypto/core/subscribable"
)

type testIdGenerator struct {
	currentId int
}

func (generator *testIdGenerator) NextId() string {
	id := fmt.Sprintf("[%d]", generator.currentId)
	generator.currentId += 1

	return id
}
func newIdGenerator() communication.IdGenerator {
	return &testIdGenerator{}
}

type TestVerificationCodeGeneratorProps struct {
	Generator func() (string, error)
}
type testVerificationCodeGenerator struct {
	generate func() (string, error)
}

func (generator *testVerificationCodeGenerator) GenerateCode() (string, error) {
	return generator.generate()
}

func newVerificationCodeGenerator(override func(*TestVerificationCodeGeneratorProps)) communication.VerificationCodeGenerator {
	code := 0
	props := TestVerificationCodeGeneratorProps{
		Generator: func() (string, error) {
			generated := fmt.Sprintf("%d", code)
			code += 1

			return generated, nil
		},
	}

	override(&props)

	return &testVerificationCodeGenerator{
		generate: props.Generator,
	}
}

type testLocalEncryption struct{}

func (encryption *testLocalEncryption) PublicKeyBytes() []byte {
	return []byte("local encryption key")
}
func (encryption *testLocalEncryption) Decrypt(handler asymetric.DecryptHandler, message []byte) {
	handler.Success(string(message))
}

type testLocalEncryptionProvider struct{}

func (provider testLocalEncryptionProvider) NewRSAContainer() (asymetric.LocalRSAContainer, error) {
	fmt.Println("localEncryptionProvider.NewRSAContainer")
	return &testLocalEncryption{}, nil
}

func newLocalEncryptionProvider() communication.LocalEncryptionProvider {
	return &testLocalEncryptionProvider{}
}

type testRemoteEncryption struct{}

func (encryption *testRemoteEncryption) Encrypt(message []byte) ([]byte, error) {
	return message, nil
}

type testRemoteEncryptionProvider struct{}

func (provider testRemoteEncryptionProvider) NewRSAContainer(string) (asymetric.RemoteRSAContainer, error) {
	fmt.Println("remoteEncryptionProvider.NewRSAContainer")
	return &testRemoteEncryption{}, nil
}

func newRemoteEncryptionProvider() communication.RemoteEncryptionProvider {
	return &testRemoteEncryptionProvider{}
}

type testSocket struct {
	incomingMessageChan <-chan subscribable.Message
	outgoingMessageChan chan<- subscribable.OutgoingMessage
}

func (socket *testSocket) ReadJSON(container interface{}) error {
	incoming := <-socket.incomingMessageChan
	bytes, err := json.Marshal(&incoming)
	if err != nil {
		fmt.Println("Error Marshalling", err)
		return err
	}

	err = json.Unmarshal(bytes, container)
	if err != nil {
		fmt.Println("Error Unmarshalling", err)
		return err
	}

	return nil
}
func (socket *testSocket) WriteJSON(outgoing interface{}) error {
	asOutgoingMessage, ok := outgoing.(subscribable.OutgoingMessage)
	if !ok {
		fmt.Println("failed to cast message to outgoing message")
		return fmt.Errorf("failed to cast [%T] as communication.OutgoingMessage", outgoing)
	}

	socket.outgoingMessageChan <- asOutgoingMessage
	return nil
}

func newSocket(incomingMessageChan <-chan subscribable.Message, outgoingMessageChan chan<- subscribable.OutgoingMessage) subscribable.Socket {
	return &testSocket{
		incomingMessageChan: incomingMessageChan,
		outgoingMessageChan: outgoingMessageChan,
	}
}

type handshakeWorkflowProvider struct{}

func (handlerProvider handshakeWorkflowProvider) NewHandler(
	responder handshake.HandshakeWorkflowResponder,
	provider handshake.HandshakeWorkflowDependenciesProvider,
) handshake.HandshakeWorkflowHandler {
	return handshake.NewHandler(responder, provider)
}

func newWorkflowProvider() communication.HandshakeWorkflowProvider {
	return handshakeWorkflowProvider{}
}

type connection struct{}

func (connection connection) Id() string {
	return ""
}
func (connection connection) Subscribe(func(shared.FromMessage)) {
}
func (connection connection) Send(receiver string, data shared.Data) {
}

type acceptConnections struct{}

func (ac acceptConnections) Join() communication.Connection {
	return connection{}
}

func (ac acceptConnections) ReJoin(id string) (communication.Connection, error) {
	return connection{}, nil
}

func newAcceptConnection() communication.AcceptConnections {
	return acceptConnections{}
}

type testHelper struct {
	t *testing.T
}

func (helper testHelper) waitForResponse(label string, responseChan <-chan subscribable.OutgoingMessage) (*subscribable.OutgoingMessage, error) {
	t := helper.t

	select {
	case result := <-responseChan:
		return &result, nil
	case <-time.After(time.Millisecond * 500):
		t.Logf("Timed out waiting for response [%s]\n", label)
		return nil, errors.New("timed out")
	}
}

func (helper testHelper) failIfError(err error, message string) {
	if err != nil {
		helper.t.Log(message, err)
		helper.t.FailNow()
	}
}

func newTestHelper(t *testing.T) testHelper {
	return testHelper{t: t}
}

func toBytes(ints []int8) []byte {
	result := make([]byte, len(ints))
	for i := range ints {
		result[i] = byte(ints[i])
	}
	return result
}

func toInt64(ints []int8) []int64 {
	result := make([]int64, len(ints))
	for i := range ints {
		result[i] = int64(ints[i])
	}
	return result
}
