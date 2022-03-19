package subscribable

import (
	"encoding/json"
	"fmt"
)

type Message struct {
	Varient string
	Data    json.RawMessage
}

type Subscription interface {
	ConnectionDropped()
	ReceivedMessage(Message)
}

type SubscriptionId struct {
	id int64
}

func (subscriptionId SubscriptionId) Id() int64 {
	return subscriptionId.id
}

func NewSubscriptionId(id int64) SubscriptionId {
	return SubscriptionId{
		id: id,
	}
}

type Connection interface {
	WriteMessage(OutgoingMessage) error
	Subscribe(Subscription) SubscriptionId
	UnSubscribe(SubscriptionId)
}

type Socket interface {
	ReadJSON(interface{}) error
	WriteJSON(interface{}) error
}

type SubscriptionAndId struct {
	SubscriptionId SubscriptionId
	Subscription   *Subscription
}

func subscriptionLoop(
	subscriptions map[int64]*Subscription,
	incomingMessageChan <-chan Message,
	disconnectChan <-chan bool,
	addSubscription <-chan SubscriptionAndId,
	removeSubscription <-chan SubscriptionId,
) {
	disconnected := false
	for {
		if disconnected {
			fmt.Println("subscriptionLoop Ending")
			break
		}

		select {
		case <-disconnectChan:
			fmt.Println("Socket was disconnected")
			disconnected = true
			for _, subscription := range subscriptions {
				(*subscription).ConnectionDropped()
			}
		case message := <-incomingMessageChan:
			fmt.Println("Dispatching message to subscribers: ", len(subscriptions))
			for _, subscription := range subscriptions {
				(*subscription).ReceivedMessage(message)
			}
			fmt.Println("Done dispatching messages")
		case newSubscription := <-addSubscription:
			fmt.Printf("Subscribing -> [%d]\n", newSubscription.SubscriptionId.id)
			subscriptions[newSubscription.SubscriptionId.Id()] = newSubscription.Subscription
		case removeId := <-removeSubscription:
			fmt.Printf("Unsubscribing -> [%d]\n", removeId.Id())
			delete(subscriptions, removeId.Id())
		}
	}
}

func listenForIncomingMessages(
	connection Socket,
	incomingMessageChan chan<- Message,
	disconnectChan chan<- bool,
) {
	fmt.Println("Ready to read some incoming messages...")
	for {
		message := &Message{}
		err := connection.ReadJSON(message)

		fmt.Println("Read some incoming message!")

		if err != nil {
			fmt.Println("Error reading message", err)
			break
		}

		incomingMessageChan <- *message
	}

	disconnectChan <- true
}

func NewConnection(connection Socket) Connection {
	incomingMessageChan := make(chan Message)
	disconnectChan := make(chan bool)

	addSubscriptionChan := make(chan SubscriptionAndId)
	removeSubscriptionChan := make(chan SubscriptionId)
	subscriptions := make(map[int64]*Subscription)

	go listenForIncomingMessages(connection, incomingMessageChan, disconnectChan)
	go subscriptionLoop(
		subscriptions,
		incomingMessageChan,
		disconnectChan,
		addSubscriptionChan,
		removeSubscriptionChan,
	)

	return newSocketConnection(connection, addSubscriptionChan, removeSubscriptionChan)
}
