.PHONY: build run test test-verbose migrate migrate-fresh docker-build docker-run k8s-deploy clean fmt


build:
	go build -o bin/prherder ./cmd/prherder

run:
	go run cmd/prherder/main.go


test:
	go test ./...


test-verbose:
	go test ./... -v


migrate:
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql $(DATABASE_URL) -f $$f; \
	done


migrate-fresh:
	docker run -d --name prherder-pg \
		-e POSTGRES_USER=prherder -e POSTGRES_PASSWORD=prherder -e POSTGRES_DB=prherder \
		-p 5432:5432 postgres:16
	sleep 5
	$(MAKE) migrate DATABASE_URL="postgres://prherder:prherder@localhost:5432/prherder?sslmode=disable"


docker-build:
	docker build -t prherder:latest .
docker-run:
	docker run --rm --env-file .env \
		--add-host=host.docker.internal:host-gateway \
		-e DATABASE_URL="postgres://prherder:prherder@host.docker.internal:5432/prherder?sslmode=disable" \
		-e OLLAMA_URL="http://host.docker.internal:11434" \
		-p 8090:8090 \
		prherder:latest


k8s-deploy:
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/postgres.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml


fmt:
	gofmt -w .


clean:
	rm -rf bin/
