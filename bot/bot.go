package bot

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/bwmarrin/discordgo"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

// Log a received message event to standard output
func logMsg(s *discordgo.Session, m *discordgo.MessageCreate) {
	guild, _ := s.Guild(m.GuildID)
	channel, _ := s.Channel(m.ChannelID)
	log.Printf("[%s/%s] %s: %s\n", guild.Name, channel.Name, m.Author.Username, m.Message.Content)
}

// Middleware that adds the database to commands' context.
// Upon first use, the Discord guild is automatically initialized in the
// databse.
func dbMiddleware(db *gorm.DB) exrouter.MiddlewareFunc {
	return func(fn exrouter.HandlerFunc) exrouter.HandlerFunc {
		return func(ctx *exrouter.Context) {
			ctx.Set("db", db)

			// Initialize the guild in the DB if it doesn't exist yet
			if err := createGuild(ctx); err != nil {
				internalError(ctx, err)
				return
			}

			if fn != nil {
				fn(ctx)
			}
		}
	}
}

// Run runs the bot.
func Run() {
	dg, err := discordgo.New("Bot " + viper.GetString("bot.token"))

	if err != nil {
		fmt.Println("couldn't create Discord session:", err)
		return
	}

	db, err := gorm.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		fmt.Println("couldn't connect to db:", err)
		return
	}

	router := exrouter.New()

	router.On("players", func(*exrouter.Context) {}).Group(func(r *exrouter.Route) {
		r.Use(dbMiddleware(db))
		r.On("list", listPlayers).Desc("list known players (alias: ls)").Alias("ls")
		r.On("rename", renamePlayer).Desc("rename a player")
		r.On("bind", bindPlayer).Desc("bind a player to a Discord user")
		r.On("remove", removePlayer).Desc("delete a player (alias: rm)").Alias("rm")
	}).Desc("handle players of the guild (alias: p)").Alias("p")

	router.On("champions", func(*exrouter.Context) {}).Group(func(r *exrouter.Route) {
		r.Use(dbMiddleware(db))
		r.On("list", listChampions).Desc("list current champions (alias: ls)").Alias("ls")
		r.On("export", exportChampions).Desc("export champions to a csv file")
		r.On("update", readChampions).Desc("read & update champions from screenshots (alias: up)").Alias("up")
		r.On("set", setChampion).Desc("manually set a champion")
		r.On("remove", removeChampion).Desc("remove a champion (alias: rm)").Alias("rm")
	}).Desc("handle champions (alias: c)").Alias("c")

	router.Default = router.On("help", func(ctx *exrouter.Context) {
		var f func(depth int, r *exrouter.Route) string
		f = func(depth int, r *exrouter.Route) string {
			text := ""
			for _, v := range r.Routes {
				text += strings.Repeat("  ", depth) + v.Name + ": " + v.Description + "\n"
				text += f(depth+1, &exrouter.Route{Route: v})
			}
			return text
		}
		ctx.Reply("```" + f(0, router) + "```")
	}).Desc("print this help menu (aliases: [h])").Alias("h")

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		logMsg(s, m)
		router.FindAndExecute(s, ".", s.State.User.ID, m.Message)
	})

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection:", err)
		return
	}

	fmt.Println("Up & running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}
