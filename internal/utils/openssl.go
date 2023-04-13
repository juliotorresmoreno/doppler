package utils

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

func publicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

func Generate() (keyOut, certOut *bytes.Buffer, err error) {
	keyOut, certOut = bytes.NewBufferString(""), bytes.NewBufferString("")
	var (
		host     = "localhost,127.0.0.1"
		validFor = 365 * 24 * time.Hour
		rsaBits  = 2048
	)

	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return
	}

	keyUsage := x509.KeyUsageDigitalSignature
	keyUsage |= x509.KeyUsageKeyEncipherment

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return keyOut, certOut, fmt.Errorf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return keyOut, certOut, fmt.Errorf("failed to create certificate: %v", err)
	}

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return keyOut, certOut, fmt.Errorf("failed to write data to cert.pem: %v", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return keyOut, certOut, fmt.Errorf("unable to marshal private key: %v", err)
	}
	err = pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if err != nil {
		return keyOut, certOut, fmt.Errorf("failed to write data to key.pem: %v", err)
	}
	return
}
