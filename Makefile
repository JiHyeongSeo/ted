LDFLAGS := -s -w

.PHONY: build install clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/ted ./cmd/ted

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/ted

clean:
	rm -rf bin/ dist/
