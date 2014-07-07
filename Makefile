DEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
PACKAGES = $(shell go list ./...)

all: deps format
	@mkdir -p bin/
	@bash --norc -i ./scripts/build.sh

deps:
	@echo "--> Installing build dependencies"
	@go get -d -v ./...
	@echo $(DEPS) | xargs -n1 go get -d

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

test: deps
	go list ./... | xargs -n1 go test

clean:
	rm -rf broome/pkg
	rm -rf bin

.PNONY: all deps test format
