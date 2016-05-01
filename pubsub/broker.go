package pubsub

type Marshaller interface {
	MarshalSSEvent() ([]byte, error)
}

type SSE interface {
	join(ResponseWriteFlusher) <-chan chan []byte
	leave(ResponseWriteFlusher)
	Done() <-chan struct{}
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

// Close effectively shuts down listening process
func (es *PubSub) Close() {
	select {
	case <-es.closed:
	default:
		close(es.closed)
	}
}

func (es *PubSub) Done() <-chan struct{} {
	return es.closed
}

// Listen handles client join/leaves as well as Event multiplexing to connections
func (es *PubSub) Listen() {

out:
	for {
		select {
		case <-es.closed:
			//TODO: test this
			break out
		case c := <-es.leavec:
			close(es.conns[c])
			delete(es.conns, c)
		case opv := <-es.joinc:
			// TODO: pool of channels
			ch := make(chan []byte)
			es.conns[opv.c] = ch
			opv.done <- ch
		case push := <-es.sendc:
			for _, ch := range es.conns {
				ch <- push.data
			}
			//TODO: move this into test client
			close(push.done)
		}
	}
}
