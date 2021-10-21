SHELL := /bin/bash

run:
	go run app/sales-api/main.go

run-admin:
	go run app/sales-admin/keygen.go

run-help:
	go run app/sales-api/main.go -h

build:
	go build -o app/sales-api/sales-api app/sales-api/main.go

tidy:
	go mod tidy
	go mod vendor