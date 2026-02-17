/*
Copyright © 2026 Adrian
*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Initiates connecting and signing in to your Google account",

	Run: func(cmd *cobra.Command, args []string) {
		b, err := os.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}

		_, err = getClient(config)

		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// connectCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// connectCmd.Flags().BoolP("toggle", "t", false, "tvoja mamas")
}
