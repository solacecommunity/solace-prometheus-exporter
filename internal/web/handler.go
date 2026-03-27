package web

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/index.html
var templateFS embed.FS

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
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
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
