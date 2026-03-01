package main

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
)

const chunkSize = 32 * 1024

type streamSender interface {
	Send(*Chunk) error
}

type streamReceiver interface {
	Recv() (*Chunk, error)
}

type streamCloseSender interface {
	CloseSend() error
}

type tunnelStream interface {
	streamSender
	streamReceiver
	Context() context.Context
}

func proxyConnAndStream(conn net.Conn, stream tunnelStream) error {
	errCh := make(chan error, 2)
	var once sync.Once

	closeSend := func() {
		if cs, ok := stream.(streamCloseSender); ok {
			_ = cs.CloseSend()
		}
	}

	go func() {
		errCh <- pipeConnToStream(conn, stream)
	}()

	go func() {
		errCh <- pipeStreamToConn(stream, conn)
	}()

	err := <-errCh
	once.Do(closeSend)
	_ = conn.Close()
	secondErr := <-errCh

	if err == nil || errors.Is(err, io.EOF) {
		err = secondErr
	}

	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func pipeConnToStream(conn net.Conn, stream streamSender) error {
	buf := make([]byte, chunkSize)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			payload := make([]byte, n)
			copy(payload, buf[:n])
			if sendErr := stream.Send(&Chunk{Data: payload}); sendErr != nil {
				return sendErr
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return stream.Send(&Chunk{Fin: true})
			}
			return err
		}
	}
}

func pipeStreamToConn(stream streamReceiver, conn net.Conn) error {
	for {
		chunk, err := stream.Recv()
		if err != nil {
			return err
		}
		if chunk.Fin {
			return io.EOF
		}
		if len(chunk.Data) == 0 {
			continue
		}
		if _, err := conn.Write(chunk.Data); err != nil {
			return err
		}
	}
}
