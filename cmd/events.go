/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
)

func listEvents(srv *calendar.Service, max int) {
	t := time.Now().Format(time.RFC3339)

	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()

	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	lenEvents := len(events.Items)
	if lenEvents == 0 {
		fmt.Println("No upcoming events found.")
	} else {
		fmt.Println("Upcoming events:")
		if lenEvents > max {
			for i := 0; i < max; i++ {
				date := events.Items[i].Start.DateTime
				if date == "" {
					date = events.Items[i].Start.Date
				}
				fmt.Printf("%v (%v)\n", events.Items[i].Summary, date)
			}
		} else {
			for i := 0; i < lenEvents; i++ {
				date := events.Items[i].Start.DateTime
				if date == "" {
					date = events.Items[i].Start.Date
				}
				fmt.Printf("%v (%v)\n", events.Items[i].Summary, date)
			}
		}

	}

}

// eventsCmd represents the events command
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "The \"events\" command is an interface for working with events for the calendars",

	Run: func(cmd *cobra.Command, args []string) {
		list, _ := cmd.Flags().GetBool("list")
		create, _ := cmd.Flags().GetBool("create")

		if list {
			srv := EnsureConnected()
			listEvents(srv, 7)
		}

		if create {
			fmt.Print("creating event")
		}
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// eventsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	eventsCmd.Flags().BoolP("list", "l", false, "Lists all events for the selected calendar")
	eventsCmd.Flags().BoolP("create", "c", false, "Creates an event and adds it to the selected calendar")
}
