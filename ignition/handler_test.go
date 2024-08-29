package ignition

import (
	"github.com/clarketm/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestIgnitionHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /configs/{osname}/{name}", &Handler{
		ConfigRoot: filepath.Join("./testdir"),
	})

	r := httptest.NewRequest("GET", "/configs/coreos/standard", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("got status %d wanted 200", w.Code)
		t.Logf("Body:\n %s", w.Body.String())
	}
	m := make(map[string]any)
	err := json.Unmarshal(w.Body.Bytes(), &m)
	if err != nil {
		t.Errorf("json.Unmarshal() got err: %s", err)
	}
	t.Logf("Got: %+v", m)
}
