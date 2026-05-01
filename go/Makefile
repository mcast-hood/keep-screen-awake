.PHONY: build test clean build-darwin

# On Windows (via git bash / make), OS=Windows_NT; on macOS/Linux it is unset.
ifeq ($(OS),Windows_NT)
  EXT := .exe
else
  EXT :=
endif

build:
	go build -o bin/ksad$(EXT) ./cmd/ksad
	go build -o bin/ksa$(EXT)  ./cmd/ksa

# Cross-compile macOS arm64 binaries (run from Windows or any host).
# Output goes to bin/darwin/ with plain names so they are ready to install on Mac.
build-darwin:
	GOOS=darwin GOARCH=arm64 go build -o bin/darwin/ksad ./cmd/ksad
	GOOS=darwin GOARCH=arm64 go build -o bin/darwin/ksa  ./cmd/ksa

test:
	go test ./...

clean:
	rm -rf bin/
