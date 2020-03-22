package input

import (
	"bytes"
	"image"
	"image/png"
	"regexp"
	"strconv"

	"github.com/ArnaudCalmettes/hephaestos/imp"
	"github.com/disintegration/imaging"
	"github.com/otiai10/gosseract/v2"
)

var powerRegexp = regexp.MustCompile(`\d{4,}`)

func getPower(img image.Image, roi image.Rectangle) (int, error) {
	txt, err := getText(img, roi)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(powerRegexp.FindString(txt))
}

func getText(img image.Image, roi image.Rectangle) (string, error) {
	bin := imp.ToGray(imaging.Crop(img, roi))
	imp.ThresholdInv(bin, bin, 120)
	var b bytes.Buffer
	if err := png.Encode(&b, bin); err != nil {
		return "", err
	}

	ocr := gosseract.NewClient()
	defer ocr.Close()
	ocr.SetImageFromBytes(b.Bytes())
	return ocr.Text()
}
