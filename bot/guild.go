package bot

import (
	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"
)

func createGuild(ctx *exrouter.Context) error {
	guild, err := ctx.Guild(ctx.Msg.GuildID)
	if err != nil {
		return err
	}

	return transaction(ctx, func(tx *gorm.DB) error {
		g := models.Guild{}
		if tx.Where("id = ?", guild.ID).First(&g).RecordNotFound() {
			g.ID = guild.ID
			g.Name = guild.Name
			if err := tx.Create(&g).Error; err != nil {
				return err
			}
			sendInfo(ctx, "This Discord server is now associated to guild **", g.Name, "**.")
		}
		return nil
	})
}
