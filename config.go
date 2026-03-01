package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	modeClient = "client"
	modeServer = "server"
)

type Config struct {
	Mode   string       `json:"mode"`
	Client ClientConfig `json:"client"`
	Server ServerConfig `json:"server"`
	TLS    TLSConfig    `json:"tls"`
}

type ClientConfig struct {
	LocalListen        string `json:"local_listen"`
	ServerAddr         string `json:"server_addr"`
	DialTimeoutSeconds int    `json:"dial_timeout_seconds"`
}

type ServerConfig struct {
	ListenAddr string `json:"listen_addr"`
	TargetAddr string `json:"target_addr"`
}

type TLSConfig struct {
	CertFile                string `json:"cert_file"`
	KeyFile                 string `json:"key_file"`
	CAFile                  string `json:"ca_file"`
	ServerName              string `json:"server_name"`
	PinnedFingerprintSHA256 string `json:"pinned_fingerprint_sha256"`
	InsecureSkipVerify      bool   `json:"insecure_skip_verify"`
	AllowPlaintext          bool   `json:"allow_plaintext"`
}

func loadConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
	switch cfg.Mode {
	case modeClient:
		if cfg.Client.LocalListen == "" {
			return fmt.Errorf("client.local_listen is required")
		}
		if cfg.Client.ServerAddr == "" {
			return fmt.Errorf("client.server_addr is required")
		}
	case modeServer:
		if cfg.Server.ListenAddr == "" {
			return fmt.Errorf("server.listen_addr is required")
		}
		if cfg.Server.TargetAddr == "" {
			return fmt.Errorf("server.target_addr is required")
		}
	default:
		return fmt.Errorf("mode must be %q or %q", modeClient, modeServer)
	}

	if cfg.TLS.AllowPlaintext {
		return nil
	}

	if (cfg.TLS.CertFile == "") != (cfg.TLS.KeyFile == "") {
		return fmt.Errorf("tls.cert_file and tls.key_file must be set together")
	}

	if cfg.TLS.CertFile != "" {
		if filepath.Ext(cfg.TLS.CertFile) != ".pem" || filepath.Ext(cfg.TLS.KeyFile) != ".pem" {
			return fmt.Errorf("tls.cert_file and tls.key_file must use .pem suffix")
		}
	}

	return nil
}
