package common

import (
	"io"
	"net/http"
)

type Service interface {
	http.Handler
}

type Commander interface {
	Execute(cmd string) (string, error)
}

type Streamer interface {
	Connect() (Stream, error)
}

type Stream interface {
	io.ReadWriteCloser
}

type Activator interface {
	Start() error
	Stop() error
}
