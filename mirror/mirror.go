package mirror

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type ImageAsset interface {
	RelativePath() string
	Download(dir string) error
}

type ImageMirror struct {
	RootDir string
}

func (h *ImageMirror) ServeAsset(w http.ResponseWriter, r *http.Request, asset ImageAsset) {
	localFile := filepath.Join(h.RootDir, asset.RelativePath())
	_, err := os.Stat(localFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		http.Error(w, fmt.Sprintf("Stat Error: %s", err), http.StatusInternalServerError)
		return
	}
	if err == nil {
		http.ServeFile(w, r, localFile)
		return
	}
	err = os.MkdirAll(filepath.Dir(localFile), 0755)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error Creating local dir: %s", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[Image] Fetching %s", asset.RelativePath())
	if err = asset.Download(h.RootDir); err != nil {
		http.Error(w, fmt.Sprintf("Remote Error: %s", err), http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, localFile)
}

type urlAsset struct {
	remote  *url.URL
	relpath string
}

func (a *urlAsset) RelativePath() string { return a.relpath }

func (a *urlAsset) Download(dir string) error {
	u := *a.remote
	var err error
	u.Path, err = url.JoinPath(u.Path, a.relpath)
	if err != nil {
		return err
	}
	localpath := filepath.Join(dir, a.relpath)
	log.Printf("Fetch %s -> %s", u.String(), localpath)

	resp, err := http.Get(u.String())
	if err != nil {
		log.Printf("Error getting Image: %s: %s", u.String(), err)
		return err
	}
	if err := errRemote(resp); err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(localpath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func errRemote(r *http.Response) error {
	if r.StatusCode == 200 {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error dumping response: %s", err)
	}
	if len(body) > 100 {
		body = body[:100]
	}
	return fmt.Errorf("error remote GET (%d): %s", r.StatusCode, body)
}
