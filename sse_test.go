package sse

import (
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

	for _, c := range cases {
		data, err := c.m.MarshalSSEvent()
		if err != nil {
			t.Fatal(err)
		}

		exp := c.exp
		got := string(data)
		if exp != got {
			t.Error("\nExp", exp, "\nGot", got)
		}
	}
}
