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

circleci:
	mkdir -p  /home/ubuntu/.go_workspace/src/github.com/Bowery/
	git clone git@github.com:Bowery/broome.git
	cd broome
	go test
	cd db
	go test
	cd ../util
	go test

test: deps
	go list ./... | xargs -n1 go test #  too fancy for circleci

clean:
	rm -rf crosby/pkg
	rm -rf bin

.PNONY: all deps test format
