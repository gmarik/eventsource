package eventsource

import (
	"net/http"

	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
)

var (
	// Vlog - verbose log; Discards messages by default
	Vlog = log.New(ioutil.Discard, "[ES]", log.LstdFlags)
)

// Event represents data to be pushed
type Event struct {
	// identifier of the event
	ID string
	// name of the event
	Event string
	// payload
	Data string
	// in millis
	Retry uint
}

// WriteEvent writes an Event into an io.Writer
func WriteEvent(w io.Writer, evt Event) (err error) {
	if evt.Retry > 0 {
		_, err = fmt.Fprintf(w, "retry:%d\n", evt.Retry)
		if err != nil {
			return err
		}
	}

	if len(evt.ID) > 0 {
		_, err = fmt.Fprintf(w, "id: %s\n", strings.Replace(evt.ID, "\n", "", -1))
		if err != nil {
			return err
		}
	}

	if len(evt.Event) > 0 {
		_, err = fmt.Fprintf(w, "event: %s\n", strings.Replace(evt.Event, "\n", "", -1))
		if err != nil {
			return err
		}
	}

	if len(evt.Data) > 0 {
		for _, line := range strings.Split(evt.Data, "\n") {
			_, err := fmt.Fprintf(w, "data: %s\n", line)
			if err != nil {
				return err
			}
		}
	}

	_, err = fmt.Fprint(w, "\n")
	return err
}

// WriteFlushCloseNotifier is a composite of required interfaces
// for EventSource protocol to work
type WriteFlushCloseNotifier interface {
	io.Writer
	http.Flusher
	http.CloseNotifier
}

// Conn represents single client connection
type Conn struct {
	WriteFlushCloseNotifier

	pushes      chan Event
	LastEventID string
}

// NewConn creating Conn;
// w can't be nil
func NewConn(w WriteFlushCloseNotifier) *Conn {
	return &Conn{
		WriteFlushCloseNotifier: w,
		pushes:                  make(chan Event),
	}
}

// Push delivers an Event to the Conn
func (c *Conn) Push(evt Event) {
	c.pushes <- evt
}

// EventSource handles all the clients and Event delivery
type EventSource struct {
	closed chan bool
	conns  map[*Conn]bool
	joins  chan *Conn
	leaves chan *Conn
	pushes chan Event
}

// New creates EventSource
func New() *EventSource {
	return &EventSource{
		closed: make(chan bool),
		conns:  make(map[*Conn]bool),
		joins:  make(chan *Conn),
		leaves: make(chan *Conn),
		pushes: make(chan Event),
	}
}

// Push delivers and Event to all current connections
func (es *EventSource) Push(evt Event) {
	es.pushes <- evt
}

// Close effectively shuts down listening process
func (es *EventSource) Close() {
	select {
	case <-es.closed:
	default:
		close(es.closed)
	}
}

// Listen handles client join/leaves as well as Event multiplexing to connections
func (es *EventSource) Listen() {
	for {
		select {
		case <-es.closed:
			break
		case c := <-es.leaves:
			delete(es.conns, c)
			Vlog.Println("Client left;", len(es.conns), "total")
		case c := <-es.joins:
			es.conns[c] = true
			Vlog.Println("Client joined;", len(es.conns), "total")
		case evt := <-es.pushes:
			Vlog.Println("Message", evt)
			for conn := range es.conns {
				// do not wait for slow recipients
				conn.pushes <- evt
			}
		}
	}
}

// EventWriterFunc is an adapter to enable custome EventWriter implementations
type EventWriterFunc func(io.Writer, Event) error

// Serve handles single connection and associated events like
// disconnect, event source termination and Event delivery
func (es *EventSource) Serve(conn *Conn, ewf EventWriterFunc) error {
	es.joins <- conn

	for {
		select {
		case <-es.closed:
			return nil
		case <-conn.CloseNotify():
			es.leaves <- conn
			return nil
		case evt := <-conn.pushes:
			if err := ewf(conn, evt); err != nil {
				// TODO: what errors cause disconnect?
				es.leaves <- conn
				return err
			}
			conn.Flush()
		}
	}
}

func (es *EventSource) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	wfcn, ok := w.(WriteFlushCloseNotifier)
	if !ok {
		return http.ErrNotSupported
	}

	conn := NewConn(wfcn)
	conn.LastEventID = r.Header.Get("Last-Event-ID")

	w.WriteHeader(http.StatusOK)
	conn.Flush()

	return es.Serve(conn, WriteEvent)
}

// implements http.Handler interface
func (es *EventSource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := es.serveHTTP(w, r); err != nil {
		Vlog.Println("Error:", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	}
}

// Headers is as default headers Middleware
func Headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		next.ServeHTTP(w, r)
	})
}
