alias b := clean-build
cc := "go build"
out := "build"
build OS ARCH:
	GOOS={{OS}} GOARCH={{ARCH}} {{cc}} -o {{out}}/droidrunco-{{OS}}-{{ARCH}}

clean-build:
	rm {{out}} bridge/extractor/{{out}} -rf
	just build-embedded
	just build linux amd64
	just build linux 386
	just build linux arm
	just build darwin amd64
	just build windows amd64
	just build windows 386

build-embedded:
	cd bridge/extractor && just build

# amd64-linux:
# 	GOOS=linux GOARCH=amd64 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

# 386-linux:
# 	GOOS=linux GOARCH=386 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

# amd64-darwin:
# 	GOOS=darwin GOARCH=amd64 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

# amd64-windows:
# 	GOOS=windows GOARCH=amd64 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

# 386-windows:
# 	GOOS=windows GOARCH=386 $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@

# arm-linux:
# 	GOOS=linux GOARCH=arm $(CC) ${LDFLAGS} -o ${BUILD_DIR}/droidrunco-$@
