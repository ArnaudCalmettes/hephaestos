package bot

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"
)

// List champions that are set for war
func listWarChampions(ctx *exrouter.Context) {
	champs, err := getChampions(ctx)
	if err != nil {
		return
	}

	var b strings.Builder
	var inWar int
	w := tabwriter.NewWriter(&b, 5, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tHEROES\tTITANS\tST\t")
	for _, c := range champs {
		if !c.InWar {
			continue
		}
		inWar++
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\n",
			c.Player.Name, c.HeroPower, c.TitanPower, c.SuperTitans,
		)
	}
	w.Flush()
	ctx.Reply("```" + b.String() + "```")

	if inWar != 15 && inWar != 20 {
		sendWarning(ctx, "There are currently ", inWar, " champions in war (not 15 or 20).")
	}
}

// Export current war champion list as a csv file
func exportWarChampions(ctx *exrouter.Context) {
	champs, err := getChampions(ctx)
	if err != nil {
		return
	}

	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Write([]string{"Name", "Heroes", "Titans", "ST"})
	for _, c := range champs {
		if !c.InWar {
			continue
		}
		w.Write([]string{
			c.Player.Name,
			strconv.Itoa(c.HeroPower),
			strconv.Itoa(c.TitanPower),
			strconv.Itoa(c.SuperTitans),
		})
	}
	w.Flush()

	ctx.Ses.ChannelFileSend(
		ctx.Msg.ChannelID,
		fmt.Sprintf("%s war champions.csv", champs[0].Guild.Name),
		&b,
	)
}

func setInWar(ctx *exrouter.Context) {
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
				if err := models.SetChampion(tx, player, true); err != nil {
					return internalError(ctx, err)
				}
			case '-':
				if err := models.SetChampion(tx, player, false); err != nil {
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
