package bot

import (
	"log"

	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/bwmarrin/discordgo"
	"github.com/jinzhu/gorm"
)

// Log a received message event to standard output
func logMsg(s *discordgo.Session, m *discordgo.Message) {
	guild, _ := s.Guild(m.GuildID)
	channel, _ := s.Channel(m.ChannelID)
	log.Printf("[%s/%s] %s: %s\n", guild.Name, channel.Name, m.Author.Username, m.Content)
}

// Middleware that logs processed messages to stdout
func logMiddleware(fn exrouter.HandlerFunc) exrouter.HandlerFunc {
	return func(ctx *exrouter.Context) {
		logMsg(ctx.Ses, ctx.Msg)
		if fn != nil {
			fn(ctx)
		}
	}
}

// Middleware that adds the database to commands' context.
func dbMiddleware(db *gorm.DB) exrouter.MiddlewareFunc {
	return func(fn exrouter.HandlerFunc) exrouter.HandlerFunc {
		return func(ctx *exrouter.Context) {
			ctx.Set("db", db)
			if fn != nil {
				fn(ctx)
			}
		}
	}
}

// Middleware that ensures the Discord guild is associated to a guild in the DB
func guildInitMiddleware(fn exrouter.HandlerFunc) exrouter.HandlerFunc {
	return func(ctx *exrouter.Context) {
		// Initialize the guild in the DB if it doesn't exist yet
		if err := createGuild(ctx); err != nil {
			return
		}

		if fn != nil {
			fn(ctx)
		}
	}
}
