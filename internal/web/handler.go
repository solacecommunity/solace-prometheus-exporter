package web

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type EndpointView struct {
	Path string
	Meta string
}

type TemplateData struct {
	IsHWBroker bool
	Endpoints  []EndpointView
}

type Handler struct {
	tmpl *template.Template
	data TemplateData
}

func NewHandler(data TemplateData) (*Handler, error) {
	path := filepath.Join("web", "templates", "index.html")
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return nil, err
	}

	return &Handler{
		tmpl: tmpl,
		data: data,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = h.tmpl.Execute(w, h.data)
}
