CGO_ENABLED=0
export CGO_ENABLED

.PHONY: all clean build
all: clean build

clean:
	rm -f build/*

build:
	go build -o build/count cmd/count/main.go
	go build -o build/pipe cmd/pipe/main.go
	go build -o build/repl cmd/repl/main.go
	go build -o build/static cmd/static/main.go
