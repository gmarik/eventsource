package sse

import (
	"testing"

	"net/http/httptest"

	"log"
	"os"
	// "fmt"
	"sync"
)

type TestSSE struct {
	*Broker

	wgstart, wgstop *sync.WaitGroup
}

func (t *TestSSE) join(c *Conn) <-chan (chan []byte) {
	defer t.wgstart.Done()
	return t.Broker.join(c)
}

func (t *TestSSE) leave(c *Conn) {
	defer t.wgstop.Done()
	t.Broker.leave(c)
}

type ResponseRecorder struct {
	*httptest.ResponseRecorder
	gone chan bool
}

func NewResponseRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		httptest.NewRecorder(),
		make(chan bool),
	}
}

func (m *ResponseRecorder) CloseNotify() <-chan bool { return m.gone }
func (m *ResponseRecorder) Close()                   { m.gone <- true }

func (m *ResponseRecorder) Write(data []byte) (int, error) {
	return m.ResponseRecorder.Write(data)
}

func TestClients(t *testing.T) {
	if testing.Verbose() {
		Vlog = log.New(os.Stdout, "", log.LstdFlags)
	}

	Vlog.Println("Starting sse")
	sse := &TestSSE{
		Broker:  New(),
		wgstart: &sync.WaitGroup{},
		wgstop:  &sync.WaitGroup{},
	}
	go sse.Serve()

	nclients := 10000

	sse.wgstart.Add(nclients)
	sse.wgstop.Add(nclients)

	clients := make([]*Conn, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewConn(NewResponseRecorder())
		go func(c *Conn, i int) {
			if err := c.Serve(sse); err != nil {
				t.Fatal(err)
				return
			}
		}(clients[i], i)
	}

	events := []Event{
		{Data: "Hello", ID: "1", Event: "e1"},
		{Data: "World", ID: "2", Event: "e2"},
		{Data: "!!", ID: "3", Event: "e3"},
	}

	// wait for clients to connnect
	Vlog.Println("Clients connecting")
	sse.wgstart.Wait()

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
	sse.wgstop.Wait()

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

func Benchmark100(t *testing.B) {
	benchmarkN(t, 100)
}
func Benchmark1000(t *testing.B) {
	benchmarkN(t, 1000)
}
func Benchmark10000(t *testing.B) {
	benchmarkN(t, 10000)
}

func benchmarkN(t *testing.B, nclients int) {
	sse := &TestSSE{
		Broker:  New(),
		wgstart: &sync.WaitGroup{},
		wgstop:  &sync.WaitGroup{},
	}
	go sse.Serve()

	sse.wgstart.Add(nclients)

	clients := make([]*ResponseRecorder, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewResponseRecorder()
		go func(w *ResponseRecorder) {
			conn := NewConn(w)
			conn.Serve(sse)
		}(clients[i])
	}

	sse.wgstart.Wait()

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
