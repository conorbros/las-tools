package conf

import (
	"encoding/json"
	"log"
	"os"
)

var spotifyAuthScopes = []string{"user-follow-read", "user-read-recently-played", "playlist-read-private", "user-follow-read", "user-top-read", "user-library-read", "user-library-modify", "playlist-modify-private", "playlist-modify-public"}

// Config stores constant variables for the applicaiton
var Config *Configuration

func init() {
	Config = New()
}

// LastFmConfig holds configuration options for the LastFm API
type LastFmConfig struct {
	APIKey                string
	UserTopTracksEndpoint string
	UserTopAlbumsEndpoint string
}

// SpotifyConfig holds configuration options for the Spotify API
type SpotifyConfig struct {
	LoginURL                 string
	RedirectURI              string
	AuthScopes               []string
	TokenEndpoint            string
	UserInfoEndpoint         string
	UserPlaylistEndpoint     string
	AddItemsPlaylistEndpoint string
	ClientID                 string
	ClientSecret             string
	SearchEndpoint           string
}

// Configuration holds the configuration data for this instance of the app
type Configuration struct {
	Spotify SpotifyConfig
	LastFm  LastFmConfig
	Port    string
}

// New creates a new configuration struct for the application
func New() *Configuration {
	config := Configuration{}

	file, err := os.Open("./conf/conf.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}

	spotifyRedirectURI := os.Getenv("SPOTIFY_REDIRECT_URL")
	if spotifyRedirectURI == "" {
		spotifyRedirectURI = "http://localhost:8080/playlist"
	}

	config.Spotify.RedirectURI = spotifyRedirectURI

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	config.Port = port

	return &config
}
