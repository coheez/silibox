APP=sili

.PHONY: build run test lint install clean

build:
	go build -ldflags "-X github.com/coheez/silibox/internal/cli.version=$$(git describe --tags --always --dirty) -X github.com/coheez/silibox/internal/cli.commit=$$(git rev-parse --short HEAD) -X github.com/coheez/silibox/internal/cli.buildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/$(APP) ./cmd/sili

run: build
	./bin/$(APP)

test:
	go test ./...

lint:
	@golangci-lint run || echo "Install golangci-lint: brew install golangci-lint"

install: build
	install -m 0755 bin/$(APP) /usr/local/bin/$(APP)

clean:
	rm -rf bin