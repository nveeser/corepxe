package ignition

import (
	"bytes"
	"github.com/clarketm/json"
	"github.com/google/go-cmp/cmp"
	"net/http"
	"net/http/httptest"
	"os"
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
		return
	}
	got := w.Body.Bytes()

	want, err := os.ReadFile("./testdir/want.json")
	if err != nil {
		t.Errorf("ReadFile(want.json) got err: %s", err)
	}

	if diff := cmp.Diff(mustUnmarshal(t, want), mustUnmarshal(t, got)); diff != "" {
		t.Errorf("request got diff: -want/+got: %s", diff)
		var b bytes.Buffer
		if err := json.Indent(&b, got, "", "   "); err != nil {
			t.Errorf("json.Indent() got err: %s", err)
		}
		t.Logf("GOT:\n%s\n", b.String())
	}
}

func mustUnmarshal(t *testing.T, d []byte) map[string]any {
	t.Helper()
	m := make(map[string]any)
	if err := json.Unmarshal(d, &m); err != nil {
		t.Fatalf("json.Unmarshal() got err: %s", err)
	}
	return m
}
