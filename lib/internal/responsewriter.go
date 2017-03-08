package internal

import (
	"encoding/json"
	"errors"
	"io"
	"time"
)

type ResponseWriter struct {
	io      *json.Encoder
	calls   chan func() error
	stopper chan bool
	stopped chan bool

	Errors chan error
}

func NewResponseWriter(w io.WriteCloser, chSize int) *ResponseWriter {
	writer := ResponseWriter{
		io:      json.NewEncoder(w),
		calls:   make(chan func() error, chSize),
		stopper: make(chan bool, 1),
		stopped: make(chan bool, 1),

		Errors: make(chan error, chSize),
	}

	go func() {
		for {
			select {
			case _, more := <-writer.stopper:
				if !more {
					close(writer.stopped)
					close(writer.calls)
					close(writer.Errors)
					return
				}
			case fn, _ := <-writer.calls:
				if err := fn(); err != nil {
					writer.Errors <- err
				}
			}
		}
	}()

	return &writer
}

func (writer ResponseWriter) StopServing(max time.Duration) error {
	if max != 0 {
		close(writer.stopper)
	} // NOTE: Don't close if no timeout, makes tests predictable

	select {
	case _, _ = <-writer.stopped:
		return nil
	case <-time.After(max):
		if max == 0 {
			close(writer.stopper)
		}
		return errors.New("ResponseWriter timed out during close")
	}
}

func (writer ResponseWriter) respond(id int64, err *Error, result interface{}) error {
	return writer.io.Encode(Response{ID: id, Error: err, Result: result})
	// TODO wrap error message
}

func (writer ResponseWriter) Respond(id int64, jsonrpcErr *Error, result interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = io.EOF // NOTE: If writer.calls was closed, assume EOF
		}
	}()

	select {
	case writer.calls <- func() error { return writer.respond(id, jsonrpcErr, result) }:
		return nil
	default:
		return errors.New("Too many outstanding requests")
	}
}
