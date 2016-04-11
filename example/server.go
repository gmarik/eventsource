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
				Event: "count",
				Data:  fmt.Sprintf("%d", i),
			}

			done, _ := es.Push(evt)
			<-done

			<-time.After(1 * time.Second)
		}
	}()

	http.Handle("/sharedc", sse.Headers(es))
	http.Handle("/parallelc", sse.Headers(http.HandlerFunc(counter)))

	http.HandleFunc("/", index)

	log.Println("Listening on http://localhost:7070")
	log.Println("Listening on https://localhost:7071")
	go http.ListenAndServe(":7070", nil)
	log.Fatal(http.ListenAndServeTLS(":7071", "example/cert.pem", "example/key.pem", nil))
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
	render(w, "example/index.html", struct{}{})
}

func render(w http.ResponseWriter, path string, kvs interface{}) {
	t, err := template.ParseFiles(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	t.Execute(w, kvs)
}
