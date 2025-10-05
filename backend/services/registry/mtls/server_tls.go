package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// ServerTLSConfig represents the mTLS configuration for the server
type ServerTLSConfig struct {
	CertFile    string
	KeyFile     string
	CAFile      string
	ClientAuth  tls.ClientAuthType
	MinTLS      uint16
	MaxTLS      uint16
}

// DefaultServerTLSConfig returns a default mTLS configuration
func DefaultServerTLSConfig() *ServerTLSConfig {
	return &ServerTLSConfig{
		ClientAuth: tls.RequireAndVerifyClientCert,
		MinTLS:     tls.VersionTLS12,
		MaxTLS:     tls.VersionTLS13,
	}
}

// LoadServerTLSConfig loads mTLS configuration from files
func LoadServerTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	config := DefaultServerTLSConfig()
	config.CertFile = certFile
	config.KeyFile = keyFile
	config.CAFile = caFile

	return config.ToTLSConfig()
}

// ToTLSConfig converts ServerTLSConfig to tls.Config
func (c *ServerTLSConfig) ToTLSConfig() (*tls.Config, error) {
	// Load server certificate
	serverCert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Load CA certificate for client verification
	caCert, err := ioutil.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   c.ClientAuth,
		ClientCAs:    caCertPool,
		MinVersion:   c.MinTLS,
		MaxVersion:   c.MaxTLS,
		// Additional security settings
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	return tlsConfig, nil
}

// CreateHTTPServer creates an HTTPS server with mTLS configuration
func CreateHTTPServer(addr string, handler http.Handler, tlsConfig *tls.Config) *http.Server {
	server := &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	return server
}

// ClientTLSConfig represents the mTLS configuration for the client
type ClientTLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// LoadClientTLSConfig loads client mTLS configuration
func LoadClientTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	// Load client certificate
	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate for server verification
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
	}

	return tlsConfig, nil
}

// CreateHTTPClient creates an HTTP client with mTLS configuration
func CreateHTTPClient(tlsConfig *tls.Config) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
	}
}

// ValidateCertificates validates that all required certificate files exist
func ValidateCertificates(certFile, keyFile, caFile string) error {
	files := map[string]string{
		"server certificate": certFile,
		"server key":         keyFile,
		"CA certificate":     caFile,
	}

	for name, file := range files {
		if file == "" {
			return fmt.Errorf("%s file path is empty", name)
		}

		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("%s file does not exist: %s", name, file)
		}

		// Check if file is readable
		if _, err := ioutil.ReadFile(file); err != nil {
			return fmt.Errorf("cannot read %s file %s: %w", name, file, err)
		}
	}

	return nil
}

// LogTLSInfo logs TLS configuration information
func LogTLSInfo(tlsConfig *tls.Config) {
	log.Printf("TLS Configuration:")
	log.Printf("  Min Version: %x", tlsConfig.MinVersion)
	log.Printf("  Max Version: %x", tlsConfig.MaxVersion)
	log.Printf("  Client Auth: %v", tlsConfig.ClientAuth)
	log.Printf("  Server Certificates: %d", len(tlsConfig.Certificates))
	
	if tlsConfig.ClientCAs != nil {
		log.Printf("  Client CAs loaded: %t", len(tlsConfig.ClientCAs.Subjects()) > 0)
	}
	
	if tlsConfig.RootCAs != nil {
		log.Printf("  Root CAs loaded: %t", len(tlsConfig.RootCAs.Subjects()) > 0)
	}
}

// ExtractClientCert extracts client certificate information from HTTP request
func ExtractClientCert(r *http.Request) (*x509.Certificate, error) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no client certificate found")
	}

	clientCert := r.TLS.PeerCertificates[0]
	return clientCert, nil
}

// GetClientCertSubject returns the subject of the client certificate
func GetClientCertSubject(r *http.Request) (string, error) {
	cert, err := ExtractClientCert(r)
	if err != nil {
		return "", err
	}

	return cert.Subject.String(), nil
}

// GetClientCertCommonName returns the common name of the client certificate
func GetClientCertCommonName(r *http.Request) (string, error) {
	cert, err := ExtractClientCert(r)
	if err != nil {
		return "", err
	}

	return cert.Subject.CommonName, nil
}

// Middleware for extracting client certificate information
func ClientCertMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client certificate info and add to request context
		if cert, err := ExtractClientCert(r); err == nil {
			// Add certificate info to request headers for downstream handlers
			w.Header().Set("X-Client-Cert-Subject", cert.Subject.String())
			w.Header().Set("X-Client-Cert-Issuer", cert.Issuer.String())
			w.Header().Set("X-Client-Cert-Serial", cert.SerialNumber.String())
		}

		next.ServeHTTP(w, r)
	})
}

// RequireClientCertMiddleware ensures client certificate is present
func RequireClientCertMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := ExtractClientCert(r); err != nil {
			http.Error(w, "Client certificate required", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}





