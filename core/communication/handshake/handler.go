package handshake

import (
	"fmt"

	"util.tim/encrypto/core/asymetric"
)

type Verification interface {
	matches(string) bool
	wasSent() bool
}

type notSentVerification struct{}

func (verification notSentVerification) matches(code string) bool {
	return false
}
func (verification notSentVerification) wasSent() bool {
	return false
}

func newNotSentVerification() Verification {
	return notSentVerification{}
}

type sentVerification struct {
	code string
}

func (verification *sentVerification) matches(code string) bool {
	return code == verification.code
}
func (verification *sentVerification) wasSent() bool {
	return true
}

func newSentVerification(code string) Verification {
	return &sentVerification{
		code: code,
	}
}

type ClientKey interface {
	hasClientKey() bool
	withClientKey(func(asymetric.RemoteRSAContainer))
}

type notSetClientKey struct{}

func (clientKey notSetClientKey) hasClientKey() bool {
	return false
}
func (clientKey notSetClientKey) withClientKey(ifClientKey func(asymetric.RemoteRSAContainer)) {}

func newNotSetClientKey() ClientKey {
	return notSetClientKey{}
}

type setClientKey struct {
	container asymetric.RemoteRSAContainer
}

func (clientKey *setClientKey) hasClientKey() bool {
	return true
}
func (clientKey *setClientKey) withClientKey(ifClientKey func(asymetric.RemoteRSAContainer)) {
	ifClientKey(clientKey.container)
}

func newSetClientKey(container asymetric.RemoteRSAContainer) ClientKey {
	return &setClientKey{
		container: container,
	}
}

type handler struct {
	responder HandshakeWorkflowResponder
	provider  HandshakeWorkflowDependenciesProvider

	serverKeySent bool

	verification Verification
	clientKey    ClientKey

	onSendVerify func(string)
}

type decryptHandler struct {
	onFailure func(error)
	onSuccess func(string)
}

func (handler *decryptHandler) Failure(err error) {
	handler.onFailure(err)
}
func (handler *decryptHandler) Success(message string) {
	handler.onSuccess(message)
}

func newDecryptHandler(onSuccess func(string), onFailure func(error)) asymetric.DecryptHandler {
	return &decryptHandler{
		onSuccess: onSuccess,
		onFailure: onFailure,
	}
}

func getOnSendVerifyFailureResponse(responder HandshakeWorkflowResponder) func(string) {
	return func(string) {
		responder.ErrorResponse("Error: Can not send a verification code without the client public key")
	}
}

func (handler *handler) SignalReady() {
	handler.responder.SignalReady()
}

func (handler *handler) SendKey() {
	handler.serverKeySent = true
	handler.responder.PublicKeyResponse(handler.provider.GetServerKeyContainer().PublicKeyBytes())
}

func (handler *handler) ReceiveKey(clientKeyBytes []byte) {
	clientKeyContainer, err := handler.provider.GetClientKeyContainer(clientKeyBytes)
	if err != nil {
		handler.responder.ErrorResponse(err.Error())
		handler.onSendVerify = getOnSendVerifyFailureResponse(handler.responder)

		handler.clientKey = newNotSetClientKey()
		handler.verification = newNotSentVerification()

		return
	}

	handler.onSendVerify = func(code string) {
		codeBytes := []byte(code)
		encrypted, err := clientKeyContainer.Encrypt(codeBytes)
		if err != nil {
			handler.responder.ErrorResponse(err.Error())

			return
		}

		handler.responder.VerificationResponse(encrypted)
	}
	handler.verification = newNotSentVerification()
	handler.clientKey = newSetClientKey(clientKeyContainer)

	handler.responder.KeyReceived()
}

func (handler *handler) SendVerification(code string) {
	handler.verification = newSentVerification(code)

	handler.onSendVerify(code)
}

func (handler *handler) Verify(codeToVerify []byte) {
	if !handler.clientKey.hasClientKey() && !handler.serverKeySent {
		handler.responder.ErrorResponse("Verification not possible before keys have been exchanged")
		return
	}

	if !handler.clientKey.hasClientKey() {
		handler.responder.ErrorResponse("Verification not possible before client key received")
		return
	}

	if !handler.serverKeySent {
		handler.responder.ErrorResponse("Verification not possible before server key sent")
		return
	}

	if !handler.verification.wasSent() {
		handler.responder.ErrorResponse("Verification not possible before verification message sent")
		return
	}

	handler.clientKey.withClientKey(func(remoteContainer asymetric.RemoteRSAContainer) {
		decryptionSuccess := func(decrypted string) {
			if handler.verification.matches(decrypted) {
				handler.responder.Verified(
					handler.provider.GetConnection(),
					remoteContainer,
					handler.provider.GetServerKeyContainer(),
				)
			} else {
				handler.responder.ErrorResponse("verification failed")
			}
		}
		decryptionFailure := func(err error) {
			handler.responder.ErrorResponse(fmt.Sprintf("verification failed because [%s]", err))
		}

		handler.provider.GetServerKeyContainer().Decrypt(
			newDecryptHandler(decryptionSuccess, decryptionFailure),
			codeToVerify,
		)
	})
}

func newHandler(
	responder HandshakeWorkflowResponder,
	provider HandshakeWorkflowDependenciesProvider,
) HandshakeWorkflowHandler {
	return &handler{
		responder:    responder,
		provider:     provider,
		onSendVerify: getOnSendVerifyFailureResponse(responder),

		clientKey:    newNotSetClientKey(),
		verification: newNotSentVerification(),
	}
}
