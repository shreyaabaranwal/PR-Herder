.PHONY: build run test test-verbose migrate migrate-fresh docker-build docker-run k8s-deploy clean fmt

# Build the binary
build:
	go build -o bin/prherder ./cmd/prherder

# Run the server locally
run:
	go run cmd/prherder/main.go

# Run all tests
test:
	go test ./...

# Run all tests with verbose output
test-verbose:
	go test ./... -v

# Apply all migrations in order against the local Postgres instance
migrate:
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql $(DATABASE_URL) -f $$f; \
	done

# Start a fresh local Postgres container and apply migrations
migrate-fresh:
	docker run -d --name prherder-pg \
		-e POSTGRES_USER=prherder -e POSTGRES_PASSWORD=prherder -e POSTGRES_DB=prherder \
		-p 5432:5432 postgres:16
	sleep 5
	$(MAKE) migrate DATABASE_URL="postgres://prherder:prherder@localhost:5432/prherder?sslmode=disable"

# Build the Docker image
docker-build:
	docker build -t prherder:latest .

# Run the Docker image locally (expects Postgres + Ollama reachable via host.docker.internal)
docker-run:
	docker run --rm --env-file .env \
		--add-host=host.docker.internal:host-gateway \
		-e DATABASE_URL="postgres://prherder:prherder@host.docker.internal:5432/prherder?sslmode=disable" \
		-e OLLAMA_URL="http://host.docker.internal:11434" \
		-p 8090:8090 \
		prherder:latest

# Deploy to Kubernetes (expects prherder-secrets already created)
k8s-deploy:
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/postgres.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml

# Format all Go code
fmt:
	gofmt -w .

# Remove build artifacts
clean:
	rm -rf bin/
