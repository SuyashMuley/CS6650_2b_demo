# HW05 Plan: Product API, Terraform/ECS, Load Testing, Observability

## Spec summary (Product section of `api.yaml`)

| Endpoint | Success | Errors |
|---|---|---|
| `GET /products/{productId}` | 200 + `Product` body | 404, 500 |
| `POST /products/{productId}/details` | 204, no body | 400, 404, 500 |

- **Product** (all required): `product_id` (int32 ≥1), `sku` (string 1–100), `manufacturer` (string 1–200), `category_id` (int32 ≥1), `weight` (int32 ≥0), `some_other_id` (int32 ≥1).
- **Error**: `{error, message, details?}`.
- Ignore shopping-cart, warehouse and payment endpoints.

### Decisions (made; explain in the interview)
- [x] POST returns 404 for unknown products, so it does not create them. Decision: seed products 1-3 at startup.
- [x] Non-integer or ≤0 `productId` on GET: returns 404. On POST: returns 400.
- [x] Body `product_id` differs from path ID: rejected with 400.
- [x] Trigger a 500 on purpose: send header `X-Simulate-Error: true`.

---

## Part II: Product API (Go + Gin)

- [x] `go mod init` in the project folder
- [x] `product-service.go` with `map[int32]Product` guarded by `sync.RWMutex`
- [x] GET handler: 200 / 404 / 500
- [x] POST handler with full validation: 204 / 400 / 404 / 500
- [x] Error responses use the `Error` schema
- [x] `Dockerfile` (multi-stage), server listens on 8080
- [x] Test with `go run` using curl (done)
- [x] Test in Docker (image builds; GET 200/404, POST 204/400, and 500 header verified)
- [x] Trigger every code with `go run`: 200, 204, 400, 404, 500
- [x] Trigger every code again inside Docker (main codes verified; other 400/404 variants only tested under `go run`)
- [x] README: curl example and output for each response code (README.MD; Part IV and V sections still TODO)

## Part III: Terraform deploy to ECS

- [x] Fork the starter infrastructure repo
- [x] Read every file: `provider.tf`, `variables.tf`, `main.tf`, `output.tf`, `modules/`
- [x] In `src/`: delete `main.go`, add `product-service.go`, `go.mod`, `go.sum`
- [x] Replace `src/Dockerfile` with the Part II Dockerfile (name and location unchanged)
- [x] `docker build` from `src/` works (Dockerfile pins linux/amd64 for Fargate; image removed)
- [x] `container_port` in `terraform/variables.tf` matches the server port (8080)
- [x] `terraform init`, `plan`, `apply` (needs DOCKER_HOST export on macOS)
- [x] Confirm the deployed server responds (200/404/204/400/500 verified on ECS)
- [x] `.gitignore`: `*.tfstate*`, `.terraform/`, `*.tfvars`, `.env`, keys, binaries
- [x] README: deploy steps for a different machine (prerequisites, AWS credentials, commands, how to find the URL, how to call the API)
- [ ] Prepare to explain what each Terraform file does and what changed from the starter

## Part IV: Locust testing

- [ ] Reuse the HW3 locustfile against the deployed URL
- [ ] Run `HttpUser` vs `FastHttpUser`; research and document why they differ
- [ ] Design a stress test (user counts, spawn rate, request mix) and find where the server struggles
- [ ] Screenshot all runs, including at least one stress test
- [ ] Write answers: most common real-world operation (reads) and its effect on the data structure choice

## Part V: Observability

- [ ] Add `prometheus/client_golang`; expose `/metrics`
- [ ] Request counter (rate), error counter, duration histogram, labelled by endpoint and status
- [ ] Gin middleware: one JSON log line per request (endpoint, status, duration)
- [ ] `docker-compose.yml` with app, Prometheus (with config) and Grafana, local only
- [ ] Grafana dashboard: panels for rate, error %, p50/p95 latency
- [ ] Run Locust against the local stack; screenshot idle vs loaded; save a log excerpt from the same run
- [ ] Write 1–2 sentences on when to use the dashboard vs the logs

## Final deliverables (repo and README)

- [ ] Server code, Dockerfile and infrastructure locations documented
- [ ] Curl or Postman examples for every response code
- [ ] Deployment instructions
- [ ] Locust screenshots
- [ ] Grafana screenshot, log excerpt and dashboard-vs-logs note
- [ ] `.gitignore` in place, no secrets or state files committed
- [ ] Submit: Piazza post by Wed, git repo link by Monday 9am

## Interview prep

- [ ] Walk through the Product section of `api.yaml`
- [ ] Trace a request: Gin → handler → map → response
- [ ] Why a map, and what breaks under concurrency (mutex / RWMutex)
- [ ] How each response code was triggered
- [ ] What each Terraform file does and how they connect
- [ ] Locust test design, breaking point and why, HttpUser vs FastHttpUser
- [ ] Grafana dashboard and request logs
- [ ] Design a scalable backend for the full store in `api.yaml`
- [ ] "Terraform is declarative": what that means vs imperative, and why it helps
