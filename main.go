package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	flag.Usage = func() {
		fmt.Println("gRPC -cfg <config.json>")
		fmt.Println()
		flag.PrintDefaults()
	}

	configPath := flag.String("cfg", "", "Path to JSON config file")
	flag.Parse()

	if flag.NArg() == 0 && *configPath == "" {
		flag.Usage()
		return
	}

	if *configPath == "" {
		log.Fatal("-cfg is required")
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	switch cfg.Mode {
	case modeClient:
		if err := runClient(cfg); err != nil {
			log.Fatalf("client error: %v", err)
		}
	case modeServer:
		if err := runServer(cfg); err != nil {
			log.Fatalf("server error: %v", err)
		}
	default:
		log.Fatalf("unsupported mode: %s", cfg.Mode)
	}
}

func runServer(cfg Config) error {
	var (
		tlsConfig   *tls.Config
		fingerprint string
		err         error
	)
	if !cfg.TLS.AllowPlaintext {
		tlsConfig, fingerprint, err = buildServerTLSConfig(cfg.TLS)
		if err != nil {
			return err
		}
		if fingerprint != "" {
			log.Printf("auto TLS certificate fingerprint (sha256): %s", fingerprint)
		}
	}

	if cfg.TLS.AllowPlaintext {
		log.Printf("plaintext mode enabled (no TLS)")
	}

	listener, err := net.Listen("tcp", cfg.Server.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	serverOptions := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
		grpc.ForceServerCodec(tunnelCodec),
	}
	if cfg.TLS.AllowPlaintext {
		serverOptions = append(serverOptions, grpc.Creds(insecure.NewCredentials()))
	} else {
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	server := grpc.NewServer(serverOptions...)

	RegisterTunnelServiceServer(server, &tunnelServer{targetAddr: cfg.Server.TargetAddr})

	log.Printf("Tunnel Connect Success,gRPC server listening on %s, forwarding to %s", cfg.Server.ListenAddr, cfg.Server.TargetAddr)
	return server.Serve(listener)
}

func runClient(cfg Config) error {
	var tlsConfig *tls.Config
	if !cfg.TLS.AllowPlaintext {
		var err error
		tlsConfig, err = buildClientTLSConfig(cfg.TLS)
		if err != nil {
			return err
		}
	}

	timeout := time.Duration(cfg.Client.DialTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultDialTimeout
	}

	dialCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dialOptions := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.ForceCodec(tunnelCodec)),
		grpc.WithBlock(),
	}
	if cfg.TLS.AllowPlaintext {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	conn, err := grpc.DialContext(
		dialCtx,
		cfg.Client.ServerAddr,
		dialOptions...,
	)
	if err != nil {
		return fmt.Errorf("gRPC dial failed: %w", err)
	}
	defer conn.Close()

	listener, err := net.Listen("tcp", cfg.Client.LocalListen)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	client := NewTunnelServiceClient(conn)
	log.Printf("Tunnel Connect Success,client listening on %s, tunneling to %s", cfg.Client.LocalListen, cfg.Client.ServerAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept failed: %w", err)
		}
		go handleClientConn(client, conn)
	}
}

func handleClientConn(client TunnelServiceClient, conn net.Conn) {
	defer conn.Close()

	stream, err := client.Tunnel(context.Background())
	if err != nil {
		log.Printf("open tunnel failed: %v", err)
		return
	}

	if err := proxyConnAndStream(conn, stream); err != nil {
		log.Printf("proxy error: %v", err)
	}
}
