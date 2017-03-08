package internal

import (
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"
)

type MethodWriter struct {
	io      *json.Encoder
	calls   chan func() error
	stopper chan bool
	stopped chan bool

	Errors chan error
}

func NewMethodWriter(w io.WriteCloser, chSize int) *MethodWriter {
	writer := MethodWriter{
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

func (writer *MethodWriter) StopServing(max time.Duration) error {
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
		return errors.New("MethodWriter timed out during close")
	}
}

func (writer *MethodWriter) call(id int64, method string, params ...interface{}) error {
	err := writer.io.Encode(Method{ID: id, Method: method, Params: params})

	if err == io.EOF || err == io.ErrClosedPipe {
		return io.EOF
	} else if err != nil {
		return errors.Wrap(err, "MethodWriter failed to encode method call")
	}

	return nil
}

func (writer *MethodWriter) Call(id int64, method string, params ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = io.EOF // NOTE: If writer.calls was closed, assume EOF
		}
	}()

	select {
	case writer.calls <- func() error { return writer.call(id, method, params...) }:
		err = nil
	default:
		err = errors.New("Too many outstanding requests")
	}

	return
}
