BINARY_NAME=raco
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

.PHONY: all build install clean

all: build

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} .

install: build
	sudo mv ${BINARY_NAME} /usr/local/bin/

clean:
	rm -f ${BINARY_NAME}
