package input

import (
	"image"
	"image/color"
	"log"
	"sort"

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

// GuildChampionsScanner is used to extract champions from screenshots of the
// guild's champions (for local wars in qualifier/bronze/silver/gold leagues).
type GuildChampionsScanner struct {
	champs map[string]models.Champion
}

// NewGuildChampionsScanner creates a new GuildChampionsScanner.
func NewGuildChampionsScanner() *GuildChampionsScanner {
	s := &GuildChampionsScanner{}
	s.champs = make(map[string]models.Champion)
	return s
}

// Scan scans an input image to extract local guild champions.
// Returns the number of new champions that were found in this image.
func (s *GuildChampionsScanner) Scan(input image.Image) (int, error) {

	// Detect the frame within the screenshot, isolate and resize it to the
	// reference size (this usually results in HD screenshots being downsampled).
	frame := imaging.Resize(
		imaging.Crop(input, findChampionsFrame(input)),
		frameWidth, frameHeight, imaging.Box,
	)

	found := 0

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
			return 0, err
		}

		// Avoid duplicating existing champions
		c, ok := s.champs[name]
		if name == "" || ok {
			continue
		}
		c.Player.Name = name

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
		s.champs[c.Player.Name] = c
	}
	return found, nil
}

// Champions returns a sorted list of local guild champions
func (s GuildChampionsScanner) Champions() []models.Champion {
	res := make([]models.Champion, 0, len(s.champs))
	for _, c := range s.champs {
		res = append(res, c)
	}
	sort.Sort(sort.Reverse(models.ByTitanPower(res)))
	return res
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
