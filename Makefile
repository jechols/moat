VERSION := $(shell git describe --tags || echo "dev")

.PHONY: bin
bin:
	go build -ldflags "-X main.Version=$(VERSION)" -o bin/moat

.PHONY: clean
clean:
	rm -f bin/*


.PHONY: test
test:
	go test ./...
