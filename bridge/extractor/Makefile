CC=go build
MAKE=make
BUILD_DIR=build
LDFLAGS=-ldflags="-w -s"

build:
	GOARCH=arm GOOS=linux $(CC) $(LDFLAGS) -o $(BUILD_DIR)/arm
	GOARCH=386 GOOS=linux $(CC) $(LDFLAGS) -o $(BUILD_DIR)/x86
