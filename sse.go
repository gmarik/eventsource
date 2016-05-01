package sse

import (
	"fmt"
	"io"
	"strings"

	"bytes"
)

// Event represents data to be pushed
type Event struct {
	// identifier of the event
	ID string
	// name of the event
	Event string
	// payload
	Data string
	// in millis
	Retry uint
}

func (e Event) MarshalSSEvent() ([]byte, error) {
	var buf = &bytes.Buffer{}

	if err := WriteEvent(buf, e); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil

}

// WriteEvent writes an Event into an io.Writer
func WriteEvent(w io.Writer, evt Event) (err error) {
	if evt.Retry > 0 {
		_, err = fmt.Fprintf(w, "retry:%d\n", evt.Retry)
		if err != nil {
			return err
		}
	}

	if len(evt.ID) > 0 {
		_, err = fmt.Fprintf(w, "id: %s\n", strings.Replace(evt.ID, "\n", "", -1))
		if err != nil {
			return err
		}
	}

	if len(evt.Event) > 0 {
		_, err = fmt.Fprintf(w, "event: %s\n", strings.Replace(evt.Event, "\n", "", -1))
		if err != nil {
			return err
		}
	}

	if len(evt.Data) > 0 {
		for _, line := range strings.Split(evt.Data, "\n") {
			_, err := fmt.Fprintf(w, "data: %s\n", line)
			if err != nil {
				return err
			}
		}
	}

	_, err = fmt.Fprint(w, "\n")
	return err
}
