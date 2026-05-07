package handler

import (
	"embed"
	"net/http"
	"path"
	"strings"
)

//go:embed apidocs/*
var apiDocsFS embed.FS

type DocsHandler struct{}

func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

func (h *DocsHandler) Serve(w http.ResponseWriter, r *http.Request) {
	name := path.Base(strings.TrimSpace(r.PathValue("name")))
	switch name {
	case "openapi.yaml":
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	case "postman_collection.json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	default:
		http.NotFound(w, r)
		return
	}

	data, err := apiDocsFS.ReadFile("apidocs/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
