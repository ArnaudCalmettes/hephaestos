package bot

import (
	"errors"
	"fmt"

	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/jinzhu/gorm"
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

// Report an error
func sendError(ctx *exrouter.Context, err error) {
	ctx.Reply("📛 ", err)
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
func internalError(ctx *exrouter.Context, err error) {
	sendError(ctx, fmt.Errorf("Internal error (`%w`)", err))
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
