package playlist

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/conorbros/las-tools/conf"
	"github.com/conorbros/las-tools/middleware"
	"github.com/conorbros/las-tools/spotify"
)

type portPlaylistData struct {
	LastFmUsername string
	SongNumber     string
	TimePeriod     string
}

// LastFmUserTopTracks represents the results retrieved from the LastFm API user top tracks
type LastFmUserTopTracks struct {
	Toptracks struct {
		Tracks []struct {
			Artist struct {
				Name string `json:"name"`
			} `json:"artist"`
			Name string `json:"name"`
		} `json:"track"`
	} `json:"toptracks"`
}

// PageHandler gets a user's top tracks from Last.fm and converts them into a Spotify playlist
func PageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tpl = template.Must(template.ParseFiles("./web/template/playlist.html"))
	tpl.Execute(w, nil)
}

// PortTopTracksHandler gets a users top track from Last.fm and ports them to a spotify playlist
func PortTopTracksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var portData portPlaylistData

	err := json.NewDecoder(r.Body).Decode(&portData)
	if err != nil {
		http.Error(w, "Malformed JSON", http.StatusBadRequest)
		return
	}

	topTracks, err := getTopTracksLastFm(portData)
	if err != nil {
		http.Error(w, "Could not get top tracks data from LastFm", http.StatusInternalServerError)
		return
	}

	if len(topTracks) <= 0 {
		http.Error(w, "No songs found on Last.fm. Check the username", http.StatusBadRequest)
		return
	}

	// Get the track's Spotify URIs
	err = getTracksSpotifyURIs(topTracks)
	if err != nil {
		http.Error(w, "Could not get Spotify URIs for tracks", http.StatusInternalServerError)
		return
	}

	spotifyAuthDetails := r.Context().Value(middleware.AuthCxtKey).(spotify.AuthDetails)

	// Get User Info
	userID, err := spotify.GetUserID(&spotifyAuthDetails)
	if err != nil {
		http.Error(w, "Could not get Spotify user info", http.StatusInternalServerError)
		return
	}

	// Create playlist on Spotify
	playlist, err := spotify.CreatePlaylist(userID, &spotifyAuthDetails)
	if err != nil {
		http.Error(w, "Could not create playlist on spotify", http.StatusInternalServerError)
		return
	}

	// Add tracks to Spotify
	tracksNotFound, err := spotify.AddTracksToPlaylist(playlist, topTracks, &spotifyAuthDetails)
	if err != nil {
		http.Error(w, "Could not add the tracks to the new playlist on spotify", http.StatusInternalServerError)
		return
	}

	// Put Spotify Auth details back into the body
	values := map[string][]spotify.Track{"tracksNotFound": tracksNotFound}
	jsonValue, err := json.Marshal(values)

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonValue)
}

func getTopTracksLastFm(portData portPlaylistData) ([]spotify.Track, error) {
	urlParams := fmt.Sprintf("&user=%s&api_key=%s&format=json&period=%s&limit=%s", portData.LastFmUsername, conf.Config.LastFm.APIKey, portData.TimePeriod, portData.SongNumber)
	req, err := http.NewRequest(http.MethodGet, conf.Config.LastFm.UserTopTracksEndpoint+urlParams, strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// extract required information from response
	var lastFmTopTracks LastFmUserTopTracks
	err = json.Unmarshal(body, &lastFmTopTracks)
	if err != nil {
		return nil, err
	}

	// convert to a more suitable data structure
	var tracks = make([]spotify.Track, len(lastFmTopTracks.Toptracks.Tracks))
	for i, t := range lastFmTopTracks.Toptracks.Tracks {
		track := spotify.Track{
			Artist: t.Artist.Name,
			Title:  t.Name,
		}
		tracks[i] = track
	}

	return tracks, nil
}

// GetTracksSpotifyURIs gets the Spotify URI for each track in a slice of tracks
func getTracksSpotifyURIs(tracks []spotify.Track) error {

	clientAccessToken, err := spotify.GetClientAccessToken()
	if err != nil {
		return err
	}

	for i := 0; i < len(tracks); i++ {
		uri, err := spotify.GetTrackSpotifyURI(tracks[i].Artist, tracks[i].Title, clientAccessToken)
		if err == nil {
			tracks[i].SpotifyURI = uri
		}
	}
	return nil
}
