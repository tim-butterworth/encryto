package communication

import (
	"encoding/json"
	"fmt"

	"util.tim/encrypto/core/subscribable"
)

func keyExchange(
	conn subscribable.Connection,
	keyExchangeSuccess chan<- subscribable.Connection,
	localEncryptionProvider LocalEncryptionProvider,
	remoteEncryptionProvider RemoteEncryptionProvider,
	verificationCodeGenerator VerificationCodeGenerator,
	handshakeWorkflowProvider HandshakeWorkflowProvider,
) {
	messageChannel := make(chan subscribable.Message)
	disonnectChannel := make(chan bool)
	subscriptionId := conn.Subscribe(newSubscription(messageChannel, disonnectChannel))
	defer conn.UnSubscribe(subscriptionId)
	defer close(keyExchangeSuccess)

	guid, err := verificationCodeGenerator.GenerateCode()
	if err != nil {
		return
	}
	verificationMessage := fmt.Sprintf("[%s]", guid)

	localEncryption, err := localEncryptionProvider.NewRSAContainer()
	if err != nil {
		fmt.Println("Error creating local rsa container", err)
		return
	}

	responder := newWorkflowResponder(conn, keyExchangeSuccess)
	handshakeWorkflowHandler := handshakeWorkflowProvider.NewHandler(
		responder,
		newWorkflowProvider(
			conn,
			localEncryption,
			remoteEncryptionProvider,
		),
	)

	handshakeWorkflowHandler.SignalReady()

	disconnected := false
	hadError := false
	verificationAttempted := false
	for {
		if disconnected {
			break
		}

		if hadError {
			break
		}

		if verificationAttempted {
			break
		}

		select {
		case request := <-messageChannel:
			{
				if request.Varient == "GetPublicKey" {
					handshakeWorkflowHandler.SendKey()
				} else if request.Varient == "GetVerification" {
					handshakeWorkflowHandler.SendVerification(verificationMessage)
				} else if request.Varient == "SetPublicKey" {
					dataBytes := request.Data
					keyData := keyData{}
					err = json.Unmarshal(dataBytes, &keyData)
					if err != nil {
						fmt.Println("Error parsing [SetPublicKey] data", err)
						hadError = true
						return
					}

					if keyData.PublicKey == "" {
						fmt.Println("Error [PublicKey] may not be empty")
						hadError = true
						return
					}

					handshakeWorkflowHandler.ReceiveKey([]byte(keyData.PublicKey))

					hadError = responder.hadError
				} else if request.Varient == "Verify" {
					dataBytes := request.Data
					verificationRequest := VerificationRequest{}
					err = json.Unmarshal(dataBytes, &verificationRequest)
					if err != nil {
						fmt.Println("Error parsing [Verify] data", err)
						hadError = true
						return
					}

					handshakeWorkflowHandler.Verify(toBytes(verificationRequest.Message))

					hadError = responder.hadError
					verificationAttempted = true
				}
			}
		case <-disonnectChannel:
			fmt.Println("Connection dropped")
			disconnected = true
		}
	}

	fmt.Println("Handshake Workflow Complete")
}
