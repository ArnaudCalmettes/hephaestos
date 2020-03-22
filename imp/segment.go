package imp

import (
	"errors"
	"image"
	"image/color"
)

var (
	Black = color.Gray{0}
	White = color.Gray{255}
)

// Threshold performs simple binarization of a grayscale image.
func Threshold(src, dst *image.Gray, level uint8) error {
	if src.Bounds() != dst.Bounds() {
		return errors.New("src and dst should have the same bounds")
	}

	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			if src.GrayAt(x, y).Y < level {
				dst.SetGray(x, y, Black)
			} else {
				dst.SetGray(x, y, White)
			}
		}
	}
	return nil
}

// ThresholdInv performs inverted binarization of a grayscale image.
func ThresholdInv(src, dst *image.Gray, level uint8) error {
	if src.Bounds() != dst.Bounds() {
		return errors.New("src and dst should have the same bounds")
	}

	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			if src.GrayAt(x, y).Y > level {
				dst.SetGray(x, y, Black)
			} else {
				dst.SetGray(x, y, White)
			}
		}
	}
	return nil
}
