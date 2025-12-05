// this file contains functions for the firewall system
package models

import (
    "net"
    "strings"
)


var blockedNets []*net.IPNet


// Extract client IP from RemoteAddr or X-Forwarded-For
func GetClientIP(remoteAddr string, forwardedFor string) string {
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		clientIP := strings.TrimSpace(parts[0])
		return clientIP
	}

	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return ip
}


// IsBlocked checks if an IP matches either a plain IP or a CIDR
func IsBlocked(ip string, blockedIPs []string, blockedCIDRs []string) bool {
	ip = strings.TrimSpace(ip)

	// Check exact IP matches first
	for _, b := range blockedIPs {
		if ip == strings.TrimSpace(b) {
			return true
		}
	}

	// Parse the client IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check CIDR ranges
	for _, cidr := range blockedCIDRs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// IsPriority works similarly for priority IPs
func IsPriority(ip string, priorityIPs []string, priorityCIDRs []string) bool {
	ip = strings.TrimSpace(ip)

	// Check exact IPs
	for _, p := range priorityIPs {
		if ip == strings.TrimSpace(p) {
			return true
		}
	}

	// Parse the client IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check CIDR ranges
	for _, cidr := range priorityCIDRs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil {
			continue
		}
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}