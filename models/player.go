package models

import "github.com/jinzhu/gorm"

// A Player modelizes a player from a guild
type Player struct {
	gorm.Model
	GuildID   string
	Guild     Guild `gorm:"association_autoupdate:false;association_autocreate:false"`
	Name      string
	DiscordID string
}

// ListPlayers returns all players from given guild
func ListPlayers(db *gorm.DB, guildID string) (players []Player, err error) {
	err = db.Where("guild_id = ?", guildID).Find(&players).Error
	return
}

// FindPlayer finds a player from his Name and GuildID
func FindPlayer(db *gorm.DB, guildID string, name string) (*Player, error) {
	p := Player{}
	err := db.Where("guild_id = ?", guildID).Where("name = ?", name).First(&p).Error
	return &p, err
}

// Delete deletes current player from the DB
func (p *Player) Delete(db *gorm.DB) error {
	c := Champion{PlayerID: p.ID}
	c.Delete(db)
	return db.Unscoped().Where("id = ?", p.ID).Delete(*p).Error
}
