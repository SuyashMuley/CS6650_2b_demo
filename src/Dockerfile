# Target CPU architecture. Fargate (the Terraform deployment) runs amd64, so that is the default.
# For a native local build on Apple Silicon: docker compose build --build-arg TARGET_ARCH=arm64
ARG TARGET_ARCH=amd64

FROM golang:1.27-alpine AS build
ARG TARGET_ARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY product-service.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGET_ARCH} go build -o /product-service product-service.go

FROM --platform=linux/${TARGET_ARCH} alpine:3.20
COPY --from=build /product-service /product-service
EXPOSE 8080
ENTRYPOINT ["/product-service"]
