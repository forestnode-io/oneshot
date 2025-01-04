package ssl

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
)

type PrivateKey interface {
	Public() crypto.PublicKey
}

type KeyType string

const (
	KeyTypeRSA2048  KeyType = "rsa-2048"
	KeyTypeRSA3072  KeyType = "rsa-3072"
	KeyTypeRSA7680  KeyType = "rsa-7680"
	KeyTypeECDSA224 KeyType = "ecdsa-p224"
	KeyTypeECDSA256 KeyType = "ecdsa-p256"
	KeyTypeECDSA384 KeyType = "ecdsa-p384"
	KeyTypeECDSA521 KeyType = "ecdsa-p521"
)

func (kt KeyType) GenerateKey(randReader io.Reader) (PrivateKey, error) {
	rr := randReader
	if rr == nil {
		rr = rand.Reader
	}

	switch kt {
	case KeyTypeRSA2048:
		return rsa.GenerateKey(rr, 2048)
	case KeyTypeRSA3072:
		return rsa.GenerateKey(rr, 3072)
	case KeyTypeRSA7680:
		return rsa.GenerateKey(rr, 7680)
	case KeyTypeECDSA224:
		return ecdsa.GenerateKey(elliptic.P224(), rr)
	case KeyTypeECDSA256:
		return ecdsa.GenerateKey(elliptic.P256(), rr)
	case KeyTypeECDSA384:
		return ecdsa.GenerateKey(elliptic.P384(), rr)
	case KeyTypeECDSA521:
		return ecdsa.GenerateKey(elliptic.P521(), rr)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", kt)
	}
}

func (kt KeyType) toSignatureAlgorithm() x509.SignatureAlgorithm {
	switch kt {
	case KeyTypeRSA2048, KeyTypeRSA3072, KeyTypeRSA7680:
		return x509.SHA256WithRSA
	case KeyTypeECDSA224, KeyTypeECDSA256, KeyTypeECDSA384, KeyTypeECDSA521:
		return x509.ECDSAWithSHA256
	default:
		return x509.UnknownSignatureAlgorithm
	}
}

func (kt KeyType) toPEMBlockType() string {
	switch kt {
	case KeyTypeRSA2048, KeyTypeRSA3072, KeyTypeRSA7680:
		return "RSA PRIVATE KEY"
	case KeyTypeECDSA224, KeyTypeECDSA256, KeyTypeECDSA384, KeyTypeECDSA521:
		return "EC PRIVATE KEY"
	default:
		return ""
	}
}
