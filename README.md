# 🟢 WhatsApp SaaS API — Go

Plataforma SaaS para envio e recebimento de mensagens WhatsApp, construída em **Go** com **Clean Architecture**, integração real via **WhatsMeow**, suporte a múltiplos tenants e um frontend moderno em **Next.js** para operação do cliente final.

Estado atual do projeto:

- multi-instância por tenant
- dashboard web com autenticação própria
- campanhas imediatas e agendadas
- inbox operacional por conversa
- envio de texto, mídia, grupos, status e mensagens interativas
- **localização, cartão de contato, sticker, resposta citada, reação, edição e encaminhamento de mensagens**
- **gerenciamento completo de grupos** (criar, participantes, info, link de convite, sair)
- **operações de chat** (arquivar, silenciar, fixar, marcar como lido/não lido)
- **perfil e privacidade** (foto, descrição, visto por último, recibos, bloquear contatos)
- **pareamento por código de telefone** (sem QR) e reinício de instância
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
│   ├── src/components/          # Container do dashboard, módulos visuais e providers
│   │   └── dashboard/           # Módulos separados: overview, operations, automation e settings
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

- Índice HTML: `GET /docs`
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
| `GET` | `/docs` | Índice HTML da documentação embutida |
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
| `POST` | `/whatsapp/pair-phone` | Parear por código de telefone (sem QR) |
| `POST` | `/whatsapp/restart` | Reiniciar instância WhatsApp |
| `GET` | `/whatsapp/profile` | Obter perfil da instância conectada |
| `PATCH` | `/whatsapp/profile` | Atualizar descrição do perfil |
| `GET` | `/whatsapp/privacy` | Consultar configurações de privacidade |
| `PATCH` | `/whatsapp/privacy` | Atualizar configurações de privacidade |
| `POST` | `/messages/send` | Enviar mensagem de texto |
| `POST` | `/messages/send-bulk` | Enviar mensagem em massa para vários números |
| `POST` | `/messages/send-media` | Enviar imagem, video, audio ou documento por URL |
| `POST` | `/messages/send-interactive` | Enviar botões, lista ou enquete |
| `POST` | `/messages/send-group` | Enviar mensagem para um ou mais grupos |
| `POST` | `/messages/send-location` | Enviar localização geográfica |
| `POST` | `/messages/send-contact` | Enviar cartão de contato (vCard) |
| `POST` | `/messages/send-sticker` | Enviar sticker por URL |
| `POST` | `/messages/send-quoted` | Enviar mensagem com citação |
| `POST` | `/messages/react` | Reagir a uma mensagem com emoji |
| `POST` | `/messages/delete` | Apagar mensagem enviada |
| `POST` | `/messages/edit` | Editar texto de uma mensagem enviada |
| `POST` | `/messages/forward` | Encaminhar mensagem com flag de encaminhamento |
| `POST` | `/messages/star` | Marcar/desmarcar mensagem com estrela |
| `GET` | `/messages` | Listar mensagens do tenant |
| `GET` | `/messages/{id}` | Consultar uma mensagem específica |
| `GET` | `/messages/{id}/media` | Baixar o binário de uma mídia recebida |
| `POST` | `/status/post` | Publicar texto ou mídia no status |
| `POST` | `/contacts/resolve` | Reconhecer quais números existem no WhatsApp |
| `GET` | `/contacts` | Listar contatos importados da instância |
| `POST` | `/contacts/{phone}/block` | Bloquear contato |
| `DELETE` | `/contacts/{phone}/block` | Desbloquear contato |
| `GET` | `/contacts/{phone}/avatar` | Obter foto de perfil de um contato |
| `POST` | `/chats/{phone}/archive` | Arquivar ou desarquivar conversa |
| `POST` | `/chats/{phone}/mute` | Silenciar ou dessilenciar conversa |
| `POST` | `/chats/{phone}/pin` | Fixar ou desafixar conversa |
| `POST` | `/chats/{phone}/read` | Marcar conversa como lida ou não lida |
| `GET` | `/conversations` | Listar conversas operacionais por instância |
| `POST` | `/conversations/{phone}` | Atualizar estado e nota de uma conversa |
| `GET` | `/groups` | Listar grupos da instância conectada |
| `POST` | `/groups` | Criar novo grupo |
| `GET` | `/groups/{jid}` | Obter informações de um grupo |
| `PATCH` | `/groups/{jid}` | Editar nome ou descrição do grupo |
| `POST` | `/groups/{jid}/participants` | Adicionar, remover, promover ou rebaixar participantes |
| `GET` | `/groups/{jid}/invite` | Obter link de convite do grupo |
| `POST` | `/groups/{jid}/leave` | Sair do grupo |
| `POST` | `/webhook` | Registrar URL de webhook |
| `GET` | `/webhook` | Listar webhooks ativos |
| `DELETE` | `/webhook/{id}` | Desativar webhook |
| `GET` | `/webhook/deliveries` | Listar histórico recente de entregas |
| `POST` | `/webhook/deliveries/{id}/replay` | Reenfileirar uma entrega de webhook |
| `GET` | `/usage` | Consultar uso mensal atual |
| `GET` | `/queue` | Snapshot da fila interna com jobs recentes |
| `GET` | `/queue/dead-letters` | Listar jobs atualmente no DLQ |
| `POST` | `/queue/dead-letters/{id}/requeue` | Recolocar um job do DLQ na fila |
| `GET` | `/audit` | Listar trilha de auditoria persistida do tenant |
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
| `POST` | `/app/messages/send-location` | Enviar localização no dashboard |
| `POST` | `/app/messages/send-contact` | Enviar cartão de contato no dashboard |
| `POST` | `/app/messages/send-sticker` | Enviar sticker no dashboard |
| `POST` | `/app/messages/send-quoted` | Enviar resposta citada no dashboard |
| `POST` | `/app/messages/react` | Reagir a mensagem no dashboard |
| `POST` | `/app/messages/delete` | Apagar mensagem no dashboard |
| `POST` | `/app/messages/edit` | Editar mensagem no dashboard |
| `POST` | `/app/messages/forward` | Encaminhar mensagem no dashboard |
| `POST` | `/app/messages/star` | Estrelar/desestrellar mensagem no dashboard |
| `POST` | `/app/status/post` | Publicar status usando sessão de usuário |
| `POST` | `/app/contacts/resolve` | Reconhecer contatos válidos para envio |
| `GET` | `/app/contacts` | Listar contatos importados no dashboard |
| `POST` | `/app/contacts/{phone}/block` | Bloquear contato no dashboard |
| `DELETE` | `/app/contacts/{phone}/block` | Desbloquear contato no dashboard |
| `GET` | `/app/contacts/{phone}/avatar` | Foto de perfil de contato no dashboard |
| `POST` | `/app/chats/{phone}/archive` | Arquivar/desarquivar chat no dashboard |
| `POST` | `/app/chats/{phone}/mute` | Silenciar/dessilenciar chat no dashboard |
| `POST` | `/app/chats/{phone}/pin` | Fixar/desafixar chat no dashboard |
| `POST` | `/app/chats/{phone}/read` | Marcar chat como lido/não lido no dashboard |
| `GET` | `/app/conversations` | Inbox operacional por conversa |
| `POST` | `/app/conversations/{phone}` | Atualizar estado/nota da conversa |
| `GET` | `/app/groups` | Listar grupos no dashboard |
| `POST` | `/app/groups` | Criar grupo no dashboard |
| `GET` | `/app/groups/{jid}` | Obter info do grupo no dashboard |
| `PATCH` | `/app/groups/{jid}` | Editar grupo no dashboard |
| `POST` | `/app/groups/{jid}/participants` | Gerenciar participantes no dashboard |
| `GET` | `/app/groups/{jid}/invite` | Link de convite no dashboard |
| `POST` | `/app/groups/{jid}/leave` | Sair do grupo no dashboard |
| `GET` | `/app/whatsapp/profile` | Perfil da instância no dashboard |
| `PATCH` | `/app/whatsapp/profile` | Atualizar perfil no dashboard |
| `GET` | `/app/whatsapp/privacy` | Privacidade no dashboard |
| `PATCH` | `/app/whatsapp/privacy` | Atualizar privacidade no dashboard |
| `POST` | `/app/whatsapp/pair-phone` | Parear por código no dashboard |
| `POST` | `/app/whatsapp/restart` | Reiniciar instância no dashboard |
| `GET` | `/app/webhooks` | Listar webhooks no dashboard |
| `POST` | `/app/webhooks` | Criar webhook no dashboard |
| `DELETE` | `/app/webhooks/{id}` | Remover webhook no dashboard |
| `GET` | `/app/webhooks/deliveries` | Histórico de entregas no dashboard |
| `POST` | `/app/webhooks/deliveries/{id}/replay` | Replay manual de entrega |
| `GET` | `/app/apikeys` | Listar API keys no dashboard |
| `POST` | `/app/apikeys` | Criar API key no dashboard |
| `DELETE` | `/app/apikeys/{id}` | Revogar API key no dashboard |
| `GET` | `/app/ws` | WebSocket do dashboard |
| `GET` | `/app/instances` | Listar instancias no dashboard |
| `POST` | `/app/instances` | Criar instancia no dashboard |
| `GET` | `/app/campaigns` | Listar campanhas da instancia atual |
| `POST` | `/app/campaigns` | Criar campanha no dashboard |
| `POST` | `/app/campaigns/{id}/run` | Rodar campanha manualmente |
| `GET` | `/app/queue` | Fila observável no dashboard |
| `GET` | `/app/queue/dead-letters` | DLQ operacional no dashboard |
| `POST` | `/app/queue/dead-letters/{id}/requeue` | Requeue manual de um dead-letter |
| `GET` | `/app/audit` | Auditoria persistida do dashboard |
| `GET` | `/app/usage` | Uso mensal no dashboard |

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

O WebSocket do dashboard (`/app/ws`) e os endpoints de QR/imagem autenticam via cookie `HttpOnly` definido automaticamente pelo servidor no login — nenhum token precisa ser incluído na URL.

---

## 📡 Exemplos de Uso

### Bootstrap inicial

O endpoint exige o header `X-Bootstrap-Secret` com o valor definido em `BOOTSTRAP_SECRET`. Se a variável estiver vazia, o endpoint retorna `404`.

```bash
curl -X POST http://localhost:8080/auth/bootstrap \
  -H "Content-Type: application/json" \
  -H "X-Bootstrap-Secret: sua_chave_bootstrap" \
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

### Replay de webhook

```bash
curl -X POST http://localhost:8080/webhook/deliveries/delivery_123/replay \
  -H "Authorization: Bearer sua_api_key"
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

O token de sessão é enviado automaticamente via cookie `HttpOnly` — não inclua token na URL. O tenant é identificado pelo header `X-Tenant-ID`.

```javascript
// O cookie de sessão é enviado automaticamente pelo browser.
// Para clients não-browser (ex: Node.js), envie o header Authorization.
const ws = new WebSocket('ws://localhost:8080/app/ws');

ws.onmessage = (e) => {
  const event = JSON.parse(e.data);
  console.log(event.type, event.payload);
  // event.type: "message.received" | "message.sent" | "message.status" | "connection.update"
};
```

---

## 📊 Planos de Billing

Posicionamento sugerido para o produto:

- abaixo das plataformas enterprise mais caras
- acima das APIs mais cruas que vendem apenas instância
- simples o suficiente para venda consultiva curta ou autosserviço

| Plano | Oferta | Preço | Webhook |
|-------|--------|-------|---------|
| Trial | Todas as funcionalidades por 48h | R$ 0 | ✅ |
| Starter | 3.000 mensagens/mês | R$ 79/mês | ❌ |
| Growth | 15.000 mensagens/mês | R$ 149/mês | ✅ |
| Pro | 60.000 mensagens/mês | R$ 299/mês | ✅ |

Leitura comercial rápida:

- `Starter`: entrada competitiva para pequenas operações que precisam profissionalizar o WhatsApp
- `Growth`: plano principal para clientes com atendimento, campanhas e integração
- `Pro`: opção para operações com rotina intensa, múltiplos fluxos e maior previsibilidade
- `Trial`: degustação gratuita para testar a plataforma completa antes da ativação comercial

Tabela comercial final:

| Nome comercial | Código interno | Ideal para quem | Faixa de uso |
|-------|-------|-------|---------|
| Degustação | `trial` | quem quer validar o produto, testar integrações e apresentar a operação antes de contratar | prova rápida de valor |
| Essencial | `starter` | pequenos negócios, consultórios, operações locais e times iniciando estrutura comercial | validação e operação enxuta |
| Profissional | `growth` | empresas em crescimento com atendimento ativo, campanhas recorrentes e integrações | melhor custo-benefício |
| Escala | `pro` | operações com maior volume, múltiplos fluxos, rotinas intensas e necessidade de previsibilidade | crescimento com mais fôlego |

Quando o limite é excedido, a API retorna `HTTP 402 Payment Required`.

Billing autosserviço agora inclui:

- checkout para ativação do plano pago
- webhook de confirmação da assinatura
- upgrade/downgrade direto no dashboard
- portal de cobrança do cliente
- degustação gratuita de 2 dias com todos os recursos liberados

Variáveis para Stripe:

- `APP_BASE_URL`
- `STRIPE_SECRET_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `STRIPE_PRICE_STARTER_ID`
- `STRIPE_PRICE_GROWTH_ID`
- `STRIPE_PRICE_PRO_ID`

---

## 🔐 Segurança

- **API Keys** são armazenadas apenas como hash SHA-256 (salt + key); a chave bruta é retornada **uma única vez** na criação
- **Senhas de usuário** são armazenadas com `bcrypt` (DefaultCost)
- **Sessão do dashboard**: access token vive somente em memória no browser (nunca em `localStorage`); o refresh token é `HttpOnly` cookie. Em page reload a sessão é recuperada automaticamente via refresh cookie
- **Webhooks** são assinados via HMAC-SHA256 no header `X-Webhook-Signature`. URLs de webhook são validadas contra SSRF: apenas `http/https` e hosts públicos são aceitos (ranges RFC1918, loopback, link-local e AWS metadata bloqueados)
- **Bootstrap** (`POST /auth/bootstrap`) exige o header `X-Bootstrap-Secret: <valor>`. Se `BOOTSTRAP_SECRET` não estiver definido, o endpoint retorna `404` (desativado por padrão)
- **Rate limiting** por tenant (token bucket) em rotas de API — configurável via `RATE_LIMIT_RPS`. Rotas de autenticação (`login`, `signup`, `refresh`) têm rate limit adicional por IP — configurável via `AUTH_RATE_LIMIT_RPM`
- **`GET /metrics`**: bloqueado para IPs externos se `METRICS_TOKEN` não estiver definido; se definido, exige `Authorization: Bearer <token>`
- **`Content-Disposition`** é gerado via `mime.FormatMediaType` — nomes de arquivo inbound não podem injetar headers
- **Corpo das requisições** limitado a 1 MiB via `http.MaxBytesReader`; webhook responses limitadas a 1 MiB
- **`X-Request-ID`** do cliente é aceito apenas se for UUID canônico — outros valores são substituídos para prevenir log injection
- **Validação de e-mail** via `net/mail.ParseAddress` no signup e bootstrap
- **CORS** configurável via `CORS_ALLOWED_ORIGINS`

## 📈 Observabilidade

- `GET /metrics` expõe métricas Prometheus de requests HTTP, latência, inflight requests e estado básico dos componentes.
- `GET /readyz` valida readiness do banco e worker pool.
- `GET /livez` responde liveness simples para orquestradores.
- `GET /queue` e `GET /app/queue` expõem jobs recentes, retries e dead letters.
- `GET /queue/dead-letters` e `POST /queue/dead-letters/{id}/requeue` fecham o ciclo operacional do DLQ.
- Entregas de webhook ficam persistidas em histórico com status, tentativas, erro da última tentativa e replay manual.
- Logs de auditoria agora também ficam persistidos em `audit_logs`, com consulta por `/audit` e `/app/audit`.

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
| `BOOTSTRAP_SECRET` | vazio | Segredo do header `X-Bootstrap-Secret`. Vazio = endpoint desativado (retorna 404) |
| `RATE_LIMIT_RPS` | `10` | Requisições/segundo por tenant |
| `AUTH_RATE_LIMIT_RPM` | `20` | Requisições/minuto por IP em rotas de login/signup/refresh |
| `METRICS_TOKEN` | vazio | Token bearer para `GET /metrics`. Vazio = apenas loopback (127.0.0.1) permitido |
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
- envio avançado: localização, cartão de contato, sticker, resposta citada, reação, edição, encaminhamento e estrela
- gestão completa de grupos: criar, participantes, info, link de convite, sair
- operações de chat: arquivar, silenciar (com duração), fixar, marcar lido/não lido
- perfil e privacidade: ver dados da conta, atualizar descrição, configurar visibilidade e recibos, bloquear contatos
- pareamento por código de telefone (sem precisar escanear QR) e reinício de instância

Organização atual do dashboard:

- `dashboard-client.tsx` atua como container principal de estado, queries e mutations
- `src/components/dashboard/dashboard-header.tsx` concentra o topo executivo do workspace
- `src/components/dashboard/overview-module.tsx` exibe navegação e KPIs iniciais
- `src/components/dashboard/operations-module.tsx` agrupa conexão, inbox, campanhas e operação diária
- `src/components/dashboard/automation-module.tsx` separa recursos avançados e gestão de grupos
- `src/components/dashboard/settings-module.tsx` organiza conta, privacidade e ferramentas de manutenção
- `src/components/dashboard/shared.tsx` reúne componentes reutilizáveis como cards, painéis e estados vazios

Essa divisão mantém a lógica atual intacta, mas deixa a interface preparada para futuras evoluções por domínio sem voltar ao arquivo monolítico anterior.

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

Release automatizado disponível em `.github/workflows/release.yml`:

- push em `main` publica imagens em `ghcr.io/<owner>/slakezapi-api`
- tags `v*` também publicam imagem com `latest`
- usa `GITHUB_TOKEN` nativo do GitHub Actions, sem segredo extra

Com isso, a API continua apta para integrações via API key e o produto ganha uma camada SaaS pronta para operação humana.

Para rodar os testes de integração com PostgreSQL real, defina `TEST_DATABASE_URL` ou reutilize `DATABASE_URL`.

---

## 📝 Licença

MIT — use, modifique e distribua livremente.
