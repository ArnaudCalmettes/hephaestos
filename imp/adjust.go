package imp

import (
	"errors"
	"image"
)

// Normalize adjusts a grayscale image so it spans the whole colorspace
func Normalize(src, dst *image.Gray) error {
	if src.Bounds() != dst.Bounds() {
		return errors.New("src and dst should have the same bounds")
	}

	var min uint8 = 255
	var max uint8 = 0

	rect := src.Bounds()
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			val := src.GrayAt(x, y).Y
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
		}
	}

	alpha := float32(max - min)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := src.GrayAt(x, y)
			c.Y = uint8((float32(c.Y-min) / alpha) * 255)
			dst.SetGray(x, y, c)
		}
	}
	return nil
}
