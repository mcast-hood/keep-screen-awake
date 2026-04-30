//go:build windows

package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/Microsoft/go-winio"
)

const pipeSDDL = "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GRGW;;;AU)"

// PipeServer implements Server over a Windows named pipe.
type PipeServer struct {
	pipeName string
}

// NewPipeServer creates a PipeServer for the given pipe name (without \\.\pipe\ prefix).
func NewPipeServer(pipeName string) *PipeServer {
	return &PipeServer{pipeName: pipeName}
}

// Serve starts listening on the named pipe and dispatches each connection to handler.
func (s *PipeServer) Serve(ctx context.Context, handler HandlerFunc) error {
	path := fmt.Sprintf(`\\.\pipe\%s`, s.pipeName)

	cfg := &winio.PipeConfig{
		SecurityDescriptor: pipeSDDL,
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	ln, err := winio.ListenPipe(path, cfg)
	if err != nil {
		return fmt.Errorf("pipe server: listen %q: %w", path, err)
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("pipe server: accept: %w", err)
			}
		}
		go handleConn(conn, handler)
	}
}

func handleConn(conn net.Conn, handler HandlerFunc) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Printf("pipe server: read: %v", err)
		}
		return
	}

	var req Request
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		resp := Response{OK: false, Error: fmt.Sprintf("bad request: %v", err)}
		writeResponse(conn, resp)
		return
	}

	resp := handler(req)
	writeResponse(conn, resp)
}

func writeResponse(w io.Writer, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("pipe server: marshal response: %v", err)
		return
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		log.Printf("pipe server: write response: %v", err)
	}
}
