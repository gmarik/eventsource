package main

import (
	"github.com/gmarik/eventsource"

	"html/template"

	"log"
	"net/http"
	"os"
	"time"
)

func main() {

	eventsource.Vlog = log.New(os.Stdout, "ES", log.LstdFlags)

	es := eventsource.New()
	go es.Serve()

	go func() {
		for i := 0; ; i += 1 {
			evt := eventsource.Event{
				Event: "time",
				Data:  time.Now().Format(time.RFC3339),
			}

			done, _ := es.Push(evt)

			<-done
			<-time.After(1 * time.Second)
		}
	}()

	http.Handle("/Stream", eventsource.Headers(es))

	// sends initial event
	http.Handle("/Custom", eventsource.Headers(custom(es)))
	http.HandleFunc("/", index)

	log.Println("Listening on localhost:7070")
	log.Fatal(http.ListenAndServe(":7070", nil))
}

func custom(es eventsource.EventSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wfcn, ok := w.(eventsource.WriteFlushCloseNotifier)
		if !ok {
			log.Println(http.ErrNotSupported)
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		wfcn.Flush()

		err := eventsource.WriteEvent(wfcn, eventsource.Event{Event: "test", Data: "Hello"})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}

		conn := eventsource.NewConn(wfcn)
		es.Join(conn)
		defer es.Leave(conn)

		if err := conn.Serve(es); err != nil {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		}
	})
}

func index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("example/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	t.Execute(w, struct{}{})
}
