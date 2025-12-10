package config

const (
	// Load Balancer HTTPS Configuration
	EnableHTTPS    = true // enable HTTPS for load balancer
	CertFile       = "./certs/server.crt"  // TLS certificate location
	KeyFile        = "./certs/server.key"  // TLS private key location
	HTTPSPort      = 8443  // HTTPS port 
	EnableCustomTLSConfig = false // enable custom secure TLS configuration
	
	// Redirect HTTP to HTTPS
	RedirectHTTPToHTTPS = true
	HTTPPort            = 8080  // HTTP port for redirect

	// Custom TLS settings
	EnableCustomTLS         = true  // enable custom TLS endpoint
	CustomTLSPort           = 7443  // custom TLS listener port

)
