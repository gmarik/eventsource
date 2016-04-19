package sse

// Conn represents single client connection
type Conn struct {
	c WriteFlushCloseNotifier

	LastEventID string

	pushes chan []byte
	closed chan struct{}
}

// NewConn creates connection wrapper;
func NewConn(wfcn WriteFlushCloseNotifier) *Conn {
	return &Conn{
		c: wfcn,

		pushes: make(chan []byte),
		closed: make(chan struct{}),
	}
}

// Push pushes data to be written into underlying connection
func (c *Conn) Push(eventData []byte) {
	c.pushes <- eventData
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
	)

	for {
		select {
		case <-joined:
			joined = nil
			defer es.leave(c)
		case <-es.Done():
			return nil
		case <-c.closed:
			return nil
		case <-disconnected:
			close(c.closed)
			return nil
		case data := <-c.pushes:
			_, err := c.c.Write(data)
			if err != nil {
				return err
			}
			c.c.Flush()
		}
	}
}
