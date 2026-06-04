# gofluent — the library itself is pure Go with zero dependencies, so `make test`
# runs natively. Only CLDR code-generation is version-sensitive and therefore
# pinned to a hermetic Docker toolchain (see gen/Dockerfile); never run the
# generators or their Node fixture scripts on the host.

GEN_IMAGE := gofluent-cldrgen
DOCKER_RUN := docker run --rm \
	-v "$(CURDIR)":/work \
	-u "$(shell id -u):$(shell id -g)" \
	"$(GEN_IMAGE)"

.PHONY: gen gen-image test vet fmt

gen-image:        ## Build the pinned CLDR generation image
	docker build --load -t "$(GEN_IMAGE)" gen

gen: gen-image    ## Regenerate all cldr/* tables + Intl golden fixtures (CLDR pinned in gen/package.json)
	$(DOCKER_RUN) sh gen/generate.sh

test:             ## Run the test suite (pure Go, zero deps — host is fine)
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
