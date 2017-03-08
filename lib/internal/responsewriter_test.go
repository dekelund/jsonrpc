package internal

import (
	"io"
	"sync"
	"testing"
	"time"
)

func TestNewResponseWriter_response(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	msg := make([]byte, 28)
	go func() {
		io.ReadFull(r, msg)
		wg.Done()
	}()

	writer := NewResponseWriter(w, 1)
	writer.Respond(1, nil, "success")

	wg.Wait()

	expectedMSG := `{"id":1,"result":"success"}`

	if string(msg[:27]) != expectedMSG {
		t.Errorf("Expected `%s`, received `%s`", expectedMSG, msg[:27])
		return
	}

	writer.StopServing(1 * time.Second)
}

func TestNewResponseWriter_error(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	msg := make([]byte, 62)
	go func() {
		io.ReadFull(r, msg)
		wg.Done()
	}()

	writer := NewResponseWriter(w, 1)
	writer.Respond(1, &Error{Code: -32601, Message: "Method not found"}, nil)

	wg.Wait()
	expectedMSG := `{"id":1,"error":{"code":-32601,"message":"Method not found"}}`

	if string(msg[:61]) != expectedMSG {
		t.Errorf("Expected %s, received %s", expectedMSG, msg[:61])
		return
	}

	writer.StopServing(1 * time.Second)
}

func TestNewResponseWriter_firstAndSecondRespond(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	msg := make([]byte, 122)
	go func() {
		io.ReadFull(r, msg)
		wg.Done()
	}()

	writer := NewResponseWriter(w, 2) // Two indicates buffer for two outstanding requests

	writer.Respond(1, &Error{Code: -32601, Message: "Method not found"}, nil)
	writer.Respond(3, &Error{Code: -32603, Message: "Internal error"}, nil)

	wg.Wait()

	expectedMSG := `{"id":1,"error":{"code":-32601,"message":"Method not found"}}
{"id":3,"error":{"code":-32603,"message":"Internal error"}}
`

	if string(msg) != expectedMSG {
		t.Errorf("Expected\n`%s`, received\n`%s`", expectedMSG, msg)
		return
	}

	writer.StopServing(1 * time.Second)
}

func TestNewResponseWriter_outstandingRequests(t *testing.T) {
	r, w := io.Pipe()

	msg := make([]byte, 122)
	go io.ReadFull(r, msg)

	writer := NewResponseWriter(w, 1)
	defer writer.StopServing(1 * time.Second)

	if err := writer.Respond(1, &Error{Code: -32601, Message: "Method not found"}, nil); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	if err := writer.Respond(3, &Error{Code: -32603, Message: "Internal error"}, nil); err == nil {
		t.Error("Expected error, received none")
		return
	}
}


func TestResponseWriter_StopServing(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	msg := make([]byte, 60)
	go func() {
		io.ReadFull(r, msg)
		wg.Done()
	}()

	writer := NewResponseWriter(w, 1)
	writer.Respond(3, &Error{Code: -32603, Message: "Internal error"}, nil)

	wg.Wait()

	expectedMSG := `{"id":3,"error":{"code":-32603,"message":"Internal error"}}`

	if string(msg[:60]) == expectedMSG {
		t.Errorf("Expected %s, received %s", expectedMSG, msg[:60])
		return
	}

	writer.StopServing(1 * time.Second)

	err := writer.Respond(3, &Error{Code: -32603, Message: "Internal error"}, nil)
	if err == nil {
		t.Errorf("Expected error, received none")
	}
}


func TestResponseWriter_StopServingWithTimeout(t *testing.T) {
	r, w := io.Pipe()

	r.Close()

	msg := make([]byte, 1024)
	go io.ReadFull(r, msg)

	writer := NewResponseWriter(w, 1)
	if err := writer.Respond(3, &Error{Code: -32603, Message: "Internal error"}, nil); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	if err := writer.StopServing(0 * time.Second); err == nil {
		t.Error("Expected timeout error, received: nil")
		return
	}

	time.Sleep(2 * time.Second)

	if err := writer.Respond(3, &Error{Code: -32603, Message: "Internal error"}, nil); err != io.EOF {
		t.Errorf("Expected io.EOF, received %v", err)
		return
	}
}
