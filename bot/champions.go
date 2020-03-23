package bot

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ArnaudCalmettes/hephaestos/imp"
	"github.com/ArnaudCalmettes/hephaestos/input"
	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// List currently recorded champions
func listChampions(ctx *exrouter.Context) {
	guild, err := ctx.Guild(ctx.Msg.GuildID)
	if err != nil {
		internalError(ctx, err)
		return
	}

	var champs []models.Champion
	err = transaction(ctx, func(tx *gorm.DB) error {
		err := tx.Where("guild_id = ?", guild.ID).Preload("Player").Find(&champs).Error
		if err != nil {
			internalError(ctx, err)
		}
		return err
	})
	if err != nil {
		return
	}
	if len(champs) == 0 {
		sendWarning(ctx, "The guild doesn't have any champions yet. Use `champions read` to set them.")
		return
	}

	// Sort champions by titan (+ 20% rule) order
	sort.Sort(sort.Reverse(models.ByTitanPower(champs)))

	var b strings.Builder
	w := tabwriter.NewWriter(&b, 5, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tHEROES\tTITANS\tST\t")
	for _, c := range champs {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t\n", c.Player.Name, c.HeroPower, c.TitanPower, c.SuperTitans)
	}
	w.Flush()
	ctx.Reply("```" + b.String() + "```")

	if len(champs) != 15 && len(champs) != 20 {
		sendWarning(ctx, "There are currently ", len(champs), " champions (not 15 or 20).")
	}
}

func findClosestPlayer(name string, players []models.Player) (best models.Player, score int) {
	score = len(name)
	for _, p := range players {
		d := levenshtein.DistanceForStrings([]rune(name), []rune(p.Name), levenshtein.DefaultOptions)
		if d < score {
			best = p
			score = d
		}
	}
	return
}

func readChampions(ctx *exrouter.Context) {
	scanner := input.NewGuildChampionsScanner()

	for _, att := range ctx.Msg.Attachments {
		log.Println("Downloading attachment", att.URL)
		resp, err := http.Get(att.URL)
		if err != nil {
			sendWarning(ctx, fmt.Sprintf("Couldn't download <%s>: `%s`\n", att.URL, err))
			continue
		}
		defer resp.Body.Close()

		img, err := imp.Read(resp.Body)
		if err != nil {
			sendWarning(ctx, fmt.Sprintf("Couldn't open <%s>: `%s`\n", att.URL, err))
			continue
		}

		if _, err := scanner.Scan(img); err != nil {
			sendWarning(ctx, fmt.Sprintf("While scanning <%s>: `%s`\n", att.URL, err))
		}
	}

	champs := scanner.Champions()
	if len(champs) == 0 {
		markPoop(ctx)
		return
	}

	transaction(ctx, func(tx *gorm.DB) error {
		guild, _ := ctx.Guild(ctx.Msg.GuildID)
		players, err := models.ListPlayers(tx, guild.ID)
		if err != nil {
			return internalError(ctx, err)
		}
		create := make([]string, 0, 20)
		update := make([]string, 0, 20)

		// Decide whether each detected champion should be created or updated
		for _, c := range champs {
			c.GuildID = guild.ID

			// Find the closest existing player according to name similarities
			p, score := findClosestPlayer(c.Player.Name, players)
			log.Printf("Closest to %v is %v (%d) (score = %d)", c.Player.Name, p.Name, p.ID, score)
			if score < len(p.Name) {
				// The champion is more likely a known player

				// Is he already a champion?
				tmp, err := models.FindChampion(tx, guild.ID, p.ID)
				if err != nil {
					// Nope, create a new champion from this player
					log.Println("Creating new champion: ", c)
					c.PlayerID, c.Player.Name = p.ID, ""
					if err := c.Create(tx); err != nil {
						return internalError(ctx, err)
					}
					create = append(create, p.Name)
				} else {
					// Yep, check that this update isn't suspicious
					if !tmp.UpdateSeemsLegit(&c) {
						// If anything's fishy, instruct the user to perform a
						// manual update.
						sendWarning(ctx,
							fmt.Sprintf("Suspicious update:\n`%v -> %v`\n", tmp, c),
							fmt.Sprintf(
								"Use `c set %s %d %d %d` to do it manually",
								c.Player.Name, c.HeroPower, c.TitanPower, c.SuperTitans,
							),
						)
						continue
					}

					// Perform the actual update
					tmp.HeroPower = c.HeroPower
					tmp.TitanPower = c.TitanPower
					tmp.SuperTitans = c.SuperTitans

					log.Println("Updating existing champion: ", tmp)

					if err := tmp.Update(tx); err != nil {
						return internalError(ctx, err)
					}
					update = append(update, p.Name)
				}
			} else {
				// The player doesn't exist yet: associate him to the guild.
				c.Player.GuildID = guild.ID
				log.Println("Creating new champion: ", c)
				if err := c.Create(tx); err != nil {
					return internalError(ctx, err)
				}
				create = append(create, c.Player.Name)
			}
		}

		if len(create) > 0 {
			sendInfo(ctx, "New champions: ", strings.Join(create, ", "))
		}
		if len(update) > 0 {
			sendInfo(ctx, "Updated champions: ", strings.Join(update, ", "))
		}
		return nil
	})
}

func removeChampion(ctx *exrouter.Context) {
	if len(ctx.Args) != 2 {
		sendUsage(ctx, "<name>")
		return
	}
	playerName := ctx.Args[1]

	transaction(ctx, func(tx *gorm.DB) error {
		p, err := models.FindPlayer(tx, ctx.Msg.GuildID, playerName)
		if err == gorm.ErrRecordNotFound {
			return sendError(ctx, errors.New("No such user"))
		} else if err != nil {
			return internalError(ctx, err)
		}
		c := models.Champion{PlayerID: p.ID}
		if err := c.Delete(tx); err != nil {
			return internalError(ctx, err)
		}
		markOk(ctx)
		return nil
	})
}

func setChampion(ctx *exrouter.Context) {
	if len(ctx.Args) != 5 {
		sendUsage(ctx, "<name> <heroes> <titans> <super titans>")
		return
	}

	var err error
	name := ctx.Args[1]
	update := models.Champion{}

	// Parse hero power
	if ctx.Args[2] != "*" {
		update.HeroPower, err = strconv.Atoi(ctx.Args[2])
		if err != nil {
			markPoop(ctx)
			sendError(ctx, errors.New("Hero power must be an integer (or `*` to leave unchanged)"))
			return
		}
	}

	// Parse titan power
	if ctx.Args[3] != "*" {
		update.TitanPower, err = strconv.Atoi(ctx.Args[3])
		if err != nil {
			markPoop(ctx)
			sendError(ctx, errors.New("Titan power must be an integer (or `*` to leave unchanged)"))
			return
		}
	}

	// Parse super titans
	if ctx.Args[4] != "*" {
		update.SuperTitans, err = strconv.Atoi(ctx.Args[4])
		if err != nil {
			markPoop(ctx)
			sendError(ctx, errors.New("Number of super titans must be an integer (or `*` to leave unchanged)"))
			return
		}
	}

	err = transaction(ctx, func(tx *gorm.DB) error {
		p, err := models.FindPlayer(tx, ctx.Msg.GuildID, name)
		if err == gorm.ErrRecordNotFound {
			markPoop(ctx)
			return sendError(ctx, fmt.Errorf("No such player (%v)", name))
		}
		if err != nil {
			return internalError(ctx, err)
		}

		tmp, err := models.FindChampion(tx, p.GuildID, p.ID)
		if err == nil {
			// Update champion
			if update.HeroPower > 0 {
				tmp.HeroPower = update.HeroPower
			}
			if update.TitanPower > 0 {
				tmp.TitanPower = update.TitanPower
			}
			if update.SuperTitans > 0 {
				tmp.SuperTitans = update.SuperTitans
			}
			if err = tmp.Update(tx); err != nil {
				return internalError(ctx, err)
			}
		} else if err == gorm.ErrRecordNotFound {
			update.GuildID, update.PlayerID = p.GuildID, p.ID
			if err = update.Create(tx); err != nil {
				return internalError(ctx, err)
			}
		} else {
			return internalError(ctx, err)
		}
		return nil
	})
	if err == nil {
		markOk(ctx)
	}

}
