//go:build windows

package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Microsoft/go-winio"
)

// PipeClient implements Client over a Windows named pipe.
type PipeClient struct {
	pipeName string
}

// NewPipeClient creates a PipeClient for the given pipe name (without \\.\pipe\ prefix).
func NewPipeClient(pipeName string) *PipeClient {
	return &PipeClient{pipeName: pipeName}
}

// Send opens a new pipe connection, sends req, reads a single response, then closes.
func (c *PipeClient) Send(req Request) (Response, error) {
	path := fmt.Sprintf(`\\.\pipe\%s`, c.pipeName)

	conn, err := winio.DialPipe(path, durationPtr(5*time.Second))
	if err != nil {
		return Response{}, fmt.Errorf("pipe client: dial %q: %w", path, err)
	}
	defer conn.Close()

	data, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("pipe client: marshal request: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return Response{}, fmt.Errorf("pipe client: write: %w", err)
	}

	// Signal end of write so server can read all bytes.
	if tc, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = tc.CloseWrite()
	}

	buf, err := io.ReadAll(conn)
	if err != nil {
		return Response{}, fmt.Errorf("pipe client: read: %w", err)
	}
	buf = bytes.TrimRight(buf, "\n")

	var resp Response
	if err := json.Unmarshal(buf, &resp); err != nil {
		return Response{}, fmt.Errorf("pipe client: unmarshal response: %w", err)
	}
	return resp, nil
}

// Close is a no-op; each Send uses a fresh connection.
func (c *PipeClient) Close() error { return nil }

func durationPtr(d time.Duration) *time.Duration { return &d }
