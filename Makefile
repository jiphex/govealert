all: build/govealert_%

clean:
	rm -vfr build/*

build/govealert_%: govealert.go
	mkdir -p build
	${GOPATH}/bin/gox -output "build/govealert_{{.OS}}_{{.Arch}}" -osarch "linux/amd64 linux/386 windows/amd64 windows/386 darwin/amd64"

.PHONY: all clean
