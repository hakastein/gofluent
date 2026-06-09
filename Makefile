# gofluent — the library is pure Go; CLDR formatting comes from the external
# github.com/hakastein/gocldr module, so there is nothing to generate here.
# `make test` / `make lint` run natively on the host.

# Keep in sync with .github/workflows/ci.yml.
STATICCHECK_VERSION := 2026.1

.PHONY: test vet fmt lint

test:             ## Run the test suite
	go test ./...

vet:
	go vet ./...

lint:             ## go vet + staticcheck (pinned, matching CI) + gofmt check
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...
	@if [ -n "$$(gofmt -l .)" ]; then echo "gofmt needed in:"; gofmt -l .; exit 1; fi

fmt:
	gofmt -w .
