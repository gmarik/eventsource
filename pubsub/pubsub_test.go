package pubsub

import (
	"testing"

	"net/http/httptest"

	"log"
	"os"
	// "fmt"
	"sync"

	"github.com/gmarik/eventsource"

	"golang.org/x/net/context"
)

type ResponseRecorder struct {
	*httptest.ResponseRecorder
	done     chan bool
	ctx      context.Context
	cancelFn context.CancelFunc
}

func NewResponseRecorder() *ResponseRecorder {
	ctx, cancelFn := context.WithCancel(context.Background())
	return &ResponseRecorder{
		httptest.NewRecorder(),
		make(chan bool),
		ctx,
		cancelFn,
	}
}

func (m *ResponseRecorder) CloseNotify() <-chan bool { return m.done }
func (m *ResponseRecorder) Close() {
	close(m.done)
	m.cancelFn()
}

func TestClients(t *testing.T) {
	if testing.Verbose() {
		Vlog = log.New(os.Stdout, "", log.LstdFlags)
	}

	Vlog.Println("Starting sse")
	ps := New()

	wgstart := &sync.WaitGroup{}
	wgstop := &sync.WaitGroup{}

	ps.joinCallback = func() {
		wgstart.Done()
	}
	ps.leaveCallback = func() {
		wgstop.Done()
	}

	go ps.Listen()

	nclients := 10000

	wgstart.Add(nclients)
	wgstop.Add(nclients)

	clients := make([]*ResponseRecorder, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewResponseRecorder()
		go func(rr *ResponseRecorder, i int) {
			if err := ps.Serve(rr.ctx, rr); err != nil {
				t.Fatal(err)
				return
			}
		}(clients[i], i)
	}

	events := []sse.Event{
		{Data: "Hello", ID: "1", Event: "e1"},
		{Data: "World", ID: "2", Event: "e2"},
		{Data: "!!", ID: "3", Event: "e3"},
	}

	// wait for clients to connnect
	Vlog.Println("Clients connecting")
	wgstart.Wait()

	for _, e := range events {
		done, err := ps.Push(e)
		if err != nil {
			t.Error(err)
		}
		<-done
	}

	// disconnect clients
	Vlog.Println("Clients leaving")
	for _, rr := range clients {
		rr.Close()
	}
	Vlog.Println("Clients disconnecting")
	wgstop.Wait()

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
	for i, rr := range clients {
		got := rr.Body.String()
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
	ps := New()
	wgstart := &sync.WaitGroup{}

	ps.joinCallback = func() {
		wgstart.Done()
	}

	go ps.Listen()

	wgstart.Add(nclients)

	clients := make([]*ResponseRecorder, nclients, nclients)

	// process client
	for i := 0; i < nclients; i += 1 {
		clients[i] = NewResponseRecorder()
		go func(rr *ResponseRecorder) {
			ps.Serve(context.Background(), rr)
		}(clients[i])
	}

	wgstart.Wait()

	Vlog.Println("Sending out events")

	evt := sse.Event{Data: "Hello", ID: "1", Event: "e1"}

	t.StartTimer()
	for i := 1; i < t.N; i += 1 {
		done, err := ps.Push(evt)
		if err != nil {
			t.Error(err)
		}
		<-done
	}
	t.StopTimer()
}
