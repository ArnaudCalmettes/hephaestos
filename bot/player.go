package bot

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"text/tabwriter"

	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"
)

func listPlayers(ctx *exrouter.Context) {
	var players []models.Player
	var err error

	err = transaction(ctx, func(tx *gorm.DB) error {
		players, err = models.ListPlayers(tx, ctx.Msg.GuildID)
		return err
	})

	if err != nil {
		return
	}

	if len(players) == 0 {
		sendWarning(ctx, "The guild wasn't populated yet")
		return
	}

	var b strings.Builder
	w := tabwriter.NewWriter(&b, 5, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tDISCORD ID\t")
	for _, p := range players {
		fmt.Fprintf(w, "%s\t%s\t\n", p.Name, p.DiscordID)
	}
	w.Flush()
	ctx.Reply("```" + b.String() + "```")
}

func renamePlayer(ctx *exrouter.Context) {
	if len(ctx.Args) != 3 {
		sendUsage(ctx, "<old_name> <new_name>")
		log.Println(ctx.Args)
		return
	}
	oldName := ctx.Args[1]
	newName := ctx.Args[2]

	db, _ := ctx.Get("db").(*gorm.DB)
	if db == nil {
		internalError(ctx, errNoDB)
		return
	}

	db.Transaction(func(tx *gorm.DB) error {
		p := models.Player{}
		if tx.Where("guild_id = ?", ctx.Msg.GuildID).Where("name = ?", oldName).First(&p).RecordNotFound() {
			sendError(ctx, fmt.Errorf(`No such player ("%s")`, oldName))
			return errors.New("no such user")
		}
		p.Name = newName
		if err := tx.Table("players").Where("id = ?", p.ID).Updates(p).Error; err != nil {
			internalError(ctx, err)
			return err
		}
		return nil
	})

	markOk(ctx)
}

func bindPlayer(ctx *exrouter.Context) {
	if len(ctx.Args) != 3 {
		sendUsage(ctx, "<name> <@mention>")
		return
	}

	name := ctx.Args[1]
	discordID := ""
	for _, u := range ctx.Msg.Mentions {
		if ctx.Ses.State.User.ID != u.ID {
			discordID = u.ID
			break
		}
	}

	if discordID == "" {
		ctx.Ses.MessageReactionAdd(ctx.Msg.ChannelID, ctx.Msg.ID, "🖕")
		ctx.Reply("I see what you did there...")
		return
	}

	db, _ := ctx.Get("db").(*gorm.DB)
	if db == nil {
		internalError(ctx, errNoDB)
		return
	}

	db.Transaction(func(tx *gorm.DB) error {
		p, err := models.FindPlayer(tx, ctx.Msg.GuildID, name)
		if err == gorm.ErrRecordNotFound {
			sendError(ctx, fmt.Errorf(`No such player ("%s")`, name))
			return err
		} else if err != nil {
			internalError(ctx, err)
			return err
		}

		if err := tx.Model(p).Update("discord_id", discordID).Error; err != nil {
			internalError(ctx, err)
			return err
		}
		return nil
	})

	markOk(ctx)
}

func removePlayer(ctx *exrouter.Context) {
	if len(ctx.Args) != 2 {
		sendUsage(ctx, "<name>")
		return
	}

	transaction(ctx, func(tx *gorm.DB) error {
		p, err := models.FindPlayer(tx, ctx.Msg.GuildID, ctx.Args[1])
		if err == gorm.ErrRecordNotFound {
			sendError(ctx, errors.New("no such player"))
			return err
		}
		if err != nil {
			internalError(ctx, err)
			return err
		}

		if err = p.Delete(tx); err != nil {
			internalError(ctx, err)
			return err
		}
		markOk(ctx)
		return nil
	})

}
