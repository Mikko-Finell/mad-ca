run:
go run ./cmd/ca

run-life:
go run ./cmd/ca -sim=life

build:
go build -o bin/ca ./cmd/ca

lint:
golangci-lint run

wasm:
GOOS=js GOARCH=wasm go build -o web/ca.wasm ./cmd/ca
