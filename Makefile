CC=go build
MAKE=make
BUILD_DIR=build
LDFLAGS=-ldflags="-w -s"

all: clean build

build: embed amd64-linux 386-linux arm-linux amd64-darwin amd64-windows 386-windows

embed:
	$(MAKE) -C extractor build

amd64-linux:
	GOOS=linux GOARCH=amd64 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

386-linux:
	GOOS=linux GOARCH=386 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

amd64-darwin:
	GOOS=darwin GOARCH=amd64 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

amd64-windows:
	GOOS=windows GOARCH=amd64 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

386-windows:
	GOOS=windows GOARCH=386 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

arm-linux:
	GOOS=linux GOARCH=arm $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

clean:
	go clean
	rm ${BUILD_DIR}/* -f
