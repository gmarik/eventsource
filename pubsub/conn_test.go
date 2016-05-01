package pubsub

import (
	"testing"
	"time"
)

func TestConn_Cancel(t *testing.T) {
	w := NewResponseRecorder()

	ps := New()
	go ps.Listen()

	done := make(chan struct{})
	go func() {
		ps.Serve(w.ctx, w)
		close(done)
	}()
	w.Close()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out")
	}
}
