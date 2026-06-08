package protocol

import "encoding/json"

const (
	TypeOpenURL   = "open-url"
	TypePing      = "ping"
	TypePong      = "pong"
	TypeProxyData = "proxy-data"
)

type OpenURL struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Ping struct {
	Type string `json:"type"`
	Port int    `json:"port"`
}

type Pong struct {
	Type  string `json:"type"`
	Port  int    `json:"port"`
	Alive bool   `json:"alive"`
}

type ProxyData struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Data string `json:"data"`
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
	case TypePing:
		var m Ping
		json.Unmarshal(data, &m)
		return m, nil
	case TypePong:
		var m Pong
		json.Unmarshal(data, &m)
		return m, nil
	case TypeProxyData:
		var m ProxyData
		json.Unmarshal(data, &m)
		return m, nil
	default:
		return g, nil
	}
}
