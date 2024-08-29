package ignition

import (
	"errors"
	"fmt"
	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
)

type Handler struct {
	ConfigRoot string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	osname := r.PathValue("osname")
	base := r.PathValue("name")

	d, err := httputil.DumpRequest(r, true)
	if err == nil {
		log.Printf("Ignition Request:\n %s\n", d)
	}

	osDir := filepath.Join(h.ConfigRoot, osname)
	if _, err := os.Stat(osDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, fmt.Sprintf("invalid osname: %s", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("error reading source config ConfigRoot: %s", err), http.StatusInternalServerError)
		return
	}
	butaneFile := filepath.Join(h.ConfigRoot, osname, base+".yaml")
	log.Printf("Butate[%s/%s]: %s", osname, base, butaneFile)
	bdata, err := os.ReadFile(butaneFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading file: %s", err), http.StatusInternalServerError)
		return
	}

	data, report, err := config.TranslateBytes(bdata, common.TranslateBytesOptions{
		TranslateOptions: common.TranslateOptions{
			FilesDir: osDir,
		},
		Pretty: false,
		Raw:    false,
	})
	if err != nil {
		log.Printf("Error during translate: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(report.String()))
		return
	}
	_, err = w.Write(data)
	if err != nil {
		log.Printf("Error writing Ignition: %s", err)
		http.Error(w, fmt.Sprintf("butane error file: %s", err), http.StatusInternalServerError)
		return
	}
}
