package main

//go:generate go-bindata -nocompress -pkg=bindata -o=bindata/bindata.go data/...

import "github.com/ArnaudCalmettes/hephaestos/cmd"

func main() {
	cmd.Execute()
}
