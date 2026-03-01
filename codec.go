package main

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

type jsonCodec struct{}

func (c *jsonCodec) Name() string {
	return "json"
}

func (c *jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (c *jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

var tunnelCodec = &jsonCodec{}

func init() {
	encoding.RegisterCodec(tunnelCodec)
}
