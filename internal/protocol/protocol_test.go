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

func TestPingMarshal(t *testing.T) {
	data, _ := json.Marshal(Ping{Type: "ping", Port: 3000})
	expected := `{"type":"ping","port":3000}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", string(data), expected)
	}
}

func TestPongMarshal(t *testing.T) {
	data, _ := json.Marshal(Pong{Type: "pong", Port: 3000, Alive: true})
	expected := `{"type":"pong","port":3000,"alive":true}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", string(data), expected)
	}
}

func TestPongUnmarshal(t *testing.T) {
	m, _ := Unmarshal([]byte(`{"type":"pong","port":3000,"alive":false}`))
	p := m.(Pong)
	if p.Alive {
		t.Error("expected Alive=false")
	}
}

func TestProxyDataMarshal(t *testing.T) {
	data, _ := json.Marshal(ProxyData{Type: "proxy-data", Port: 8080, Data: "aGVsbG8="})
	expected := `{"type":"proxy-data","port":8080,"data":"aGVsbG8="}`
	if string(data) != expected {
		t.Errorf("got %s, want %s", string(data), expected)
	}
}

func TestRoundTrip(t *testing.T) {
	orig := OpenURL{Type: "open-url", URL: "http://localhost:8000/docs"}
	data, _ := json.Marshal(orig)
	m, _ := Unmarshal(data)
	o := m.(OpenURL)
	if o.URL != orig.URL {
		t.Errorf("round trip: %q != %q", o.URL, orig.URL)
	}
}
