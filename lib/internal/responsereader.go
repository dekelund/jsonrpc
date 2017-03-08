package internal

import (
	"encoding/json"
	"errors"
	"io"
	"time"
)

type ResponseReader struct {
	stopper chan bool
	stopped chan bool

	Responses chan Response
	Errors    chan error
}

func NewResponseReader(r io.ReadCloser, chSize int) *ResponseReader {
	reader := ResponseReader{
		stopper: make(chan bool, 1),
		stopped: make(chan bool, 1),

		Responses: make(chan Response, chSize),
		Errors:    make(chan error, chSize),
	}

	go func() {
		dec := json.NewDecoder(r)
		dec.UseNumber()

		for {
			response := Response{}

			if err := dec.Decode(&response); err != nil {
				if err == io.EOF || err == io.ErrClosedPipe {
					reader.Errors <- io.EOF
					break
				}

				reader.Errors <- err

				continue
			}

			reader.Responses <- response
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

func (reader ResponseReader) StopServing(max time.Duration) error {
	if max != 0 {
		close(reader.stopper)
	} // NOTE: Don't close if no timeout, makes tests predictable

	select {
	case _, _ = <-reader.stopped:
		return nil
	case <-time.After(max): // For instance EOF has been reached
		if max == 0 {
			close(reader.stopper)
		} // NOTE: Don't close if no timeout, makes tests predictable

		return errors.New("ResponseReader timed out during stop serving")
	}
}
