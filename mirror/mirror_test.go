package mirror

import (
	rand "math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// TODO test invalid remote URLS
func TestMirrorHandler(t *testing.T) {
	var called int
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 1<<10)
		if _, err := rand.Read(data); err != nil {
			t.Fatalf("error creating random data")
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
		called++
	}))
	remoteu, err := url.Parse(remote.URL)
	if err != nil {
		t.Fatalf("Error parsing url from servertest: %s", err)
	}

	imageDir := t.TempDir()
	mirror := ImageMirror{imageDir}

	asset := &urlAsset{
		remote: remoteu,
		rpath:  "foo/data",
	}
	{
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		mirror.ServeAsset(w, r, asset)
		if w.Code != http.StatusOK {
			t.Errorf("Status is not OK: %d", w.Code)
		}
	}
	mirrorFile := filepath.Join(imageDir, "/foo/data")
	_, err = os.Stat(mirrorFile)
	if err != nil {
		t.Errorf("stat(%s) got err %s", mirrorFile, err)
	}
	{
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		mirror.ServeAsset(w, r, asset)
		if w.Code != http.StatusOK {
			t.Errorf("Status is not OK: %d", w.Code)
		}
	}
	if called != 1 {
		t.Errorf("remote got called %d times wanted %d", called, 1)
	}
}
