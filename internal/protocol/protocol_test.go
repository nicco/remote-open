package protocol

import (
	"encoding/json"
	"testing"
)

func TestOpenURLMarshal(t *testing.T) {
	m := OpenURL{Type: "open-url", URL: "http://localhost:3000/path"}
	data, _ := json.Marshal(m)
	expected := `{"type":"open-url","url":"http://localhost:3000/path"}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", data, expected)
	}
}

func TestOpenURLUnmarshal(t *testing.T) {
	m, _ := Unmarshal([]byte(`{"type":"open-url","url":"https://github.com"}`))
	o := m.(OpenURL)
	if o.URL != "https://github.com" {
		t.Errorf("URL = %q", o.URL)
	}
}

func TestStartProxyMarshal(t *testing.T) {
	data, _ := json.Marshal(StartProxy{Type: "start-proxy", Port: 3000})
	expected := `{"type":"start-proxy","port":3000}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", data, expected)
	}
}

func TestProxyStartedUnmarshal(t *testing.T) {
	m, _ := Unmarshal([]byte(`{"type":"proxy-started","port":3000,"proxy_port":55497}`))
	p := m.(ProxyStarted)
	if p.ProxyPort != 55497 || p.Port != 3000 {
		t.Errorf("got port=%d proxy=%d", p.Port, p.ProxyPort)
	}
}

func TestRoundTrip(t *testing.T) {
	orig := StartProxy{Type: "start-proxy", Port: 8080}
	data, _ := json.Marshal(orig)
	m, _ := Unmarshal(data)
	o := m.(StartProxy)
	if o.Port != orig.Port {
		t.Errorf("round trip: %d != %d", o.Port, orig.Port)
	}
}
