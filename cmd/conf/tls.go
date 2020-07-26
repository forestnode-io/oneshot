package conf

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/spf13/pflag"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func genCertAndKey(port string) (location string, err error) {
	ips := []net.IP{}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	var home string
	for _, addr := range addrs {
		saddr := addr.String()

		if strings.Contains(saddr, "::") {
			continue
		}

		parts := strings.Split(saddr, "/")

		if parts[0] == "127.0.0.1" || parts[0] == "localhost" {
			home = parts[0]
			continue
		}

		ips = append(ips, net.ParseIP(parts[0]))
	}
	if len(ips) == 0 {
		ips = append(ips, net.ParseIP(home))
	}

	max := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, max)
	subject := pkix.Name{
		Organization:       []string{"Raphael Reyna"},
		OrganizationalUnit: []string{"oneshot"},
		CommonName:         "oneshot",
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  ips,
	}
	pk, _ := rsa.GenerateKey(rand.Reader, 2048)

	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &pk.PublicKey, pk)

	tempDir, err := ioutil.TempDir("", "oneshot")
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}
	certOut, _ := os.Create(filepath.Join(tempDir, "cert.pem"))
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, _ := os.Create(filepath.Join(tempDir, "key.pem"))
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)})
	keyOut.Close()

	return tempDir, nil
}

// setupCertAndKey checks to see if we need to self-sign any certificates, and if so it returns their location
func (c *Conf) SetupCertAndKey(fs *pflag.FlagSet) (location string, err error) {

	if (fs.Changed("tls-key") && fs.Changed("tls-cert") &&
		c.CertFile == "" && c.KeyFile == "") || c.Sstls {
		location, err := genCertAndKey(c.Port)
		if err != nil {
			return "", err
		}

		c.CertFile = filepath.Join(location, "cert.pem")
		c.KeyFile = filepath.Join(location, "key.pem")

		return location, nil
	}

	return "", nil
}
