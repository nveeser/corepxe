package ignition

import (
	"github.com/clarketm/json"
	"github.com/nveeser/corepxe/server"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIgnitionHandler(t *testing.T) {
	s := &server.IPXE{DataDir: "/home/nicholas/pxe-files/"}
	h, err := s.buildHandler()
	if err != nil {
		t.Fatalf("buildHandler returned an error")
	}

	r := httptest.NewRequest("GET", "/configs/coreos/flat", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("got status %d wanted 200", w.Code)
		t.Logf("Body:\n %s", w.Body.String())
	}
	m := make(map[string]any)
	err = json.Unmarshal(w.Body.Bytes(), &m)
	if err != nil {
		t.Errorf("json.Unmarshal() got err: %s", err)
	}
	t.Logf("Got: %+v", m)
}
