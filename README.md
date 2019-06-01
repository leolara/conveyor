# Conveyor, an idiomatic and asynchronous Go message broker abstraction

This Go module provides an abstraction for message queues, brokers, buses and the sort. It is idiomatic and asynchronous
because it uses Go channels everywhere

In another repo there are implementations for redis, rabbitmq, ...

This Go module includes an in-memory message broker implementation that is useful for testing

## How does it work

With conveyor you send and receive messages using Go channels.

### Sending messages

To write to a queue called `myqueue` you would do:

```go
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
err := <-pubChanErr
if err != nil {
    fmt.Errorf("got publication error: %s", err)
}

// here repeat as many times as necessary writing on pubChan and reading from pubChanErr

// Closing pubChan will finish the go routine in the broker that handles this publication, releasing resources
close(pubChan)

```

### Receiving messages

In the other side is as idiomatic, asynchronous and channel based as when sending messages.

To start receiving messages from `myqueue`:

```go

// The object sub is a subscription to "exampleTopic", as you can see the subscription is
// async as it returns a channel that will eventually return the subscription object
sub := <-b.Subscribe("exampleTopic")

// We should check if there was an error subscribing
if sub.Error() != nil {
    panic(sub.Error())
}

// sub.Receive() give us a channel from which receive messages
envelope := <-sub.Receive()

// envelope.Body() returns the content of the message
if len(envelope.Body()) != 1 && envelope.Body()[0] != 24 {
    t.Error("received wrong data")
}

// We should ack the message when we are done with it
envelope.Ack() <- nil

// after this, we can repeat reading from sub.Receive() and writing to envelope.Ack()

// Once we are done, we can unsubscribe to stop receiving messages
sub.Unsubscribe()

```

### Implementation specific details

Both `broker.Subscribe` and `broker.Publish` allow optional parameters that are dependent on the broker implementation.
Writing `nil` into `envelope.Ack()` means a successful processing of the message, sending other values are implementation dependent.

## Why is this different from matryer/vice

The go module vice goal is to being able to use Go channels over message brokers transparently, so the code reading and
writing does not have to know that there are actually a distributed messages underneath. Conveyor goal is to being able
to use message and event brokers, buses and the sort using an idiomatic and asynchonous paradigm, which in Go means using
channels.

In summary they are the sides of the same coin:

Vice advantages: works with any code that uses `chan []byte`

Conveyor advantages: the user controls acknowledgments and can respond to publication errors, also access broker implementation details

Vice disadvantages: user cannot control acknowledgments and cannot respond to publication errors

Conveyor disadvantages: code that uses it needs to be aware of Conveyor interfaces

## Contributing

If you have some idea for some changes, please create an issue explaining your idea before sending a pull request. We are
happy to have your help.

