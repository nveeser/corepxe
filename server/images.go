package server

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
	"strings"
)

func NewImageHandler(rootDir string) (http.Handler, error) {
	localCache, err := NewCacheFileHandler(rootDir, coreosURL)
	if err != nil {
		return nil, err
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		relpath, err := ImagePath(r)
		if err != nil {
			fmt.Fprintf(w, "Invalid Request: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Printf("[Image] %s -> %s\n", r.URL.Path, relpath)
		rr := r.Clone(r.Context())
		rr.URL.Path = relpath
		localCache.ServeHTTP(w, rr)
	}
	return http.HandlerFunc(handler), nil
}

// URL /images/coreos/...
// FS  $BASEDIR/images/coreos/...

// https://builds.coreos.fedoraproject.org/browser?stream=stable&arch=x86_64
var (
	coreosURL = "https://builds.coreos.fedoraproject.org/prod"

	basePath = "streams/${STREAM}/builds/${VERSION}/${ARCH}/"
	kernel   = "fedora-coreos-${VERSION}-live-kernel-${ARCH}"
	rootfs   = "fedora-coreos-${VERSION}-live-rootfs.${ARCH}.img"
	initrd   = "fedora-coreos-${VERSION}-live-initramfs.${ARCH}.img"
)

func ImagePath(r *http.Request) (relpath string, error error) {
	q := r.URL.Query()
	switch r.PathValue("ostype") {
	case "coreos":
		relpath = basePath
	default:
		return "", fmt.Errorf("invalid OS type: %s", r.PathValue("ostype"))
	}

	switch r.PathValue("filetype") {
	case "kernel":
		relpath += kernel
	case "rootfs":
		relpath += rootfs
	case "initrd":
		relpath += initrd
	default:
		return "", fmt.Errorf("invalid path type: %s", r.PathValue("filetype"))
	}
	for _, k := range []string{"stream", "version", "arch"} {
		if !q.Has(k) {
			return "", fmt.Errorf("request is missing param: %s", k)
		}
		v := q.Get(k)
		kx := fmt.Sprintf("${%s}", strings.ToUpper(k))
		relpath = strings.ReplaceAll(relpath, kx, v)
	}
	return relpath, nil
}

func NewCacheFileHandler(rootDir string, baseURL string) (http.Handler, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %s: %w", baseURL, err)
	}
	return &cachedFile{
		root:   rootDir,
		remote: base,
	}, nil
}

type cachedFile struct {
	root   string
	remote *url.URL
}

func (h *cachedFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	relpath := r.URL.Path
	localFile := filepath.Join(h.root, relpath)
	_, err := os.Stat(localFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(w, "Stat Error: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err == nil {
		http.ServeFile(w, r, localFile)
		return
	}
	fmt.Printf("[Image] Fetching %s\n", relpath)
	if err = h.fetch(relpath); err != nil {
		fmt.Fprintf(w, "Remote Error: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, localFile)
}

func (h *cachedFile) fetch(relpath string) error {
	u := *h.remote
	var err error
	u.Path, err = url.JoinPath(u.Path, relpath)
	if err != nil {
		return err
	}
	localpath := filepath.Join(h.root, relpath)
	fmt.Printf("Fetch %s -> %s", u.String(), localpath)

	// Get the data
	resp, err := http.Get(u.String())
	if err != nil {
		log.Printf("Error getting Image: %s: %s", u.String(), err)
		return err
	}
	if err := errRemote(resp); err != nil {
		return err
	}
	defer resp.Body.Close()

	dir := filepath.Dir(localpath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
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
	return &errRemoteFetch{
		code:    r.StatusCode,
		headers: r.Header,
		body:    body,
	}
}

type errRemoteFetch struct {
	code    int
	headers http.Header
	body    []byte
}

func (e *errRemoteFetch) Error() string {
	body := e.body
	if len(e.body) > 100 {
		body = body[:100]
	}
	return fmt.Sprintf("error remote GET (%d): %s", e.code, body)
}
