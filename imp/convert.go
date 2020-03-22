package imp

import (
	"image"
)

// ToGray converts any image in a grayscale picture of the same size
func ToGray(src image.Image) *image.Gray {
	if dst, ok := src.(*image.Gray); ok {
		return dst
	}

	bounds := src.Bounds()
	dst := image.NewGray(bounds)
	model := dst.ColorModel()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(x, y, model.Convert(src.At(x, y)))
		}
	}
	return dst
}
