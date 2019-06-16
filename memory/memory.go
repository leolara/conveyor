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
	topic string

	id     uint
	parent *memoryBroker

	ch     chan conveyor.ReceiveEnvelope
	stopCh chan interface{}
}

func (ms *memorySubscription) Receive() <-chan conveyor.ReceiveEnvelope {
	return ms.ch
}

func (ms *memorySubscription) Unsubscribe() {
	ms.parent.unsubscribe(ms)
	close(ms.stopCh)
}

func (ms *memorySubscription) publication(envelope conveyor.ReceiveEnvelope) {
	select {
	case <-ms.stopCh:
		return
	default:
	}
	select {
	case <-ms.stopCh:
	case ms.ch <- envelope:
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
		parent: b,
		ch:     make(chan conveyor.ReceiveEnvelope),
		stopCh: make(chan interface{}),
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
			go s.publication(conveyor.NewReceiveEnvelopCopy(msg.Body(), b.ackBlackHole))
		}

		go func() { msg.Error() <- nil }() // signal that the publication had no errors
	}
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
