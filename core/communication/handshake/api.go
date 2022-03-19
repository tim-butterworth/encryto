package handshake

import (
	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/subscribable"
)

type HandshakeWorkflowHandler interface {
	SignalReady()
	SendKey()
	ReceiveKey([]byte)
	SendVerification(string)
	Verify([]byte)
}

type HandshakeWorkflowResponder interface {
	SignalReady()
	PublicKeyResponse([]byte)
	VerificationResponse([]byte)
	Verified(subscribable.Connection, asymetric.RemoteRSAContainer, asymetric.LocalRSAContainer)
	KeyReceived()
	ErrorResponse(string)
}

type HandshakeWorkflowDependenciesProvider interface {
	GetServerKeyContainer() asymetric.LocalRSAContainer
	GetClientKeyContainer([]byte) (asymetric.RemoteRSAContainer, error)
	GetConnection() subscribable.Connection
}

func NewHandler(responder HandshakeWorkflowResponder, provider HandshakeWorkflowDependenciesProvider) HandshakeWorkflowHandler {
	return newHandler(responder, provider)
}
