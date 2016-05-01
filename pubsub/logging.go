package pubsub

import (
	"io/ioutil"
	"log"
)

var (
	// Vlog - verbose log; Discards messages by default
	Vlog = log.New(ioutil.Discard, "[SSE]", log.LstdFlags)
)
