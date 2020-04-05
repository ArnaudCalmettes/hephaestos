package models

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
)

// A Champion is represented by his name, the power of his teams, and the
// number of supertitans he has.
type Champion struct {
	GuildID     string
	Guild       Guild `gorm:"association_autoupdate:false;association_autocreate:false"`
	PlayerID    uint
	Player      Player `gorm:"association_autoupdate:false"`
	HeroPower   int
	TitanPower  int
	SuperTitans int
	InWar       bool
	InRotation  bool
}

func (c Champion) String() string {
	return fmt.Sprintf("Heroes:%d, Titans:%d, ST:%d", c.HeroPower, c.TitanPower, c.SuperTitans)
}

// Create creates a new champion in the DB
func (c *Champion) Create(db *gorm.DB) error {
	return db.Create(c).Error
}

// Update updates a champion in the DB
func (c *Champion) Update(db *gorm.DB) error {
	c.Player = Player{}
	return db.Table("champions").Where("player_id = ?", c.PlayerID).Updates(
		map[string]interface{}{
			"hero_power":   c.HeroPower,
			"titan_power":  c.TitanPower,
			"super_titans": c.SuperTitans,
			"in_war":       c.InWar,
		}).Error
}

func SetChampion(db *gorm.DB, player Player, inWar bool) error {
	return db.Table("champions").Where("player_id = ?", player.ID).Update("in_war", inWar).Error
}

// Delete deletes current champion from the DB
func (c *Champion) Delete(db *gorm.DB) error {
	return db.Where("player_id = ?", c.PlayerID).Delete(*c).Error
}

// ByTitanPower allows to sort champions by titan power using the so-called "20% rule".
type ByTitanPower []Champion

func (a ByTitanPower) Len() int      { return len(a) }
func (a ByTitanPower) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTitanPower) Less(i, j int) bool {
	pi := float32(a[i].TitanPower)
	pi += float32(a[i].SuperTitans) * (pi / 5)
	pj := float32(a[j].TitanPower)
	pj += float32(a[j].SuperTitans) * (pj / 5)
	return pi < pj
}

// ByHeroPower allows to sort champions by hero power
type ByHeroPower []Champion

func (a ByHeroPower) Len() int      { return len(a) }
func (a ByHeroPower) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByHeroPower) Less(i, j int) bool {
	return a[i].HeroPower < a[j].HeroPower
}

// FindChampion finds a champion given its GuildID and PlayerID
func FindChampion(db *gorm.DB, guildID string, playerID uint) (Champion, error) {
	c := Champion{}
	err := db.Where(&Champion{GuildID: guildID, PlayerID: playerID}).First(&c).Error
	if err != nil {
		return c, err
	}
	err = db.Where("id = ?", playerID).First(&c.Player).Error
	return c, err
}

// Diff computes a diff to qualify a possible champion update.
func (c *Champion) Diff(cNew *Champion) ChampionDiff {
	inWar := 0
	if !c.InWar && cNew.InWar {
		inWar = 1
	} else if c.InWar && !cNew.InWar {
		inWar = -1
	}
	return ChampionDiff{
		c.Player.Name,
		cNew.HeroPower - c.HeroPower,
		cNew.TitanPower - c.TitanPower,
		cNew.SuperTitans - c.SuperTitans,
		inWar,
	}
}

// ChampionDiff represents a difference between two versions of a champion.
type ChampionDiff struct {
	Name        string
	HeroPower   int
	TitanPower  int
	SuperTitans int
	InWar       int
}

// IsNull returns true if the diff is null
func (c ChampionDiff) IsNull() bool {
	return c.HeroPower == 0 && c.TitanPower == 0 && c.SuperTitans == 0
}

// SeemsLegit returs true if the update is an actual improvement
func (c ChampionDiff) SeemsLegit() bool {
	return c.HeroPower >= 0 && c.TitanPower >= 0 && c.SuperTitans >= 0
}

func (c ChampionDiff) String() string {
	updates := make([]string, 0, 3)
	if c.HeroPower != 0 {
		updates = append(updates, fmt.Sprintf("Heroes: %+d", c.HeroPower))
	}
	if c.TitanPower != 0 {
		updates = append(updates, fmt.Sprintf("Titans: %+d", c.TitanPower))
	}
	if c.SuperTitans != 0 {
		updates = append(updates, fmt.Sprintf("ST: %+d", c.SuperTitans))
	}
	if c.InWar != 0 {
		updates = append(updates, fmt.Sprintf("In war: %+d", c.InWar))
	}
	return strings.Join(updates, ", ")
}
