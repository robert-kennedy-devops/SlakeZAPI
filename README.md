# 🟢 WhatsApp SaaS API — Go

Plataforma SaaS para envio e recebimento de mensagens WhatsApp, construída em **Go** com **Clean Architecture**, integração real via **WhatsMeow** e suporte a múltiplos tenants.

---

## 📁 Estrutura do Projeto

```
whatsapp-saas/
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint — graceful shutdown via OS signals
├── internal/
│   ├── app/
│   │   └── app.go               # Bootstrap — injeção de dependências
│   ├── config/
│   │   └── config.go            # Variáveis de ambiente
│   ├── domain/
│   │   ├── entities.go          # Entidades e DTOs
│   │   ├── interfaces.go        # Contratos (repositórios + serviços)
│   │   └── errors.go            # Erros de domínio (sentinelas)
│   ├── usecase/
│   │   ├── auth.go              # Criação e validação de API Keys
│   │   ├── message.go           # Envio de mensagens + billing check
│   │   ├── whatsapp.go          # Conexão de sessão WhatsApp
│   │   └── webhook.go           # Registro de webhooks
│   ├── repository/
│   │   ├── postgres.go          # Pool de conexão PostgreSQL
│   │   ├── tenant.go            # CRUD de tenants
│   │   ├── apikey.go            # API Keys (hash SHA-256)
│   │   ├── message.go           # Persistência de mensagens
│   │   └── billing.go           # Subscriptions + usage
│   ├── transport/
│   │   ├── http/
│   │   │   ├── router.go        # Registro de rotas + middleware chain
│   │   │   └── handler/
│   │   │       ├── auth.go      # POST /auth/apikey
│   │   │       ├── message.go   # POST /messages/send
│   │   │       ├── whatsapp.go  # POST /whatsapp/connect
│   │   │       └── misc.go      # POST /webhook  GET /health
│   │   └── ws/
│   │       └── hub.go           # WebSocket hub — fan-out por tenant
│   ├── middleware/
│   │   ├── auth.go              # Validação de Bearer token
│   │   └── http.go              # RequestID, Logging, RateLimit, Recover
│   ├── whatsapp/
│   │   └── manager.go           # Gerenciador de sessões WhatsMeow
│   ├── billing/
│   │   └── service.go           # Checagem de limites de plano
│   ├── events/
│   │   └── bus.go               # Event bus in-memory (pub/sub por tenant)
│   ├── queue/
│   │   └── pool.go              # Worker pool com retry + dead-letter
│   └── webhook/
│       └── dispatcher.go        # Entrega de webhooks via HTTP POST + HMAC
├── pkg/
│   ├── logger/
│   │   └── logger.go            # Logger JSON estruturado
│   └── httputil/
│       └── response.go          # Helpers de resposta HTTP
├── migrations/
│   └── 001_initial.sql          # Schema completo + índices
├── docker/
│   ├── Dockerfile               # Multi-stage build (scratch)
│   └── docker-compose.yml       # API + PostgreSQL
├── .env.example
└── go.mod
```

---

## 🚀 Como Rodar

### Pré-requisitos

- Go 1.25+
- Docker & Docker Compose

### 1. Clonar e configurar

```bash
git clone https://github.com/seu-usuario/whatsapp-saas
cd whatsapp-saas

cp .env.example .env
# Edite o .env — altere API_KEY_SALT para um valor seguro!
```

### 2. Subir com Docker Compose

```bash
cd docker
docker compose up --build
```

A API estará disponível em `http://localhost:8080`.

### 3. Rodar localmente (sem Docker)

```bash
# Suba apenas o banco
docker compose up postgres -d

# Configure o .env e rode a API
go run ./cmd/api
```

---

## 🔌 Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Health check |
| `GET` | `/livez` | Liveness probe |
| `GET` | `/readyz` | Readiness probe com dependências |
| `GET` | `/metrics` | Métricas Prometheus |
| `POST` | `/auth/bootstrap` | Criar tenant, assinatura inicial e primeira API key |
| `GET` | `/auth/me` | Consultar resumo do tenant autenticado |
| `POST` | `/auth/apikey` | Criar nova API Key para o tenant |
| `GET` | `/auth/apikey` | Listar API Keys do tenant |
| `DELETE` | `/auth/apikey/{id}` | Revogar API Key |
| `POST` | `/whatsapp/connect` | Gerar QR Code para conexão |
| `GET` | `/whatsapp/status` | Status da sessão WhatsApp |
| `GET` | `/whatsapp/qr` | Página HTML com o QR code atual |
| `GET` | `/whatsapp/qr.png` | PNG do QR code atual |
| `POST` | `/whatsapp/disconnect` | Desconectar sessão WhatsApp |
| `POST` | `/whatsapp/logout` | Desparear a sessão do WhatsApp |
| `POST` | `/messages/send` | Enviar mensagem de texto |
| `POST` | `/messages/send-media` | Enviar imagem, video, audio ou documento por URL |
| `GET` | `/messages` | Listar mensagens do tenant |
| `GET` | `/messages/{id}` | Consultar uma mensagem específica |
| `GET` | `/messages/{id}/media` | Baixar o binário de uma mídia recebida |
| `POST` | `/webhook` | Registrar URL de webhook |
| `GET` | `/webhook` | Listar webhooks ativos |
| `DELETE` | `/webhook/{id}` | Desativar webhook |
| `GET` | `/usage` | Consultar uso mensal atual |
| `GET` | `/ws` | WebSocket — eventos em tempo real |

### Autenticação

Todas as rotas, exceto `/health` e `/auth/bootstrap`, requerem o header:

```
Authorization: Bearer <API_KEY>
```

---

## 📡 Exemplos de Uso

### Bootstrap inicial

```bash
curl -X POST http://localhost:8080/auth/bootstrap \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Tenant Demo",
    "email": "demo@example.com",
    "plan": "starter"
  }'
```

### Enviar mensagem

```bash
curl -X POST http://localhost:8080/messages/send \
  -H "Authorization: Bearer sua_api_key" \
  -H "Content-Type: application/json" \
  -d '{"phone": "5511999999999", "message": "Olá do SaaS!"}'
```

**Resposta:**
```json
{
  "message_id": "3EB0...",
  "status": "sent"
}
```

### Enviar mídia

```bash
curl -X POST http://localhost:8080/messages/send-media \
  -H "Authorization: Bearer sua_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999999999",
    "type": "image",
    "url": "https://example.com/banner.png",
    "caption": "Arquivo de teste"
  }'
```

### Conectar WhatsApp

```bash
curl -X POST http://localhost:8080/whatsapp/connect \
  -H "Authorization: Bearer sua_api_key"
```

**Resposta:**
```json
{
  "qr_code": "string-do-qr-real",
  "qr_png_url": "http://localhost:8080/whatsapp/qr.png",
  "qr_page_url": "http://localhost:8080/whatsapp/qr",
  "status": "connecting"
}
```

Para abrir o QR como imagem local:

```powershell
Invoke-WebRequest `
  -Uri "http://localhost:8080/whatsapp/qr.png" `
  -Headers @{ Authorization = "Bearer sua_api_key" } `
  -OutFile ".\whatsapp-qr.png"

Start-Process ".\whatsapp-qr.png"
```

### Registrar Webhook

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Authorization: Bearer sua_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://meuapp.com/hook",
    "events": ["message.received", "message.sent"]
  }'
```

### Baixar mídia inbound

```bash
curl -L http://localhost:8080/messages/msg_123/media \
  -H "Authorization: Bearer sua_api_key" \
  -o arquivo.bin
```

### Payload do Webhook

```json
{
  "id": "evt_123",
  "version": "v1",
  "type": "message.received",
  "tenant_id": "tenant_123",
  "timestamp": "2026-05-04T12:00:00Z",
  "payload": {
    "id": "wamid.HBgL...",
    "tenant_id": "tenant_123",
    "whatsapp_id": "wamid.HBgL...",
    "phone": "5511999999999",
    "body": "arquivo recebido",
    "type": "document",
    "mime_type": "application/pdf",
    "file_name": "contrato.pdf",
    "media_url": "https://mmg.whatsapp.net/...",
    "direct_path": "/v/t62.7118-24/...",
    "file_length": 182944,
    "direction": "inbound",
    "status": "delivered",
    "sent_at": "2026-05-04T12:00:00Z",
    "created_at": "2026-05-04T12:00:01Z"
  }
}
```

Headers enviados:

- `X-Webhook-Event`: tipo do evento
- `X-Webhook-Id`: id único do envelope
- `X-Webhook-Signature`: assinatura HMAC-SHA256 do body

### WebSocket — Eventos em Tempo Real

```javascript
const ws = new WebSocket('ws://localhost:8080/ws', [], {
  headers: { Authorization: 'Bearer sua_api_key' }
});

ws.onmessage = (e) => {
  const event = JSON.parse(e.data);
  console.log(event.type, event.payload);
  // event.type: "message.received" | "message.sent" | "message.status" | "connection.update"
};
```

---

## 📊 Planos de Billing

| Plano | Mensagens/mês | Preço | Webhook |
|-------|--------------|-------|---------|
| Starter | 1.000 | Grátis | ❌ |
| Growth | 10.000 | R$ ~145/mês | ✅ |
| Pro | 100.000 | R$ ~495/mês | ✅ |

Quando o limite é excedido, a API retorna `HTTP 402 Payment Required`.

---

## 🔐 Segurança

- **API Keys** são armazenadas apenas como hash SHA-256 (salt + key)
- A chave bruta é retornada **uma única vez** na criação
- **Webhooks** são assinados via HMAC-SHA256 no header `X-Webhook-Signature`
- Eventos de webhook são entregues em envelope versionado (`version: "v1"`) com `id` e `timestamp`
- **Rate limiting** por tenant (token bucket, configurável via `RATE_LIMIT_RPS`)

## 📈 Observabilidade

- `GET /metrics` expõe métricas Prometheus de requests HTTP, latência, inflight requests e estado básico dos componentes.
- `GET /readyz` valida readiness do banco e worker pool.
- `GET /livez` responde liveness simples para orquestradores.
- Logs de auditoria são emitidos para operações críticas como bootstrap, criação/revogação de API key, envio de mensagens, mudanças de sessão e webhooks.

---

## 🏗️ Arquitetura

O projeto segue **Clean Architecture** com separação rígida de camadas:

```
Transport (HTTP/WS)
      ↓
  Use Cases          ← toda regra de negócio está aqui
      ↓
  Domain             ← interfaces + entidades + erros
      ↓
  Repository         ← persistência (PostgreSQL)
      ↓
  Infrastructure     ← WhatsApp, EventBus, Queue, Webhook
```

**Regras:**
- Handlers **não contêm** lógica de negócio
- Use Cases **não conhecem** HTTP
- Domain **não depende** de nenhuma camada externa
- Injeção de dependência **manual** (sem frameworks)
- `context.Context` em **todas** as chamadas

---

## 🔧 Integração WhatsApp

O projeto já integra `whatsmeow` de forma real. As tabelas internas do `whatsmeow` são criadas automaticamente no mesmo PostgreSQL da aplicação durante o bootstrap do servidor.

Fluxo operacional:
- `POST /auth/bootstrap` cria tenant, assinatura inicial e primeira API key.
- `POST /whatsapp/connect` inicia o pareamento e retorna o QR real quando necessário.
- Eventos inbound, receipts e mudanças de conexão são persistidos e publicados para WebSocket e webhook.
- `GET /whatsapp/status` também expõe `last_event`, `last_error` e links do QR quando a sessão estiver em pareamento.

---

## 🧪 Variáveis de Ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `HTTP_PORT` | `8080` | Porta do servidor HTTP |
| `DATABASE_URL` | — | DSN do PostgreSQL (obrigatório) |
| `API_KEY_SALT` | `change-me` | Salt para hash de API Keys |
| `WHATSAPP_DATA_PATH` | `./data/whatsapp` | Diretório de sessões WhatsMeow |
| `WORKER_COUNT` | `10` | Workers do pool de tarefas |
| `QUEUE_BUFFER_SIZE` | `500` | Buffer da fila de jobs |
| `WEBHOOK_TIMEOUT` | `10s` | Timeout de entrega de webhook |
| `WEBHOOK_RETRIES` | `3` | Tentativas de retry (backoff exp.) |
| `RATE_LIMIT_RPS` | `10` | Requisições/segundo por tenant |
| `SHUTDOWN_TIMEOUT` | `10s` | Timeout para graceful shutdown |

---

## ✅ Testes

```bash
go test ./...
```

Para rodar os testes de integração com PostgreSQL real, defina `TEST_DATABASE_URL` ou reutilize `DATABASE_URL`.

---

## 📝 Licença

MIT — use, modifique e distribua livremente.
