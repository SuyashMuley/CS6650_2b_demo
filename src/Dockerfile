FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY product-service.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /product-service product-service.go

FROM --platform=linux/amd64 alpine:3.20
COPY --from=build /product-service /product-service
EXPOSE 8080
ENTRYPOINT ["/product-service"]
