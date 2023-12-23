alias b := build
alias ba := build-all

cc := "go build"
out := "build"

build-for OS ARCH:
	CGO_ENABLED=0 GOOS={{OS}} GOARCH={{ARCH}} {{cc}} -o {{out}}/droidrunco-{{OS}}-{{ARCH}} -ldflags "-w -s"

clean:
	rm {{out}} bridge/extractor/{{out}} -rf

build: clean build-embedded
	CGO_ENABLED=0 {{cc}} -o {{out}}/droidrunco -ldflags "-w -s"

build-all: clean build-embedded
	just build-for linux amd64
	just build-for linux 386
	just build-for linux arm
	just build-for darwin amd64
	just build-for windows amd64
	just build-for windows 386
	mv {{out}}/droidrunco-windows-amd64 {{out}}/droidrunco-windows-amd64.exe
	mv {{out}}/droidrunco-windows-386 {{out}}/droidrunco-windows-386.exe

build-embedded:
	cd bridge/extractor && just build
