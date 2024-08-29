package server

import (
	"fmt"
	"net/http"
	"text/template"
)

func NewIPXEHandler(configDir string) (http.Handler, error) {
	tmplSet, err := template.New("").ParseGlob(configDir + "*.cfg.tmpl")
	if err != nil {
		return nil, fmt.Errorf("error parsing template(s): %w", err)
	}
	return &ipxeConfigHandler{}
}

type ipxeConfigHandler struct {
	tmplSet *template.Template
}
