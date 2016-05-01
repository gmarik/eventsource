package pubsub

import (
	"golang.org/x/net/context"
)

type Marshaller interface {
	MarshalSSEvent() ([]byte, error)
}

// Push helper struct
type push struct {
	data []byte
	done chan struct{}
}

// join
type op struct {
	c    ResponseWriteFlusher
	done chan (chan []byte)
}

// PubSub handles all the clients and Event delivery
type PubSub struct {
	closed chan struct{}
	conns  map[ResponseWriteFlusher]chan []byte
	joinc  chan op
	leavec chan ResponseWriteFlusher
	sendc  chan push

	joinCallback, leaveCallback func()
}

// New creates PubSub
func New() *PubSub {
	return &PubSub{
		closed: make(chan struct{}),
		conns:  make(map[ResponseWriteFlusher](chan []byte)),
		joinc:  make(chan op),
		leavec: make(chan ResponseWriteFlusher),
		sendc:  make(chan push),
	}
}

func (es *PubSub) join(c ResponseWriteFlusher) <-chan chan []byte {
	opv := op{c, make(chan chan []byte)}
	es.joinc <- opv
	return opv.done
}

func (es *PubSub) leave(c ResponseWriteFlusher) {
	es.leavec <- c
}

// Push delivers and Event to all current connections
func (es *PubSub) Push(evt Marshaller) (<-chan struct{}, error) {

	data, err := evt.MarshalSSEvent()
	if err != nil {
		return nil, err
	}

	p := push{data, make(chan struct{})}
	es.sendc <- p
	return p.done, nil
}

// Listen handles client join/leaves as well as Event multiplexing to connections
func (es *PubSub) Listen(ctx context.Context) {

out:
	for {
		select {
		case <-ctx.Done():
			//TODO: test this
			break out
		case c := <-es.leavec:
			close(es.conns[c])
			delete(es.conns, c)
			if es.leaveCallback != nil {
				es.leaveCallback()
			}
		case opv := <-es.joinc:
			// TODO: pool of channels
			ch := make(chan []byte)
			es.conns[opv.c] = ch
			opv.done <- ch
			if es.joinCallback != nil {
				es.joinCallback()
			}
		case push := <-es.sendc:
			for _, ch := range es.conns {
				ch <- push.data
			}
			//TODO: move this into test client
			close(push.done)
		}
	}
}

// Serve handles single connection and associated events like
// disconnect, event source termination and Event delivery
func (es *PubSub) Serve(ctx context.Context, rwf ResponseWriteFlusher) error {

	var (
		err error

		joinc = es.join(rwf)
		recvc chan []byte
	)

out:
	for {
		select {
		case ch := <-joinc:
			recvc = ch
			joinc = nil
		case <-ctx.Done():
			break out
		case data, ok := <-recvc:
			if !ok {
				return nil
			}
			_, err = rwf.Write(data)
			if err != nil {
				break out
			}
			rwf.Flush()
		}
	}

	// never joined
	if recvc == nil {
		return err
	}

	// if joined and leaving - drain the channel
	// to not block the sender
	go drain(recvc)

	es.leave(rwf)

	// wait until closed
	<-recvc

	return err
}

func drain(ch chan []byte) {
	for range ch {
	}
}