all: build

GIT_VERSION := $(shell git rev-parse --short HEAD)

build: $(wildcard *.go)
	GOOS=linux  GOARCH=amd64 go build -ldflags -w -o build/allthefirmwares-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags -w -o build/allthefirmwares-darwin-amd64
	GOOS=windows GOARCH=386 go build -ldflags -w -o build/allthefirmwares-windows-x32.exe

archive: build
	cp README.md build
	cd build
	zip allthefirmwares-win-$(GIT_VERSION).zip build/allthefirmwares-windows-x32.exe README.md
	zip allthefirmwares-osx-$(GIT_VERSION).zip build/allthefirmwares-darwin-amd64 README.md
	zip allthefirmwares-lin-$(GIT_VERSION).zip build/allthefirmwares-linux-amd64 README.md
	cd ..

clean:
	rm -rf build