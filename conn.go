package eventsource

// Conn represents single client connection
type Conn struct {
	c WriteFlushCloseNotifier

	LastEventID string

	pushes chan []byte
	done   chan struct{}
}

// NewConn creates connection wrapper;
func NewConn(wfcn WriteFlushCloseNotifier) *Conn {
	return &Conn{
		c: wfcn,

		pushes: make(chan []byte),
		done:   make(chan struct{}),
	}
}

// Push writes event data to underlying connection
func (c *Conn) Push(eventData []byte) {
	c.pushes <- eventData
}

// Serve handles single connection and associated events like
// disconnect, event source termination and Event delivery
func (c *Conn) Serve(es EventSource) error {
	for {
		select {
		case <-es.Done():
			return nil
		case <-c.c.CloseNotify():
			close(c.done)
			return nil
		case data := <-c.pushes:
			select {
			case <-c.done:
				// TODO: race conditnon
				// client disconnected
				// but c is still registered with the Broker
				Vlog.Println("Client Disconnected already")
			default:
				_, err := c.c.Write(data)
				if err != nil {
					return err
				}
				c.c.Flush()
			}
		}
	}
}
