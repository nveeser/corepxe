package ignition

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"github.com/nveeser/srvsrv/jsonwalk"
	"gopkg.in/yaml.v3"
	"iter"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var butaneFilePaths = []string{
	".local",
	".contents_local",
	".ssh_authorized_keys_local",
}

type Handler struct {
	ConfigRoot string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	osname := r.PathValue("osname")
	host := r.PathValue("name")

	log.Printf("RemoteHost: %s\n", r.RemoteAddr)

	butaneData, err := h.readHostYAML(osname, host)
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

func (h *Handler) readHostYAML(osname, host string) ([]byte, error) {
	if _, err := os.Stat(filepath.Join(h.ConfigRoot, osname)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("invalid osname[%s]: %w", osname, err)
		}
		return nil, fmt.Errorf("error reading source config ConfigRoot: %w", err)
	}
	hostMap, err := h.readYAMLObject(osname, filepath.Join(host, "host.yaml"))
	if err != nil {
		return nil, err
	}
	var files []string
	for p, v := range find(hostMap, "$.ignition.config.merge.*.local") {
		if local, ok := v.(string); ok {
			files = append(files, local)
		} else {
			log.Printf("Invalid invalid type %T at path %s", v, p)
		}
	}
	if err := deletePath(hostMap, "$.ignition.config.merge"); err != nil {
		return nil, err
	}
	updatePaths(hostMap, host)

	for _, file := range files {
		importMap, err := h.readYAMLObject(osname, file)
		if err != nil {
			return nil, err
		}
		updatePaths(importMap, filepath.Dir(file))
		if err := jsonwalk.Merge(hostMap, importMap); err != nil {
			return nil, fmt.Errorf("error merging %s: %w", file, err)
		}
	}
	return json.Marshal(hostMap)
}

func (h *Handler) readYAMLObject(osname, relpath string) (map[string]any, error) {
	yamlPath := filepath.Join(h.ConfigRoot, osname, relpath)
	data, err := os.ReadFile(yamlPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("invalid path: %s, %w", relpath, err)
	}
	yamlMap := make(map[string]any)
	if err := yaml.Unmarshal(data, &yamlMap); err != nil {
		return nil, fmt.Errorf("error Unmarshal yaml: %w", err)
	}
	return yamlMap, nil
}

func find(m jsonwalk.Mapping, path string) iter.Seq2[jsonwalk.Path, any] {
	matchPath := jsonwalk.ParsePath(path)
	return func(yield func(jsonwalk.Path, any) bool) {
		v := func(p jsonwalk.Path, _ jsonwalk.Mapping, v any) jsonwalk.Result {
			switch {
			case !p.Prefix(matchPath):
				return jsonwalk.Skip
			case p.Match(matchPath):
				if !yield(p, v) {
					return jsonwalk.Exit
				}
				return jsonwalk.Continue
			default:
				return jsonwalk.Continue
			}
		}
		jsonwalk.Walk(m, jsonwalk.NodeVisitor(v))
	}
}

func deletePath(m map[string]any, path string) error {
	jpath := jsonwalk.ParsePath(path)
	f := func(p jsonwalk.Path, m jsonwalk.Mapping, v any) jsonwalk.Result {
		switch {
		case !p.Prefix(jpath):
			return jsonwalk.Skip
		case jpath.Equal(p):
			delete(m, p.Key())
			return jsonwalk.Exit
		default:
			return jsonwalk.Continue
		}
	}
	jsonwalk.Walk(m, jsonwalk.NodeVisitor(f))
	return nil
}

func updatePaths(m map[string]any, prefix string) {
	updateFunc := func(p jsonwalk.Path, m jsonwalk.Mapping, v any) jsonwalk.Result {
		if matchesAny(butaneFilePaths, p) {
			fmt.Printf("Updating: %s\n", p)
			m[p.Key()] = filepath.Join(prefix, v.(string))
		} else {
			fmt.Printf("Ignore: %s\n", p)
		}
		return jsonwalk.Continue
	}
	jsonwalk.Walk(m, jsonwalk.ScalarVisitor(updateFunc))
}

func matchesAny(paths []string, p jsonwalk.Path) bool {
	curr := p.String()
	return slices.ContainsFunc(paths, func(s string) bool {
		return strings.HasSuffix(curr, s)
	})
}
