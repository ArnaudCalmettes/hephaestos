package bot

import (
	"fmt"

	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"
)

func setInRotation(ctx *exrouter.Context) {
	if len(ctx.Args) < 2 {
		sendUsage(ctx, "[+<name>|-<name>]...")
		return
	}

	transaction(ctx, func(tx *gorm.DB) error {
		players, err := models.ListPlayers(tx, ctx.Msg.GuildID)
		if err != nil {
			return internalError(ctx, err)
		}
		for _, op := range ctx.Args[1:] {
			if len(op) == 0 {
				continue
			}
			if len(op) < 2 {
				return sendError(ctx, fmt.Errorf("Invalid argument: `%s`", op))
			}
			player, score := findClosestPlayer(op[1:], players)
			if score > 2 {
				return sendError(ctx, fmt.Errorf("No such player: `%s`", op[1:]))
			}
			switch op[0] {
			case '+':
				if err := models.SetInRotation(tx, player, true); err != nil {
					return internalError(ctx, err)
				}
			case '-':
				if err := models.SetInRotation(tx, player, false); err != nil {
					return internalError(ctx, err)
				}
			default:
				return sendError(ctx, fmt.Errorf("Invalid arg: `%s` doesn't start with `+` or `-`", op))
			}
		}
		markOk(ctx)
		return nil
	})
}
