package input

import (
	"image"
	"image/color"
	"log"

	"github.com/ArnaudCalmettes/hephaestos/imp"
	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/deluan/lookup"
	"github.com/disintegration/imaging"
)

const (
	frameWidth     = 931 // Reference frame width
	frameHeight    = 491 // Reference frame height
	frameThreshold = 127 // Threshold used when segmenting the frame

	// X coordinates are relative to the top-left corner of the frame
	// Y coordinates are offsets, relative to the position of the checkmarks

	// Where to look for champions' names
	nameXMin       = 120
	nameXMax       = 250
	nameYMinOffset = -15
	nameYMaxOffset = 30

	// Where to look for hero and titan power numbers
	heroPowerXMin   = 360
	heroPowerXMax   = 460
	titanPowerXMin  = 615
	titanPowerXMax  = 715
	powerYMinOffset = -35
	powerYMaxOffset = 0

	// Where to look for titan icons
	titanIconsXMin       = 525
	titanIconsXMax       = 765
	titanIconsYMinOffset = 10
	titanIconsYMaxOffset = 50
)

var (
	// Region containing checkboxes within the guild champions frame
	checkboxROI = image.Rectangle{
		Min: image.Point{
			X: 815,
			Y: 150,
		},
		Max: image.Point{
			X: 845,
			Y: 470,
		},
	}

	// Image templates to be detected using normalized cross-correlation
	checkmarkTemplate image.Image
	arajiTemplate     image.Image
	edenTemplate      image.Image
	hyperionTemplate  image.Image
)

func init() {
	var err error
	if checkmarkTemplate, err = imp.ReadAsset("data/checkmark.png"); err != nil {
		panic(err)
	}
	if arajiTemplate, err = imp.ReadAsset("data/st/araji.png"); err != nil {
		panic(err)
	}
	if edenTemplate, err = imp.ReadAsset("data/st/eden.png"); err != nil {
		panic(err)
	}
	if hyperionTemplate, err = imp.ReadAsset("data/st/hyperion.png"); err != nil {
		panic(err)
	}
}

// ExtractChampions scans an image for champion information and extracts it.
func ExtractChampions(input image.Image) ([]models.Champion, error) {
	champs := make([]models.Champion, 0, 3)

	// Detect the frame within the screenshot, isolate and resize it to the
	// reference size (this usually results in HD screenshots being downsampled).
	frame := imaging.Resize(
		imaging.Crop(input, findChampionsFrame(input)),
		frameWidth, frameHeight, imaging.Box,
	)

	found := 0

SCAN_LOOP:
	// Find ticked checkboxes, and extract champions based on them.
	for _, p := range findCheckmarks(frame) {

		// Extract the name
		name, err := getText(
			frame,
			image.Rectangle{
				Min: image.Point{X: nameXMin, Y: p.Y + nameYMinOffset},
				Max: image.Point{X: nameXMax, Y: p.Y + nameYMaxOffset},
			},
		)
		if err != nil {
			// An error in here means something really wrong has happenned.
			return champs, err
		} else if name == "" {
			continue
		}

		// Make sure the champion wasn't already detected
		for _, c := range champs {
			if c.Player.Name == name {
				continue SCAN_LOOP
			}
		}

		c := models.Champion{
			Player: models.Player{
				Name: name,
			},
		}

		// Find the champions' hero power
		c.HeroPower, err = getPower(
			frame,
			image.Rectangle{
				Min: image.Point{X: heroPowerXMin, Y: p.Y + powerYMinOffset},
				Max: image.Point{X: heroPowerXMax, Y: p.Y + powerYMaxOffset},
			},
		)
		if err != nil || c.HeroPower < 1000 {
			log.Println(err)
			continue
		}

		// Find the champions' titan power
		c.TitanPower, err = getPower(
			frame,
			image.Rectangle{
				Min: image.Point{X: titanPowerXMin, Y: p.Y + powerYMinOffset},
				Max: image.Point{X: titanPowerXMax, Y: p.Y + powerYMaxOffset},
			},
		)
		if err != nil || c.TitanPower < 1000 {
			log.Println(err)
			continue
		}
		found++

		// Count the number of super titans
		c.SuperTitans = findSuperTitans(
			frame,
			image.Rectangle{
				Min: image.Point{X: titanIconsXMin, Y: p.Y + titanIconsYMinOffset},
				Max: image.Point{X: titanIconsXMax, Y: p.Y + titanIconsYMaxOffset},
			},
		)

		// All went well up to this point, keep the extracted data
		champs = append(champs, c)
	}
	return champs, nil
}

func findChampionsFrame(img image.Image) image.Rectangle {
	frame := image.Rectangle{}
	bounds := img.Bounds()

	midX := (bounds.Min.X + bounds.Max.X) / 2
	midY := (bounds.Min.Y + bounds.Max.Y) / 2

	for x := bounds.Min.X; x < midX; x++ {
		if color.GrayModel.Convert(img.At(x, midY)).(color.Gray).Y > frameThreshold {
			frame.Min.X = x
			break
		}
	}

	for y := bounds.Min.Y; y < midY; y++ {
		if color.GrayModel.Convert(img.At(midX, y)).(color.Gray).Y > frameThreshold {
			frame.Min.Y = y
			break
		}
	}

	for x := bounds.Max.X; x > midX; x-- {
		if color.GrayModel.Convert(img.At(x, midY)).(color.Gray).Y > frameThreshold {
			frame.Max.X = x
			break
		}
	}

	for y := bounds.Max.Y; y > midY; y-- {
		if color.GrayModel.Convert(img.At(midX, y)).(color.Gray).Y > frameThreshold {
			frame.Max.Y = y
			break
		}
	}
	return frame
}

// Returns true if both points belong are within the same rectangular region
func sameRegion(a, b lookup.GPoint, w, h int) bool {
	return a.X > b.X-w && a.X < b.X+w && a.Y > b.Y-h && a.Y < b.Y+h
}

// Remove duplicate matches
func pruneMatches(m []lookup.GPoint, r image.Rectangle) []lookup.GPoint {
	res := make([]lookup.GPoint, 0, len(m))
	w, h := r.Size().X, r.Size().Y
LOOP_MATCHES:
	for _, match := range m {
		for idx, best := range res {
			if sameRegion(match, best, w, h) {
				if match.G > best.G {
					res[idx] = match
				}
				continue LOOP_MATCHES
			}
		}
		res = append(res, match)
	}
	return res
}

func findCheckmarks(frame image.Image) []lookup.GPoint {
	lkp := lookup.NewLookup(frame)
	matches, _ := lkp.FindAllInRect(checkmarkTemplate, checkboxROI, 0.9)
	return pruneMatches(matches, checkmarkTemplate.Bounds())
}

func findSuperTitans(frame image.Image, roi image.Rectangle) int {
	res := 0
	lkp := lookup.NewLookup(frame)
	matches, _ := lkp.FindAllInRect(edenTemplate, roi, 0.7)
	if len(matches) > 0 {
		res++
	}
	matches, _ = lkp.FindAllInRect(arajiTemplate, roi, 0.7)
	if len(matches) > 0 {
		res++
	}
	matches, _ = lkp.FindAllInRect(hyperionTemplate, roi, 0.7)
	if len(matches) > 0 {
		res++
	}

	return res
}
