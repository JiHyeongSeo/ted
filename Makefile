VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/ted ./cmd/ted

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/ted

clean:
	rm -rf bin/ dist/
