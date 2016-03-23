package sse

import (
	"net/http"
	"testing"
	"time"
)

func TestBroker_ServeHTTP(t *testing.T) {
	w := NewTestResponseWriter()
	r := &http.Request{}

	sse := New()
	go sse.Serve()
	go sse.ServeHTTP(w, r)

	<-time.After(100 * time.Millisecond)

	done, _ := sse.Push(Event{Data: "Hello"})
	<-done

	w.Close()
	got := w.Body.String()
	exp := "data: Hello\n\n"

	if got != exp {
		t.Errorf("\nExp: %v\nGot: %v", exp, got)
	}
}
