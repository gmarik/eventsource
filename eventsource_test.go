package eventsource

import (
	"testing"

	"net/http"
	"net/http/httptest"

	"bytes"
	"log"
	"os"
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

func (m *TestResponseWriter) CloseNotify() <-chan bool {
	return m.gone
}

func (m *TestResponseWriter) Close() {
	m.gone <- true
}

func TestEventStream(t *testing.T) {
	if testing.Verbose() {
		Vlog = log.New(os.Stdout, "", log.LstdFlags)
	}

	es := New()
	go es.Listen()

	req, err := http.NewRequest("GET", "/", bytes.NewBufferString(""))
	if err != nil {
		t.Fatal(err)
	}

	w := NewTestResponseWriter()

	done, started := make(chan struct{}), make(chan struct{})

	// process client
	go func() {
		close(started)
		es.ServeHTTP(w, req)
		close(done)
	}()

	events := []Event{
		{Data: "Hello", ID: "1", Event: "e1"},
		{Data: "World", ID: "2", Event: "e2"},
		{Data: "!!", ID: "3", Event: "e3"},
	}

	// wait for client goroutine
	<-started

	for _, e := range events {
		es.Push(e)
	}

	// disconnect client
	w.Close()
	<-done

	got := w.Body.String()
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
	if got != exp {
		t.Errorf("\nExp: %v\nGot: %v", exp, got)
	}
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
