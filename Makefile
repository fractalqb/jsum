BUILDFLAGS:=-trimpath -ldflags "-s -w"

.PHONY: clean

all: build/jsum-linux-amd64 \
	build/jsum-linux-386 \
	build/jsum-windows-amd64.exe \
	build/jsum-windows-386.exe

clean:
	rm -rf build

build/jsum-linux-amd64:
	GOOS=linux GOARCH=amd64 go build ${BUILDFLAGS} -o $@ ./cmd/jsum

build/jsum-linux-386:
	GOOS=linux GOARCH=386 go build ${BUILDFLAGS} -o $@ ./cmd/jsum

build/jsum-windows-amd64.exe:
	GOOS=windows GOARCH=amd64 go build ${BUILDFLAGS} -o $@ ./cmd/jsum

build/jsum-windows-386.exe:
	GOOS=windows GOARCH=386 go build ${BUILDFLAGS} -o $@ ./cmd/jsum
