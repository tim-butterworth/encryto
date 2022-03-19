package communication

import (
	"encoding/json"
	"fmt"

	"util.tim/encrypto/core/asymetric"
	"util.tim/encrypto/core/subscribable"
)

type decryptionHandler struct {
	subscription subscribable.Subscription
}

func (handler *decryptionHandler) Success(decrypted string) {
	message := subscribable.Message{}
	err := json.Unmarshal([]byte(decrypted), &message)
	if err == nil {
		handler.subscription.ReceivedMessage(message)
		return
	}

	fmt.Println("ParsingError", err)
}

func (handler *decryptionHandler) Failure(err error) {
	fmt.Println("encryptedConnection.decryptionHandler.Failure -> EncryptionError", err)
}

func newDecryptionHandler(subscription subscribable.Subscription) asymetric.DecryptHandler {
	return &decryptionHandler{
		subscription: subscription,
	}
}

type decryptSubscription struct {
	encryption             asymetric.LocalRSAContainer
	underlyingSubscription subscribable.Subscription
	decryptionHandler      asymetric.DecryptHandler
}

func (subscription *decryptSubscription) ConnectionDropped() {
	subscription.underlyingSubscription.ConnectionDropped()
}

type M struct {
	Message []int16 `json:"message"`
}

func (subscription *decryptSubscription) ReceivedMessage(message subscribable.Message) {
	ints := []int16{}
	err := json.Unmarshal(message.Data, &ints)
	if err != nil {
		fmt.Println("Error unmarshalling message body", err)
		fmt.Println("received", string(message.Data))

		m := M{}
		err = json.Unmarshal(message.Data, &m)
		if err != nil {
			fmt.Println("Failed on the second try", err)
			return
		}

		ints = m.Message
	}

	subscription.encryption.Decrypt(subscription.decryptionHandler, toBytes(ints))
}

func newDecryptSubscription(
	localEncryption asymetric.LocalRSAContainer,
	innerSubscription subscribable.Subscription,
) subscribable.Subscription {
	return &decryptSubscription{
		encryption:             localEncryption,
		underlyingSubscription: innerSubscription,
		decryptionHandler:      newDecryptionHandler(innerSubscription),
	}
}

type encryptedConnection struct {
	underlyingConnection subscribable.Connection
	localEncryption      asymetric.LocalRSAContainer
	remoteEncryption     asymetric.RemoteRSAContainer
}

func (conn *encryptedConnection) WriteMessage(outgoing subscribable.OutgoingMessage) error {
	bytes, err := json.Marshal(outgoing)
	if err != nil {
		return err
	}

	encrypted, err := conn.remoteEncryption.Encrypt(bytes)
	if err != nil {
		return err
	}

	conn.underlyingConnection.WriteMessage(
		subscribable.OutgoingMessage{
			Variant: "Message",
			Body:    toIntArray(encrypted),
		},
	)

	return nil
}

func (conn *encryptedConnection) Subscribe(subscription subscribable.Subscription) subscribable.SubscriptionId {
	fmt.Println("Subscribing to an encrypted connection")
	return conn.underlyingConnection.Subscribe(newDecryptSubscription(conn.localEncryption, subscription))
}

func (conn *encryptedConnection) UnSubscribe(subscriptionId subscribable.SubscriptionId) {
	conn.underlyingConnection.UnSubscribe(subscriptionId)
}
