package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

const (
	defaultDialTimeout = 10 * time.Second
	maxMessageSize     = 16 << 20
)

func buildServerTLSConfig(cfg TLSConfig) (*tls.Config, string, error) {
	var (
		cert tls.Certificate
		err  error
		fp   string
	)

	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err = tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, "", fmt.Errorf("load tls cert/key: %w", err)
		}
	} else {
		cert, fp, err = generateSelfSignedCert([]string{"localhost", "127.0.0.1"})
		if err != nil {
			return nil, "", err
		}
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2"},
	}, fp, nil
}

func buildClientTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: cfg.ServerName,
		NextProtos: []string{"h2"},
	}

	if cfg.CAFile != "" {
		pool := x509.NewCertPool()
		caBytes, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read tls ca_file: %w", err)
		}
		if !pool.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("invalid tls ca_file")
		}
		tlsCfg.RootCAs = pool
	}

	pinned := normalizeFingerprint(cfg.PinnedFingerprintSHA256)
	if pinned != "" {
		tlsCfg.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("no server certificate")
			}
			got := sha256Fingerprint(rawCerts[0])
			if got != pinned {
				return fmt.Errorf("server certificate fingerprint mismatch: got %s", got)
			}
			return nil
		}
	}

	if tlsCfg.RootCAs == nil {
		switch {
		case pinned != "":
			tlsCfg.InsecureSkipVerify = true
		case cfg.InsecureSkipVerify:
			tlsCfg.InsecureSkipVerify = true
		default:
			return nil, fmt.Errorf("tls requires ca_file, pinned_fingerprint_sha256, or insecure_skip_verify")
		}
	}

	return tlsCfg, nil
}

func generateSelfSignedCert(hosts []string) (tls.Certificate, string, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, "", fmt.Errorf("generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, "", fmt.Errorf("generate serial: %w", err)
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"gRPC auto TLS"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else if host != "" {
			tmpl.DNSNames = append(tmpl.DNSNames, host)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, "", fmt.Errorf("create certificate: %w", err)
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, "", fmt.Errorf("marshal private key: %w", err)
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return tls.Certificate{}, "", fmt.Errorf("load key pair: %w", err)
	}

	return cert, sha256Fingerprint(derBytes), nil
}

func sha256Fingerprint(derBytes []byte) string {
	sum := sha256.Sum256(derBytes)
	return hex.EncodeToString(sum[:])
}

func normalizeFingerprint(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "sha256:")
	return strings.ReplaceAll(value, ":", "")
}
