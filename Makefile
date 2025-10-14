SIMS := briansbrain ecology elementary life

.PHONY: run $(SIMS) run-% build lint wasm

run:
	./scripts/devsync.sh

run-%:
	./scripts/devsync.sh -sim=$*

$(SIMS):
	./scripts/devsync.sh -sim=$@

build:
	mkdir -p bin
	go build -o bin/ca ./cmd/ca

lint:
	golangci-lint run

wasm:
	mkdir -p web
	GOOS=js GOARCH=wasm go build -o web/ca.wasm ./cmd/ca
