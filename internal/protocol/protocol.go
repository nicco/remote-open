package protocol

import "encoding/json"

const (
	TypeOpenURL = "open-url"
)

type OpenURL struct {
	Type string `json:"type"`
	URL  string `json:"url"`
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
	default:
		return g, nil
	}
}
