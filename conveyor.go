// Package conveyor provides an abstraction for message queues, brokers, buses and the sort. It is idiomatic and asynchronous because it uses Go channels everywhere
//
//In another repo there are implementations for redis, rabbitmq, ...
//
//This repository includes an in-memory message broker implementation that is useful for testing
//
// Please, check README.md for an overview https://github.com/leolara/conveyor/README.md
package conveyor

// Broker interface for message brokers
type Broker interface {
	// Subscribe to a topic/queue
	//  + target is the name of what you are subscribing to
	//  + options are implementation dependant
	// returns a Subscription object asynchronously
	Subscribe(target string, options ...interface{}) <-chan Subscription
	// Publish to a topic/queue
	//  + target is the name of what you are publishing to
	//  + msgs is a channel on which you will send SendEnvelop objects
	//  + options are implementation dependant
	// After calling this method you can publish as many messages as necessary using msgs channel
	Publish(target string, msgs <-chan SendEnvelop, options ...interface{})
}

// Message contains a body, both ReceiveEnvelope and SendEnvelop extend this
type Message interface {
	Body() []byte
}

// ReceiveEnvelope encapsulates a message that is being received
type ReceiveEnvelope interface {
	Message
	// Ack MUST return a channel to which the message receiver must write once to ACK, writing nil means a to ACK in
	// all implementations, other values are implementation dependent
	Ack() chan<- interface{}
}

// SendEnvelop encapsulates a message that is being received
type SendEnvelop interface {
	Message
	// Error MUST return a channel, the broker will write nil on success or an error if failure
	Error() chan<- error
}

// Subscription encapsulates a subscription to a topic/queue
type Subscription interface {
	// Receive returns a channel to read and receive messages. The returned channel is always the same, so it is not necessary
	// to call this method for every read.
	Receive() <-chan ReceiveEnvelope
	// Unsubscribe lives up to its name
	Unsubscribe()
	Error() error
}

type message struct {
	body []byte
}

func (m message) Body() []byte {
	return m.body
}

var _ Message = (*message)(nil)

type sendEnvelop struct {
	message
	err chan error
}

func (m sendEnvelop) Error() chan<- error {
	return m.err
}

var _ SendEnvelop = (*sendEnvelop)(nil)

type receiveEnvelope struct {
	message
	ack chan<- interface{}
}

func (m receiveEnvelope) Ack() chan<- interface{} {
	return m.ack
}

var _ ReceiveEnvelope = (*receiveEnvelope)(nil)

// NewSendEnvelop creates an immutable SendEnvelope.
// Useful when you are sending messages, you do not have to create your own implementation of SendEnvelope.
// It is immutable but contains *references* to body and err, so you should be aware of that.
func NewSendEnvelop(body []byte, err chan error) SendEnvelop {
	return sendEnvelop{
		message: message{body: body},
		err:     err,
	}
}

// NewReceiveEnvelop creates an immutable ReceiveEnvelope.
// Useful when writing a broker implementation and need to send messages to subscribers, you do not have to create your
// own implementation of ReceiveEnvelope.
// It is immutable but contains *references* to body and ack, so you should be aware of that
func NewReceiveEnvelop(body []byte, ack chan<- interface{}) ReceiveEnvelope {
	return receiveEnvelope{
		message: message{body: body},
		ack:     ack,
	}
}

// NewReceiveEnvelopCopy does like NewReceiveEnvelop but using a copy of body instead of keeping the reference
// It is immutable but contains a *reference* to ack, so you should be aware of that
func NewReceiveEnvelopCopy(body []byte, ack chan<- interface{}) ReceiveEnvelope {
	copied := make([]byte, len(body))
	copy(copied, body)
	return NewReceiveEnvelop(copied, ack)
}
