package spotify

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/conorbros/las-tools/conf"
	"github.com/conorbros/las-tools/util"
)

// AuthDetails contains the auth details extracted from a request to the server from a logged in user
type AuthDetails struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TimeObtained int64  `json:"time_obtained"`
}

type trackURIResponse struct {
	Tracks struct {
		Items []struct {
			URI string `json:"uri"`
		} `json:"items"`
	} `json:"tracks"`
}

// User represents the response from the get user info endpoint of the Spotify API
type User struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

// Playlist represents the createed playlist object returned after creating a playlist
type Playlist struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

// Track represents the a track to be added to the user's new spotify playlist
type Track struct {
	Artist     string
	Title      string
	SpotifyURI string
}

func getClientIDClientSecretHeader() string {
	clientIDSecret := []byte(fmt.Sprintf("%s:%s", conf.Config.Spotify.ClientID, conf.Config.Spotify.ClientSecret))
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(clientIDSecret))
}

// LoginHandler redirects the user to the Spotify login screen
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	params := url.Values{}
	params.Add("client_id", conf.Config.Spotify.ClientID)

	scopeStr := ""
	for _, s := range conf.Config.Spotify.AuthScopes {
		scopeStr = scopeStr + "," + s
	}

	params.Add("scope", scopeStr)

	url := conf.Config.Spotify.LoginURL + "?response_type=code" + "&" + params.Encode() + "&redirect_uri=" + conf.Config.Spotify.RedirectURI

	http.Redirect(w, r, url, http.StatusFound)
}

// GetUserAccessTokenHandler gets an access token from the Spotify Token API endpoint and returns to the applicaton frontend
func GetUserAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	params, ok := r.URL.Query()["code"]

	if !ok || len(params[0]) < 1 {
		http.Error(w, "Code is missing", http.StatusBadRequest)
		return
	}
	code := params[0]

	reqBody := url.Values{"code": {code}, "redirect_uri": {conf.Config.Spotify.RedirectURI}, "grant_type": {"authorization_code"}}

	req, err := http.NewRequest(http.MethodPost, conf.Config.Spotify.TokenEndpoint, strings.NewReader(reqBody.Encode()))
	if err != nil {
		http.Error(w, "Error sending code to Spotify Token Endpoint", http.StatusInternalServerError)
	}
	req.Header.Add("Content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", getClientIDClientSecretHeader())

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error sending code to Spotify Token Endpoint", http.StatusInternalServerError)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, "Error reading response from Spotify Token Endpoint", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// GetClientAccessToken gets an access token for reading public data
func GetClientAccessToken() (string, error) {
	reqBody := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequest(http.MethodPost, conf.Config.Spotify.TokenEndpoint, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", getClientIDClientSecretHeader())

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}
	accessToken, ok := data["access_token"].(string)
	if !ok {
		return "", nil
	}
	return "Bearer " + accessToken, nil
}

// RefreshAuth refreshes spotify auth details using the refresh token
func RefreshAuth(spotifyAuthDetails *AuthDetails) error {
	reqBody := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {spotifyAuthDetails.RefreshToken}}

	req, err := http.NewRequest(http.MethodPost, conf.Config.Spotify.TokenEndpoint, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", getClientIDClientSecretHeader())

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &spotifyAuthDetails)
	if err != nil {
		return err
	}

	spotifyAuthDetails.TimeObtained = util.EpochUTC()
	return nil
}

// GetTrackSpotifyURI gets the spotify URI for a track matching the given artist and title
// https://api.spotify.com/v1/search?type=track&limit=10&q=Death+in+Vegas+-+Girls
func GetTrackSpotifyURI(artist string, title string, clientAccessToken string) (string, error) {
	var endpoint = conf.Config.Spotify.SearchEndpoint + "?type=track&limit=10&q="
	endpoint = fmt.Sprintf(endpoint+"artist:%s+track:%s", url.QueryEscape(artist), url.QueryEscape(title))

	req, err := http.NewRequest(http.MethodGet, endpoint, strings.NewReader(""))
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", clientAccessToken)

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response trackURIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response.Tracks.Items) == 0 {
		return "", nil
	}

	// Select the first (or only) result as the URI to use
	uri := response.Tracks.Items[0].URI
	return uri, nil
}

// GetUserID returns the user id for the auth details of the logged in user
func GetUserID(authDetails *AuthDetails) (string, error) {
	req, err := http.NewRequest(http.MethodGet, conf.Config.Spotify.UserInfoEndpoint, strings.NewReader(""))
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+authDetails.AccessToken)

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var user User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// CreatePlaylist creates a playlist on the user account supplied
func CreatePlaylist(userID string, authDetails *AuthDetails) (Playlist, error) {
	var playlist Playlist

	values := map[string]string{"name": "Lastools Playlist", "description": "This playlist was generated automatically with conorb.dev lastools"}
	jsonValue, err := json.Marshal(values)

	req, err := http.NewRequest(http.MethodPost, strings.ReplaceAll(conf.Config.Spotify.UserPlaylistEndpoint, "{user_id}", userID), bytes.NewBuffer(jsonValue))
	if err != nil {
		return playlist, err
	}
	req.Header.Add("Authorization", "Bearer "+authDetails.AccessToken)
	req.Header.Add("Content-type", "application/json")

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return playlist, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return playlist, nil
	}

	err = json.Unmarshal(body, &playlist)
	if err != nil {
		return playlist, err
	}
	return playlist, nil
}

// AddTracksToPlaylist adds the tracks to the supplied playlist
func AddTracksToPlaylist(playlist Playlist, tracks []Track, authDetails *AuthDetails) ([]Track, error) {
	var trackURIs []string
	var tracksNotFound []Track

	for _, t := range tracks {
		if t.SpotifyURI == "" {
			tracksNotFound = append(tracksNotFound, t)
			continue
		}
		trackURIs = append(trackURIs, t.SpotifyURI)
	}
	values := map[string][]string{"uris": trackURIs}

	jsonValue, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.ReplaceAll(conf.Config.Spotify.AddItemsPlaylistEndpoint, "{playlist_id}", playlist.ID), bytes.NewBuffer(jsonValue))

	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+authDetails.AccessToken)
	req.Header.Add("Content-type", "application/json")

	client := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 {
		return nil, errors.New("Tracks were not successfully added to playlist")
	}

	return tracksNotFound, nil
}
