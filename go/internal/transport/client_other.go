//go:build !windows

package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTPClient implements Client over HTTP for non-Windows platforms.
type HTTPClient struct {
	port   int
	client *http.Client
}

// NewHTTPClient creates an HTTPClient targeting 127.0.0.1:<port>.
func NewHTTPClient(port int) *HTTPClient {
	return &HTTPClient{
		port:   port,
		client: &http.Client{},
	}
}

// Send posts req to the daemon's /command endpoint and returns the response.
func (c *HTTPClient) Send(req Request) (Response, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/command", c.port)

	body, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("http client: marshal request: %w", err)
	}

	httpResp, err := c.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("http client: post %s: %w", url, err)
	}
	defer httpResp.Body.Close()

	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("http client: decode response: %w", err)
	}
	return resp, nil
}

// Close is a no-op for the HTTP client.
func (c *HTTPClient) Close() error { return nil }
