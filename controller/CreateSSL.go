package controller

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/sunvc/NoLets/common"
)

// CreateSSL generates a self-signed TLS certificate if it doesn't exist.
func CreateSSL() {
	if err := os.MkdirAll(common.BaseDir(), 0755); err != nil {
		log.Fatalf("CreateSSL: failed to create data dir %s: %v", common.BaseDir(), err)
	}

	keyPath := common.BaseDir("key.pem")
	certPath := common.BaseDir("cert.pem")

	// Reuse the existing pair only when BOTH files are present; a missing
	// key means regenerating, otherwise the server would later fail to load TLS.
	if _, errCert := os.Stat(certPath); errCert == nil {
		if _, errKey := os.Stat(keyPath); errKey == nil {
			log.Printf("CreateSSL: cert already exists at %s, skip generation", certPath)
			common.LocalConfig.System.Key = keyPath
			common.LocalConfig.System.Cert = certPath
			return
		}
	}

	// Generate ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	// Set certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Province:     []string{"Beijing"},
			Locality:     []string{"Beijing"},
			Organization: []string{"NoLetter Inc."},
			CommonName:   "localhost",
			SerialNumber: "001",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(5 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"*"},
		EmailAddresses:        []string{"to@wzs.app"},
	}

	// Self-sign the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(err)
	}

	// Write certificate and private key to files
	certOut, err := os.Create(certPath)
	if err != nil {
		panic(err)
	}
	defer certOut.Close()

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	// Private key: 0600, it must not be world-readable
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}
	defer keyOut.Close()

	marshaledKey, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}

	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: marshaledKey})

	log.Printf("CreateSSL: self-signed TLS certificate generated at %s", certPath)

	common.LocalConfig.System.Key = keyPath
	common.LocalConfig.System.Cert = certPath
}
