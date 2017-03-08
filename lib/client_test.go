package jsonrpc

import (
	"io"
	"sync"
	"testing"
	"time"
)

func TestClient_Call(t *testing.T) {
	cr, sw := io.Pipe()
	sr, cw := io.Pipe()

	go func() {
		expectedMSG := `{"id":1,"method":"system.info","params":["cpu","mem"]}`
		b := make([]byte, len(expectedMSG))
		io.ReadFull(sr, b)

		if string(b) != expectedMSG {
			t.Errorf("Expected `%s`, received `%s`", expectedMSG, string(b))
		}

		sw.Write([]byte(`{"id":1,"result":[75,"1GB"]}`))
	}()

	client := NewClient(cr, cw)

	result, err := client.Call(1, "system.info", "cpu", "mem")
	if err != nil {
		t.Errorf("Expected no errors, received %s", err.Error())
		return
	}

	switch r := result.(type) {
	case []interface{}:
		if v, ok := r[0].(Number); !ok {
			t.Errorf("Expected a Number to be returned from call, received %T", v)
			return
		} else if v.String() != "75" {
			cpu, _ := v.Int64()
			t.Errorf("Expected 75 to be returned from call, received %d", cpu)
			return
		}

		if v, ok := r[1].(string); !ok {
			t.Errorf("Expected a string to be returned from call, received %T", v)
			return
		} else if v != "1GB" {
			t.Errorf("Expected `1GB` to be returned from call, received `%s`", v)
			return
		}
	default:
		t.Errorf("Expected an array to be returned from call, received %T", r)
		return
	}
}

func TestClient_StopServing(t *testing.T) {
	cr, sw := io.Pipe()
	sr, cw := io.Pipe()

	go func() {
		expectedMSG := `{"id":1,"method":"system.info","params":["cpu","mem"]}`
		b := make([]byte, len(expectedMSG))
		io.ReadFull(sr, b)

		if string(b) != expectedMSG {
			t.Errorf("Expected `%s`, received `%s`", expectedMSG, string(b))
		}

		sw.Write([]byte(`{"id":1,"result":[75,"1GB"]}`))
	}()

	client := NewClient(cr, cw)

	client.Call(1, "system.info", "cpu", "mem") // Ignore result
	client.StopServing(0 * time.Second)
	_, err := client.Call(2, "system.info", "cpu", "mem") // Ignore result

	if err != io.EOF {
		t.Errorf("Expected EOF, received %v", err)
	}
}

func TestClient_TooManyOutstandingReq(t *testing.T) {
	cr, _ := io.Pipe()
	_, cw := io.Pipe()

	client := NewClient(cr, cw)

	wg := sync.WaitGroup{}
	wg.Add(1)

	for i := int64(1); i <= 12; i++ {
		go func(i int64) {
			if _, err := client.Call(i, "system.info", "cpu", "mem"); err == nil {
				t.Error("Expected error, received none")
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestClient_ReuseID(t *testing.T) {
	cr, _ := io.Pipe()
	_, cw := io.Pipe()

	client := NewClient(cr, cw)

	wg := sync.WaitGroup{}
	wg.Add(1)

	for i := 0; i < 2; i++ {
		go func() {
			if _, err := client.Call(1, "system.info", "cpu", "mem"); err == nil {
				t.Error("Expected error, received none")
			}
			wg.Done()
		}()
	}

	wg.Wait()

}

/*
func TestClient_CallBrokenPipe(t *testing.T) {
	cr, _ := io.Pipe()
	_, cw := io.Pipe()

	client := NewClient(cr, cw)

	cr.Close()
	client.Call(1, "system.info", "cpu", "mem")
	anoeuh
}
*/

func TestClient_Call_rpcError(t *testing.T) {
	cr, sw := io.Pipe()
	sr, cw := io.Pipe()

	go func() {
		expectedMSG := `{"id":1,"method":"system.info","params":["cpu","mem"]}`
		b := make([]byte, len(expectedMSG))
		io.ReadFull(sr, b)

		if string(b) != expectedMSG {
			t.Errorf("Expected `%s`, received `%s`", expectedMSG, string(b))
		}

		sw.Write([]byte(`{"id":1,"error":{"code":-32603,"message":"Internal error"}}`))
	}()

	client := NewClient(cr, cw)

	result, err := client.Call(1, "system.info", "cpu", "mem")
	if err == nil {
		t.Errorf("Expected an error, received nil and result = %#v", result)
		return
	}

	if t.Failed() {
		return // Return if the other go-routine failed
	}

	switch e := err.(type) {
	case Error:
		if e.Code() != -32603 {
			t.Errorf("Expected error code -32603, received %d", e.Code())
			return
		} else if e.Error() != "Internal error" {
			t.Errorf("Expected error string `Internal Error`, received `%s;", e.Error())
			return
		}
	default:
		t.Errorf("Expected an internal.Error to be returned from call, received %T", e)
		return
	}
}

func TestClient_Call_dsServerSideClosed(t *testing.T) {
	cr, sw := io.Pipe() // DS (downstream)
	sr, cw := io.Pipe() // UP (upstream)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		expectedMSG := `{"id":1,"method":"system.info","params":["cpu","mem"]}`
		b := make([]byte, len(expectedMSG))
		io.ReadFull(sr, b)

		if string(b) != expectedMSG {
			t.Errorf("Expected `%s`, received `%s`", expectedMSG, string(b))
		}

		sw.Write([]byte(`{"id":1,"result":[75,"1GB"]}`))
		sw.Close()
		wg.Done()
	}()

	client := NewClient(cr, cw)

	if _, err := client.Call(1, "system.info", "cpu", "mem"); err != nil {
		t.Errorf("Expected no error, received %s", err.Error())
		return
	}

	if t.Failed() {
		return // Return if the other go-routine failed
	}

	wg.Wait()

	if _, err := client.Call(1, "system.info", "cpu", "mem"); err == nil {
		t.Errorf("Expected an error, received nil")
		return
	}

	/*
		switch e := err.(type) {
		case Error:
			if e.Code() != -32603 {
				t.Errorf("Expected error code -32603, received %d", e.Code())
				return
			} else if e.Error() != "Internal error" {
				t.Errorf("Expected error string `Internal Error`, received `%s;", e.Error())
				return
			}
		default:
			t.Errorf("Expected an internal.Error to be returned from call, received %T", e)
			return
		}
	*/
}
