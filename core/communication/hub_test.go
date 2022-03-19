package communication_test

import (
	"encoding/json"
	"testing"

	"util.tim/encrypto/core/communication"
	"util.tim/encrypto/core/subscribable"
)

func TestEverythingGoesWell(t *testing.T) {
	helper := newTestHelper(t)
	hub := communication.NewHub(
		newIdGenerator(),
		newVerificationCodeGenerator(func(tvcgp *TestVerificationCodeGeneratorProps) {}),
		newLocalEncryptionProvider(),
		newRemoteEncryptionProvider(),
		newWorkflowProvider(),
		newAcceptConnection(),
	)

	fromClient := make(chan subscribable.Message)
	toClient := make(chan subscribable.OutgoingMessage)

	hub.AddConnection(subscribable.NewConnection(newSocket(fromClient, toClient)))

	_, err := helper.waitForResponse("Ready", toClient)
	helper.failIfError(err, "Error")

	requestKeyMessage := subscribable.Message{
		Varient: "GetPublicKey",
	}

	fromClient <- requestKeyMessage

	getKeyResponse, err := helper.waitForResponse("GetPublicKey", toClient)
	helper.failIfError(err, "Error")

	serverKey, ok := getKeyResponse.Body.(communication.ServerKey)
	if !ok {
		t.Log("Failed to cast response")
		t.FailNow()
	}

	publicKey := serverKey.PublicKey
	t.Log("---SERVER KEY---")
	t.Log(string(toBytes(publicKey)))
	t.Log("---SERVER KEY---")

	publicKeyPayload, err := json.Marshal(map[string]string{
		"publicKey": "someKey",
	})
	helper.failIfError(err, "Error parsing publicKeyPayload")

	setKeyMessage := subscribable.Message{
		Varient: "SetPublicKey",
		Data:    publicKeyPayload,
	}

	fromClient <- setKeyMessage

	setKeyResponse, err := helper.waitForResponse("SetPublicKey", toClient)
	helper.failIfError(err, "Error")

	t.Log(setKeyResponse.Variant)

	requestVerification := subscribable.Message{
		Varient: "GetVerification",
	}
	fromClient <- requestVerification

	getVerificationResponse, err := helper.waitForResponse("SetPublicKey", toClient)
	helper.failIfError(err, "Error")

	verificationResponse := getVerificationResponse.Body.(communication.VerificationResponse)

	verificationData, err := json.Marshal(map[string][]int64{
		"message": toInt64(verificationResponse.Message),
	})
	helper.failIfError(err, "Error parsing verification data")

	verify := subscribable.Message{
		Varient: "Verify",
		Data:    verificationData,
	}
	fromClient <- verify

	verificationSubmittedResponse, err := helper.waitForResponse("Verify", toClient)
	helper.failIfError(err, "Error")

	t.Log(verificationSubmittedResponse.Variant)
	innerMessage := &subscribable.OutgoingMessage{}
	err = json.Unmarshal(toBytes(verificationSubmittedResponse.Body.([]int8)), innerMessage)
	t.Log(err)
	t.Log(innerMessage)
	t.Log(verificationSubmittedResponse)

	commandData, err := json.Marshal(subscribable.Message{
		Varient: "Connect",
	})
	helper.failIfError(err, "Parsing command data")

	fromClient <- subscribable.Message{
		Varient: "Command",
		Data:    commandData,
	}
	availableActionResponse, err := helper.waitForResponse("AvailableActions", toClient)
	helper.failIfError(err, "Error")

	t.Log(availableActionResponse)
}
