package internal

import (
	"testing"
)

func TestMethod_String(t *testing.T) {
	expected := `{"id":10,"method":"system.login","params":["root","changeme"]}`
	method := Method{ID: 10, Method: "system.login", Params:[]interface{}{"root", "changeme"}}
	if method.String() != expected {
		t.Errorf("expected %s received %s", expected, method.String())
	}
}

func TestResponse_String(t *testing.T) {
	expected := `{"id":10,"result":[true,"success"]}`
	response := Response{ID: 10, Result: []interface{}{true, "success"}}
	if response.String() != expected {
		t.Errorf("expected %s received %s", expected, response.String())
	}
}

func TestError_Error(t *testing.T) {
	expected := `{"id":10,"error":{"code":-1,"message":"EOF"}}`
	response := Response{ID: 10, Error: EOF}

	if response.String() != expected {
		t.Errorf("expected %s received %s", expected, response.String())
	}

	err := response.Error

	if err.Code != -1 {
		t.Errorf("expected -1 as code received %d", err.Code)
	}

	if err.Message != "EOF" {
		t.Errorf("expected error message EOF received %s", err.Message)
	}

	if err.Error() != "EOF" {
		t.Errorf("expected error interface to contain string EOF, received %v", err)
	}
}

func TestResponse_StringInvalidResult(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from Stringer, panic didn't happen")
		}
	}()

	Response{ID: 10, Result: make(chan int, 1)}.String()
}

func TestMethod_StringInvalidParam(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from Stringer, panic didn't happen")
		}
	}()

	Method{ID: 10, Method: "system.login", Params: []interface{}{make(chan int, 1)}}.String()
}