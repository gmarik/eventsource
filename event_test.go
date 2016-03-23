package sse

import (
	"bytes"
	"testing"
)

func TestEventWriter(t *testing.T) {
	cases := []struct {
		m   Event
		exp string
	}{
		{
			m: Event{
				ID:    "1",
				Event: "test\nevent",
				Data:  "hello\nworld",
			},
			exp: "id: 1\nevent: testevent\ndata: hello\ndata: world\n\n",
		},
	}

	var buf bytes.Buffer

	for _, c := range cases {

		buf.Reset()
		err := WriteEvent(&buf, c.m)

		if err != nil {
			t.Fatal(err)
		}

		exp := c.exp
		got := buf.String()
		if exp != got {
			t.Error("\nExp", exp, "\nGot", got)
		}
	}

}
