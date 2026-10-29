# VIRA developer commands. Requires Docker (with Compose v2), Go 1.27, Node 22, Flutter (stable).
COMPOSE = docker compose --env-file .env -f deploy/compose/docker-compose.yml

.PHONY: setup dev-up dev-down dev-reset dev-logs dev-admin migrate-status test test-backend test-web test-mobile smoke contracts

setup:            ## create .env and local JWT keys (random values; never committed)
	./scripts/dev-setup.sh

dev-up: setup     ## build and start the full local stack, wait until healthy
	$(COMPOSE) up -d --build --wait

dev-down:         ## stop the stack (keeps data)
	$(COMPOSE) down

dev-reset:        ## stop the stack and delete local data volumes
	$(COMPOSE) down -v

dev-logs:
	$(COMPOSE) logs -f --tail=100

dev-admin:        ## create a local Super Admin: make dev-admin EMAIL=you@example.com (password from VIRA_NEW_ADMIN_PASSWORD)
	@test -n "$$VIRA_NEW_ADMIN_PASSWORD" || (echo "set VIRA_NEW_ADMIN_PASSWORD"; exit 2)
	$(COMPOSE) exec -T -e VIRA_NEW_ADMIN_PASSWORD="$$VIRA_NEW_ADMIN_PASSWORD" admin-api admin-api create-admin -email "$(EMAIL)" -name "Local Admin" -roles super_admin | tail -n1

migrate-status:
	$(COMPOSE) run --rm migrate migrate status

test: test-backend test-web test-mobile

test-backend:     ## needs a PostgreSQL superuser DSN in VIRA_TEST_PG_URL (the dev stack's works)
	cd backend && go vet ./... && go test -count=1 ./...

test-web:
	cd web && npm ci && npm run typecheck && npm run lint && npm test && npm run build

test-mobile:
	cd mobile && flutter pub get && flutter analyze && flutter test

smoke:            ## end-to-end checks against the running stack (see backend/scripts/smoke)
	cd backend && go run ./scripts/smoke

contracts:
	python3 scripts/gen_openapi.py
