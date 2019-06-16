package memory

import (
	"runtime"
	"testing"
	"time"

	"github.com/leolara/conveyor"
)

func TestMemoryBroker(t *testing.T) {
	b := NewBroker()

	sub := <-b.Subscribe("testTopic")

	if sub.Error() != nil {
		t.Fatal(sub.Error())
	}

	pubChan := make(chan conveyor.SendEnvelop)
	pubChanErr := make(chan error)
	b.Publish("testTopic", pubChan)

	pubChan <- conveyor.NewSendEnvelop([]byte{24}, pubChanErr)

	select {
	case err := <-pubChanErr:
		if err != nil {
			t.Errorf("got publication error: %s", err)
		}
	case <-time.After(time.Microsecond):
		t.Error("Did not receive empty error")
	}

	select {
	case envelope, ok := <-sub.Receive():
		if !ok {
			t.Error("channel closed")
		}
		if len(envelope.Body()) != 1 && envelope.Body()[0] != 24 {
			t.Error("received wrong data")
		}
		envelope.Ack() <- nil
	case <-time.After(time.Microsecond):
		t.Error("Did not receive message")
	}

	pubChan <- conveyor.NewSendEnvelop([]byte{42}, pubChanErr)
	runtime.Gosched()

	sub.Unsubscribe()
	runtime.Gosched()

	select {
	case _, ok := <-sub.Receive():
		if ok {
			t.Error("shouldn't receive anything")
		}
	case <-time.After(10 * time.Millisecond):
		// OK
	}

	pubChan <- conveyor.NewSendEnvelop([]byte{42}, pubChanErr)
	runtime.Gosched()

	select {
	case _, ok := <-sub.Receive():
		if ok {
			t.Error("shouldn't receive anything")
		}
	case <-time.After(10 * time.Millisecond):
		// OK
	}
}
