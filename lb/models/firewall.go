package models

import (
    "net"
    "strings"
)

var blockedIPs = map[string]bool{
	"10.20.30.40": true,
    "123.45.67.89": true,
}

var blockedNets []*net.IPNet


// Extract client IP from RemoteAddr or X-Forwarded-For
func GetClientIP(remoteAddr string, forwardedFor string) string {
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		// actual first client IP
        return strings.TrimSpace(parts[0])
	}

	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return ip
}

// Returns true if the IP is exactly in BlockedIPs
func IsIPBlocked(ip string) bool {
    return blockedIPs[ip]
}
