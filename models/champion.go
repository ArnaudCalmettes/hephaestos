package models

import (
	"fmt"

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
}

func (c Champion) String() string {
	return fmt.Sprintf("%s (Heroes:%d, Titans:%d, ST:%d)", c.Player.Name, c.HeroPower, c.TitanPower, c.SuperTitans)
}

// Create creates a new champion in the DB
func (c *Champion) Create(db *gorm.DB) error {
	return db.Create(c).Error
}

// Update updates a champion in the DB
func (c *Champion) Update(db *gorm.DB) error {
	return db.Table("champions").Where("player_id = ?", c.PlayerID).Updates(*c).Error
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

// FindChampion finds a champion given its GuildID and PlayerID
func FindChampion(db *gorm.DB, guildID string, playerID uint) (*Champion, error) {
	c := &Champion{}
	err := db.Where("guild_id = ?", guildID).Where("player_id", playerID).First(c).Error

	return c, err
}