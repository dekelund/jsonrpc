package internal

import (
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestNewResponseReader_firstResponse(t *testing.T) {
	r, w := io.Pipe()
	go io.WriteString(w, `{"id": 1, "result": "success"}`)

	reader := NewResponseReader(r, 1)
	rpcmsg := <-reader.Responses

	if rpcmsg.ID != 1 {
		t.Errorf("Expected ID 1, received %d", rpcmsg.ID)
		return
	}

	if result, ok := rpcmsg.Result.(string); !ok {
		t.Errorf("Expected result of type string, received %#v", rpcmsg.Result)
		return
	} else if result != "success" {
		t.Errorf("Expected result success, received %s", result)
		return
	}

	reader.StopServing(1 * time.Second)
	err := <-reader.Errors
	if err != io.EOF {
		t.Errorf("Expected io.EOF, received %#v", err)
		return
	}
}

func TestNewResponseReader_firstAndSecondResponses(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		io.WriteString(w, `{"id": 1, "result": "success"}{"id": 2, "result": 42}`)
		w.Close()
	}()

	reader := NewResponseReader(r, 1)

	rpcmsg := <-reader.Responses

	if rpcmsg.ID != 1 {
		t.Errorf("Expected ID 1, received %d", rpcmsg.ID)
		return
	}

	if result, ok := rpcmsg.Result.(string); !ok {
		t.Errorf("Expected result of type string, received %#v", rpcmsg.Result)
		return
	} else if result != "success" {
		t.Errorf("Expected result success, received %s", result)
		return
	}

	rpcmsg = <-reader.Responses

	if rpcmsg.ID != 2 {
		t.Errorf("Expected ID 2, received %d", rpcmsg.ID)
		return
	}

	if result, ok := rpcmsg.Result.(json.Number); !ok {
		t.Errorf("Expected result of type int64, received %#v", rpcmsg.Result)
		return
	} else if v, err := result.Int64(); err != nil {
		t.Errorf("Expected no error received %s", err.Error())
		return
	} else if v != 42 {
		t.Errorf("Expected result 42, received %d", result)
		return
	}

	reader.StopServing(1 * time.Second)
	err := <-reader.Errors
	if err != io.EOF {
		t.Errorf("Expected io.EOF, received %#v", err)
		return
	}
}

func TestNewResponseReader_methodWithError(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		io.WriteString(w, `"id": 1, "result": "success"}`)
		w.Close()
	}()

	reader := NewResponseReader(r, 1)

	rpcErr := <-reader.Errors

	if rpcErr == nil {
		t.Error("Expected an error, received none")
		return
	}

	err := reader.StopServing(time.Second)
	if err != nil {
		t.Errorf("Expected no error from stop serving, received %#v", err)
		return
	}
}

func TestResponseReader_StopServing(t *testing.T) {
	r, w := io.Pipe()
	reader := NewResponseReader(r, 1)

	io.WriteString(w, `{"id": 1, "result": "success"}`)
	<-reader.Responses

	if err := reader.StopServing(2 * time.Second); err != nil {
		t.Errorf("Expected no errors, received: %s", err.Error())
	}

	if err := <-reader.Errors; err != io.EOF {
		t.Errorf("Expected io.EOF, received: %s", err.Error())
		return
	}

	select {
	case r, _ := <-reader.Responses:
		t.Errorf("Expected no response, received: %#v", r)
		return
	default:
		// All ok
	}
}


func TestResponseReader_StopServingWithTimeout(t *testing.T) {
	r, w := io.Pipe()
	reader := NewResponseReader(r, 1)

	io.WriteString(w, `{"id": 1, "result": "success"}`)
	<-reader.Responses

	if err := reader.StopServing(0 * time.Second); err == nil {
		t.Error("Expected timeout error, received: nil")
	}

	// In this test we know that EOF will come after StopServing timed out
	if err := <-reader.Errors; err != io.EOF {
		t.Errorf("Expected io.EOF, received: %s", err.Error())
		return
	}

	select {
	case r, _ := <-reader.Responses:
		t.Errorf("Expected no response, received: %#v", r)
		return
	default:
		// All ok
	}
}
