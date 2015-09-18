# What: [EventSource] for [Go](lang)

[Go]'s [net/http] compatible simple [EventSource] implementation.

# Why

TODO: blog post is coming...

# Where
## Installation

```console
go get github.com/gmarik/eventsource
```

## Example: running a real-time clock

```console
$ cd $GOPATH/src/github.com/gmarik/eventsource/
$ go run example/server.go
$ open http://localhost:7070
```

## Quick start

see [example/server.go](https://github.com/gmarik/eventsource/blob/master/example/server.go)

[Go]:http://golang.org
[EventSource]:http://www.w3.org/TR/2011/WD-eventsource-20110208/
[net/http]:https://golang.org/pkg/net/http/
