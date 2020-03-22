package models

import (
	"errors"
	"fmt"
)

// A Guild models both a guild in HWM and a Discord guild.
// Hephaestos assumes there's a 1-1 correspondence between both.
type Guild struct {
	ID   string `gorm:"primary_key"`
	Name string
}

func (g Guild) String() string {
	return fmt.Sprintf("Guild{id=%v, name=%v}", g.ID, g.Name)
}

// BeforeSave is executed just before a Guild is saved into the DB
func (g *Guild) BeforeSave() error {
	if g.ID == "" {
		return errors.New("missing guild ID")
	}
	if g.Name == "" {
		return errors.New("guild name can't be empty")
	}
	return nil
}
