package main

import (
	"net/url"
	"strings"
)

// natsAddr turns nats://host:4222 into host:4222.
func natsAddr(u string) string {
	if p, err := url.Parse(u); err == nil && p.Host != "" {
		return p.Host
	}
	return strings.TrimPrefix(u, "nats://")
}

// originHosts converts origins (https://seller.example) to the host patterns websocket.Accept expects.
func originHosts(origins []string) []string {
	var out []string
	for _, o := range origins {
		if p, err := url.Parse(o); err == nil && p.Host != "" {
			out = append(out, p.Host)
		}
	}
	return out
}
