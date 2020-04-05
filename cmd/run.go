package cmd

import (
	"github.com/ArnaudCalmettes/hephaestos/bot"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var token string

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the bot.",
	Run: func(cmd *cobra.Command, args []string) {
		migrateDB()
		if token != "" {
			viper.Set("bot.token", token)
		}
		bot.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&token, "token", "t", "", "discord token")
}
