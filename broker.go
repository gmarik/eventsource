package sse

import (
	"net/http"

	"bytes"
)

// WriteFlushCloseNotifier is a composite of required interfaces
// for EventSource protocol to work
type WriteFlushCloseNotifier interface {
	http.ResponseWriter
	http.Flusher
	http.CloseNotifier
}

type SSE interface {
	Join(*Conn)
	Leave(*Conn)
	Done() <-chan struct{}
}

type push struct {
	data []byte
	done chan struct{}
}

// Broker handles all the clients and Event delivery
type Broker struct {
	closed chan struct{}
	conns  map[*Conn]bool
	joins  chan *Conn
	leaves chan *Conn
	pushes chan push
}

// New creates Broker
func New() *Broker {
	return &Broker{
		closed: make(chan struct{}),
		conns:  make(map[*Conn]bool),
		joins:  make(chan *Conn),
		leaves: make(chan *Conn),
		pushes: make(chan push),
	}
}

func (es *Broker) Join(c *Conn) {
	es.joins <- c
}

func (es *Broker) Leave(c *Conn) {
	es.leaves <- c
}

// Push delivers and Event to all current connections
func (es *Broker) Push(evt Event) (<-chan struct{}, error) {

	buf := &bytes.Buffer{}
	if err := WriteEvent(buf, evt); err != nil {
		return nil, err
	}

	p := push{buf.Bytes(), make(chan struct{})}
	es.pushes <- p
	return p.done, nil
}

// Close effectively shuts down listening process
func (es *Broker) Close() {
	select {
	case <-es.closed:
	default:
		close(es.closed)
	}
}

func (es *Broker) Done() <-chan struct{} {
	return es.closed
}

// Listen handles client join/leaves as well as Event multiplexing to connections
func (es *Broker) Serve() error {
	for {
		select {
		case <-es.closed:
			break
		case c := <-es.leaves:
			delete(es.conns, c)
		case c := <-es.joins:
			es.conns[c] = true
		case push := <-es.pushes:
			for conn := range es.conns {
				// do not wait for slow recipients
				// TODO: do not push to clients that left
				conn.Push(push.data)
			}
			// TODO: should this be part of the api?
			close(push.done)
		}
	}
}
