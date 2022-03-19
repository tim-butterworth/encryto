package main

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"util.tim/encrypto/adapters/asymetric/local"
	"util.tim/encrypto/adapters/asymetric/remote"

	"util.tim/encrypto/core/actors"
	"util.tim/encrypto/core/actors/concurrent"
	"util.tim/encrypto/core/actors/presentation"
	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/communication"
	"util.tim/encrypto/core/communication/handshake"
	"util.tim/encrypto/core/subscribable"
)

func websocketHandler(upgrader *websocket.Upgrader, hub communication.Hub) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			fmt.Println("Failed to upgrade")
			return
		}

		hub.AddConnection(subscribable.NewConnection(connection))
	}
}

func index(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte(`<!DOCTYPE html>
		<div>
			<div>Index, should have js eventually</div>
			<div id="app"/>
		</div>
	`))
}

type simpleIdGenerator struct {
	current int
}

func (generator *simpleIdGenerator) NextId() string {
	value := generator.current

	generator.current += 1

	return fmt.Sprintf("%d", value)
}

func newIdGenerator() communication.IdGenerator {
	return &simpleIdGenerator{}
}

type localEncryptionProvider struct{}

func (provider localEncryptionProvider) NewRSAContainer() (asymetric.LocalRSAContainer, error) {
	return local.NewRSAContainer()
}

func newLocalEncryptionProvider() communication.LocalEncryptionProvider {
	return &localEncryptionProvider{}
}

type remoteEncryptionProvider struct{}

func (provider remoteEncryptionProvider) NewRSAContainer(remotePemString string) (asymetric.RemoteRSAContainer, error) {
	return remote.NewRSARemoteContainer(remotePemString)
}

func newRemoteEncryptionProvider() communication.RemoteEncryptionProvider {
	return &remoteEncryptionProvider{}
}

type verificationCodeGenerator struct{}

func (generator verificationCodeGenerator) GenerateCode() (string, error) {
	guid, err := uuid.NewRandom()

	if err != nil {
		return "", err
	}

	return guid.String(), nil
}
func newVerificationCodeGenerator() communication.VerificationCodeGenerator {
	return &verificationCodeGenerator{}
}

type handshakeWorkflowHandlerProvider struct{}

func (provider handshakeWorkflowHandlerProvider) NewHandler(
	responder handshake.HandshakeWorkflowResponder,
	workflowProvider handshake.HandshakeWorkflowDependenciesProvider,
) handshake.HandshakeWorkflowHandler {
	return handshake.NewHandler(responder, workflowProvider)
}

func newHandshakeWorkflowHandlerProvider() communication.HandshakeWorkflowProvider {
	return handshakeWorkflowHandlerProvider{}
}

type acceptConnectionAdapter struct {
	exchange actors.Exchange
}

func (adapter *acceptConnectionAdapter) Join() communication.Connection {
	return adapter.exchange.Connect()
}
func (adapter *acceptConnectionAdapter) ReJoin(id string) (communication.Connection, error) {
	return adapter.exchange.Reconnect(id)
}

func newAcceptConnectionAdapter(exchange actors.Exchange) communication.AcceptConnections {
	return &acceptConnectionAdapter{
		exchange: exchange,
	}
}

func main() {
	upgrader := &websocket.Upgrader{}
	exchange := concurrent.NewConcurrentExchange(newIdGenerator())
	communicationHub := communication.NewHub(
		newIdGenerator(),
		newVerificationCodeGenerator(),
		newLocalEncryptionProvider(),
		newRemoteEncryptionProvider(),
		newHandshakeWorkflowHandlerProvider(),
		newAcceptConnectionAdapter(exchange),
	)

	connection := exchange.Connect()
	fmt.Println(connection.Id())
	go presentation.Coordinate(connection)

	http.HandleFunc("/ws", websocketHandler(upgrader, communicationHub))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./js/dist/"))))

	port := 8181
	fmt.Printf("Started on port [%d]\n", port)
	http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), nil)
}
