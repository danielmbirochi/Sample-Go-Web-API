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
# Running from within k8s/dev

kind-up:
	kind create cluster --image kindest/node:v1.22.1 --name go-sample-service --config ops/k8s/dev/kind-config.yaml

kind-down:
	kind delete cluster --name go-sample-service

kind-load:
	kind load docker-image sales-api-amd64:v1.0.0 --name go-sample-service

kind-services:
	kustomize build ops/k8s/dev | kubectl apply -f -

kind-status:
	kubectl get nodes
	kubectl get pods --watch

kind-status-full:
	kubectl describe pod -lapp=sales-api

kind-logs:
	kubectl logs -lapp=sales-api --all-containers=true -f

kind-sales-api-update: sales-api
	kind load docker-image sales-api-amd64:v1.0.0 --name go-sample-service
	kubectl delete pods -lapp=sales-api


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