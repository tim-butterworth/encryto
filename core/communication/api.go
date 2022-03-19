package communication

import (
	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/communication/handshake"
	"util.tim/encrypto/core/shared"
	"util.tim/encrypto/core/subscribable"
)

type Hub interface {
	AddConnection(subscribable.Connection)
}

type IdGenerator interface {
	NextId() string
}

type VerificationCodeGenerator interface {
	GenerateCode() (string, error)
}

type LocalEncryptionProvider interface {
	NewRSAContainer() (asymetric.LocalRSAContainer, error)
}

type RemoteEncryptionProvider interface {
	NewRSAContainer(string) (asymetric.RemoteRSAContainer, error)
}

type HandshakeWorkflowProvider interface {
	NewHandler(
		responder handshake.HandshakeWorkflowResponder,
		provider handshake.HandshakeWorkflowDependenciesProvider,
	) handshake.HandshakeWorkflowHandler
}

type Connection interface {
	Id() string
	Subscribe(func(shared.FromMessage))
	Send(receiver string, data shared.Data)
}

type AcceptConnections interface {
	Join() Connection
	ReJoin(id string) (Connection, error)
}

func NewHub(
	idGenerator IdGenerator,
	verificationCodeGenerator VerificationCodeGenerator,
	localEncryptionProvider LocalEncryptionProvider,
	remoteEncryptionProvider RemoteEncryptionProvider,
	handshakeWorkflowHandlerProvider HandshakeWorkflowProvider,
	exchange AcceptConnections,
) Hub {
	return newHub(
		idGenerator,
		verificationCodeGenerator,
		localEncryptionProvider,
		remoteEncryptionProvider,
		handshakeWorkflowHandlerProvider,
		exchange,
	)
}
