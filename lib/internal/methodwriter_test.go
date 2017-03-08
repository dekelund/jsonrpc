package internal

import (
	"io"
	"sync"
	"testing"
	"time"
)

func TestNewMethodWriter_methodCall(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	msg := make([]byte, 65)
	go func() {
		io.ReadFull(r, msg)
		wg.Done()
	}()

	writer := NewMethodWriter(w, 1)
	writer.Call(1, "test.method", 42, "second parameter")

	wg.Wait()

	expectedMSG := `{"id":1,"method":"test.method","params":[42,"second parameter"]}`

	if string(msg[:64]) != expectedMSG {
		t.Errorf("Expected `%s`, received `%s`", expectedMSG, msg[:64])
		return
	}

	writer.StopServing(1 * time.Second)
}

func TestNewMethodWriter_methodCallWithEOF(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	r.Close()

	writer := NewMethodWriter(w, 1)
	writer.Call(1, "test.method", 42, "second parameter")
	if err := <- writer.Errors; err != io.EOF {
		t.Error("Expected an error, received none")
		return
	}


	writer.StopServing(1 * time.Second)
}

func TestNewMethodWriter_firstAndSecondMethods(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	msg := make([]byte, 126)
	go func() {
		io.ReadFull(r, msg)
		wg.Done()
	}()

	writer := NewMethodWriter(w, 2) // Two indicates buffer for two outstanding requests

	if err := writer.Call(1, "test.method", 42, "second parameter"); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	if err := writer.Call(3, "test.method2", 911, "help value"); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	expectedMSG := `{"id":1,"method":"test.method","params":[42,"second parameter"]}
{"id":3,"method":"test.method2","params":[911,"help value"]}
`
	wg.Wait()

	if string(msg) != expectedMSG {
		t.Errorf("Expected\n`%s`, received\n`%s`", expectedMSG, msg)
		return
	}

	writer.StopServing(1 * time.Second)
}

func TestNewMethodWriter_outstandingRequests(t *testing.T) {
	r, w := io.Pipe()

	msg := make([]byte, 128)
	go io.ReadFull(r, msg)

	writer := NewMethodWriter(w, 1)
	defer writer.StopServing(1 * time.Second)

	if err := writer.Call(1, "test.method", 42, "second parameter"); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	if err := writer.Call(3, "test.method2", 911, "help value"); err == nil {
		t.Error("Expected error, received none")
		return
	}
}

func TestMethodWriter_StopServing(t *testing.T) {
	r, w := io.Pipe()

	wg := sync.WaitGroup{}
	wg.Add(1)

	msg := make([]byte, 65)
	go func() {
		io.ReadFull(r, msg)
		wg.Done()
	}()

	writer := NewMethodWriter(w, 1)
	if err := writer.Call(1, "test.method", 42, "second parameter"); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	wg.Wait()

	expectedMSG := `{"id":1,"method":"test.method","params":[42,"second parameter"]}`

	if string(msg[:64]) != expectedMSG {
		t.Errorf("Expected `%s`, received `%s`", expectedMSG, msg[:64])
		return
	}

	writer.StopServing(1 * time.Second)

	if err := writer.Call(2, "system.something", 42, "going to fail"); err != io.EOF {
		t.Errorf("Expected io.EOF, received %v", err)
		return
	}
}

func TestMethodWriter_StopServingWithTimeout(t *testing.T) {
	r, w := io.Pipe()

	r.Close()

	writer := NewMethodWriter(w, 1)
	if err := writer.Call(1, "test.method", 42, "second parameter"); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	if err := writer.StopServing(0 * time.Second); err == nil {
		t.Error("Expected timeout error, received: nil")
		return
	}

	time.Sleep(2 * time.Second)

	if err := writer.Call(2, "system.something", 42, "going to fail"); err != io.EOF {
		t.Errorf("Expected io.EOF, received %v", err)
		return
	}
}


func TestNewMethodWriter_callWithInvalidParam(t *testing.T) {
	_, w := io.Pipe()
	writer := NewMethodWriter(w, 1)

	if err := writer.Call(1, "invalid.call", make(chan bool)); err != nil {
		t.Errorf("No error expected, received %s", err.Error())
		return
	}

	if err := <- writer.Errors; err == nil {
		t.Errorf("Expected an error, received %s", err.Error())
		return
	}
}
