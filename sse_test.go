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

type TestResponseWriter struct {
	*httptest.ResponseRecorder
	gone  chan bool
	sleep time.Duration
}

func NewTestResponseWriter() *TestResponseWriter {
	return &TestResponseWriter{
		httptest.NewRecorder(),
		make(chan bool),
		0,
	}
}

func (m *TestResponseWriter) CloseNotify() <-chan bool { return m.gone }
func (m *TestResponseWriter) Close()                   { m.gone <- true }

func (m *TestResponseWriter) Write(data []byte) (int, error) {
	if m.sleep > 0 {
		time.Sleep(m.sleep)
	}
	return m.ResponseRecorder.Write(data)
}

func TestJoinLeave(t *testing.T) {
	sse := New()

	c1 := NewConn(NewTestResponseWriter())

	go sse.Serve()
	if len(sse.conns) > 0 {
		t.Error("None expected")
	}

	<-sse.Join(c1)
	if _, ok := sse.conns[c1]; !ok {
		t.Error("Join expected")
	}

	<-sse.Leave(c1)
	if _, ok := sse.conns[c1]; ok {
		t.Error("Leave expected", sse.conns)
	}
}

func TestClients(t *testing.T) {
	if testing.Verbose() {
		Vlog = log.New(os.Stdout, "", log.LstdFlags)
	}

	sse := New()

	nclients := 1000

	stopping := &sync.WaitGroup{}
	stopping.Add(nclients)
	starting := &sync.WaitGroup{}
	starting.Add(nclients)

	clients := make([]*Conn, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewConn(NewTestResponseWriter())
		go func(c *Conn, i int) {
			<-sse.Join(c)
			starting.Done()
			if err := c.Serve(sse); err != nil {
				t.Fatal(err)
				return
			}
			stopping.Done()
		}(clients[i], i)
	}

	Vlog.Println("Starting sse")
	go sse.Serve()

	events := []Event{
		{Data: "Hello", ID: "1", Event: "e1"},
		{Data: "World", ID: "2", Event: "e2"},
		{Data: "!!", ID: "3", Event: "e3"},
	}

	// wait for clients to connnect
	Vlog.Println("Clients connecting")
	starting.Wait()

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
		c.c.(*TestResponseWriter).Close()
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
		got := c.c.(*TestResponseWriter).Body.String()
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

	clients := make([]*TestResponseWriter, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewTestResponseWriter()
		go func(w *TestResponseWriter) {
			conn := NewConn(w)
			<-sse.Join(conn)
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
