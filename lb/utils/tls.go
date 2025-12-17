package utils

import (
	"crypto/tls"
)

// TLS configuration for https library
func GetSecureTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		
		// Ciphers for data exchange
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		
		// Use ciphers by order of preference
		PreferServerCipherSuites: true,
		
		// Curve preferences for ECDHE (key exchange)
		CurvePreferences: []tls.CurveID{
            tls.X25519,    // Classical ECDH 
            tls.CurveP256, // NIST P-256
            tls.CurveP384, // NIST P-384
        },
	}
}