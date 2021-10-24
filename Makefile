SHELL := /bin/bash

export PROJECT = go-sample-service

# ==============================================================================
# CLI Help

admin-help:
	go run app/sales-admin/main.go

run-help:
	go run app/sales-api/main.go -h

# ==============================================================================
# Building containers

all: sales-api

sales-api:
	docker build \
		-f ops/docker/dockerfile.sales-api \
		-t sales-api-amd64:v1.0.0 \
		--build-arg PACKAGE_NAME=sales-api \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +”%Y-%m-%dT%H:%M:%SZ”` \
		.


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