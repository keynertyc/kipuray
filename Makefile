.PHONY: build web test browser-test ui-test check demo clean

VERSION ?= dev

web:
	cd web && npm ci && npm run build

build: web
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/kipuray ./cmd/kipuray

test:
	go test -race ./...

browser-test:
	KIPURAY_BROWSER_TEST=1 go test -race ./internal/browser

ui-test: build
	cd web && npx playwright test

check:
	go vet ./...
	cd web && npm run check

demo: build
	./bin/kipuray demo

clean:
	go clean
