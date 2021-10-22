SHELL := /bin/bash

# ==============================================================================
# CLI Help

admin-help:
	go run app/sales-admin/main.go

run-help:
	go run app/sales-api/main.go -h

# ==============================================================================
# Running locally

run:
	go run app/sales-api/main.go

build:
	go build -o app/sales-api/sales-api app/sales-api/main.go

# ==============================================================================
# Administration

generate-keys:
	go run app/sales-admin/main.go keygen

generate-token:
	go run app/sales-admin/main.go tokengen ${EMAIL}

# ==============================================================================
# Running local tests

test:
	go test -v ./... -count=1
	staticcheck ./...


# ==============================================================================
# Modules support

tidy:
	go mod tidy
	go mod vendor