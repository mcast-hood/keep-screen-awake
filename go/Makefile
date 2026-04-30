.PHONY: build test clean build-darwin

GOOS_DARWIN=GOOS=darwin GOARCH=arm64

build:
	go build -o bin/ksad.exe ./cmd/ksad
	go build -o bin/ksa.exe  ./cmd/ksa

build-darwin:
	$(GOOS_DARWIN) go build -o bin/ksad-darwin ./cmd/ksad
	$(GOOS_DARWIN) go build -o bin/ksa-darwin  ./cmd/ksa

test:
	go test ./...

clean:
	rm -rf bin/
