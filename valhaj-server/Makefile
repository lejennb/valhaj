CGO_ENABLED=0
export CGO_ENABLED

.PHONY: all clean build stripped
all: clean build

clean:
	rm -f build/*

build:
	go build -o build/valhaj cmd/valhaj/main.go

stripped:
	go build -o build/valhaj -ldflags="-s -w" cmd/valhaj/main.go
