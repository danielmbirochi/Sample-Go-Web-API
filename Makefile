SHELL := /bin/bash

CLUSTER_NAME := go-sample-service

# ==============================================================================
# Testing the running system 
#
# curl --user "admin@example.com:gophers" http://localhost:3000/v1/users/token/32bc1165-24t2-61a7-af3e-9da4agf2h1p1
# export TOKEN="YOUR_TOKEN_HERE"
# curl -H "Authorization: Bearer ${TOKEN}" http://localhost:3000/v1/users/1/2
#
# hey -m GET -c 100 -n 10000 -H "Authorization: Bearer ${TOKEN}" http://localhost:3000/v1/users/1/2
# zipkin: http://localhost:9411
# expvarmon -ports 4000 -vars build,requests,goroutines,errors,mem:memstats.Alloc
#

# ==============================================================================
# CLI Help

admin-help:
	go run app/services/sales-admin/main.go -h

run-help:
	go run app/services/sales-api/main.go -h

# ==============================================================================
# Modules support

tidy:
	go mod tidy
	go mod vendor

# ==============================================================================
# Running locally

run:
	go run app/services/sales-api/main.go

run-admin:
	go run app/services/sales-admin/main.go 

build:
	go build -o app/services/sales-api/sales-api app/services/sales-api/main.go

# ==============================================================================
# Administration

generate-keys:
	go run app/services/sales-admin/main.go keygen

generate-token:
	go run app/services/sales-admin/main.go tokengen ${EMAIL}

db-migrations:
	go run app/services/sales-admin/main.go migrate

seed-db:
	go run app/services/sales-admin/main.go seed

# ==============================================================================
# Running local tests

test:
	go test -v ./... -count=1
	staticcheck ./...

test-coverage:
	go test -coverprofile cover.out -v ./... -count=1

test-coverage-detail:
	go tool cover -html cover.out

test-crud:
	cd app/services/sales-api/tests && go test -run TestUsers/crud -v 


# ==============================================================================
# Building containers

all: sales-api

sales-api:
	docker build \
		-f ops/docker/dockerfile.sales-api \
		-t sales-api-amd64:${VERSION} \
		--build-arg PACKAGE_NAME=sales-api \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +”%Y-%m-%dT%H:%M:%SZ”` \
		.

# ==============================================================================
# Running from within k8s/kind

kind-up:
	kind create cluster --image kindest/node:v1.22.1 --name ${CLUSTER_NAME} --config ops/k8s/kind/kind-config.yaml
# Runs the command below to set a default namespace
# kubectl config set-context --current --namespace=sales-system

kind-down:
	kind delete cluster --name ${CLUSTER_NAME}

kind-load:
	cd ops/k8s/kind && kustomize edit set image sales-api-image=sales-api-amd64:${VERSION}
	kind load docker-image sales-api-amd64:${VERSION} --name ${CLUSTER_NAME}

kind-apply:
	kustomize build ops/k8s/kind | kubectl apply -f -

kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide

kind-status-full:
	kubectl describe pod -l app=sales --namespace=sales-system

kind-status-service:
	kubectl get pods -o wide --namespace=sales-system

kind-logs:
	kubectl logs -l app=sales --all-containers=true -f --tail=10000 --namespace=sales-system

kind-restart:
	kubectl rollout restart deployment sales-api --namespace=sales-system

kind-sales-api-update: sales-api # kind-load kind-restart   
	kind load docker-image sales-api-amd64:${VERSION} --name ${CLUSTER_NAME}
	kubectl delete pods -lapp=sales --namespace=sales-system

