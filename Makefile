run:
	cd ui && go run ./cmd/ca

run-life:
	cd ui && go run ./cmd/ca -sim=life

build:
	mkdir -p bin
	cd ui && go build -o ../bin/ca ./cmd/ca

lint:
	golangci-lint run

wasm:
	mkdir -p web
	cd ui && GOOS=js GOARCH=wasm go build -o ../web/ca.wasm ./cmd/ca
