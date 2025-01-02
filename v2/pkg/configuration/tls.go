package configuration

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"strings"
)

type PKIXName struct {
	CommonName         string                   `mapstructure:"commonName" yaml:"commonName"`
	Country            SingletonOrSlice[string] `mapstructure:"country" yaml:"country"`
	Organization       SingletonOrSlice[string] `mapstructure:"organization" yaml:"organization"`
	OrganizationalUnit SingletonOrSlice[string] `mapstructure:"organizationalUnit" yaml:"organizationalUnit"`
	Locality           SingletonOrSlice[string] `mapstructure:"locality" yaml:"locality"`
	Province           SingletonOrSlice[string] `mapstructure:"province" yaml:"province"`
	StreetAddress      SingletonOrSlice[string] `mapstructure:"streetAddress" yaml:"streetAddress"`
	PostalCode         SingletonOrSlice[string] `mapstructure:"postalCode" yaml:"postalCode"`
}

func (pn PKIXName) ToStdLib() pkix.Name {
	return pkix.Name{
		CommonName:         pn.CommonName,
		Country:            pn.Country,
		Organization:       pn.Organization,
		OrganizationalUnit: pn.OrganizationalUnit,
		Locality:           pn.Locality,
		Province:           pn.Province,
		StreetAddress:      pn.StreetAddress,
		PostalCode:         pn.PostalCode,
	}
}

type GeneratedCertificate struct {
	Enabled                 *bool     `mapstructure:"enabled" yaml:"enabled"`
	Subject                 *PKIXName `mapstructure:"subject" yaml:"subject"`
	NotBefore               string    `mapstructure:"notBefore" yaml:"notBefore"`
	NotAfter                string    `mapstructure:"notAfter" yaml:"notAfter"`
	SubjectAlternativeNames []string  `mapstructure:"subjectAlternativeNames" yaml:"subjectAlternativeNames"`
	PrivateKeyAlgorithm     string    `mapstructure:"privateKeyAlgorithm" yaml:"privateKeyAlgorithm"`
	GenerateAtStartup       bool      `mapstructure:"generateAtStartup" yaml:"generateAtStartup"`
	// Invalid if SigningMaterial is set to prevent misuse
	ExportCA *FileExport `mapstructure:"exportCA" yaml:"exportCA"`
	// Invalid if ExportCA is set to prevent misuse
	SigningMaterial *SigningMaterial `mapstructure:"signWith" yaml:"signWith"`
}

func (gc *GeneratedCertificate) Validate() error {
	if gc.ExportCA != nil && gc.SigningMaterial != nil {
		return errors.New("only one of exportCA or signWith can be specified")
	}

	return nil
}

type SigningMaterial struct {
	Certificate *PathOrContent `mapstructure:"certificate" yaml:"certificate"`
	PrivateKey  *PathOrContent `mapstructure:"key" yaml:"key"`
}

func (sm *SigningMaterial) Validate() error {
	if (!sm.Certificate.IsZero() && sm.PrivateKey.IsZero()) || (sm.Certificate.IsZero() && !sm.PrivateKey.IsZero()) {
		return errors.New("both certificate and private key must be specified when using signing material")
	}

	return nil
}

const defaultPrivatyeKeyAlgorithm = "ecdsa-p384"

func (gc *GeneratedCertificate) GetPrivateKeyAlgorithm() string {
	if gc.PrivateKeyAlgorithm == "" {
		return defaultPrivatyeKeyAlgorithm
	}
	return gc.PrivateKeyAlgorithm
}

type StaticOrGeneratedCertificate struct {
	Certificate          *PathOrContent        `mapstructure:"certificate" yaml:"certificate"`
	GeneratedCertificate *GeneratedCertificate `mapstructure:"generatedCertificate" yaml:"generatedCertificate"`
}

func (sogc *StaticOrGeneratedCertificate) Validate() error {
	if !sogc.Certificate.IsZero() && sogc.GeneratedCertificate != nil {
		return fmt.Errorf("cert: %+v, generatedCert: %+v", sogc.Certificate, sogc.GeneratedCertificate)
	}

	return nil
}

type MTLSMode string

const (
	MTLSModeDisabled MTLSMode = "disabled"
	MTLSModeRequest  MTLSMode = "request"
	MTLSModeRequire  MTLSMode = "require"
)

type CertPool struct {
	UseSystemRoots bool            `mapstructure:"useSystemRoots" yaml:"useSystemRoots"`
	Certs          []PathOrContent `mapstructure:"certs" yaml:"certs"`
}

type MTLS struct {
	Mode     MTLSMode  `mapstructure:"mode" yaml:"mode"`
	CertPool *CertPool `mapstructure:"certPool" yaml:"certPool"`
}

type TLS struct {
	Certificate *StaticOrGeneratedCertificate `mapstructure:"certificate" yaml:"certificate"`
	PrivateKey  *PathOrContent                `mapstructure:"privateKey" yaml:"privateKey"`
	MTLS        *MTLS                         `mapstructure:"mtls" yaml:"mtls"`
	MaxVersion  string                        `mapstructure:"maxVersion" yaml:"maxVersion"`
	MinVersion  string                        `mapstructure:"minVersion" yaml:"minVersion"`
}

const (
	defaultMaxVersion = tls.VersionTLS13
	defaultMinVersion = tls.VersionTLS12
)

func (t *TLS) GetMaxVersion() (uint16, error) {
	if t.MaxVersion == "" {
		return defaultMaxVersion, nil
	}

	return parseTLSVersion(t.MaxVersion)
}

func (t *TLS) GetMinVersion() (uint16, error) {
	if t.MinVersion == "" {
		maxV, err := t.GetMaxVersion()
		if err != nil {
			return 0, fmt.Errorf("invalid max version: %w", err)
		}

		if maxV == tls.VersionTLS13 {
			return defaultMinVersion, nil
		}

		return maxV, nil
	}

	return parseTLSVersion(t.MinVersion)
}

func (t *TLS) IsEnabled() bool {
	if t == nil {
		return false
	}
	x := t.Certificate != nil && (t.Certificate.Certificate != nil || t.Certificate.GeneratedCertificate != nil)
	if t.Certificate != nil && t.Certificate.GeneratedCertificate != nil && t.Certificate.GeneratedCertificate.Enabled != nil {
		x = *t.Certificate.GeneratedCertificate.Enabled
	}
	return x
}

func (t *TLS) Validate() error {
	if t == nil {
		return nil
	}

	if t.MTLS == nil && t.Certificate == nil && t.PrivateKey == nil {
		return nil
	}

	if (t.MTLS != nil && t.MTLS.Mode != MTLSModeDisabled) && t.Certificate == nil {
		return errors.New("mTLS cannot be used without a server certificate")
	}

	if !t.Certificate.Certificate.IsZero() && t.PrivateKey.IsZero() {
		return errors.New("private key must be specified when using a static certificate")
	}

	if t.Certificate.GeneratedCertificate != nil && !t.PrivateKey.IsZero() {
		return errors.New("private key cannot be specified when using a generated certificate")
	}

	maxV, err := t.GetMaxVersion()
	if err != nil {
		return fmt.Errorf("invalid max version: %w", err)
	}
	minV, err := t.GetMinVersion()
	if err != nil {
		return fmt.Errorf("invalid min version: %w", err)
	}
	if maxV < minV {
		return errors.New("max version cannot be less than min version")
	}

	return nil
}

func parseTLSVersion(s string) (uint16, error) {
	mv := strings.ToLower(s)
	mv = strings.ReplaceAll(mv, " ", "")
	mv = strings.TrimPrefix(mv, "tls")

	switch mv {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	}

	return 0, fmt.Errorf("invalid TLS version: %q", s)
}
