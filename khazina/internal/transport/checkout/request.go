package checkout

import (
	"fmt"
	"net"
	"net/http"
)

func ExtractPayerIP(r *http.Request) (string, error) {
	ip := r.RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		ip = host
	}
	if ip == "::1" {
		ip = "0000:0000:0000:0000:0000:0000:0000:0001"
	}
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid ip address: %s", ip)
	}
	return ip, nil
}
