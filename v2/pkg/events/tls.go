package events

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"strconv"

	"github.com/forestnode-io/oneshot/v2/pkg/log"
)

var (
	cipherSuiteNames = map[uint16]string{}

	signatureSchemeNames = map[tls.SignatureScheme]string{
		tls.PKCS1WithSHA256: "PKCS1WithSHA256",
		tls.PKCS1WithSHA384: "PKCS1WithSHA384",
		tls.PKCS1WithSHA512: "PKCS1WithSHA512",

		tls.PSSWithSHA256: "PSSWithSHA256",
		tls.PSSWithSHA384: "PSSWithSHA384",
		tls.PSSWithSHA512: "PSSWithSHA512",

		tls.ECDSAWithP256AndSHA256: "ECDSAWithP256AndSHA256",
		tls.ECDSAWithP384AndSHA384: "ECDSAWithP384AndSHA384",
		tls.ECDSAWithP521AndSHA512: "ECDSAWithP521AndSHA512",

		tls.Ed25519: "Ed25519",

		tls.PKCS1WithSHA1: "PKCS1WithSHA1",
		tls.ECDSAWithSHA1: "ECDSAWithSHA1",
	}

	versionNames = map[uint16]string{
		tls.VersionSSL30: "SSL 3.0",
		tls.VersionTLS10: "TLS 1.0",
		tls.VersionTLS11: "TLS 1.1",
		tls.VersionTLS12: "TLS 1.2",
		tls.VersionTLS13: "TLS 1.3",
	}

	curveNames = map[tls.CurveID]string{
		tls.CurveP256: "P-256",
		tls.CurveP384: "P-384",
		tls.CurveP521: "P-521",
		tls.X25519:    "X25519",
		0x6399:        "X25519Kyber768Draft00",
	}

	keyUsageNames = map[x509.KeyUsage]string{
		x509.KeyUsageDigitalSignature:  "DigitalSignature",
		x509.KeyUsageContentCommitment: "ContentCommitment",
		x509.KeyUsageKeyEncipherment:   "KeyEncipherment",
		x509.KeyUsageDataEncipherment:  "DataEncipherment",
		x509.KeyUsageKeyAgreement:      "KeyAgreement",
		x509.KeyUsageCertSign:          "CertSign",
		x509.KeyUsageCRLSign:           "CRLSign",
		x509.KeyUsageEncipherOnly:      "EncipherOnly",
		x509.KeyUsageDecipherOnly:      "DecipherOnly",
	}

	extKeyUsageNames = map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageAny:                        "Any",
		x509.ExtKeyUsageServerAuth:                 "ServerAuth",
		x509.ExtKeyUsageClientAuth:                 "ClientAuth",
		x509.ExtKeyUsageCodeSigning:                "CodeSigning",
		x509.ExtKeyUsageEmailProtection:            "EmailProtection",
		x509.ExtKeyUsageIPSECEndSystem:             "IPSECEndSystem",
		x509.ExtKeyUsageIPSECTunnel:                "IPSECTunnel",
		x509.ExtKeyUsageIPSECUser:                  "IPSECUser",
		x509.ExtKeyUsageTimeStamping:               "TimeStamping",
		x509.ExtKeyUsageOCSPSigning:                "OCSPSigning",
		x509.ExtKeyUsageMicrosoftServerGatedCrypto: "MicrosoftServerGatedCrypto",
		x509.ExtKeyUsageNetscapeServerGatedCrypto:  "NetscapeServerGatedCrypto",
		x509.ExtKeyUsageMicrosoftKernelCodeSigning: "MicrosoftKernelCodeSigning",
	}
)

func init() {
	cipherSuites := tls.CipherSuites()
	for _, cs := range cipherSuites {
		cipherSuiteNames[cs.ID] = cs.Name
	}
}

type clientHelloKey struct{}

func withClientHelloRecorder(ctx context.Context) context.Context {
	var chr ClientHello
	return context.WithValue(ctx, clientHelloKey{}, &chr)
}

func getClientHello(ctx context.Context) *ClientHello {
	chr, _ := ctx.Value(clientHelloKey{}).(*ClientHello)
	return chr
}

func SetClientHello(ctx context.Context, chi *tls.ClientHelloInfo) {
	originalChr := getClientHello(ctx)
	if originalChr == nil {
		return
	}

	chr := ClientHello{
		ServerName:      chi.ServerName,
		SupportedProtos: chi.SupportedProtos,
	}

	for _, cs := range chi.CipherSuites {
		name := cipherSuiteNames[cs]
		if name != "" {
			chr.CipherSuites = append(chr.CipherSuites, name)
		} else {
			log.Logger().Warn().
				Uint16("cipher_suite", cs).
				Msg("unknown cipher suite")
		}
	}

	for _, curve := range chi.SupportedCurves {
		name := curveNames[curve]
		if name != "" {
			chr.SupportedCurves = append(chr.SupportedCurves, name)
		} else {
			log.Logger().Warn().
				Uint16("curve", uint16(curve)).
				Msg("unknown curve")
		}
	}

	for _, point := range chi.SupportedPoints {
		chr.SupportedPoints = append(chr.SupportedPoints, strconv.Itoa(int(point)))
	}

	for _, sig := range chi.SignatureSchemes {
		name := signatureSchemeNames[sig]
		if name != "" {
			chr.SignatureSchemes = append(chr.SignatureSchemes, name)
		} else {
			log.Logger().Warn().
				Uint16("signature_scheme", uint16(sig)).
				Msg("unknown signature scheme")
		}
	}

	for _, ver := range chi.SupportedVersions {
		name := versionNames[ver]
		if name != "" {
			chr.SupportVersions = append(chr.SupportVersions, name)
		} else {
			log.Logger().Warn().
				Uint16("version", ver).
				Msg("unknown version")
		}
	}

	*originalChr = chr
}

type ClientHello struct {
	CipherSuites     []string `json:",omitempty"`
	ServerName       string   `json:",omitempty"`
	SupportedCurves  []string `json:",omitempty"`
	SupportedPoints  []string `json:",omitempty"`
	SignatureSchemes []string `json:",omitempty"`
	SupportedProtos  []string `json:",omitempty"`
	SupportVersions  []string `json:",omitempty"`
}

type Certificate struct {
	SignatureAlgorithm string   `json:",omitempty"`
	PublicKeyAlgorithm string   `json:",omitempty"`
	Version            int      `json:",omitempty"`
	SerialNumber       string   `json:",omitempty"`
	Issuer             string   `json:",omitempty"`
	Subject            string   `json:",omitempty"`
	NotBefore          string   `json:",omitempty"`
	NotAfter           string   `json:",omitempty"`
	KeyUsage           string   `json:",omitempty"`
	ExtendedKeyUsage   []string `json:",omitempty"`
	SubjectAltNames    []string `json:",omitempty"`
}

func certificateFromStdLib(c *x509.Certificate) *Certificate {
	cert := Certificate{
		SignatureAlgorithm: c.SignatureAlgorithm.String(),
		PublicKeyAlgorithm: c.PublicKeyAlgorithm.String(),
		Version:            c.Version,
		SerialNumber:       c.SerialNumber.String(),
		Issuer:             c.Issuer.String(),
		Subject:            c.Subject.String(),
		NotBefore:          c.NotBefore.String(),
		NotAfter:           c.NotAfter.String(),
		KeyUsage:           keyUsageNames[c.KeyUsage],
	}

	for _, ku := range c.ExtKeyUsage {
		name := extKeyUsageNames[ku]
		if name != "" {
			cert.ExtendedKeyUsage = append(cert.ExtendedKeyUsage, name)
		} else {
			log.Logger().Warn().
				Int("ext_key_usage", int(ku)).
				Msg("unknown extended key usage")
		}
	}

	cert.SubjectAltNames = append(cert.SubjectAltNames, c.DNSNames...)
	cert.SubjectAltNames = append(cert.SubjectAltNames, c.EmailAddresses...)
	for _, ip := range c.IPAddresses {
		cert.SubjectAltNames = append(cert.SubjectAltNames, ip.String())
	}
	for _, uri := range c.URIs {
		cert.SubjectAltNames = append(cert.SubjectAltNames, uri.String())
	}

	return &cert
}

type TLS struct {
	Version            string       `json:",omitempty"`
	Resumed            bool         `json:",omitempty"`
	CipherSuite        string       `json:",omitempty"`
	NegotiatedProtocol string       `json:",omitempty"`
	ClientHello        *ClientHello `json:",omitempty"`
	PeerCertificates   []*Certificate
}
