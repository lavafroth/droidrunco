BUILD_DIR=build

all: clean build

build: amd64-linux 386-linux amd64-darwin amd64-windows 386-windows

amd64-linux:
	GOOS=linux GOARCH=amd64 go build -o ${BUILD_DIR}/droidrunco-amd64-linux

386-linux:
	GOOS=linux GOARCH=386 go build -o ${BUILD_DIR}/droidrunco-386-linux

amd64-darwin:
	GOOS=darwin GOARCH=amd64 go build -o ${BUILD_DIR}/droidrunco-amd64-darwin

amd64-windows:
	GOOS=windows GOARCH=amd64 go build -o ${BUILD_DIR}/droidrunco-amd64.exe

386-windows:
	GOOS=windows GOARCH=386 go build -o ${BUILD_DIR}/droidrunco-386.exe

clean:
	go clean
	rm ${BUILD_DIR}/* -f
