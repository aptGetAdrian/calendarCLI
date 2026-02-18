package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// EnsureConnected checks if the user is already authenticated.
// Returns a ready-to-use calendar service if token.json exists, otherwise prints a message and exits.
func EnsureConnected() *calendar.Service {
	ctx := context.Background()
	tokFile := "token.json"
	if _, err := os.Stat(tokFile); os.IsNotExist(err) {
		fmt.Println("You are not connected. Please run the `connect` command first to authenticate.")
		os.Exit(1)
	}

	b, err := os.ReadFile("credentials.json")
	if err != nil {
		fmt.Printf("Unable to read client secret file: %v\n", err)
		os.Exit(1)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		fmt.Printf("Unable to parse client secret file to config: %v\n", err)
		os.Exit(1)
	}

	client, err := getClient(config)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		fmt.Printf("Unable to create Calendar service: %v\n", err)
		os.Exit(1)
	}

	return srv
}

////////////////////////////////////////////////////////////////
// GOOGLE DOCS FUNCTIONS
////////////////////////////////////////////////////////////////

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
		fmt.Printf("You are now authenticated and have your access and refresh tokens stored in a \"tokens.json\" file\n")
		return config.Client(context.Background(), tok), nil
	} else {
		return config.Client(context.Background(), tok), nil
	}

}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	// TODO: tu naredi, da bo neki lep line vmes, recimo to "------------"
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	fmt.Println()
	fmt.Printf("Google will redirect you to something like this:\n" +
		"http://localhost/?code=4/0AfJohXABC123&scope=...\n" +
		"\n" +
		"Copy the code=... value of the URL and paste it into the terminal:\n")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
