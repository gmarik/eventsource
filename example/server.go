package main

import (
	"github.com/gmarik/sse"

	"html/template"

	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {

	sse.Vlog = log.New(os.Stdout, "ES", log.LstdFlags)

	es := sse.New()
	go es.Serve()

	go func() {
		for i := 0; ; i += 1 {
			evt := sse.Event{
				Event: "time",
				Data:  time.Now().Format(time.RFC3339),
			}

			done, _ := es.Push(evt)

			<-done
			<-time.After(1 * time.Second)
		}
	}()

	http.Handle("/clock", sse.Headers(es))

	// sends initial event
	http.Handle("/custom", sse.Headers(custom(es)))
	http.Handle("/counter", sse.Headers(http.HandlerFunc(counter)))
	http.Handle("/counters", http.HandlerFunc(counters))
	http.HandleFunc("/", index)

	log.Println("Listening on localhost:7070")
	go http.ListenAndServe(":7070", nil)
	log.Fatal(http.ListenAndServeTLS(":7071", "example/cert.pem", "example/key.pem", nil))
}

func custom(es sse.SSE) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wfcn, ok := w.(sse.WriteFlushCloseNotifier)
		if !ok {
			log.Println(http.ErrNotSupported)
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		wfcn.Flush()

		err := sse.WriteEvent(wfcn, sse.Event{Event: "test", Data: "Hello"})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}

		conn := sse.NewConn(wfcn)
		es.Join(conn)
		defer es.Leave(conn)

		if err := conn.Serve(es); err != nil {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	})
}

func counter(w http.ResponseWriter, r *http.Request) {
	wfcn, ok := w.(sse.WriteFlushCloseNotifier)
	if !ok {
		log.Println(http.ErrNotSupported)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	wfcn.Flush()

	for count := 0; ; count += 1 {
		select {
		case <-wfcn.CloseNotify():
			return
		case <-time.After(1 * time.Second):
			err := sse.WriteEvent(wfcn, sse.Event{Event: "count", Data: fmt.Sprintf("%d", count)})
			if err != nil {
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			wfcn.Flush()
		}
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("example/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	t.Execute(w, struct{}{})
}
func counters(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("example/counters.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	t.Execute(w, struct{}{})
}
