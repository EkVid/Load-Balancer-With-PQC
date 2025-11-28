package config

// BlockedIPs contains IPs that are blocked
var BlockedIPs = []string{
	"10.20.30.40",
    "123.45.67.89",
	"256.256.256.256",
}

// BlockedCIDRs contains CIDRS that are blocked
var BlockedCIDRs = []string{
	"123.45.67.88/24",
}

// PriorityIPs contains IPs that should automatically get high priority
var PriorityIPs = []string{
	"203.0.113.42",
	"203.0.113.99",
}

// PriorityCIDRs contains CIDRS that should automatically get high priority
var PriorityCIDRs = []string{
	"203.0.113.41/24",
}