package ignition

import (
	"errors"
	"fmt"
	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Handler struct {
	ConfigRoot string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	osname := r.PathValue("osname")
	host := r.PathValue("name")

	log.Printf("RemoteHost: %s\n", r.RemoteAddr)

	osDir := filepath.Join(h.ConfigRoot, osname)
	if _, err := os.Stat(osDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, fmt.Sprintf("invalid osname: %s", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("error reading source config ConfigRoot: %s", err), http.StatusInternalServerError)
		return
	}

	// /path/to/config/root
	//
	merge := &merge{
		base: osDir,
		pathKeys: []string{
			".local",
			".contents_local",
			".ssh_authorized_keys_local",
		},
	}
	butaneData, err := merge.Merge("base/base.yaml", filepath.Join(host, "host.yaml"))

	if err != nil {
		http.Error(w, fmt.Sprintf("error reading file: %s", err), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Has("debug") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(butaneData)
		if err != nil {
			log.Printf("Error writing Butane: %s", err)
		}
		return
	}

	data, report, err := config.TranslateBytes(butaneData, common.TranslateBytesOptions{
		TranslateOptions: common.TranslateOptions{
			FilesDir: osDir,
		},
	})
	if err != nil {
		log.Printf("Error during translate: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(report.String()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	if err != nil {
		log.Printf("Error writing Ignition: %s", err)
		return
	}
}

type merge struct {
	base      string
	config    map[string]any
	pathKeys  []string
	overwrite bool
	append    bool
}

func (m *merge) Merge(path ...string) ([]byte, error) {
	for _, f := range path {
		if err := m.mergeFile(f); err != nil {
			return nil, fmt.Errorf("file[%s]: %w", path, err)
		}
	}
	return yaml.Marshal(m.config)
}

func (m *merge) mergeFile(path string) error {
	d, err := os.ReadFile(filepath.Join(m.base, path))
	if err != nil {
		return fmt.Errorf("error file[%s]: %w", path, err)
	}

	config := map[string]any{}
	if err := yaml.Unmarshal(d, &config); err != nil {
		return fmt.Errorf("error reading yaml: %w", err)
	}

	m.resolvePathsObject(config, path, "$")

	if m.config == nil {
		m.config = config
		return nil
	}
	if err := m.mergeObjects(m.config, config, "$"); err != nil {
		return fmt.Errorf("error during merge[%s]: %w", path, err)
	}
	return nil
}

func (m *merge) resolvePathsObject(object map[string]any, relpath, ctxpath string) {
	for k, v := range object {
		cpath := ctxpath + "." + k
		if vv, ok := m.resolvePathsValue(v, relpath, cpath); ok {
			object[k] = vv
		}
	}
}

func (m *merge) resolvePathsValue(v any, relpath, ctxpath string) (any, bool) {
	log.Printf("File[%s] ContextPath: %s", relpath, ctxpath)
	switch v := v.(type) {
	case []any:
		var updated []any
		for _, vi := range v {
			if upv, ok := m.resolvePathsValue(vi, relpath, ctxpath); ok {
				updated = append(updated, upv)
			}
		}
		// only return true if all values in v were updated
		return updated, len(updated) == len(v)

	case map[string]any:
		m.resolvePathsObject(v, relpath, ctxpath)

	case string:
		if m.isRelativePath(ctxpath) {
			vv := filepath.Join(filepath.Dir(relpath), v)
			log.Printf("\t Update[%s] %s -> %s", ctxpath, v, vv)
			return vv, true
		}
	}
	return nil, false
}

func (m *merge) isRelativePath(contextPath string) bool {
	for _, key := range m.pathKeys {
		if strings.HasPrefix(key, ".") && strings.HasSuffix(contextPath, key) {
			return true
		}
		if key == contextPath {
			return true
		}
	}
	return false
}

func (m *merge) mergeObjects(dst, src map[string]any, path string) error {
	for key, sv := range src {
		cpath := path + "." + key
		switch sv := sv.(type) {
		case []any:
			dv, exists := dst[key]
			dvv, isSlice := dv.([]any) // if exists=false, then dv=nil and isSlice=false
			switch {
			case !exists:
				dst[key] = sv

			case exists && isSlice:
				if m.append {
					// If both are slices - copy from one slice to the other
					sv = append(dvv, sv...)
				}
				dst[key] = sv

			case exists && !isSlice:
				return fmt.Errorf("key[%s] mismatch: src(%T) vs dst(%T)", cpath, sv, dv)

			case exists && !m.overwrite:
				return fmt.Errorf("key[%s] duplicated (overrwrite=false)", cpath)
			}

		case map[string]any:
			dv, exists := dst[key]
			dvv, isMap := dv.(map[string]any) // if exists=false, then dv=nil and isMap=false
			switch {
			case !exists:
				// Dest Missing
				dv := make(map[string]any)
				dst[key] = dv
				err := m.mergeObjects(dv, sv, cpath)
				if err != nil {
					return err
				}
			case isMap:
				// Dest Merge
				err := m.mergeObjects(dvv, sv, cpath)
				if err != nil {
					return err
				}
			default:
				// Dest type mismatch
				return fmt.Errorf("key[%s] mismatch: src(%T) vs dst(%T)", cpath, sv, dv)
			}

		default:
			dv, ok := dst[key]
			switch {
			case ok && reflect.DeepEqual(sv, dv):
				continue
			case ok && !m.overwrite:
				return fmt.Errorf("duplicate Keys(overrwrite=false): %s", cpath)
			default:
				dst[key] = sv
			}
		}
	}
	return nil
}
