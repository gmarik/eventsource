package sse

import (
	"testing"
	"time"
)

func TestConn_CloseNotify(t *testing.T) {
	sse := New()
	go sse.Serve()

	w := NewResponseRecorder()
	c := NewConn(w)

	done := make(chan struct{})
	go func() {
		c.Serve(sse)
		close(done)
	}()
	w.Close()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out")
	}
}

func TestConn_Close(t *testing.T) {
	sse := New()
	go sse.Serve()

	c := NewConn(NewResponseRecorder())

	done := make(chan struct{})
	go func() {
		c.Serve(sse)
		close(done)
	}()
	c.Close()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out")
	}
}
