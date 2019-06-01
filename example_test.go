package conveyor_test

import (
	"github.com/leolara/conveyor"
	"github.com/leolara/conveyor/memory"
	"sync"
	"time"
)

func Example() {
	var wd sync.WaitGroup

	// we use a in-memory broker for testing and examples, you can find many implementations for different brokers at
	// https://github.com/leolara/conveyor-impl
	b := memory.NewBroker()

	wd.Add(2)
	go Producer(b, wd)
	go Consumer(b, wd)

	wd.Wait()
}

func Producer(b conveyor.Broker, wd sync.WaitGroup) {
	defer wd.Done()

	// This creates a channel to publish messages
	pubChan := make(chan conveyor.SendEnvelop)

	// Links the publication channel to the queue or topic, from now on, what we write on pubChan will get published
	// into "exampleTopic"
	b.Publish("exampleTopic", pubChan)

	// Creates a channel to receive publication errors, we can use one error channel for each publication
	// or one for all publications
	pubChanErr := make(chan error)

	// Sends a message to "exampleTopic" with content []byte{24}, and with errors going pubChanErr
	pubChan <- conveyor.NewSendEnvelop([]byte{24}, pubChanErr)

	// We MUST read from pubChanErr, as it is idiomatic in Go a nil error means success.
	// We could use the same error channel for each publication or use a different time each time.
	select {
	case err := <-pubChanErr:
		if err != nil {
			panic(err)
		}
	case <-time.After(10 * time.Millisecond):
		panic("Did not receive empty error")
	}

	// Closing pubChan will finish the go routine in the broker that handles this publication, releasing resources
	close(pubChan)
}

func Consumer(b conveyor.Broker, wd sync.WaitGroup) {
	defer wd.Done()

	// The object sub is a subscription to "exampleTopic", as you can see the subscription is
	// async as it returns a channel that will eventually return the subscription object
	sub := <-b.Subscribe("exampleTopic")

	// We should check if there was an error subscribing
	if sub.Error() != nil {
		panic(sub.Error())
	}

	// sub.Receive() give us a channel from which receive messages
	select {
	case envelope := <-sub.Receive():
		// envelope.Body() returns the content of the message
		if len(envelope.Body()) != 1 && envelope.Body()[0] != 24 {
			panic("received wrong data")
		}
		// We should ack the message when we are done with it
		envelope.Ack() <- nil
	case <-time.After(10 *time.Millisecond):
		panic("Did not receive message")
	}

	// after this, we can repeat reading from sub.Receive() and writing to envelope.Ack()

	// We can unsubscribe to stop receiving messages
	sub.Unsubscribe()

	select {
	case _, ok := <-sub.Receive():
		if ok {
			panic("shouldn't receive anything")
		}
	case <-time.After(10 * time.Millisecond):
		// OK, it should not receive message
	}
}
