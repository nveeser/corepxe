package coreos

import (
	"fmt"
	"github.com/coreos/stream-metadata-go/fedoracoreos"
	"github.com/coreos/stream-metadata-go/stream"
	"github.com/nveeser/corepxe/mirror"
	"log"
	"net/http"
)

type ImageHandler struct {
	Streams     *StreamCache
	ImageMirror interface {
		ServeAsset(w http.ResponseWriter, r *http.Request, asset mirror.ImageAsset)
	}
}

func (h *ImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for k, v := range r.Header {
		log.Printf("iPXE Header: %s => %q", k, v)
	}
	artifact, err := resolveCoreos(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid Request: %s\n", err), http.StatusBadRequest)
		return
	}

	name, err := artifact.Name()
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid Finding artifact name: %s\n", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[Image] %s -> %s", r.URL.String(), name)
	a := &coreosAsset{
		path:     name,
		artifact: artifact,
	}
	h.ImageMirror.ServeAsset(w, r, a)
	return
}

func resolveCoreos(r *http.Request) (artifact *stream.Artifact, err error) {
	q := r.URL.Query()
	param := func(k string) (string, error) {
		v, ok := q.Get(k), q.Has(k)
		if ok {
			return v, nil
		}
		v, ok = coreosDefaults[k]
		if ok {
			return v, nil
		}
		return "", fmt.Errorf("no value for %q", k)
	}

	streamName, err := param("stream")
	if err != nil {
		return nil, err
	}
	streamInfo, err := fedoracoreos.FetchStream(streamName)
	if err != nil {
		return nil, fmt.Errorf("error fetching coreos info: %w", err)
	}

	a, err := param("arch")
	if err != nil {
		return nil, err
	}
	arch, ok := streamInfo.Architectures[a]
	if !ok {
		return nil, fmt.Errorf("invalid architecture: %s", a)
	}
	art, ok := arch.Artifacts["metal"]
	if !ok {
		return nil, fmt.Errorf("invalid artifact: metal")
	}
	format, ok := art.Formats["pxe"]
	if !ok {
		return nil, fmt.Errorf("invalid format: pxe")
	}

	switch r.PathValue("filetype") {
	case "kernel":
		artifact = format.Kernel
	case "rootfs":
		artifact = format.Rootfs
	case "initrd":
		artifact = format.Initramfs
	default:
		return nil, fmt.Errorf("invalid path type: %s", r.PathValue("filetype"))
	}
	return artifact, nil
}

type coreosAsset struct {
	path     string
	artifact *stream.Artifact
}

func (a *coreosAsset) RelativePath() string { return a.path }
func (a *coreosAsset) Download(dir string) error {
	_, err := a.artifact.Download(dir)
	return err
}

// URL /images/coreos/...
// FS  $BASEDIR/images/coreos/...

// https://builds.coreos.fedoraproject.org/browser?stream=stable&arch=x86_64
var (
	coreosDefaults = map[string]string{
		"stream": "stable",
		"arch":   "x86_64",
	}
)
