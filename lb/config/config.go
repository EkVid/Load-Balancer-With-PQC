package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	StartServers            bool  `json:"start_servers"`
	NumServers              int   `json:"num_servers"`
	StartingPort            int   `json:"starting_port"`
	LBPort                  int   `json:"lb_port"`
	ScaleCooldownSeconds    int   `json:"scale_cooldown_seconds"`
	LowPriorityQueueLength  int   `json:"low_priority_queue_length"`
	HighPriorityQueueLength int   `json:"high_priority_queue_length"`
	MaxQueueingTimeSeconds  int   `json:"max_queueing_time_seconds"`
	MaxConcurrentRequests   int   `json:"max_concurrent_requests"`
	RateLimitPerSecond      int   `json:"rate_limit_per_second"`
	HealthCheckPeriodSeconds int  `json:"health_check_period_seconds"`
	ScaleDownPeriodSeconds  int   `json:"scale_down_period_seconds"`
	ServerStartingDelayMs   int   `json:"server_starting_delay_ms"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
