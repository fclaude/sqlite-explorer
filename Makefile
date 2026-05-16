.PHONY: build run dev vet test frontend-test frontend-build doctor install-deps gen-big-db

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

frontend-test:
	cd frontend && npm test -- --run

frontend-build:
	cd frontend && npm run build

doctor:
	wails doctor

install-deps:
	cd frontend && npm install

gen-big-db:
	go run scripts/gen_big_db.go -rows 1000000 -out /tmp/sqlite-explorer-big.db
