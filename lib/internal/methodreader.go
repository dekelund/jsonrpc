package internal

import (
	"encoding/json"
	"errors"
	"io"
	"time"
)

type MethodReader struct {
	stopper chan bool
	stopped chan bool

	Methods chan Method
	Errors  chan error
}

func NewMethodReader(r io.ReadCloser, chSize int) *MethodReader {
	reader := MethodReader{
		stopper: make(chan bool, 1),
		stopped: make(chan bool, 1),

		Methods: make(chan Method, chSize),
		Errors:  make(chan error, chSize),
	}

	go func() {
		dec := json.NewDecoder(r)
		dec.UseNumber()

		for {
			call := Method{}

			if err := dec.Decode(&call); err == io.EOF || err == io.ErrClosedPipe {
				reader.Errors <- io.EOF
				break
			} else if err != nil {
				reader.Errors <- err // NOTE: Consider to break regardless of error
				continue
			}

			reader.Methods <- call
		}
	}()

	go func() {
		for {
			select {
			case _, more := <-reader.stopper:
				if !more {
					r.Close()

					close(reader.stopped)
					return
				}
			}
		}
	}()

	return &reader
}

func (reader MethodReader) StopServing(max time.Duration) error {
	if max != 0 { close(reader.stopper) } // NOTE: Don't close if no timeout, makes tests predictable

	select {
	case _, _ = <-reader.stopped:
		return nil
	case <-time.After(max): // For instance EOF has been reached
		if max == 0 { close(reader.stopper) } // NOTE: Don't close if no timeout, makes tests predictable
		return errors.New("MethodReader timed out during stop serving")
	}
}
