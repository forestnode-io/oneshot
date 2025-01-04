package ssl

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/forestnode-io/oneshot/v2/pkg/configuration"
	"github.com/forestnode-io/oneshot/v2/pkg/events"
	"github.com/forestnode-io/oneshot/v2/pkg/log"
)

func GetTLSConfig(config *configuration.TLS) (*tls.Config, error) {
	if !config.IsEnabled() {
		return nil, nil
	}

	var (
		tc  tls.Config
		err error
	)

	// If we are using mTLS, we dont need a cert pol unless we are verifying the client
	// presented certificate.
	// We also have the option to use the system cert pool.
	if config.MTLS != nil {
		var certPool *x509.CertPool
		if config.MTLS.CertPool != nil {
			if config.MTLS.CertPool.UseSystemRoots {
				certPool, err = x509.SystemCertPool()
				if err != nil {
					return nil, fmt.Errorf("failed to get system cert pool for mTLS: %w", err)
				}
			}
			if config.MTLS.CertPool.Certs != nil {
				if certPool == nil {
					certPool = x509.NewCertPool()
				}
				for _, certPathOrContent := range config.MTLS.CertPool.Certs {
					certBytes, err := certPathOrContent.GetContent()
					if err != nil {
						return nil, fmt.Errorf("failed to get cert: %w", err)
					}
					ok := certPool.AppendCertsFromPEM(certBytes)
					if !ok {
						return nil, fmt.Errorf("failed to append cert to pool")
					}
				}
			}
		}
		if certPool != nil {
			tc.ClientCAs = certPool
			switch config.MTLS.Mode {
			case configuration.MTLSModeRequire:
				tc.ClientAuth = tls.RequireAndVerifyClientCert
			case configuration.MTLSModeRequest:
				tc.ClientAuth = tls.VerifyClientCertIfGiven
			}
		} else {
			switch config.MTLS.Mode {
			case configuration.MTLSModeRequire:
				tc.ClientAuth = tls.RequireAnyClientCert
			case configuration.MTLSModeRequest:
				tc.ClientAuth = tls.RequestClientCert
			}
		}
	}

	// If we are using static certificates, we need to load the certificate and key
	// and set the cert in the tls.Config.
	if certConf := config.Certificate.Certificate; !certConf.IsZero() {
		cert, err := certConf.GetContent()
		if err != nil {
			return nil, fmt.Errorf("failed to get cert: %w", err)
		}

		key, err := config.PrivateKey.GetContent()
		if err != nil {
			return nil, fmt.Errorf("failed to get private key: %w", err)
		}

		x509KP, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get x509 key pair: %w", err)
		}

		tc.Certificates = []tls.Certificate{x509KP}
	} else if certConf := config.Certificate.GeneratedCertificate; certConf.GenerateAtStartup {
		rootCert, rootPrivKey, err := generateRootCertAndKey(certConf)
		if err != nil {
			return nil, fmt.Errorf("failed to generate root cert and key: %w", err)
		}

		certPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCert.Raw})
		keyDERBytes, err := x509.MarshalPKCS8PrivateKey(rootPrivKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal private key: %w", err)
		}
		keyPEMBYtes := pem.EncodeToMemory(&pem.Block{Type: KeyType(certConf.GetPrivateKeyAlgorithm()).toPEMBlockType(), Bytes: keyDERBytes})

		x509KP, err := tls.X509KeyPair(certPEMBytes, keyPEMBYtes)
		if err != nil {
			return nil, fmt.Errorf("failed to get x509 key pair: %w", err)
		}

		tc.Certificates = []tls.Certificate{x509KP}
	} else if certConf := config.Certificate.GeneratedCertificate; !certConf.GenerateAtStartup {
		var (
			rootCert    *x509.Certificate
			rootPrivKey any
		)

		leafCertTemplate, err := CertFromConfig(config.Certificate.GeneratedCertificate, true)
		if err != nil {
			return nil, fmt.Errorf("failed to create certificate template: %w", err)
		}

		// Only use a root CA if we are exporting it, otherwise we can just generate a leaf cert on the fly.
		if certConf.ExportCA != nil {
			rootCert, rootPrivKey, err = generateRootCertAndKey(config.Certificate.GeneratedCertificate)
			if err != nil {
				return nil, fmt.Errorf("failed to generate root cert and key: %w", err)
			}
		} else if sgnMtrl := certConf.SigningMaterial; sgnMtrl != nil {
			rootCertBytes, err := sgnMtrl.Certificate.GetContent()
			if err != nil {
				return nil, fmt.Errorf("failed to get root cert: %w", err)
			}
			rootCertBytes = bytes.TrimSpace(rootCertBytes)
			rootCertPEM, _ := pem.Decode(rootCertBytes)
			rootCert, err = x509.ParseCertificate(rootCertPEM.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse root cert: %w\n%s", err, string(rootCertBytes))
			}

			if !rootCert.IsCA {
				return nil, fmt.Errorf("root certificate is not a CA")
			}

			rootPrivKeyBytes, err := sgnMtrl.PrivateKey.GetContent()
			if err != nil {
				return nil, fmt.Errorf("failed to get root private key: %w", err)
			}
			privKeyPEM, _ := pem.Decode(rootPrivKeyBytes)
			rootPrivKey, err = x509.ParsePKCS8PrivateKey(privKeyPEM.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse root private key: %w", err)
			}

			rootSigner, ok := rootPrivKey.(crypto.Signer)
			if !ok {
				return nil, fmt.Errorf("root private key is not a signer")
			}

			switch pt := rootSigner.Public().(type) {
			case *rsa.PublicKey:
				leafCertTemplate.SignatureAlgorithm = x509.SHA256WithRSA
			case *ecdsa.PublicKey:
				leafCertTemplate.SignatureAlgorithm = x509.ECDSAWithSHA256
			default:
				return nil, fmt.Errorf("unsupported public key type for root CA: %T", pt)
			}
		}

		certGenConfig := config.Certificate.GeneratedCertificate
		pkeyAlgorithm := certGenConfig.GetPrivateKeyAlgorithm()
		leafPrivKey, err := KeyType(pkeyAlgorithm).GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
		leafKeyBytes, err := x509.MarshalPKCS8PrivateKey(leafPrivKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal private key: %w", err)
		}
		leafKeyPEM := pem.EncodeToMemory(&pem.Block{Type: KeyType(pkeyAlgorithm).toPEMBlockType(), Bytes: leafKeyBytes})

		tc.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			ctx := hello.Context()
			events.SetClientHello(ctx, hello)

			log := log.Logger()
			log.Debug().
				Interface("client_hello", hello).
				Msg("Generating leaf cert")

			leafCert, err := generateLeafCertAndKey(leafCertTemplate, leafPrivKey, rootCert, rootPrivKey, hello)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to generate leaf cert and key")

				return nil, fmt.Errorf("failed to generate leaf cert and key: %w", err)
			}

			x509KP, err := tls.X509KeyPair(leafCert, leafKeyPEM)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to parse generated leaf and cert as x509 key pair")

				return nil, fmt.Errorf("failed to get x509 key pair: %w", err)
			}

			return &x509KP, nil
		}
	}

	tc.NextProtos = append(tc.NextProtos, "h2")
	tc.MinVersion, err = config.GetMinVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get min version: %w", err)
	}
	tc.MaxVersion, err = config.GetMaxVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get max version: %w", err)
	}

	return &tc, nil
}

func generateRootCertAndKey(config *configuration.GeneratedCertificate) (*x509.Certificate, PrivateKey, error) {
	pkeyAlgorithm := config.GetPrivateKeyAlgorithm()

	rootPrivKey, err := KeyType(pkeyAlgorithm).GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	rootCertTemplate, err := CertFromConfig(config, false)
	if err != nil {
		return nil, nil, err
	}

	rootCertBytes, err := x509.CreateCertificate(rand.Reader, rootCertTemplate, rootCertTemplate, rootPrivKey.Public(), rootPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	rootCert, err := x509.ParseCertificate(rootCertBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	if config.ExportCA != nil {
		if config.ExportCA.Path != "" {
			mode := os.FileMode(0644)
			if config.ExportCA.Mode != "" {
				modeUint, err := strconv.ParseUint(config.ExportCA.Mode, 8, 32)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse mode: %w", err)
				}
				mode = os.FileMode(modeUint)
			}
			os.WriteFile(config.ExportCA.Path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCertBytes}), mode)
		}
	}

	return rootCert, rootPrivKey, nil
}

func generateLeafCertAndKey(leafCertTemplate *x509.Certificate, leafPrivKey PrivateKey, rootCert *x509.Certificate, rootPrivKey any, hello *tls.ClientHelloInfo) ([]byte, error) {
	// If we have a client hello, we can add the server name and local address to the leaf cert.
	if hello != nil {
		leafCertTemplate.DNSNames = append(leafCertTemplate.DNSNames, hello.ServerName)
		localAddr := hello.Conn.LocalAddr().String()
		localHost, _, err := net.SplitHostPort(localAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to split local address: %w", err)
		}
		leafCertTemplate.IPAddresses = append(leafCertTemplate.IPAddresses, net.ParseIP(localHost))
	}

	// If we have a root cert and key, use them to sign the leaf cert
	// otherwise, self-sign the leaf cert.
	rc := rootCert
	rpk := rootPrivKey
	if rc == nil && rpk == nil {
		rc = leafCertTemplate
		rpk = leafPrivKey
	}

	leafCertBytes, err := x509.CreateCertificate(rand.Reader, leafCertTemplate, rc, leafPrivKey.Public(), rpk)
	if err != nil {
		log.Logger().Error().
			Err(err).
			Interface("leaf_cert_template", leafCertTemplate).
			Msg("Failed to create certificate")

		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	leafCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafCertBytes})

	return leafCertPEM, nil
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func CertFromConfig(config *configuration.GeneratedCertificate, isLeaf bool) (*x509.Certificate, error) {
	var (
		pkeyAlgorithm = KeyType(config.GetPrivateKeyAlgorithm())
		cert          = x509.Certificate{
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(1, 0, 0),
			BasicConstraintsValid: true,
			IsCA:                  !isLeaf,
		}
		err error
	)

	cert.SignatureAlgorithm = pkeyAlgorithm.toSignatureAlgorithm()
	cert.SerialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	if config.Subject != nil {
		cert.Subject = config.Subject.ToStdLib()
	} else {
		cert.Subject = pkix.Name{}
	}

	if cert.Subject.CommonName == "" {
		cert.Subject.CommonName = "oneshot"
		if !isLeaf {
			cert.Subject.CommonName += "-local-ca"
		}
	}

	if isLeaf {
		cert.KeyUsage |= x509.KeyUsageDigitalSignature
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth)
	}

	if config.NotBefore != "" {
		cert.NotBefore, err = parseTime(config.NotBefore)
		if err != nil {
			return nil, fmt.Errorf("failed to parse notBefore: %w", err)
		}
	} else {
		cert.NotBefore = time.Now()
	}

	if config.NotAfter != "" {
		cert.NotAfter, err = parseTime(config.NotAfter)
		if err != nil {
			return nil, fmt.Errorf("failed to parse notAfter: %w", err)
		}
	} else {
		cert.NotAfter = cert.NotBefore.AddDate(1, 0, 0)
	}

	return &cert, nil
}
