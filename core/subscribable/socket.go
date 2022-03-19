package subscribable

import "fmt"

type socketConnection struct {
	socket             Socket
	subscribeChannel   chan<- SubscriptionAndId
	unSubscribeChannel chan<- SubscriptionId
	nextSubscriptionId int64
}

func (wrapper *socketConnection) WriteMessage(message OutgoingMessage) error {
	return wrapper.socket.WriteJSON(message)
}

func (wrapper *socketConnection) Subscribe(subscription Subscription) SubscriptionId {
	subscriptionId := NewSubscriptionId(wrapper.nextSubscriptionId)
	wrapper.nextSubscriptionId = wrapper.nextSubscriptionId + 1
	fmt.Println("Pushing the subscription")
	wrapper.subscribeChannel <- SubscriptionAndId{
		SubscriptionId: subscriptionId,
		Subscription:   &subscription,
	}
	fmt.Println("Done pushing the subscription")

	return subscriptionId
}
func (wrapper *socketConnection) UnSubscribe(id SubscriptionId) {
	wrapper.unSubscribeChannel <- id
}

func newSocketConnection(socket Socket, subscribeChannel chan<- SubscriptionAndId, unSubscribeChannel chan<- SubscriptionId) Connection {
	return &socketConnection{
		socket:             socket,
		subscribeChannel:   subscribeChannel,
		unSubscribeChannel: unSubscribeChannel,
	}
}
