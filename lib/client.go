package jsonrpc

import (
	"errors"
	"io"
	"sync"

	"github.com/dekelund/jsonrpc/lib/internal"
	"time"
)

type Client struct {
	r       *internal.ResponseReader
	w       *internal.MethodWriter
	calls   chan func()
	waiters map[int64]func(internal.Response)
}

func NewClient(r io.ReadCloser, w io.WriteCloser) *Client {
	client := Client{
		r:       internal.NewResponseReader(r, 10),
		w:       internal.NewMethodWriter(w, 10),
		calls:   make(chan func(), 10),
		waiters: make(map[int64]func(internal.Response)),
	}

	var eof bool

	broadcastEOF := func() {
		for id, fn := range client.waiters {
			delete(client.waiters, id)
			fn(internal.Response{id, internal.EOF, nil})
		}
	}

	go func() {
		for {
			select {
			case err := <-client.r.Errors:
				if err == io.EOF {
					eof = true
					broadcastEOF()
				}
			case response := <-client.r.Responses:
				if fn, ok := client.waiters[response.ID]; ok {
					delete(client.waiters, response.ID)
					fn(response)
				}

			case fn, more := <-client.calls:
				if !more {
					return // Client.Stop() has been called
				}

				fn()

				if eof {
					broadcastEOF()
					continue
				}
			}
		}
	}()

	return &client
}

func (client *Client) StopServing(max time.Duration) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	// Don't use schedule, stop must be executed
	client.calls <- func() {

		go func() {
			client.r.StopServing(max) // Check and return err
			wg.Done()
		}()

		go func() {
			client.w.StopServing(max) // Check and return err
			wg.Done()
		}()


		return
	}

	wg.Wait()
	close(client.calls)
}

func (client *Client) schedule(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = io.EOF // NOTE: If client.calls was closed, assume EOF
		}
	}()

	select {
	case client.calls <- fn:
		return nil
	default:
		return errors.New("Too many outstanding requests, please slow down the speed")
	}
}

func (client *Client) Call(id int64, method string, params ...interface{}) (result interface{}, err error) {
	wg := sync.WaitGroup{}
	if id != 0 {
		wg.Add(1)
	}

	err = client.schedule(func() {
		if err = client.w.Call(id, method, params...); err != nil {
			wg.Done()
			return
		}

		// Since we runs the schedule at the same time as
		// we handle responses, and since we only run one
		// scheduled method at once, it's safe to read and
		// write to waiters.
		if _, ok := client.waiters[id]; ok {
			err = errors.New("ID already used for outstanding request")
			wg.Done()
			return
		}

		client.waiters[id] = func(r internal.Response) {
			if r.Error == nil {
				result = r.Result
			} else {
				err = Error{r.Error}
			}

			if id != 0 {
				wg.Done()
			}
		}

		return
	})

	if err != nil {
		wg.Done()
	}

	wg.Wait()

	result = fixResultTypes(result)

	return
}
