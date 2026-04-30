//go:build !windows

package main

import "github.com/mcast-hood/keep-screen-awake/internal/transport"

func newClient() transport.Client {
	return transport.NewHTTPClient(9877)
}
