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

func returnTimezoneString() string {
	return time.Now().Location().String()
}

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

func insertEvent(srv *calendar.Service) {
	// TODO: make pretty
	// t := time.Now().Format(time.RFC3339)

	// TODO: make sure to automate the timezone offset thing
	var title, location, description string

	fmt.Println("Write a title for the event:")
	if _, err := fmt.Scan(&title); err != nil {
		log.Fatalf("Unable to read the title")
	}

	for len(title) == 0 {
		fmt.Println("Title cannot be empty:")
		if _, err := fmt.Scan(&title); err != nil {
			log.Fatalf("Unable to read the title")
		}
	}

	fmt.Println("Write a location for the event:")
	if _, err := fmt.Scan(&location); err != nil {
		log.Fatalf("Unable to read the location!")
	}

	fmt.Println("Write a description for the event:")
	if _, err := fmt.Scan(&description); err != nil {
		log.Fatalf("Unable to read the description!")
	}

	timezone := returnTimezoneString()

	event := &calendar.Event{
		Summary:     title,
		Location:    location,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: "2026-02-09T16:00:00+01:00",
			TimeZone: timezone,
		},
		End: &calendar.EventDateTime{
			DateTime: "2026-02-09T17:00:00+01:00",
			TimeZone: timezone,
		},
		Recurrence: []string{"RRULE:FREQ=DAILY;COUNT=2"},
		Attendees: []*calendar.EventAttendee{
			&calendar.EventAttendee{Email: "lpage@example.com"},
			&calendar.EventAttendee{Email: "sbrin@example.com"},
		},
	}

	event, err := srv.Events.Insert("primary", event).Do()
	if err != nil {
		log.Fatalf("Unable to create event. %v\n", err)
	}
	fmt.Printf("Event created: %s\n", event.HtmlLink)

}

// eventsCmd represents the events command
var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "The \"events\" command is an interface for working with events for the calendars",

	Run: func(cmd *cobra.Command, args []string) {
		list, _ := cmd.Flags().GetBool("list")
		create, _ := cmd.Flags().GetBool("create")
		srv := EnsureConnected()
		if list {
			listEvents(srv, 7)
		}

		if create {
			insertEvent(srv)
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
