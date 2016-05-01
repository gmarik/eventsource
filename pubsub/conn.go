package pubsub

import (
	"golang.org/x/net/context"
)

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
		case <-es.Done():
			break out
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
