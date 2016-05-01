package pubsub

import (
	"testing"
	"time"
)

func TestBroker_Close(t *testing.T) {
	es := New()

	done := make(chan struct{})
	go func() {
		es.Serve()
		close(done)
	}()

	es.Close()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out")
	}
}
