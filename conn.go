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
		joined       = es.join(c)

		pushes chan []byte
	)

out:
	for {
		select {
		case pushes = <-joined:
			joined = nil
		case <-es.Done():
			break out
		case <-c.closed:
			break out
		case <-disconnected:
			close(c.closed)
			break out
		case data, ok := <-pushes:
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

	if pushes == nil {
		return err
	}

	go drain(pushes)

	es.leave(c)
	// wait until closed
	<-pushes
	return err
}

func drain(ch chan []byte) {
	for range ch {
	}
}
