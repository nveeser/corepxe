package ignition

import (
	"errors"
	"fmt"
	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"github.com/nveeser/butanex"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Handler struct {
	ConfigRoot string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	osname := r.PathValue("osname")
	host := r.PathValue("name")

	log.Printf("RemoteHost: %s\n", r.RemoteAddr)

	butaneData, err := h.butane(osname, host)
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Has("debug") {
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(butaneData)
		if err != nil {
			log.Printf("Error writing Butane: %s", err)
		}
		return
	}

	data, report, err := config.TranslateBytes(butaneData, common.TranslateBytesOptions{
		TranslateOptions: common.TranslateOptions{
			FilesDir: filepath.Join(h.ConfigRoot, osname),
		},
	})
	if err != nil {
		log.Printf("Error during translate: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(report.String()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		log.Printf("Error writing Ignition: %s", err)
	}
}

func (h *Handler) butane(osname, host string) ([]byte, error) {
	osDir := filepath.Join(h.ConfigRoot, osname)
	if _, err := os.Stat(osDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("invalid osname: %w", err)
		}
		return nil, fmt.Errorf("error reading source config ConfigRoot: %w", err)
	}

	// /path/to/config/root
	//
	merge := &butanex.Options{
		FilesDir: osDir,
		ResolvePath: []string{
			".local",
			".contents_local",
			".ssh_authorized_keys_local",
		},
	}
	butaneData, err := butanex.MergeFiles(merge, "base/base.yaml", filepath.Join(host, "host.yaml"))
	if err != nil {
		return nil, fmt.Errorf("merge error: %w", err)
	}
	return butaneData, nil
}
