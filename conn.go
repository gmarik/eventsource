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
		disconnected = c.c.CloseNotify()
		joined       = es.join(c)

		pushes chan []byte
	)

	for {
		select {
		case pushes = <-joined:
			joined = nil
			defer es.leave(c)
		case <-es.Done():
			return nil
		case <-c.closed:
			return nil
		case <-disconnected:
			close(c.closed)
			return nil
		case data := <-pushes:
			_, err := c.c.Write(data)
			if err != nil {
				return err
			}
			c.c.Flush()
		}
	}
}
