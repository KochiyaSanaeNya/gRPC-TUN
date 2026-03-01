package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

type tunnelServer struct {
	targetAddr string
}

func (s *tunnelServer) Tunnel(stream Tunnel_TunnelServer) error {
	ctx, cancel := context.WithTimeout(stream.Context(), 10*time.Second)
	defer cancel()

	targetConn, err := (&net.Dialer{}).DialContext(ctx, "tcp", s.targetAddr)
	if err != nil {
		return fmt.Errorf("dial target: %w", err)
	}
	defer targetConn.Close()

	if err := proxyConnAndStream(targetConn, stream); err != nil {
		log.Printf("server proxy error: %v", err)
	}

	return nil
}
