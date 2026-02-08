/*
Copyright © 2026 Adrian
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Initiates connecting and signing in to your Google account",

	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		b, err := os.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}

		client, err := getClient(config)

		if err != nil {
			fmt.Println(err)
		}

		return

		srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("Unable to retrieve Calendar client: %v", err)
		}

		// TODO: make sure to automate the timezone offset thing
		event := &calendar.Event{
			Summary:     "Google I/O 2015",
			Location:    "800 Howard St., San Francisco, CA 94103",
			Description: "A chance to hear more about Google's developer products.",
			Start: &calendar.EventDateTime{
				DateTime: "2026-02-09T16:00:00+01:00",
				TimeZone: "America/Los_Angeles",
			},
			End: &calendar.EventDateTime{
				DateTime: "2026-02-09T17:00:00+01:00",
				TimeZone: "America/Los_Angeles",
			},
			Recurrence: []string{"RRULE:FREQ=DAILY;COUNT=2"},
			Attendees: []*calendar.EventAttendee{
				&calendar.EventAttendee{Email: "lpage@example.com"},
				&calendar.EventAttendee{Email: "sbrin@example.com"},
			},
		}

		event, err = srv.Events.Insert("primary", event).Do()
		if err != nil {
			log.Fatalf("Unable to create event. %v\n", err)
		}
		fmt.Printf("Event created: %s\n", event.HtmlLink)

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
