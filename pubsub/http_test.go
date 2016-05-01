package pubsub

import (
	"net/http"
	"testing"
	"time"

	"github.com/gmarik/eventsource"
)

func TestBroker_ServeHTTP(t *testing.T) {
	w := NewResponseRecorder()
	r := &http.Request{}

	ps := New()
	go ps.Serve()
	go ps.ServeHTTP(w, r)

	<-time.After(100 * time.Millisecond)

	done, _ := ps.Push(sse.Event{Data: "Hello"})
	<-done

	w.Close()
	got := w.Body.String()
	exp := "data: Hello\n\n"

	if got != exp {
		t.Errorf("\nExp: %v\nGot: %v", exp, got)
	}
}
