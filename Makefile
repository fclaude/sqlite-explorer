.PHONY: dev build vet test frontend-build doctor

export PATH := $(shell go env GOPATH)/bin:$(PATH)

dev:
	wails dev

build:
	wails build

vet:
	go vet ./...

test:
	go test ./...

frontend-build:
	cd frontend && npm run build

doctor:
	wails doctor
