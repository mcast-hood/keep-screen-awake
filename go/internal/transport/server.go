package transport

import "context"

// HandlerFunc is the function signature for handling an IPC request.
type HandlerFunc func(req Request) Response

// Server listens for incoming IPC connections and dispatches requests.
type Server interface {
	// Serve starts the server and blocks until ctx is cancelled or a fatal
	// error occurs.
	Serve(ctx context.Context, handler HandlerFunc) error
}
