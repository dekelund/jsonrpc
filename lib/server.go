package jsonrpc

import (
	"io"

	"github.com/dekelund/jsonrpc/lib/internal"
)

type Server struct {
	r *internal.MethodReader
	w *internal.ResponseWriter
}

func NewServer(r io.ReadCloser, w io.WriteCloser) *Server {
	return &Server{internal.NewMethodReader(r, 10), internal.NewResponseWriter(w, 10)}
}
