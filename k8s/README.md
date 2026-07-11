# PR Herder — Kubernetes Deployment

## Prerequisites
- A cluster (local: `kind` or `minikube`; or any real cluster)
- `prherder:latest` image built and available to the cluster
  (for `kind`: `kind load docker-image prherder:latest`)

## Deploy

```bash
kubectl apply -f k8s/namespace.yaml

# Create secrets with real values (never commit real secrets to k8s/secret.yaml)
kubectl create secret generic prherder-secrets \
  --namespace=pr-herder \
  --from-literal=SLACK_BOT_TOKEN=xoxb-... \
  --from-literal=SLACK_SIGNING_SECRET=... \
  --from-literal=GITHUB_WEBHOOK_SECRET=... \
  --from-literal=GITHUB_READ_TOKEN=... \
  --from-literal=DATABASE_URL=postgres://prherder:prherder@postgres:5432/prherder?sslmode=disable

kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

## Verify

```bash
kubectl -n pr-herder get pods
kubectl -n pr-herder logs -f deployment/prherder
kubectl -n pr-herder port-forward svc/prherder 8090:80
curl http://localhost:8090/healthz
```

## Notes
- Postgres here is a StatefulSet for demo/local use only. In real
  production, point `DATABASE_URL` at a managed Postgres instance instead.
- Ollama is expected to run externally (not included here) -- point
  `OLLAMA_URL` in the ConfigMap at wherever it's reachable from the cluster.
- Migrations are not automated here -- run them manually against the
  target Postgres before first deploy (same `.sql` files under `migrations/`).
