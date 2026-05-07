package handler

import (
	"embed"
	"fmt"
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

func (h *DocsHandler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>SlakeZAPI Docs</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #07111f;
      --panel: rgba(12, 24, 42, 0.86);
      --line: rgba(255,255,255,0.12);
      --text: #e8eef8;
      --muted: #9db1ca;
      --accent: #58d7ff;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: ui-sans-serif, system-ui, sans-serif;
      background:
        radial-gradient(circle at top right, rgba(88, 215, 255, 0.16), transparent 28%%),
        linear-gradient(180deg, #08101d 0%%, #050a13 100%%);
      color: var(--text);
      min-height: 100vh;
    }
    .wrap { max-width: 960px; margin: 0 auto; padding: 48px 20px 72px; }
    .hero { margin-bottom: 28px; }
    h1 { margin: 0 0 10px; font-size: 2.2rem; line-height: 1.1; }
    p { color: var(--muted); line-height: 1.6; }
    .grid { display: grid; gap: 18px; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); }
    .card {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 22px;
      padding: 20px;
      backdrop-filter: blur(18px);
    }
    .card h2 { margin: 0 0 8px; font-size: 1rem; }
    .card a {
      display: inline-block;
      margin-top: 12px;
      color: var(--accent);
      text-decoration: none;
      font-weight: 600;
    }
    code {
      display: block;
      margin-top: 12px;
      padding: 12px 14px;
      border-radius: 14px;
      background: rgba(255,255,255,0.04);
      border: 1px solid var(--line);
      color: #d8e7ff;
      overflow-x: auto;
    }
  </style>
</head>
<body>
  <main class="wrap">
    <section class="hero">
      <h1>SlakeZAPI Developer Docs</h1>
      <p>Contrato vivo da API para dashboard, automações e integrações máquina-a-máquina.</p>
    </section>
    <section class="grid">
      <article class="card">
        <h2>OpenAPI</h2>
        <p>Especificação principal em YAML para geração de clientes, inspeção e importação em ferramentas.</p>
        <a href="/docs/openapi.yaml">Abrir /docs/openapi.yaml</a>
      </article>
      <article class="card">
        <h2>Postman</h2>
        <p>Coleção pronta com autenticação, instâncias, mensagens, fila, auditoria e webhooks.</p>
        <a href="/docs/postman_collection.json">Abrir /docs/postman_collection.json</a>
      </article>
      <article class="card">
        <h2>Quick Start</h2>
        <p>Bootstrap inicial e primeira chamada autenticada.</p>
        <code>curl -X POST /auth/bootstrap
curl -H "Authorization: Bearer &lt;API_KEY&gt;" /auth/me</code>
      </article>
    </section>
  </main>
</body>
</html>`)
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
