package communication

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"util.tim/encrypto/core/shared"
	"util.tim/encrypto/core/subscribable"
)

func receivedMessage(message subscribable.Message, exchangeConnection Connection) {
	exchangeMessage := shared.ToMessage{}
	err := json.Unmarshal(message.Data, &exchangeMessage)
	if err != nil {
		fmt.Println("Error", err)
		exchangeConnection.Send("ERROR", shared.Data{
			Varient: "ERROR",
			Content: err,
		})
		return
	}

	exchangeConnection.Send(exchangeMessage.To, exchangeMessage.Data)
}

type ConnectArgs struct {
	exchange   AcceptConnections
	connection subscribable.Connection
}

func connect(
	args ConnectArgs,
	messageChannel <-chan subscribable.Message,
	disconnectChan <-chan bool,
) {
	exchangeConnection := args.exchange.Join()

	exchangeConnection.Subscribe(func(m shared.FromMessage) {
		outgoing := subscribable.OutgoingMessage{
			Variant: "Message",
			Body:    m,
		}
		args.connection.WriteMessage(outgoing)
	})

	fmt.Println("Writing a welcome message!")
	fmt.Printf("Exchange mailbox id -> [%s]\n", exchangeConnection.Id())
	args.connection.WriteMessage(subscribable.OutgoingMessage{
		Variant: "Welcome",
		Body:    []byte("Welcome to the exchange!"),
	})

	disconnected := false
	for {
		if disconnected {
			break
		}
		select {
		case message := <-messageChannel:
			receivedMessage(message, exchangeConnection)
		case <-disconnectChan:
			disconnected = true
			fmt.Printf("Connection ID [%s] was dropped\n", exchangeConnection.Id())
		}
	}
}

func registerConnection(incomingConnection <-chan subscribable.Connection, exchange AcceptConnections) {
	loopId := uuid.NewString()
	for connection := range incomingConnection {
		fmt.Printf("Setting up connection in loop: [%s]\n", loopId)
		messageChannel := make(chan subscribable.Message)
		disconnectChannel := make(chan bool)

		connection.Subscribe(newSubscription(messageChannel, disconnectChannel))
		outgoingMessage := subscribable.OutgoingMessage{
			Variant: "AvailableActions",
			Body:    []string{"connect", "reconnect"},
		}
		err := connection.WriteMessage(outgoingMessage)
		if err != nil {
			fmt.Println("Error sending message", err)
			return
		}

		args := ConnectArgs{
			exchange:   exchange,
			connection: connection,
		}

		disconnected := false
		connected := false
		for {
			if disconnected {
				break
			}
			if connected {
				fmt.Println("--------------CONNECTED---")
				break
			}
			select {
			case <-disconnectChannel:
				disconnected = true
			case message := <-messageChannel:
				canonicalVarient := strings.ToLower(message.Varient)
				fmt.Printf("Got a message Varient -> [%s]\n", canonicalVarient)
				if canonicalVarient == "connect" {
					go connect(args, messageChannel, disconnectChannel)
					connected = true
					// go handler(canonicalVarient)
					// handler = noOpHandler
				}

				if canonicalVarient == "reconnect" {
					fmt.Println("Should handle reconnect here")
				}
			}
		}
	}
}
