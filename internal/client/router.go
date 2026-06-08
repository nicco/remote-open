package client

import (
	"net/url"
	"strconv"
	"strings"
)

type ActionKind int

const (
	ActionExternal ActionKind = iota
	ActionTunnel
)

type RouteAction struct {
	Kind ActionKind
	URL  string
	Port int
}

func RouteURL(rawURL string) RouteAction {
	if isLocalhost(rawURL) {
		port, _ := extractPort(rawURL)
		return RouteAction{Kind: ActionTunnel, URL: rawURL, Port: port}
	}
	return RouteAction{Kind: ActionExternal, URL: rawURL}
}

func isLocalhost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.Split(u.Host, ":")[0]
	return host == "localhost" || host == "127.0.0.1"
}

func extractPort(rawURL string) (int, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, false
	}
	if strings.Contains(u.Host, ":") {
		parts := strings.Split(u.Host, ":")
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, false
		}
		return port, true
	}
	if u.Scheme == "https" {
		return 443, true
	}
	return 80, true
}
