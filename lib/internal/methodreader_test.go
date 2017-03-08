package internal

import (
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestNewMethodReader_firstMethod(t *testing.T) {
	r, w := io.Pipe()
	go io.WriteString(w, `{"id": 1, "method": "mymethod", "params": ["firstparam"]}`)

	reader := NewMethodReader(r, 1)
	rpcmsg := <-reader.Methods

	if rpcmsg.ID != 1 {
		t.Errorf("Expected ID 1, received %d", rpcmsg.ID)
		return
	}

	if result, ok := rpcmsg.Params[0].(string); !ok {
		t.Errorf("Expected result of type string, received %#v", rpcmsg.Params[0])
		return
	} else if result != "firstparam" {
		t.Errorf("Expected result success, received %s", result)
		return
	}

	reader.StopServing(time.Second)
	err := <-reader.Errors
	if err != io.EOF {
		t.Errorf("Expected io.EOF, received %#v", err)
		return
	}
}

func TestNewMethodReader_firstAndSecondMethods(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		io.WriteString(w, `{"id": 1, "method": "mymethod", "params": ["firstparam"]}{"id": 2, "method": "anothermethod", "params": [42]}`)
		w.Close()
	}()

	reader := NewMethodReader(r, 1)

	rpcmsg := <-reader.Methods

	if rpcmsg.ID != 1 {
		t.Errorf("Expected ID 1, received %d", rpcmsg.ID)
		return
	}

	if result, ok := rpcmsg.Params[0].(string); !ok {
		t.Errorf("Expected result of type string, received %#v", rpcmsg.Params[0])
		return
	} else if result != "firstparam" {
		t.Errorf("Expected result success, received %s", result)
		return
	}

	rpcmsg = <-reader.Methods

	if rpcmsg.ID != 2 {
		t.Errorf("Expected ID 2, received %d", rpcmsg.ID)
		return
	}

	if result, ok := rpcmsg.Params[0].(json.Number); !ok {
		t.Errorf("Expected result of type int64, received %#v", rpcmsg.Params[0])
		return
	} else if v, err := result.Int64(); err != nil {
		t.Errorf("Expected no error received %s", err.Error())
		return
	} else if v != 42 {
		t.Errorf("Expected result 42, received %d", result)
		return
	}

	reader.StopServing(time.Second)
	err := <-reader.Errors
	if err != io.EOF {
		t.Errorf("Expected io.EOF, received %#v", err)
		return
	}
}

func TestNewMethodReader_methodWithError(t *testing.T) {
	r, w := io.Pipe()
	go func() {
		io.WriteString(w, `"id": 1, "method": "mymethod", "params": ["firstparam"]}`)
		w.Close()
	}()

	reader := NewMethodReader(r, 1)

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

func TestMethodReader_StopServing(t *testing.T) {
	r, w := io.Pipe()
	reader := NewMethodReader(r, 1)

	io.WriteString(w, `{"id": 1, "result": "success"}`)
	<-reader.Methods

	if err := reader.StopServing(time.Second); err != nil {
		t.Errorf("Expected no errors, received: %s", err.Error())
	}

	if err := <-reader.Errors; err != io.EOF {
		t.Errorf("Expected io.EOF, received: %s", err.Error())
		return
	}

	select {
	case c, _ := <-reader.Methods:
		t.Errorf("Expected no method call, received: %#v", c)
		return
	default:
		// All ok
	}
}

func TestMethodReader_StopServingWithTimeout(t *testing.T) {
	r, w := io.Pipe()
	reader := NewMethodReader(r, 1)

	io.WriteString(w, `{"id": 1, "result": "success"}`)
	<-reader.Methods

	if err := reader.StopServing(0 * time.Second); err == nil {
		t.Error("Expected timeout error, received: nil")
	}

	// In this test we know that EOF will come after StopServing timed out
	if err := <-reader.Errors; err != io.EOF {
		t.Errorf("Expected io.EOF, received: %s", err.Error())
		return
	}

	select {
	case c, _ := <-reader.Methods:
		t.Errorf("Expected no method call, received: %#v", c)
		return
	default:
		// All ok
	}
}
