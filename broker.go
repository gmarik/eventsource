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
	join(*Conn) <-chan chan []byte
	leave(*Conn) <-chan chan []byte
	Done() <-chan struct{}
}

// Push helper struct
type push struct {
	data []byte
	done chan struct{}
}

// Join/Leave helper struct
type op struct {
	c    *Conn
	done chan (chan []byte)
}

// Broker handles all the clients and Event delivery
type Broker struct {
	closed chan struct{}
	conns  map[*Conn]chan []byte
	joins  chan op
	leaves chan op
	pushes chan push
}

// New creates Broker
func New() *Broker {
	return &Broker{
		closed: make(chan struct{}),
		conns:  make(map[*Conn](chan []byte)),
		joins:  make(chan op),
		leaves: make(chan op),
		pushes: make(chan push),
	}
}

func (es *Broker) join(c *Conn) <-chan chan []byte {
	opv := op{c, make(chan chan []byte)}
	es.joins <- opv
	return opv.done
}

func (es *Broker) leave(c *Conn) <-chan chan []byte {
	opv := op{c, make(chan chan []byte)}
	es.leaves <- opv
	return opv.done
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
		case opv := <-es.leaves:
			delete(es.conns, opv.c)
			close(opv.done)
		case opv := <-es.joins:
			// TODO: pool of channels
			ch := make(chan []byte)
			es.conns[opv.c] = ch
			opv.done <- ch
		case push := <-es.pushes:
			for conn, ch := range es.conns {
				// TODO: do not wait for slow recipients?
				select {
				case <-conn.Done():
					//skip
				default:
					ch <- push.data
				}
			}
			// TODO: should this be part of the api?
			close(push.done)
		}
	}
}
