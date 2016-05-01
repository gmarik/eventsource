package pubsub

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gmarik/eventsource"
)

func Test_ResponseWriter_Close(t *testing.T) {
	w := NewResponseRecorder()
	w.Close()
	select {
	case <-w.done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out")
	}
	select {
	case <-w.ctx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out")
	}
}

func Test_ResponseWriter_Write(t *testing.T) {
	w := NewResponseRecorder()
	w.Write([]byte("Hello"))
	got := string(w.Body.Bytes())
	exp := "Hello"
	if got != exp {
		t.Errorf("\nExp: %v\nGot: %v", exp, got)
	}
}

func TestBroker_ServeHTTP(t *testing.T) {
	w := NewResponseRecorder()

	if nil == responseWriteFlushCloseNotifier(w) {
		t.Fatal("Not implemented")
	}

	ps := New()

	nclients := 1
	wgstart := &sync.WaitGroup{}
	wgstop := &sync.WaitGroup{}
	wgstart.Add(nclients)
	wgstop.Add(nclients)

	ps.joinCallback = func() {
		wgstart.Done()
	}
	ps.leaveCallback = func() {
		wgstop.Done()
	}

	go ps.Listen()
	go ps.ServeHTTP(w, &http.Request{})

	wgstart.Wait()

	done, _ := ps.Push(sse.Event{Data: "Hello"})
	<-done

	w.Close()

	wgstop.Wait()

	if w.Code != http.StatusOK {
		t.Errorf("%v\nWant: %v", w.Code, http.StatusOK)
	}

	got := w.Body.String()
	exp := "data: Hello\n\n"

	if got != exp {
		t.Errorf("\nExp: %v\nGot: %v", exp, got)
	}
}
