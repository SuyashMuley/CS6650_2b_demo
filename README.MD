# HW05: Product API

A small in-memory Product API built with Go and Gin, implementing the Product endpoints of `api.yaml`.

## Where things are

| What | Where |
|---|---|
| Server code | `src/product-service.go` |
| Dockerfile | `src/Dockerfile` |
| API spec | `api.yaml` (assignment handout; not in this repo) |
| Plan / progress | `plan.md` |
| Terraform infrastructure | `terraform/` |

## Endpoints

| Method and path | Success | Errors |
|---|---|---|
| `GET /products/{productId}` | 200 + Product JSON | 404, 500 |
| `POST /products/{productId}/details` | 204, no body | 400, 404, 500 |

Product JSON fields (all required): `product_id` (int ≥1), `sku` (1–100 chars), `manufacturer` (1–200 chars), `category_id` (int ≥1), `weight` (int ≥0), `some_other_id` (int ≥1).

Errors use `{"error": "...", "message": "...", "details": "..."}`.

### Behaviour notes

- Products are stored in memory (`map[int32]Product` guarded by a `sync.RWMutex`). Restarting the server resets all data.
- Products **1, 2 and 3 are seeded** at startup. POST only updates existing products; it never creates them.
- `GET` with a non-numeric or non-positive ID returns 404. `POST` with the same returns 400.
- `POST` returns 400 if the body's `product_id` differs from the path ID.
- To trigger a 500 on demand, send the header `X-Simulate-Error: true` on either endpoint.

## Run locally

```
cd src
go run product-service.go
```

The server listens on `localhost:8080`.

## Run with Docker

```
cd src
docker build -t product-api .
docker run --rm -p 8080:8080 product-api
```

## Examples for every response code

The outputs below were captured from the Docker container.

### 200 OK: get an existing product

```
curl -i localhost:8080/products/1
```
```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{"product_id":1,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}
```

### 204 No Content: add product details

```
curl -i -X POST localhost:8080/products/1/details \
  -H 'Content-Type: application/json' \
  -d '{"product_id":1,"sku":"NEW-1","manufacturer":"Acme","category_id":5,"weight":10,"some_other_id":2}'
```
```
HTTP/1.1 204 No Content
```

A following `curl localhost:8080/products/1` returns the updated product:
```
{"product_id":1,"sku":"NEW-1","manufacturer":"Acme","category_id":5,"weight":10,"some_other_id":2}
```

### 400 Bad Request: invalid input (POST)

Missing or invalid fields:
```
curl -i -X POST localhost:8080/products/1/details -H 'Content-Type: application/json' -d '{"sku":""}'
```
```
HTTP/1.1 400 Bad Request

{"error":"INVALID_INPUT","message":"The provided input data is invalid","details":"product_id is required and must be an integer >= 1"}
```

Malformed JSON:
```
curl -i -X POST localhost:8080/products/1/details -H 'Content-Type: application/json' -d '{bad'
```
```
HTTP/1.1 400 Bad Request

{"error":"INVALID_INPUT","message":"The provided input data is invalid","details":"Malformed JSON or wrong field types"}
```

Body `product_id` does not match the path:
```
curl -i -X POST localhost:8080/products/1/details -H 'Content-Type: application/json' \
  -d '{"product_id":2,"sku":"X","manufacturer":"Y","category_id":1,"weight":0,"some_other_id":1}'
```
```
HTTP/1.1 400 Bad Request

{"error":"INVALID_INPUT","message":"The provided input data is invalid","details":"product_id in body must match productId in path"}
```

Invalid path ID:
```
curl -i -X POST localhost:8080/products/0/details -H 'Content-Type: application/json' \
  -d '{"product_id":1,"sku":"NEW-1","manufacturer":"Acme","category_id":5,"weight":10,"some_other_id":2}'
```
```
HTTP/1.1 400 Bad Request

{"error":"INVALID_INPUT","message":"The provided input data is invalid","details":"Product ID must be a positive integer"}
```

### 404 Not Found: unknown product

GET:
```
curl -i localhost:8080/products/999
```
```
HTTP/1.1 404 Not Found

{"error":"NOT_FOUND","message":"Product not found","details":"No product with the given ID"}
```

GET with a non-numeric ID:
```
curl -i localhost:8080/products/abc
```
```
HTTP/1.1 404 Not Found

{"error":"NOT_FOUND","message":"Product not found","details":"Product ID must be a positive integer"}
```

POST:
```
curl -i -X POST localhost:8080/products/999/details -H 'Content-Type: application/json' \
  -d '{"product_id":999,"sku":"X","manufacturer":"Y","category_id":1,"weight":0,"some_other_id":1}'
```
```
HTTP/1.1 404 Not Found

{"error":"NOT_FOUND","message":"Product not found","details":"No product with the given ID"}
```

### 500 Internal Server Error: simulated

GET:
```
curl -i -H 'X-Simulate-Error: true' localhost:8080/products/1
```
```
HTTP/1.1 500 Internal Server Error

{"error":"INTERNAL_ERROR","message":"Internal server error","details":"simulated failure"}
```

POST:
```
curl -i -X POST -H 'X-Simulate-Error: true' -H 'Content-Type: application/json' \
  -d '{"product_id":1,"sku":"NEW-1","manufacturer":"Acme","category_id":5,"weight":10,"some_other_id":2}' \
  localhost:8080/products/1/details
```
```
HTTP/1.1 500 Internal Server Error

{"error":"INTERNAL_ERROR","message":"Internal server error","details":"simulated failure"}
```

## Deploying with Terraform

Terraform builds the Docker image from `src/`, pushes it to ECR, and runs it on ECS Fargate.

### Prerequisites
- [Terraform](https://developer.hashicorp.com/terraform/install), [Docker](https://docs.docker.com/get-docker/) (running) and the [AWS CLI](https://aws.amazon.com/cli/)
- AWS credentials. The config uses the existing `LabRole`, so it is meant for AWS Academy Learner Lab.

**macOS (Docker Desktop):** Terraform's Docker provider looks for `/var/run/docker.sock`, which Docker Desktop may not create. If `terraform plan` fails with `Cannot connect to the Docker daemon at unix:///var/run/docker.sock`, run this in the same terminal first:
```
export DOCKER_HOST=unix://$HOME/.docker/run/docker.sock
```
Alternatively, enable Docker Desktop → Settings → Advanced → "Allow the default Docker socket to be used".

### Prepare credentials
Get temporary credentials from Learner Lab, then:
```
aws configure
aws configure set aws_session_token <YOUR-TEMP-SESSION-TOKEN>
```
Credentials expire when the lab session ends, so repeat this each session.

### Apply infrastructure
```
cd terraform
terraform init
terraform apply -auto-approve
```

### Get the public IP
Run from the `terraform/` folder:
```
aws ec2 describe-network-interfaces \
--network-interface-ids $(
    aws ecs describe-tasks \
    --cluster $(terraform output -raw ecs_cluster_name) \
    --tasks $(
        aws ecs list-tasks \
        --cluster $(terraform output -raw ecs_cluster_name) \
        --service-name $(terraform output -raw ecs_service_name) \
        --query 'taskArns[0]' --output text
    ) \
    --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" \
    --output text
) \
--query 'NetworkInterfaces[0].Association.PublicIp' \
--output text
```

### Send requests
Use the same examples as above, replacing `localhost` with the IP:
```
curl -i http://<PUBLIC-IP-ADDRESS>:8080/products/1
```

### Clean up
```
terraform destroy -auto-approve
```

## Load testing

Tool: [Locust](https://locust.io), run from a laptop against the Product API on ECS Fargate (0.25 vCPU, 512 MB, one task, us-east-1).
Locustfile: `locust/locustfile.py`. Screenshots: `screenshots/`.

Install Locust first: `pip install locust` (a virtual environment is recommended).

"RPS at end of run" is the current requests/second shown in Locust's header when the run stopped, not an average over the run. Throughput peaked a little higher during some runs (for example about 4,000 RPS in the 500-user chart).

```
cd locust
# HttpUser (default)
locust -f locustfile.py -H http://<PUBLIC-IP>:8080
# FastHttpUser
LOCUST_CLIENT=fast locust -f locustfile.py -H http://<PUBLIC-IP>:8080
```
Optional: `LOCUST_WAIT=1` adds 0.1-0.5 s of think time between requests (default is back-to-back).

### Test design
- **Request mix (about 8:1:1):** `GET /products/{id}` on seeded products 1-3, `GET` on a missing ID (expected 404, counted as success), and `POST /products/{id}/details` (expected 204). A real store is read-heavy, so most traffic is reads.
- **Users:** 50 (baseline), 100, 200, 500, 1000. Each level ran about one minute. Spawn rate was 10/s for 50-500 users and 25/s for the 1000-user run.
- **No wait time:** every user fires back to back, so this measures the server's limit instead of simulating human pacing.

### Results (FastHttpUser)

| Users | Requests | RPS at end of run | Median | p95 | p99 | Max | Failures |
|---|---|---|---|---|---|---|---|
| 50 | 151,348 | 2,077 | 23 ms | 31 ms | 46 ms | 312 ms | 0 |
| 100 | 162,438 | 2,685 | 27 ms | 69 ms | 77 ms | 195 ms | 0 |
| 200 | 156,695 | 3,118 | 56 ms | 98 ms | 110 ms | 279 ms | 0 |
| 500 | 219,183 | 3,702 | 91 ms | 190 ms | 240 ms | 471 ms | 0 |
| 1000 | 205,750 | 4,021 | 180 ms | 370 ms | 480 ms | 1,102 ms | 0 |

Screenshots: `baseline_fasthttpuser.png`, `100user_fasthttp.png`, `fasthttp_200user.png`, `fasthttp_500user.png`, `fasthttp_1000user_25rampup.png`. Each has matching `*_chart.png` / `*_chart2.png` charts. The 1000-user charts show all runs on one timeline.

### Where the server struggled
- Nothing failed: 0 errors up to 1000 users. The server degraded through latency, not errors.
- The tail got worse first: going from 50 to 100 users, p95 more than doubled (31 to 69 ms) while the median barely moved.
- From 200 to 1000 users (5x), throughput rose only 29% (3,118 to 4,021 RPS) while the median went from 56 to 180 ms. Extra users just queue.
- **Cause: the Fargate task's CPU (0.25 vCPU).** CloudWatch `CPUUtilization` at 1-minute resolution reached 94-100% in every run from 50 users up (for example about 96% around 01:17, during the HttpUser 50-user run, and 99.7% during the 500-user run). Memory peaked at about 6% and the task count stayed at 1, so the task did not crash or restart (`ecs_memoryutilization.png`, `ecs_livetaskcount.png`).
- Note on `ecs_cpuutilization.png`: the console graph defaults to a 5-minute average and ends before the 1000-user run, so it shows only about 41% at its peak. The short, saturated runs are averaged with idle time. The 1-minute data is what shows the plateau.
- **Not the client:** during the 1000-user run the Locust process used about 33% of one core (Activity Monitor), so the load generator was not the limit.
- About 22 ms of every latency is the network round trip from the laptop to AWS.

### Tradeoffs
- **Most common operation:** reads (`GET /products/{id}`). A shopper browses far more than a merchant updates products.
- **Effect on the data structure:** a `map` gives O(1) reads. A `sync.RWMutex` lets many GETs run in parallel and serializes only the POSTs, which suits a read-heavy mix. A plain `Mutex` would serialize the reads too, and no lock at all would be a data race (Go maps can crash on concurrent writes).
- **Limits:** the data lives in one task's memory, so it is lost on restart and cannot be shared across several tasks. A scalable design needs a database and a cache.

### HttpUser vs FastHttpUser (50 users)

| Client | RPS at end of run | Median | p95 | Failures |
|---|---|---|---|---|
| HttpUser | 1,859 | 24 ms | 31 ms | 0 |
| FastHttpUser | 2,077 | 23 ms | 31 ms | 0 |

(`baseline_httpuser.png`, `baseline_fasthttpuser.png`)

FastHttpUser gave about 12% more throughput with the same latency. The difference is small because Locust users are closed-loop: each waits for a response before sending the next request, so throughput is about users divided by latency. Here latency is dominated by the network trip and a CPU-bound server, not by the client. `HttpUser` is built on the Python `requests` library, which does more work per call. `FastHttpUser` uses `geventhttpclient`, a lighter, faster HTTP client, so one Locust process can generate more load before its own CPU becomes the limit. We did not reach that limit at 50 users, so the gap here understates what FastHttpUser can do. The server's CPU limit mattered more than the choice of client.

## Observability

_TODO (Part V)_
