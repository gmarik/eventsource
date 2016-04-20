package sse

import (
	"testing"

	"net/http/httptest"

	"log"
	"os"
	// "fmt"
	"sync"
	"time"
)

type ResponseRecorder struct {
	*httptest.ResponseRecorder
	gone  chan bool
	sleep time.Duration
}

func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		httptest.NewRecorder(),
		make(chan bool),
		0,
	}
}

func (m *ResponseRecorder) CloseNotify() <-chan bool { return m.gone }
func (m *ResponseRecorder) Close()                   { m.gone <- true }

func (m *ResponseRecorder) Write(data []byte) (int, error) {
	if m.sleep > 0 {
		time.Sleep(m.sleep)
	}
	return m.ResponseRecorder.Write(data)
}

func TestClients(t *testing.T) {
	if testing.Verbose() {
		Vlog = log.New(os.Stdout, "", log.LstdFlags)
	}

	Vlog.Println("Starting sse")
	sse := New()
	go sse.Serve()

	nclients := 1000

	stopping := &sync.WaitGroup{}
	stopping.Add(nclients)
	starting := &sync.WaitGroup{}
	starting.Add(nclients)

	clients := make([]*Conn, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewConn(NewResponseRecorder())
		go func(c *Conn, i int) {
			starting.Done()
			if err := c.Serve(sse); err != nil {
				t.Fatal(err)
				return
			}
			stopping.Done()
		}(clients[i], i)
	}

	events := []Event{
		{Data: "Hello", ID: "1", Event: "e1"},
		{Data: "World", ID: "2", Event: "e2"},
		{Data: "!!", ID: "3", Event: "e3"},
	}

	// wait for clients to connnect
	Vlog.Println("Clients connecting")
	starting.Wait()

	// TODO: this is a smell
	<-time.After(200 * time.Millisecond)

	Vlog.Println("Sending out events")
	for _, e := range events {
		done, err := sse.Push(e)
		if err != nil {
			t.Error(err)
		}
		<-done
	}

	// disconnect clients
	Vlog.Println("Clients leaving")
	for _, c := range clients {
		c.c.(*ResponseRecorder).Close()
	}

	Vlog.Println("Clients disconnecting")
	stopping.Wait()

	exp := `id: 1
event: e1
data: Hello

id: 2
event: e2
data: World

id: 3
event: e3
data: !!

`
	for i, c := range clients {
		got := c.c.(*ResponseRecorder).Body.String()
		if got != exp {
			t.Errorf("\nClient %d\nExp: %v\nGot: %v", i, exp, got)
		}
	}
}

func BenchmarkIt(t *testing.B) {
	sse := New()

	nclients := 1000
	starting := &sync.WaitGroup{}
	starting.Add(nclients)

	clients := make([]*ResponseRecorder, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewResponseRecorder()
		go func(w *ResponseRecorder) {
			conn := NewConn(w)
			starting.Done()
			conn.Serve(sse)
		}(clients[i])
	}

	go sse.Serve()

	starting.Wait()

	Vlog.Println("Sending out events")

	evt := Event{Data: "Hello", ID: "1", Event: "e1"}

	t.StartTimer()
	for i := 1; i < t.N; i += 1 {
		done, err := sse.Push(evt)
		if err != nil {
			t.Error(err)
		}
		<-done
	}
	t.StopTimer()
}
