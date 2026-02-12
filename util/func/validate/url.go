package validate

import (
	"net"
	"net/url"
	"strings"
)

var privateCIDRs = []string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"::1/128",
	"fe80::/10",
	"fc00::/7",
}

func URL(rawURL string) bool {
	isEmpty := rawURL == ""
	if isEmpty {
		return false
	}

	hasHTTP := strings.HasPrefix(rawURL, "http://")
	if hasHTTP {
		return false
	}

	hasHTTPS := strings.HasPrefix(rawURL, "https://")
	if !hasHTTPS {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if parsed.Host == "" {
		return false
	}

	if parsed.User != nil {
		return false
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return false
	}

	if strings.ContainsAny(hostname, " \t\n\r") {
		return false
	}

	if strings.Contains(hostname, "..") {
		return false
	}

	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || strings.HasPrefix(lowerHost, "localhost.") {
		return false
	}

	ip := net.ParseIP(hostname)
	if ip != nil {
		for _, cidr := range privateCIDRs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if ipNet.Contains(ip) {
				return false
			}
		}
	}

	return true
}
