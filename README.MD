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

_TODO (Part IV)_

## Observability

_TODO (Part V)_
