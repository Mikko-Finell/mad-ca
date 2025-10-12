run:
	go run ./cmd/ca

run-life:
	go run ./cmd/ca -sim=life

build:
	mkdir -p bin
	go build -o bin/ca ./cmd/ca

lint:
	golangci-lint run

wasm:
	mkdir -p web
	GOOS=js GOARCH=wasm go build -o web/ca.wasm ./cmd/ca
