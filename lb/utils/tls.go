package utils

import (
	"crypto/tls"
	"lb/config"
)

// GetSecureTLSConfig returns a secure TLS configuration (by net/http standards)
func GetSecureTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		
		// Cipher suites for data exchange (ordered by preference)
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},
		
		// Have server cipher suite ordered by preference
		PreferServerCipherSuites: true,
		
		// Curve preferences for ECDHE (key exchange)
		CurvePreferences: buildCurvePreferences(),
	}
}

func buildCurvePreferences() []tls.CurveID {
    var curves []tls.CurveID
    
    if config.EnablePQC {
        // USe PQC key exchange (ML-KEM-512)
		curves = append(curves, tls.CurveID(0x0200)) 
	}
    
    // Classical curves for ECDHE
    curves = append(curves,
        tls.X25519,    // Classical ECDH (most common)
        tls.CurveP256, // NIST P-256
        tls.CurveP384, // NIST P-384
    )
    
    return curves
}