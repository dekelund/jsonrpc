package internal

import (
	"encoding/json"
)

type Response struct {
	ID     int64       `json:"id,omitempty"`
	Error  *Error      `json:"error,omitempty"`
	Result interface{} `json:"result,omitempty"`
}

type Method struct {
	ID     int64         `json:"id,omitempty"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err *Error) Error() string {
	return err.Message
}

func (m Method) String() string {
	b, err := json.Marshal(m)

	if err != nil {
		panic(err.Error())
	}

	return string(b)
}

func (r Response) String() string {
	var err error
	var str []byte

	if r.Error != nil {
		str, _ = json.Marshal(struct {
			ID    int64  `json:"id"`
			Error *Error `json:"error"`
		}{r.ID, r.Error})
	} else {
		str, err = json.Marshal(struct {
			ID     int64       `json:"id"`
			Result interface{} `json:"result"`
		}{r.ID, r.Result})

		if err != nil {
			panic(err.Error())
		}
	}

	return string(str)
}

var EOF = &Error{-1, "EOF"}
