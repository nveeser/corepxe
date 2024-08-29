package server

import (
	"fmt"
	"net/http"
	"net/url"
	"text/template"
)

const templateSuffxix = ".cfg.tmpl"

func NewIPXEHandler(configDir string) (http.Handler, error) {
	tmplSet, err := template.New("").ParseGlob(configDir + "*" + templateSuffxix)
	if err != nil {
		return nil, fmt.Errorf("error parsing template(s): %w", err)
	}
	return &ipxeHandler{
		tmplSet,
	}, nil
}

type ipxeHandler struct {
	tmplSet *template.Template
}

func (h *ipxeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	t := h.tmplSet.Lookup(name + templateSuffxix)
	if t == nil {
		http.Error(w, "invalid template name", http.StatusNotFound)
		return
	}
	images := &url.URL{
		Scheme: "http",
		Host:   r.Host,
		Path:   "images/coreos",
	}
	ignition := &url.URL{
		Scheme: "http",
		Host:   r.Host,
		Path:   "configs/coreos/standard",
	}
	data := &struct {
		ImageURL    string
		IgnitionURL string
		InstallDev  string
	}{
		ImageURL:    images.String(),
		IgnitionURL: ignition.String(),
		InstallDev:  "/dev/sda",
	}
	r.URL.Hostname()

	if err := t.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("error processing template: %s", err), http.StatusInternalServerError)
		return
	}
}
