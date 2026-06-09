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

func TestRoundTrip(t *testing.T) {
	orig := OpenURL{Type: "open-url", URL: "http://localhost:8000/docs"}
	data, _ := json.Marshal(orig)
	m, _ := Unmarshal(data)
	o := m.(OpenURL)
	if o.URL != orig.URL {
		t.Errorf("round trip: %q != %q", o.URL, orig.URL)
	}
}
