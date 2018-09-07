package generatesomecerts

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

var OrgName = "Acme Co"

type Cert struct {
	Template   *x509.Certificate
	DER        []byte // DER encoded
	PrivateKey *rsa.PrivateKey
}

func (cert Cert) String() string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.DER}))
}

func CA() (Cert, error) {
	var cert *x509.Certificate

	caPriv, err := rsa.GenerateKey(rand.Reader, 512)

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 10000) // nearly 30 years

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return Cert{}, fmt.Errorf("failed to generate serial number: %s", err)
	}

	caTemplate := x509.Certificate{
		IsCA:         true,
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{OrgName},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	caTemplate.KeyUsage |= x509.KeyUsageCertSign

	certBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, caPriv.Public(), caPriv)
	if err != nil {
		return Cert{}, fmt.Errorf("Failed to create certificate: %s", err)
	}

	return Cert{cert, certBytes, caPriv}, nil
}

func (ca Cert) SignedCert(hosts ...string) (Cert, error) {
	var cert Cert
	var certTemplate *x509.Certificate

	priv, err := rsa.GenerateKey(rand.Reader, 512)

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 10000) // nearly 30 years

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return cert, fmt.Errorf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{OrgName},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	template.KeyUsage |= x509.KeyUsageCertSign // not sure if correct?

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, ca.Template, priv.Public(), ca.PrivateKey)
	if err != nil {
		return cert, fmt.Errorf("Failed to create certificate: %s", err)
	}

	return Cert{certTemplate, certBytes, priv}, nil
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}
