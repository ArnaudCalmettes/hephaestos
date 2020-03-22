package imp

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/ArnaudCalmettes/hephaestos/bindata"
)

// ReadFile reads an image from a file.
func ReadFile(filename string) (image.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Read(f)
}

// ReadAsset reads an image from given packed asset name.
func ReadAsset(name string) (image.Image, error) {
	data, err := bindata.Asset(name)
	if err != nil {
		return nil, err
	}
	return ReadBytes(data)
}

// ReadBytes reads an image from raw bytes.
func ReadBytes(data []byte) (image.Image, error) {
	b := bytes.NewBuffer(data)
	return Read(b)
}

// Read reads an image from a io.Reader.
func Read(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

// Save creates a file and writes an image to it. Image format is decided based
// upon its extention (either "png" or "jpg")
func Save(filename string, img image.Image) error {
	ext := filepath.Ext(filename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	switch ext {
	case ".png":
		return png.Encode(f, img)
	case ".jpg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
	}

	return fmt.Errorf("unknown extention %v", ext)
}
