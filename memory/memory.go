package memory

import (
	"sync"

	"github.com/leolara/conveyor"
)

func NewBroker() conveyor.Broker {
	return newMemoryBroker()
}

func newMemoryBroker() *memoryBroker {
	b := &memoryBroker{
		subscribers:  make(map[string][]*memorySubscription),
		ackBlackHole: make(chan interface{}, 100),
	}

	b.run()

	return b
}

type memoryBroker struct {
	subscribers      map[string][]*memorySubscription
	subscribersMutex sync.Mutex
	subscriberIDgen  uint

	ackBlackHole chan interface{}
}

type memorySubscription struct {
	id uint

	topic string
	ch    chan conveyor.ReceiveEnvelope

	parent *memoryBroker
}

func (ms *memorySubscription) Receive() <-chan conveyor.ReceiveEnvelope {
	return ms.ch
}

func (ms *memorySubscription) Unsubscribe() {
	ms.parent.unsubscribe(ms)
	close(ms.ch)
	for range ms.ch {

	}
}

func (memorySubscription) Error() error {
	return nil
}

var _ conveyor.Subscription = (*memorySubscription)(nil)

func (b *memoryBroker) run() {
	go func() {
		for range b.ackBlackHole {
		}
	}()
}

func (b *memoryBroker) Subscribe(target string, options ...interface{}) <-chan conveyor.Subscription {
	s := &memorySubscription{
		topic:  target,
		ch:     make(chan conveyor.ReceiveEnvelope),
		parent: b,
	}

	b.subscribersMutex.Lock()
	b.subscriberIDgen++
	s.id = b.subscriberIDgen
	b.subscribers[target] = append(b.subscribers[target], s)
	b.subscribersMutex.Unlock()

	ch := make(chan conveyor.Subscription, 1)
	ch <- s
	close(ch)

	return ch
}

func (b *memoryBroker) Publish(target string, msgs <-chan conveyor.SendEnvelop, options ...interface{}) {
	go b.publishLoop(target, msgs)
}

func (b *memoryBroker) publishLoop(target string, msgs <-chan conveyor.SendEnvelop) {
	for msg := range msgs {
		b.subscribersMutex.Lock()
		subscribers := b.subscribers[target]
		b.subscribersMutex.Unlock()
		for _, s := range subscribers {
			go b.publish(
				s.ch,
				conveyor.NewReceiveEnvelopCopy(msg.Body(), b.ackBlackHole),
			)
		}

		go func () {msg.Error() <- nil}() // signal that the publication had no errors
	}
}

func (*memoryBroker) publish(ch chan conveyor.ReceiveEnvelope, envelope conveyor.ReceiveEnvelope) {
	// We will panic if the channel is closed, avoid it having an effect
	defer func() {
		recover()
	}()

	ch <- envelope
}

func (b *memoryBroker) unsubscribe(sub *memorySubscription) {
	b.subscribersMutex.Lock()

	subs := b.subscribers[sub.topic]

	var i int
	var search *memorySubscription
	for i, search = range subs {
		if search.id == sub.id {
			break
		}
	}

	if i < len(subs) {
		subs[i] = subs[len(subs)-1]
		subs[len(subs)-1] = nil
		subs = subs[:len(subs)-1]
	}

	b.subscribers[sub.topic] = subs

	b.subscribersMutex.Unlock()
}
