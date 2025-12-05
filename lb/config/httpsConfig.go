package config

const (
	// Load Balancer HTTPS Configuration
	EnableHTTPS    = true  // enable HTTPS for load balancer
	EnablePQC      = false // enable Post-Quantum Cryptography (PQC) algorithms
	CertFile       = "./certs/server.crt"  // TLS certificate location
	KeyFile        = "./certs/server.key"  // TLS private key location
	HTTPSPort      = 8443  // HTTPS port 
	EnableCustomTLSConfig = false // enable custom secure TLS configuration
	
	// Redirect HTTP to HTTPS
	RedirectHTTPToHTTPS = true
	HTTPPort            = 8080  // HTTP port for redirect
)
