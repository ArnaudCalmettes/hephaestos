package bot

import (
	"errors"
	"fmt"

	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"
	"github.com/texttheater/golang-levenshtein/levenshtein"
)

var errNoDB = errors.New("couldn't get DB from context")

// React with a poopy (indicate failure)
func markPoop(ctx *exrouter.Context) {
	ctx.Ses.MessageReactionAdd(ctx.Msg.ChannelID, ctx.Msg.ID, "💩")
}

// React with a thumbs up (indicate success)
func markOk(ctx *exrouter.Context) {
	ctx.Ses.MessageReactionAdd(ctx.Msg.ChannelID, ctx.Msg.ID, "👍")
}

// React with a hourglass (indicate a work in progress)
func markInProgress(ctx *exrouter.Context) {
	ctx.Ses.MessageReactionAdd(ctx.Msg.ChannelID, ctx.Msg.ID, "⏳")
}

// Remove hourglass (indicate the work is done)
func markDone(ctx *exrouter.Context) {
	ctx.Ses.MessageReactionRemove(ctx.Msg.ChannelID, ctx.Msg.ID, "⏳", ctx.Ses.State.User.ID)
}

// Report an error
func sendError(ctx *exrouter.Context, err error) error {
	ctx.Reply("🛑 ", err)
	return err
}

// Report an information
func sendInfo(ctx *exrouter.Context, args ...interface{}) {
	ctx.Reply("ℹ️  ", fmt.Sprint(args...))
}

// Report a warning
func sendWarning(ctx *exrouter.Context, args ...interface{}) {
	ctx.Reply("⚠️  ", fmt.Sprint(args...))
}

// Report an internal error
func internalError(ctx *exrouter.Context, err error) error {
	return sendError(ctx, fmt.Errorf("Internal error (`%w`)", err))
}

// Send correct command syntax
func sendUsage(ctx *exrouter.Context, syntax string) {
	sendWarning(ctx, fmt.Sprintf("syntax: `%s %s`", ctx.Args[0], syntax))
}

// Get database instance from the context
func getDB(ctx *exrouter.Context) (db *gorm.DB, err error) {
	db, _ = ctx.Get("db").(*gorm.DB)
	if db == nil {
		err = errNoDB
	}
	return
}

// Helper to execute a database transaction
func transaction(ctx *exrouter.Context, fn func(*gorm.DB) error) error {
	db, err := getDB(ctx)
	if err != nil {
		internalError(ctx, err)
		return err
	}
	return db.Transaction(fn)
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
