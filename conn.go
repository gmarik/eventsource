package sse

// Conn represents single client connection
type Conn struct {
	c WriteFlushCloseNotifier

	LastEventID string

	closed chan struct{}
}

// NewConn creates connection wrapper;
func NewConn(wfcn WriteFlushCloseNotifier) *Conn {
	return &Conn{
		c:      wfcn,
		closed: make(chan struct{}),
	}
}

func (c *Conn) Done() <-chan struct{} {
	return c.closed
}

func (c *Conn) Close() {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
}

// Serve handles single connection and associated events like
// disconnect, event source termination and Event delivery
func (c *Conn) Serve(es SSE) error {

	var (
		err error

		disconnected = c.c.CloseNotify()
		joinc        = es.join(c)

		recvc chan []byte
	)

out:
	for {
		select {
		case recvc = <-joinc:
			joinc = nil
		case <-es.Done():
			break out
		case <-c.closed:
			break out
		case <-disconnected:
			close(c.closed)
			break out
		case data, ok := <-recvc:
			if !ok {
				return nil
			}
			_, err = c.c.Write(data)
			if err != nil {
				break out
			}
			c.c.Flush()
		}
	}

	// never joined
	if recvc == nil {
		return err
	}

	// if joined and leaving - drain the channel
	// to not block the sender
	go drain(recvc)

	es.leave(c)

	// wait until closed
	<-recvc

	return err
}

func drain(ch chan []byte) {
	for range ch {
	}
}
