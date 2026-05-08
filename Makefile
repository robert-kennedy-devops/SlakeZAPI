.PHONY: run build docker-up docker-down migrate lint tidy web-dev web-build web-e2e

# ── Local dev ──────────────────────────────────────────────
run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o ./api ./cmd/api

tidy:
	go mod tidy

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

web-e2e:
	cd web && npm run test:e2e

lint:
	go vet ./...

# ── Docker ──────────────────────────────────────────────────
docker-up:
	docker compose --env-file .env -f docker/docker-compose.yml up --build

docker-down:
	docker compose --env-file .env -f docker/docker-compose.yml down -v

docker-db:
	docker compose --env-file .env -f docker/docker-compose.yml up postgres -d

# ── Database ────────────────────────────────────────────────
migrate:
	for file in $$(ls migrations/*.sql | sort); do psql $${DATABASE_URL} -f $$file; done

# ── Seed (create a test tenant + starter subscription) ──────
seed:
	psql $${DATABASE_URL} -c "\
	  INSERT INTO tenants (id, name, email, active) \
	  VALUES ('tenant_test', 'Tenant Teste', 'teste@saas.com', true) \
	  ON CONFLICT DO NOTHING; \
	  INSERT INTO subscriptions (id, tenant_id, plan_id, status, period_end, updated_at) \
	  VALUES ('sub_test', 'tenant_test', 'plan_growth', 'active', NOW() + INTERVAL '30 days', NOW()) \
	  ON CONFLICT (tenant_id) DO UPDATE \
	    SET plan_id='plan_growth', status='active', \
	        period_end=NOW() + INTERVAL '30 days', updated_at=NOW();"

help:
	@echo ""
	@echo "  make run          Rodar API localmente"
	@echo "  make build        Compilar binário"
	@echo "  make web-dev      Rodar frontend Next.js"
	@echo "  make web-build    Buildar frontend Next.js"
	@echo "  make docker-up    Subir API + Postgres via Docker"
	@echo "  make docker-db    Subir apenas o Postgres"
	@echo "  make migrate      Rodar migrations SQL"
	@echo "  make seed         Inserir tenant de teste"
	@echo "  make tidy         go mod tidy"
	@echo "  make lint         go vet"
	@echo ""
