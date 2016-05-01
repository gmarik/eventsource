package pubsub

import (
	"net/http"

	"golang.org/x/net/context"
)

// WriteFlushCloseNotifier is a composite of required interfaces
// for EventSource protocol to work
type ResponseWriteFlusher interface {
	http.ResponseWriter
	http.Flusher
}

type responseWriteFlushCloseNotifier interface {
	ResponseWriteFlusher
	http.CloseNotifier
}

// ServeHTTP implements http.Handler interface
func (es *PubSub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	es.ServeHTTPC(context.Background(), w, r)
}

func (es *PubSub) ServeHTTPC(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	rwfc, ok := w.(responseWriteFlushCloseNotifier)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}

	ctx, cancelFn := context.WithCancel(ctx)
	go func() {
		<-rwfc.CloseNotify()
		cancelFn()
	}()

	// conn := NewConn()
	// conn.LastEventID = r.Header.Get("Last-Event-ID")

	rwfc.WriteHeader(http.StatusOK)
	rwfc.Flush()

	if err := es.Serve(ctx, rwfc); err != nil {
		Vlog.Println("Error:", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	}
}
