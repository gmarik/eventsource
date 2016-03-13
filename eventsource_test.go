package sse

import (
	"testing"

	"net/http/httptest"

	"bytes"
	// "log"
	// "os"
	"sync"
)

type TestResponseWriter struct {
	*httptest.ResponseRecorder
	gone chan bool
}

func NewTestResponseWriter() *TestResponseWriter {
	return &TestResponseWriter{
		httptest.NewRecorder(),
		make(chan bool),
	}
}

func (m *TestResponseWriter) CloseNotify() <-chan bool { return m.gone }
func (m *TestResponseWriter) Close()                   { m.gone <- true }

func TestClients(t *testing.T) {
	// if testing.Verbose() {
	// 	Vlog = log.New(os.Stdout, "", log.LstdFlags)
	// }

	es := New()

	nclients := 1000

	starting := &sync.WaitGroup{}
	stopping := &sync.WaitGroup{}
	starting.Add(nclients)
	stopping.Add(nclients)

	clients := make([]*TestResponseWriter, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewTestResponseWriter()
		go func(w *TestResponseWriter, i int) {
			conn := NewConn(w)
			es.Join(conn)
			starting.Done()
			conn.Serve(es)
			es.Leave(conn)
			stopping.Done()
		}(clients[i], i)
	}

	Vlog.Println("Starting ES")
	go es.Serve()

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
		done, err := es.Push(e)
		if err != nil {
			t.Error(err)
		}
		<-done
	}

	// disconnect clients
	Vlog.Println("Clients leaving")
	for _, w := range clients {
		w.Close()
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
	for i, w := range clients {
		got := w.Body.String()
		if got != exp {
			t.Errorf("\nClient %d\nExp: %v\nGot: %v", i, exp, got)
		}
	}
}

func BenchmarkIt(t *testing.B) {
	es := New()

	nclients := 1000
	starting := &sync.WaitGroup{}
	starting.Add(nclients)

	clients := make([]*TestResponseWriter, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewTestResponseWriter()
		go func(w *TestResponseWriter) {
			conn := NewConn(w)
			es.Join(conn)
			starting.Done()
			conn.Serve(es)
		}(clients[i])
	}

	go es.Serve()

	starting.Wait()

	Vlog.Println("Sending out events")

	t.StartTimer()
	for i := 1; i < t.N; i += 1 {
		done, err := es.Push(Event{Data: "Hello", ID: "1", Event: "e1"})
		if err != nil {
			t.Error(err)
		}
		<-done
	}
	t.StopTimer()
}

func TestEventWriter(t *testing.T) {
	cases := []struct {
		m   Event
		exp string
	}{
		{
			m: Event{
				ID:    "1",
				Event: "test\nevent",
				Data:  "hello\nworld",
			},
			exp: "id: 1\nevent: testevent\ndata: hello\ndata: world\n\n",
		},
	}

	var buf bytes.Buffer

	for _, c := range cases {

		buf.Reset()
		err := WriteEvent(&buf, c.m)

		if err != nil {
			t.Fatal(err)
		}

		exp := c.exp
		got := buf.String()
		if exp != got {
			t.Error("\nExp", exp, "\nGot", got)
		}
	}

}
