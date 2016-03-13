package sse

import (
	"io/ioutil"
	"log"
)

var (
	// Vlog - verbose log; Discards messages by default
	Vlog = log.New(ioutil.Discard, "[ES]", log.LstdFlags)
)
