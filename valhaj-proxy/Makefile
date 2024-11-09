CGO_ENABLED=0
export CGO_ENABLED

.PHONY: all clean deps build
all: clean deps build

deps:
	rm -rf external/
	git clone --depth 1 --branch v1.0.8 git@192.168.1.2:valhaj/go-valhaj.git external/go-valhaj/
	mv external/go-valhaj/client/ external/
	rm -rf external/go-valhaj
	find ./external/ -type f -exec sed -i 's#lj.com/go-valhaj#lj.com/valhaj-proxy/external#g' {} \;

clean:
	rm -f build/*

build:
	go build -o build/proxy cmd/proxy/main.go
