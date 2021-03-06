package bot

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/ArnaudCalmettes/hephaestos/imp"
	"github.com/ArnaudCalmettes/hephaestos/input"
	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"
)

var errInvalidArgs = errors.New("Invalid arguments")
var errEmptyResult = errors.New("Nothing to get")

func boolToEmoji(v bool) string {
	if v {
		return "X"
	}
	return ""
}

// Utility function to get the list of champions
func getChampions(ctx *exrouter.Context) ([]models.Champion, error) {
	order := "by_titans"
	if len(ctx.Args) == 2 {
		order = ctx.Args[1]
		if order != "by_titans" && order != "by_heroes" {
			sendUsage(ctx, "[by_heroes|by_titans]")
			return nil, errInvalidArgs
		}
	} else if len(ctx.Args) > 2 {
		sendUsage(ctx, "[by_heroes|by_titans]")
		return nil, errInvalidArgs
	}

	champs := make([]models.Champion, 0, 20)
	err := transaction(ctx, func(tx *gorm.DB) error {
		err := tx.Set("gorm:auto_preload", true).Where("guild_id = ?", ctx.Msg.GuildID).Find(&champs).Error
		if err != nil {
			internalError(ctx, err)
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	if len(champs) == 0 {
		sendWarning(ctx, "The guild doesn't have any champions yet. Use `champions read` to set them.")
		return nil, errEmptyResult
	}

	switch order {
	case "by_titans":
		sort.Sort(sort.Reverse(models.ByTitanPower(champs)))
	case "by_heroes":
		sort.Sort(sort.Reverse(models.ByHeroPower(champs)))
	}
	return champs, nil
}

// List currently recorded champions
func listChampions(ctx *exrouter.Context) {
	champs, err := getChampions(ctx)
	if err != nil {
		return
	}

	var b strings.Builder
	var inWar int
	w := tabwriter.NewWriter(&b, 5, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tHEROES\tTITANS\tST\tIN WAR\tWANTS IN\t")
	for _, c := range champs {
		if c.InWar {
			inWar++
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%s\t%s\n",
			c.Player.Name,
			c.HeroPower,
			c.TitanPower,
			c.SuperTitans,
			boolToEmoji(c.InWar),
			boolToEmoji(c.InRotation),
		)
	}
	w.Flush()
	ctx.Reply("```" + b.String() + "```")

	if inWar != 15 && inWar != 20 {
		sendWarning(ctx, "There are currently ", inWar, " champions in war (not 15 or 20).")
	}
}

// Export current champion list as a csv file
func exportChampions(ctx *exrouter.Context) {
	champs, err := getChampions(ctx)
	if err != nil {
		return
	}

	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Write([]string{"Name", "Heroes", "Titans", "ST", "In war"})
	for _, c := range champs {
		w.Write([]string{
			c.Player.Name,
			strconv.Itoa(c.HeroPower),
			strconv.Itoa(c.TitanPower),
			strconv.Itoa(c.SuperTitans),
			fmt.Sprintf("%t", c.InWar),
		})
	}
	w.Flush()

	ctx.Ses.ChannelFileSend(
		ctx.Msg.ChannelID,
		fmt.Sprintf("%s champions.csv", champs[0].Guild.Name),
		&b,
	)
}

// scanChampions fetches images from the message, scans them and extract
// champion information
func scanChampions(ctx *exrouter.Context) []models.Champion {
	var wg sync.WaitGroup
	champStream := make(chan models.Champion)

	// Scan all images in parallel
	for _, att := range ctx.Msg.Attachments {
		wg.Add(1)
		url := att.URL

		go func() {
			defer wg.Done()

			log.Println("Downloading attachment", url)
			resp, err := http.Get(url)
			if err != nil {
				sendWarning(ctx, fmt.Sprintf("Couldn't download <%s>: `%s`\n", url, err))
				return
			}
			defer resp.Body.Close()

			img, err := imp.Read(resp.Body)
			if err != nil {
				sendWarning(ctx, fmt.Sprintf("Couldn't open <%s>: `%s`\n", url, err))
				return
			}

			champs, err := input.ExtractChampions(img)
			if err != nil {
				sendWarning(ctx, fmt.Sprintf("While scanning <%s>: `%s`\n", url, err))
			}

			for _, c := range champs {
				champStream <- c
			}
		}()
	}
	go func() {
		wg.Wait()
		close(champStream)
	}()

	champs := make([]models.Champion, 0, 20)
	seen := make(map[string]bool)
	for c := range champStream {
		if !seen[c.Player.Name] {
			seen[c.Player.Name] = true
			champs = append(champs, c)
		}
	}
	return champs
}

func readChampions(ctx *exrouter.Context) {

	markInProgress(ctx)

	champs := scanChampions(ctx)
	if len(champs) == 0 {
		markDone(ctx)
		markPoop(ctx)
		return
	}

	transaction(ctx, func(tx *gorm.DB) error {
		guild, _ := ctx.Guild(ctx.Msg.GuildID)
		players, err := models.ListPlayers(tx, guild.ID)
		if err != nil {
			return internalError(ctx, err)
		}
		create := make([]models.Champion, 0, 20)
		update := make([]models.ChampionDiff, 0, 20)
		uptodate := make([]string, 0, 20)

		// Decide whether each detected champion should be created or updated
		for _, c := range champs {
			c.GuildID = guild.ID

			// Find the closest existing player according to name similarities
			p, score := findClosestPlayer(c.Player.Name, players)
			// log.Printf("Closest to %v is %v (%d) (score = %d)", c.Player.Name, p.Name, p.ID, score)
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
					c.Player.Name = p.Name
					create = append(create, c)
				} else {
					// Yep, check that this update isn't suspicious
					diff := tmp.Diff(&c)
					if !diff.SeemsLegit() {
						// If anything's fishy, instruct the user to perform a
						// manual update.
						sendWarning(ctx,
							fmt.Sprintf("Suspicious update for **%s** (`%s`)\n", diff.Name, diff),
							fmt.Sprintf(
								"Use `c set \"%s\" %d %d %d` to do it manually",
								tmp.Player.Name, c.HeroPower, c.TitanPower, c.SuperTitans,
							),
						)
						continue
					}
					if diff.IsNull() {
						uptodate = append(uptodate, diff.Name)
						continue
					}

					// Perform the actual update
					tmp.HeroPower = c.HeroPower
					tmp.TitanPower = c.TitanPower
					tmp.SuperTitans = c.SuperTitans
					tmp.InWar = c.InWar

					log.Printf("Updating %s (%s)", diff.Name, diff)

					if err := tmp.Update(tx); err != nil {
						return internalError(ctx, err)
					}
					update = append(update, diff)
				}
			} else {
				// The player doesn't exist yet: associate him to the guild.
				c.Player.GuildID = guild.ID
				log.Println("Creating new champion: ", c)
				if err := c.Create(tx); err != nil {
					return internalError(ctx, err)
				}
				create = append(create, c)
			}
		}

		if len(create) > 0 {
			var b strings.Builder
			for _, c := range create {
				fmt.Fprintf(&b, "\n%s (`%s`)", c.Player.Name, c)
			}
			sendInfo(ctx, "Created ", len(create), " champion(s).", b.String())
		}
		if len(update) > 0 {
			var b strings.Builder
			for _, d := range update {
				fmt.Fprintf(&b, "\n%s (`%s`)", d.Name, d)
			}
			sendInfo(ctx, "Updated ", len(update), " champion(s).", b.String())
		}
		if len(uptodate) > 0 {
			sendInfo(ctx, len(uptodate), " champion(s) already up to date: ", strings.Join(uptodate, ", "), ".")
		}
		return nil
	})
	markDone(ctx)
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
			// Create new player
			update.Player.Name = name
			update.Player.GuildID = ctx.Msg.GuildID
			log.Println("Creating new champion: ", update)
			if err := update.Create(tx); err != nil {
				return internalError(ctx, err)
			}
			sendInfo(ctx, fmt.Sprintf("Created new player: `%s`", update))
			return nil
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
