.PHONY: build build-linux build-macos run dev vet test frontend-test frontend-build audit doctor install-wails install-deps gen-sample-db gen-big-db

WAILS_VERSION ?= v2.15.0

export PATH := $(shell go env GOPATH)/bin:$(PATH)

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
RUN_CMD = open build/bin/sqlite-explorer.app
else
RUN_CMD = ./build/bin/sqlite-explorer
endif

# Production build (macOS: build/bin/sqlite-explorer.app)
build:
	wails build -clean -trimpath

# Modern Linux build (GTK3 + WebKit2GTK 4.1)
build-linux:
	wails build -clean -trimpath -platform linux/amd64

# Universal Intel + Apple Silicon macOS bundle
build-macos:
	wails build -clean -trimpath -platform darwin/universal

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

frontend-test:
	cd frontend && npm test -- --run

frontend-build:
	cd frontend && npm run build

audit:
	govulncheck ./...
	cd frontend && npm audit --audit-level=low

doctor:
	wails doctor

install-wails:
	go install github.com/wailsapp/wails/v2/cmd/wails@$(WAILS_VERSION)

install-deps:
	cd frontend && npm ci

gen-sample-db:
	go run scripts/gen_sample_db.go

gen-big-db:
	go run scripts/gen_big_db.go -rows 1000000 -out /tmp/sqlite-explorer-big.db
