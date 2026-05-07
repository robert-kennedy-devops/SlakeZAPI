# 🟢 WhatsApp SaaS API — Go

Plataforma SaaS para envio e recebimento de mensagens WhatsApp, construída em **Go** com **Clean Architecture**, integração real via **WhatsMeow**, suporte a múltiplos tenants e um frontend moderno em **Next.js** para operação do cliente final.

Estado atual do projeto:

- multi-instância por tenant
- dashboard web com autenticação própria
- campanhas imediatas e agendadas
- inbox operacional por conversa
- envio de texto, mídia, grupos, status e mensagens interativas
- webhooks, WebSocket, fila observável e docs OpenAPI/Postman

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
│   ├── Dockerfile               # Multi-stage build da API
│   └── docker-compose.yml       # API + PostgreSQL + frontend
├── web/
│   ├── src/app/                 # App Router (login, signup, dashboard)
│   ├── src/components/          # Dashboard e providers React Query
│   └── src/lib/                 # API client, auth local e tipos do frontend
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
make docker-up
```

Endpoints locais:

- Frontend: `http://localhost:3000`
- API: `http://localhost:8080`

Se a porta `5432` já estiver ocupada na máquina:

```bash
POSTGRES_PORT=5433 docker compose -f docker/docker-compose.yml up --build -d
```

### 3. Rodar localmente (sem Docker)

```bash
# Terminal 1: banco
make docker-db

# Terminal 2: API
make run

# Terminal 3: frontend
make web-dev
```

### 4. Frontend

O painel web usa login de usuário em `/app/auth/*`, seleção de tenant por header `X-Tenant-ID` e realtime via `/app/ws`.

- login: `POST /app/auth/login`
- signup: `POST /app/auth/signup`
- refresh: `POST /app/auth/refresh`
- perfil/sessão: `GET /app/auth/me`
- dashboard summary: `GET /app/tenant/summary`
- mensagens, inbox, fila, grupos, campanhas, webhooks, credenciais e sessão WhatsApp via `/app/*`
- para frontend e API em subdomínios diferentes, use cookie `SameSite=None` + `Secure`

### 5. Documentação de dev

Arquivos servidos pela própria API:

- OpenAPI: `GET /docs/openapi.yaml`
- Postman: `GET /docs/postman_collection.json`

---

## 🔌 Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Health check |
| `GET` | `/livez` | Liveness probe |
| `GET` | `/readyz` | Readiness probe com dependências |
| `GET` | `/metrics` | Métricas Prometheus |
| `GET` | `/docs/openapi.yaml` | Especificação OpenAPI servida pela API |
| `GET` | `/docs/postman_collection.json` | Coleção Postman servida pela API |
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
| `POST` | `/messages/send-bulk` | Enviar mensagem em massa para vários números |
| `POST` | `/messages/send-media` | Enviar imagem, video, audio ou documento por URL |
| `POST` | `/messages/send-interactive` | Enviar botões, lista ou enquete |
| `POST` | `/messages/send-group` | Enviar mensagem para um grupo |
| `POST` | `/status/post` | Publicar texto ou mídia no status |
| `POST` | `/contacts/resolve` | Reconhecer quais números existem no WhatsApp |
| `GET` | `/messages` | Listar mensagens do tenant |
| `GET` | `/conversations` | Listar conversas operacionais por instância |
| `POST` | `/conversations/{phone}` | Atualizar estado e nota de uma conversa |
| `GET` | `/groups` | Listar grupos da instância conectada |
| `GET` | `/messages/{id}` | Consultar uma mensagem específica |
| `GET` | `/messages/{id}/media` | Baixar o binário de uma mídia recebida |
| `POST` | `/webhook` | Registrar URL de webhook |
| `GET` | `/webhook` | Listar webhooks ativos |
| `DELETE` | `/webhook/{id}` | Desativar webhook |
| `GET` | `/usage` | Consultar uso mensal atual |
| `GET` | `/queue` | Snapshot da fila interna com jobs recentes |
| `GET` | `/ws` | WebSocket — eventos em tempo real |
| `GET` | `/instances` | Listar instancias do tenant |
| `POST` | `/instances` | Criar nova instancia WhatsApp |
| `GET` | `/campaigns` | Listar campanhas da instancia |
| `POST` | `/campaigns` | Criar campanha imediata ou agendada |
| `POST` | `/campaigns/{id}/run` | Disparar campanha manualmente |
| `POST` | `/app/auth/signup` | Criar usuário, tenant owner e sessão do app |
| `POST` | `/app/auth/login` | Autenticar usuário do dashboard |
| `POST` | `/app/auth/refresh` | Renovar access token usando refresh cookie |
| `GET` | `/app/auth/me` | Consultar usuário atual e memberships |
| `GET` | `/app/tenant/summary` | Resumo do tenant selecionado |
| `GET` | `/app/messages` | Listar mensagens usando sessão de usuário |
| `POST` | `/app/messages/send` | Enviar mensagem usando sessão de usuário |
| `POST` | `/app/messages/send-bulk` | Disparo em massa usando sessão de usuário |
| `POST` | `/app/messages/send-media` | Enviar mídia usando sessão de usuário |
| `POST` | `/app/messages/send-interactive` | Enviar botões, lista ou enquete no dashboard |
| `POST` | `/app/messages/send-group` | Enviar mensagem para grupo no dashboard |
| `POST` | `/app/status/post` | Publicar status usando sessão de usuário |
| `POST` | `/app/contacts/resolve` | Reconhecer contatos válidos para envio |
| `GET` | `/app/conversations` | Inbox operacional por conversa |
| `POST` | `/app/conversations/{phone}` | Atualizar estado/nota da conversa |
| `GET` | `/app/groups` | Listar grupos no dashboard |
| `GET` | `/app/webhooks` | Listar webhooks no dashboard |
| `POST` | `/app/webhooks` | Criar webhook no dashboard |
| `GET` | `/app/apikeys` | Listar API keys no dashboard |
| `POST` | `/app/apikeys` | Criar API key no dashboard |
| `GET` | `/app/ws` | WebSocket do dashboard |
| `GET` | `/app/instances` | Listar instancias no dashboard |
| `POST` | `/app/instances` | Criar instancia no dashboard |
| `GET` | `/app/campaigns` | Listar campanhas da instancia atual |
| `POST` | `/app/campaigns` | Criar campanha no dashboard |
| `POST` | `/app/campaigns/{id}/run` | Rodar campanha manualmente |
| `GET` | `/app/queue` | Fila observável no dashboard |

### Autenticação

Todas as rotas, exceto `/health` e `/auth/bootstrap`, requerem o header:

```
Authorization: Bearer <API_KEY>
```

As rotas de dashboard em `/app/*` aceitam:

```http
Authorization: Bearer <USER_SESSION_TOKEN>
X-Tenant-ID: <tenant_id>
X-Instance-ID: <instance_id>
```

Se `X-Instance-ID` nao for enviado, a API usa a instancia padrao do tenant.

Para WebSocket do navegador e QR/image endpoints do dashboard, o token também pode ser enviado via query string:

```text
?access_token=<token>&tenant_id=<tenant_id>
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

### Reconhecer contatos

```bash
curl -X POST http://localhost:8080/contacts/resolve \
  -H "Authorization: Bearer sua_api_key" \
  -H "Content-Type: application/json" \
  -d '{"phones":["5511999999999","+55 (11) 98888-8888"]}'
```

### Envio em massa

```bash
curl -X POST http://localhost:8080/messages/send-bulk \
  -H "Authorization: Bearer sua_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "phones": ["5511999999999", "5511888888888"],
    "message": "Campanha de teste"
  }'
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

### Enviar interação por botões

```bash
curl -X POST http://localhost:8080/messages/send-interactive \
  -H "Authorization: Bearer sua_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999999999",
    "type": "buttons",
    "body": "Escolha uma opção",
    "footer": "Atendimento",
    "buttons": [
      {"id":"sales","title":"Comercial"},
      {"id":"support","title":"Suporte"}
    ]
  }'
```

### Snapshot da fila

```bash
curl http://localhost:8080/queue \
  -H "Authorization: Bearer sua_api_key"
```

### Abrir docs da API

```bash
curl http://localhost:8080/docs/openapi.yaml
curl http://localhost:8080/docs/postman_collection.json
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
const ws = new WebSocket(
  'ws://localhost:8080/app/ws?access_token=sua_sessao&tenant_id=tenant_123'
);

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
- **CORS** configurável para o frontend via `CORS_ALLOWED_ORIGINS`

## 📈 Observabilidade

- `GET /metrics` expõe métricas Prometheus de requests HTTP, latência, inflight requests e estado básico dos componentes.
- `GET /readyz` valida readiness do banco e worker pool.
- `GET /livez` responde liveness simples para orquestradores.
- `GET /queue` e `GET /app/queue` expõem jobs recentes, retries e dead letters.
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
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Origins permitidas para o frontend |
| `USER_ACCESS_TOKEN_TTL` | `15m` | Duração do access token do dashboard |
| `USER_REFRESH_TOKEN_TTL` | `168h` | Duração do refresh token rotativo |
| `USER_SESSION_COOKIE_NAME` | `slakezapi_rt` | Nome do cookie `HttpOnly` da sessão web |
| `USER_SESSION_COOKIE_SECURE` | `false` | Use `true` em produção com HTTPS |
| `USER_SESSION_COOKIE_DOMAIN` | vazio | Domínio compartilhado do cookie, ex: `.example.com` |
| `USER_SESSION_COOKIE_SAMESITE` | `lax` | `lax`, `strict` ou `none` |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Base URL usada pelo frontend Next.js |

---

## ✅ Testes

```bash
go test ./...
cd web && npm run build
cd web && npx playwright install --with-deps chromium
cd web && npm run test:e2e
```

Validação mínima antes de publicar:

```bash
docker compose -f docker/docker-compose.yml config
POSTGRES_PORT=5433 docker compose -f docker/docker-compose.yml up --build -d
curl http://localhost:8080/health
curl -I http://localhost:3000
curl -I http://localhost:8080/docs/openapi.yaml
```

---

## 🖥️ Frontend

O frontend fica em `web/` e foi construído com:

- `Next.js` + `React` + `TypeScript`
- `Tailwind CSS`
- `TanStack Query`
- autenticação por sessão de usuário
- QR code, mensagens, inbox operacional, grupos, status, campanhas, webhooks, fila, API keys e uso mensal no mesmo dashboard

Fluxo principal:

- `/signup` cria usuário + workspace
- `/login` autentica a sessão do app
- o app renova sessão automaticamente com refresh cookie `HttpOnly`
- para setup cross-domain, configure `USER_SESSION_COOKIE_DOMAIN=.seu-dominio.com`, `USER_SESSION_COOKIE_SAMESITE=none` e HTTPS
- `/dashboard` consome as rotas `/app/*`
- realtime chega por `/app/ws`

Observabilidade:

- métricas HTTP em `/metrics`
- métricas de autenticação do dashboard em `slakezapi_auth_events_total{action,outcome}`

CI disponível em `.github/workflows/ci.yml` com validação de Go, build do frontend, Playwright e build das imagens Docker.

Com isso, a API continua apta para integrações via API key e o produto ganha uma camada SaaS pronta para operação humana.

Para rodar os testes de integração com PostgreSQL real, defina `TEST_DATABASE_URL` ou reutilize `DATABASE_URL`.

---

## 📝 Licença

MIT — use, modifique e distribua livremente.
