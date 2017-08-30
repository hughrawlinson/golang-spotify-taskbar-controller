package main

import (
	"fmt"
	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
	"github.com/zmb3/spotify"
	"log"
	"net/http"
	"reflect"
)

const redirectURI = "http://localhost:8080/callback"

var html = "<h1>Close this window</h1>"

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	ch    = make(chan *spotify.Client)
	state = "toteslegitrandomstatevalue"
)

func onSystrayReady() {
	systray.SetTitle("Spotify")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
	mAuth := systray.AddMenuItem("Auth", "Authenticate on Spotify")
	go func() {
		<-mAuth.ClickedCh
		log.Print("Commencing Auth")
		open.Run(auth.AuthURL(state))
	}()
}

func prepareServer() {
	var client *spotify.Client
	var playerState *spotify.PlayerState
	fmt.Println(reflect.TypeOf(client.Play))

	http.HandleFunc("/callback", completeAuth)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Got request")
		log.Println("Got request for:", r.URL.String())
	})

	go func() {
		// wait for auth to complete
		client = <-ch

		// use the client to make calls that require authorization
		user, err := client.CurrentUser()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("You are logged in as:", user.ID)

		playerState, err = client.PlayerState()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found your %s (%s)\n", playerState.Device.Type, playerState.Device.Name)
	}()

	http.ListenAndServe(":8080", nil)
}

func main() {
	go prepareServer()
	systray.Run(onSystrayReady, func() {})
}

func setupMenuItem(title string, tooltip string, method func() error) {
	var err error
	mButton := systray.AddMenuItem(title, tooltip)
	go func() {
		<-mButton.ClickedCh
		log.Print("Pressed ", title)
		err = method()
		if err != nil {
			log.Print(err)
		}
	}()
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Login Completed!"+html)

	setupMenuItem("Play", "Play", client.Play)
	setupMenuItem("Pause", "Pause", client.Pause)
	setupMenuItem("Previous", "Previous", client.Previous)
	setupMenuItem("Next", "Next", client.Next)
	setupMenuItem("Play", "Play", client.Play)
	setupMenuItem("Play", "Play", client.Play)

	ch <- &client
}
