package sse

import (
	"net/http"
)

// ServeHTTP implements http.Handler interface
func (es *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wfcn, ok := w.(WriteFlushCloseNotifier)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}

	conn := NewConn(wfcn)
	conn.LastEventID = r.Header.Get("Last-Event-ID")

	wfcn.WriteHeader(http.StatusOK)
	wfcn.Flush()

	if err := conn.Serve(es); err != nil {
		Vlog.Println("Error:", err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	}
}
