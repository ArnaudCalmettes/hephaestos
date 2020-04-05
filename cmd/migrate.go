package cmd

import (
	"log"

	"github.com/ArnaudCalmettes/hephaestos/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Perform automatic database migration",
	Run: func(cmd *cobra.Command, args []string) {
		migrateDB()
	},
}

func migrateDB() {
	db, err := gorm.Open("sqlite3", viper.GetString("db"))
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(
		&models.Guild{},
		&models.Player{},
		&models.Champion{},
	)
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
