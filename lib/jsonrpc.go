package jsonrpc

import (
	"encoding/json"

	"github.com/dekelund/jsonrpc/lib/internal"
)

type Number struct {
	json.Number
}

type Error struct {
	e *internal.Error
}

func (err Error) Code() int {
	return err.e.Code
}

func (err Error) Error() string {
	return err.e.Error()
}

func fixResultTypes(result interface{}) interface{} {
	switch r := result.(type) {
	case json.Number:
		result = Number{r}
	case []interface{}:
		for i, v := range r {
			switch v := v.(type) {
			case json.Number:
				r[i] = Number{v}
			}
		}
	}

	return result
}
