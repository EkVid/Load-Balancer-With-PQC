package config

const (
	// Load balancer & server settings
	StartServers            = true  // whether to start backend servers automatically
	NumServers              = 5     // initial number of backend servers
	StartingPort            = 8081  // port to start backend servers from
	LBPort                  = 8080  // load balancer port

	// Queue & worker settings
	LowPriorityQueueLength  = 200
	HighPriorityQueueLength = 100
	MaxQueueingTimeSeconds  = 60
	NumberOfWorkers         = 20    // number of goroutine workers to process the req queue

	// Request handling
	MaxConcurrentRequests   = 300
	RateLimitPerSecond      = 100
	ClientTimeoutSeconds    = 5     // how long a client waits for server response

	// Scaling & health
	ScaleCooldownSeconds    = 10
	ScaleDownPeriodSeconds  = 60
	HealthCheckPeriodSeconds = 3
	ServerStartingDelayMs   = 200

	// Exponential moving average coefficient for response time
	ExpoMovingAvgCoef       = 0.3
)
