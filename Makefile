.PHONY: all bindata

all:
	go generate
	go build

bindata:
	go-bindata -nocompress -pkg=bindata -o=bindata/bindata.go data/...
