package concurrent

import (
	"fmt"
	"sync"

	api "util.tim/encrypto/core/actors"
	"util.tim/encrypto/core/shared"
)

type connection struct {
	id       string
	exchange *concurrentExchange
	outbox   *outbox
}

func (connection *connection) Id() string {
	return connection.id
}
func (connection *connection) Subscribe(subscriber func(shared.FromMessage)) {
	connection.outbox.subscribe(subscriber)
}
func (connection *connection) Send(toId string, message shared.Data) {
	outbox, found := connection.exchange.outboxMap[toId]
	fromOutbox, fromFound := connection.exchange.outboxMap[connection.id]

	if !found {
		if fromFound {
			fromOutbox.writeMessage(shared.FromMessage{
				From: "System",
				Data: shared.Data{
					Varient: "Error",
					Content: fmt.Sprintf("Destination does not exist [%s]", toId),
				},
			})
		}
		return
	}
	if found {
		if toId != connection.Id() {
			outbox.writeMessage(shared.FromMessage{
				From: connection.Id(),
				Data: message,
			})
		} else {
			outbox.writeMessage(shared.FromMessage{
				From: "System",
				Data: shared.Data{
					Varient: "Error",
					Content: "Sending a message to oneself is not supported",
				},
			})
		}
	}
}

func newConnection(id string, exchange *concurrentExchange, outbox *outbox) api.Connection {
	return &connection{
		id:       id,
		exchange: exchange,
		outbox:   outbox,
	}
}

type outbox struct {
	data            chan *shared.FromMessage
	subscriberMutex sync.RWMutex
	subscriber      func(shared.FromMessage)
}

func (outbox *outbox) writeMessage(message shared.FromMessage) {
	outbox.data <- &message
}

func (outbox *outbox) subscribe(subscriber func(shared.FromMessage)) {
	outbox.subscriberMutex.Lock()
	outbox.subscriber = subscriber
	outbox.subscriberMutex.Unlock()

	go func() {
		for {
			message := <-outbox.data

			outbox.subscriberMutex.Lock()
			outbox.subscriber(*message)
			outbox.subscriberMutex.Unlock()
		}
	}()
}

func newOutBox() *outbox {
	return &outbox{
		data:            make(chan *shared.FromMessage, 10),
		subscriberMutex: sync.RWMutex{},
		subscriber:      func(m shared.FromMessage) {},
	}
}

type concurrentExchange struct {
	idProvider      api.IdProvider
	outboxMap       map[string]*outbox
	connectionMutex sync.RWMutex
}

func (exchange *concurrentExchange) readSynchronized(query func()) {
	exchange.connectionMutex.RLock()
	query()
	exchange.connectionMutex.RUnlock()
}
func (exchange *concurrentExchange) writeSynchronized(command func()) {
	exchange.connectionMutex.Lock()
	command()
	exchange.connectionMutex.Unlock()
}

func (exchange *concurrentExchange) Connect() api.Connection {
	var connection api.Connection

	exchange.writeSynchronized(func() {
		id := exchange.idProvider.NextId()
		outbox := newOutBox()
		connection = newConnection(id, exchange, outbox)

		exchange.outboxMap[id] = outbox
	})

	return connection
}
func (exchange *concurrentExchange) Reconnect(id string) (api.Connection, error) {
	var outbox *outbox
	found := false
	exchange.readSynchronized(func() {
		outbox, found = exchange.outboxMap[id]
	})

	if found {
		return newConnection(id, exchange, outbox), nil
	}

	return nil, fmt.Errorf("there is no outbox for the provided id [%s]", id)
}

func NewConcurrentExchange(idProvider api.IdProvider) api.Exchange {
	return &concurrentExchange{
		idProvider:      idProvider,
		outboxMap:       make(map[string]*outbox),
		connectionMutex: sync.RWMutex{},
	}
}
