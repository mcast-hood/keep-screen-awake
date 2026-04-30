package transport

// Client sends IPC requests to the daemon and receives responses.
type Client interface {
	// Send transmits req to the daemon and returns the response.
	Send(req Request) (Response, error)
	// Close releases any underlying resources.
	Close() error
}
