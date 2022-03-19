package concurrent_test

import (
	"fmt"
	"testing"
	"time"

	api "util.tim/encrypto/core/actors"
	"util.tim/encrypto/core/actors/concurrent"
	"util.tim/encrypto/core/shared"
)

type testIdProvider struct {
	count int
}

func (idProvider *testIdProvider) NextId() string {
	id := fmt.Sprintf("[%d]", idProvider.count)

	idProvider.count += 1

	return id
}

func newTestIdProvider() api.IdProvider {
	return &testIdProvider{}
}

func newTestData(message string) shared.Data {
	return shared.Data{
		Varient: "TestData",
		Content: message,
	}
}

func Test_CanMultipleConnections(t *testing.T) {
	exchange := concurrent.NewConcurrentExchange(newTestIdProvider())

	outboxOne := exchange.Connect()
	outboxTwo := exchange.Connect()

	idOne := outboxOne.Id()
	idTwo := outboxTwo.Id()

	if idOne == idTwo {
		t.Log("different connections should have different ids")
		t.FailNow()
	}
}

func Test_CanSendMessagesBetweenConnections_subscribeAfterMessageSent(t *testing.T) {
	exchange := concurrent.NewConcurrentExchange(newTestIdProvider())

	connectionOne := exchange.Connect()
	connectionTwo := exchange.Connect()

	idOne := connectionOne.Id()
	idTwo := connectionTwo.Id()

	connectionOne.Send(idTwo, newTestData("Hi"))
	connectionTwo.Send(idOne, newTestData("Hello!"))

	messageToOneReceived := make(chan shared.FromMessage)
	messageToTwoReceived := make(chan shared.FromMessage)

	connectionOne.Subscribe(func(m shared.FromMessage) {
		go func() {
			messageToOneReceived <- m
		}()
	})
	connectionTwo.Subscribe(func(m shared.FromMessage) {
		go func() {
			messageToTwoReceived <- m
		}()
	})

	select {
	case messageToOne := <-messageToOneReceived:
		if messageToOne.From != idTwo {
			t.Log("Expected the message to come from connection Two")
			t.FailNow()
		}
		fmt.Println(messageToOne.From)
		fmt.Println(messageToOne.Data)
	case <-time.After(time.Millisecond * 500):
		t.Log("timedout: did not receive messageToOne")
		t.FailNow()
	}

	select {
	case messageToTwo := <-messageToTwoReceived:
		if messageToTwo.From != idOne {
			t.Log("Expected the message to come from connection One")
			t.FailNow()
		}
		fmt.Println(messageToTwo.From)
		fmt.Println(messageToTwo.Data)
	case <-time.After(time.Millisecond * 500):
		t.Log("timedout: did not receive messageToTwo")
		t.FailNow()
	}
}

func Test_CanSendMessagesBetweenConnections_subscribeBeforeMessageSent(t *testing.T) {
	exchange := concurrent.NewConcurrentExchange(newTestIdProvider())

	connectionOne := exchange.Connect()
	connectionTwo := exchange.Connect()

	messageToOneReceived := make(chan shared.FromMessage)
	connectionOne.Subscribe(func(m shared.FromMessage) {
		go func() {
			messageToOneReceived <- m
		}()
	})
	messageToTwoReceived := make(chan shared.FromMessage)
	connectionTwo.Subscribe(func(m shared.FromMessage) {
		go func() {
			messageToTwoReceived <- m
		}()
	})

	idOne := connectionOne.Id()
	idTwo := connectionTwo.Id()

	connectionOne.Send(idTwo, newTestData("Hi"))
	connectionTwo.Send(idOne, newTestData("Hello!"))

	select {
	case messageToOne := <-messageToOneReceived:
		if messageToOne.From != idTwo {
			t.Log("Expected the message to come from connection Two")
			t.FailNow()
		}
		fmt.Println(messageToOne.From)
		fmt.Println(messageToOne.Data)
	case <-time.After(time.Millisecond * 500):
		t.Log("timedout: did not receive messageToOne")
		t.FailNow()
	}

	select {
	case messageToTwo := <-messageToTwoReceived:
		if messageToTwo.From != idOne {
			t.Log("Expected the message to come from connection One")
			t.FailNow()
		}
		fmt.Println(messageToTwo.From)
		fmt.Println(messageToTwo.Data)
	case <-time.After(time.Millisecond * 500):
		t.Log("timedout: did not receive messageToTwo")
		t.FailNow()
	}
}

func Test_CanSwitchTheSubscriber(t *testing.T) {
	exchange := concurrent.NewConcurrentExchange(newTestIdProvider())

	connectionOne := exchange.Connect()
	connectionTwo := exchange.Connect()

	messageToTwoReceivedFirstSubscriber := make(chan shared.FromMessage)
	connectionTwo.Subscribe(func(m shared.FromMessage) {
		go func() {
			messageToTwoReceivedFirstSubscriber <- m
		}()
	})

	idOne := connectionOne.Id()
	idTwo := connectionTwo.Id()

	firstMessage := "message to subscriber one"
	connectionOne.Send(idTwo, newTestData(firstMessage))
	select {
	case messageToTwo := <-messageToTwoReceivedFirstSubscriber:
		if messageToTwo.Data.Content != firstMessage {
			t.Log("Did not receive the expected message, instead got", messageToTwo.Data.Content)
			t.Fail()
		}
	case <-time.After(time.Millisecond * 500):
		t.Log("timedout: did not receive messageToTwo")
	}

	messageToTwoReceivedSecondSubscriber := make(chan shared.FromMessage, 10)
	connectionTwo.Subscribe(func(m shared.FromMessage) {
		t.Log(m.Data.Content)
		messageToTwoReceivedSecondSubscriber <- m
	})

	connectionOne.Send(idTwo, newTestData("message to subscriber two"))
	connectionOne.Send(idTwo, newTestData("should be the other subscriber"))
	connectionOne.Send(idTwo, newTestData("should be the other subscriber"))
	connectionOne.Send(idTwo, newTestData("should be the other subscriber"))

	select {
	case messageToTwo := <-messageToTwoReceivedSecondSubscriber:
		if messageToTwo.From != idOne {
			t.Log("Expected the message to come from connection One")
			t.FailNow()
		}
		if messageToTwo.Data.Content != "message to subscriber two" {
			t.Log("did not receive the expected message", messageToTwo.Data.Content)
			t.FailNow()
		}
		fmt.Println(messageToTwo.From)
		fmt.Println(messageToTwo.Data)
	case <-time.After(time.Millisecond * 500):
		t.Log("timedout: did not receive messageToTwo")
		t.FailNow()
	}

	select {
	case <-messageToTwoReceivedFirstSubscriber:
		t.Log("The first subscriber should not receive any addtional messages")
		t.FailNow()
	case <-time.After(time.Millisecond * 500):
		t.Log("Did not receive message after changing subscribers")
	}
}

func Test_CanReceiveMultipleMessages(t *testing.T) {
	exchange := concurrent.NewConcurrentExchange(newTestIdProvider())

	connectionOne := exchange.Connect()
	connectionTwo := exchange.Connect()

	expectedNumberOfMessages := 9
	complete := make(chan bool, 1)
	received := make([]*shared.FromMessage, expectedNumberOfMessages)
	receivedCount := 0
	connectionTwo.Subscribe(func(m shared.FromMessage) {
		fmt.Println(receivedCount)
		received[receivedCount] = &m
		receivedCount += 1
		if receivedCount == expectedNumberOfMessages {
			complete <- true
		}
	})

	idOne := connectionOne.Id()
	idTwo := connectionTwo.Id()

	messages := make([]string, expectedNumberOfMessages)
	for i := range messages {
		messages[i] = fmt.Sprintf("message -> [%d]", i)
	}

	for _, message := range messages {
		connectionOne.Send(idTwo, newTestData(message))
	}

	select {
	case <-complete:
		break
	case <-time.After(time.Millisecond * 500):
		t.Log("did not recieve the expected number of messages in time")
		t.FailNow()
	}

	for _, m := range received {
		if m.From != idOne {
			t.Log(fmt.Sprintf("expected from to have id [%s] but was [%s]", idOne, m.From))
			t.FailNow()
		}

		t.Log(m.Data)
	}
}

func Test_CanReconnect_usingAnExistingId(t *testing.T) {
	exchange := concurrent.NewConcurrentExchange(newTestIdProvider())

	connectionOne := exchange.Connect()
	connectionTwo := exchange.Connect()

	subOne := make(chan *shared.FromMessage)
	connectionTwo.Subscribe(func(m shared.FromMessage) {
		go func() {
			subOne <- &m
		}()
	})

	idTwo := connectionTwo.Id()

	connectionOne.Send(idTwo, newTestData("hi there first subscription"))

	select {
	case message := <-subOne:
		t.Log(message)
	case <-time.After(time.Millisecond * 500):
		t.Log("timed out, did not receive the first message")
		t.FailNow()
	}

	reconnectionTwo, _ := exchange.Reconnect(idTwo)

	subTwo := make(chan *shared.FromMessage)
	reconnectionTwo.Subscribe(func(m shared.FromMessage) {
		go func() {
			subTwo <- &m
		}()
	})

	connectionOne.Send(idTwo, shared.Data{
		Varient: "TestData",
		Content: "hi there second subscription",
	})

	select {
	case message := <-subTwo:
		t.Log(message)
	case <-time.After(time.Millisecond * 500):
		t.Log("the reconnected subscriber never received a message")
		t.FailNow()
	}
}

func Test_ReceiveReconnectError_usingAnNonExistingId(t *testing.T) {
	exchange := concurrent.NewConcurrentExchange(newTestIdProvider())

	connectionOne := exchange.Connect()
	connectionTwo := exchange.Connect()

	subOne := make(chan *shared.FromMessage)
	connectionTwo.Subscribe(func(m shared.FromMessage) {
		go func() {
			subOne <- &m
		}()
	})

	idTwo := connectionTwo.Id()

	connectionOne.Send(idTwo, newTestData("hi there first subscription"))

	select {
	case message := <-subOne:
		t.Log(message)
	case <-time.After(time.Millisecond * 500):
		t.Log("timed out, did not receive the first message")
		t.FailNow()
	}

	doesNotExistId := "does not exist"
	reconnectionTwo, err := exchange.Reconnect(doesNotExistId)
	if reconnectionTwo != nil {
		t.Log("should not have received a connection")
		t.FailNow()
	}

	if err == nil {
		t.Log("expected a reconnection error")
		t.FailNow()
	}

	if err.Error() != fmt.Sprintf("there is no outbox for the provided id [%s]", doesNotExistId) {
		t.Log("expected a reconnection error but received a different error -> ", err)
		t.FailNow()
	}
}
