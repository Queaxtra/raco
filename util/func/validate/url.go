package validate

import (
	"net"
	"net/url"
	"strings"
)

// privateNets is parsed once at startup to avoid repeated ParseCIDR on every validation call.
var privateNets []*net.IPNet

func init() {
	cidrs := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"::1/128",
		"fe80::/10",
		"fc00::/7",
	}
	privateNets = make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		privateNets = append(privateNets, n)
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, n := range privateNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func isBlockedHost(hostname string) bool {
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || strings.HasPrefix(lowerHost, "localhost.") {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip != nil && isPrivateIP(ip) {
		return true
	}
	return false
}

func URL(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	if !strings.HasPrefix(rawURL, "https://") {
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

	return !isBlockedHost(hostname)
}

func WebSocketURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	if !strings.HasPrefix(rawURL, "ws://") && !strings.HasPrefix(rawURL, "wss://") {
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

	return !isBlockedHost(hostname)
}

func GRPCTarget(target string) bool {
	if target == "" {
		return false
	}
	if strings.Contains(target, "://") {
		return false
	}
	if strings.Contains(target, "/") {
		return false
	}

	host := target
	if strings.Contains(target, ":") {
		h, _, err := net.SplitHostPort(target)
		if err != nil {
			return false
		}
		host = h
	}

	if host == "" {
		return false
	}
	if strings.ContainsAny(host, " \t\n\r") {
		return false
	}

	return !isBlockedHost(host)
}
