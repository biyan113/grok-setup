.PHONY: build test fmt vet tidy install

PREFIX ?= $(HOME)/.local/bin
VERSION ?= 0.1.1

build:
	go build -ldflags "-s -w" -o gsetup ./cmd/gsetup

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

install: build
	mkdir -p "$(PREFIX)"
	install -m 755 gsetup "$(PREFIX)/gsetup"
	@echo "installed $(PREFIX)/gsetup"
