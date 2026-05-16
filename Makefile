.PHONY: build run dev vet test frontend-build doctor install-deps

export PATH := $(shell go env GOPATH)/bin:$(PATH)

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
RUN_CMD = open build/bin/sqlite-explorer.app
else
RUN_CMD = ./build/bin/sqlite-explorer
endif

# Production build (macOS: build/bin/sqlite-explorer.app)
build:
	wails build

# Build then launch the packaged app
run: build
	$(RUN_CMD)

# Hot-reload development server
dev:
	wails dev

vet:
	go vet ./...

test:
	go test ./...

frontend-build:
	cd frontend && npm run build

doctor:
	wails doctor

install-deps:
	cd frontend && npm install
