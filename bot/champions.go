package bot

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
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
			internalError(ctx, err)
			return err
		}
		create := make([]models.Champion, 0, 20)
		update := make([]models.Champion, 0, 20)

		// Decide whether each detected champion should be created or updated
		for _, c := range champs {
			c.GuildID = guild.ID

			// Find the closest existing player according to name similarities
			p, score := findClosestPlayer(c.Player.Name, players)
			log.Printf("Closest to %v is %v (%d) (score = %d)", c.Player.Name, p.Name, p.ID, score)
			if score < len(p.Name) {
				// The champion is a known player

				// Is he already a champion?
				tmp, err := models.FindChampion(tx, guild.ID, p.ID)
				if err != nil {
					log.Println("Creating new champion")
					// Nope, create a new champion from this player
					c.PlayerID, c.Player.Name = p.ID, ""
					create = append(create, c)
				} else {
					log.Println("Updating existing champion with player_id", tmp.PlayerID)
					// Yep, update the champion's characteristics
					tmp.HeroPower = c.HeroPower
					tmp.TitanPower = c.TitanPower
					tmp.SuperTitans = c.SuperTitans
					update = append(update, tmp)
				}
			} else {
				// The player doesn't exist yet: associate him to the guild.
				c.Player.GuildID = guild.ID
				create = append(create, c)
			}
		}

		// Create new champions (and players if needed)
		for _, c := range create {
			if err := c.Create(tx); err != nil {
				internalError(ctx, err)
				return err
			}
		}
		// Update existing champions
		for _, c := range update {
			if err := c.Update(tx); err != nil {
				internalError(ctx, err)
				return err
			}
		}

		if len(create) > 0 {
			sendInfo(ctx, "Created ", len(create), " champion(s)")
		}
		if len(update) > 0 {
			sendInfo(ctx, "Updated ", len(update), " champions(s)")
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
			sendError(ctx, errors.New("No such user"))
			return err
		} else if err != nil {
			internalError(ctx, err)
			return err
		}
		c := models.Champion{PlayerID: p.ID}
		if err := c.Delete(tx); err != nil {
			internalError(ctx, err)
			return err
		}
		markOk(ctx)
		return nil
	})
}
