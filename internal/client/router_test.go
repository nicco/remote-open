package client

import "testing"

func TestRouteURLLocalhost(t *testing.T) {
	tests := []struct {
		url      string
		external bool
		port     int
	}{
		{"http://localhost:3000/path", false, 3000},
		{"http://127.0.0.1:8080", false, 8080},
		{"http://localhost:9999/foo", false, 9999},
		{"http://localhost", false, 80},
		{"https://localhost", false, 443},
		{"https://github.com", true, 0},
		{"http://example.com:3000", true, 0},
		{"http://192.168.1.1:3000", true, 0},
	}

	for _, tc := range tests {
		action := RouteURL(tc.url)
		if tc.external {
			if action.Kind != ActionExternal || action.URL != tc.url {
				t.Errorf("RouteURL(%q): expected external, got %v", tc.url, action)
			}
		} else {
			if action.Kind != ActionTunnel || action.Port != tc.port {
				t.Errorf("RouteURL(%q): expected tunnel port %d, got %v",
					tc.url, tc.port, action)
			}
		}
	}
}
