# What: [SSE] for [Go](lang)


[Go]'s [net/http] compatible simple [EventSource] implementation.

SSE aka Server Sent Events aka [EventSource]

# Why

TODO: blog post is coming...

# Where
## Installation

```console
go get github.com/gmarik/sse
```

## Example: running a real-time clock

```console
$ cd $GOPATH/src/github.com/gmarik/sse/
$ go run example/server.go
$ open http://localhost:7070
```

## Quick start

see [example/server.go](https://github.com/gmarik/sse/blob/master/example/server.go)

[Go]:http://golang.org
[SSE]:http://www.w3.org/TR/2011/WD-eventsource-20110208/
[EventSource]:http://www.w3.org/TR/2011/WD-eventsource-20110208/
[net/http]:https://golang.org/pkg/net/http/


## TODO
- fix Join/Serve race condition: Joined client will stall all the Pushes
