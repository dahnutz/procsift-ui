BINARY := bin/procsift-ui

.PHONY: build test verify-binary size clean

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/procsift-ui
	@chmod 0755 $(BINARY)

test:
	CGO_ENABLED=0 go test ./...

verify-binary:
	@tmp=$$(mktemp); trap 'rm -f "$$tmp"' EXIT; \
	CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o "$$tmp" ./cmd/procsift-ui; \
	cmp -s "$$tmp" $(BINARY) || { echo "$(BINARY) differs from a current source build" >&2; exit 1; }

size: build
	@ls -lh $(BINARY)

clean:
	rm -f $(BINARY)
