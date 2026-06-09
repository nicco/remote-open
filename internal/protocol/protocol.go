package protocol

import "encoding/json"

const (
	TypeOpenURL      = "open-url"
	TypeStartProxy   = "start-proxy"
	TypeProxyStarted = "proxy-started"
	TypeProxyStop    = "proxy-stop"
)

type OpenURL struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type StartProxy struct {
	Type string `json:"type"`
	Port int    `json:"port"`
}

type ProxyStarted struct {
	Type      string `json:"type"`
	Port      int    `json:"port"`
	ProxyPort int    `json:"proxy_port"`
}

type ProxyStop struct {
	Type      string `json:"type"`
	Port      int    `json:"port"`
	ProxyPort int    `json:"proxy_port"`
}

type generic struct{ Type string `json:"type"` }

func Unmarshal(data []byte) (interface{}, error) {
	var g generic
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, err
	}
	switch g.Type {
	case TypeOpenURL:
		var m OpenURL
		json.Unmarshal(data, &m)
		return m, nil
	case TypeStartProxy:
		var m StartProxy
		json.Unmarshal(data, &m)
		return m, nil
	case TypeProxyStarted:
		var m ProxyStarted
		json.Unmarshal(data, &m)
		return m, nil
	case TypeProxyStop:
		var m ProxyStop
		json.Unmarshal(data, &m)
		return m, nil
	default:
		return g, nil
	}
}
