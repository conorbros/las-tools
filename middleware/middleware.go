package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/conorbros/conorb-dev/spotify"
	"github.com/conorbros/conorb-dev/util"
)

const (
	// AuthCxtKey represents the SpotifyAuthDetails in context
	AuthCxtKey key = iota
)

type key int

// SpotifyAuthRequired checks that a request has the required spotify authentication details
func SpotifyAuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var spotifyAuthDetails spotify.AuthDetails

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Malformed JSON.", http.StatusBadRequest)
		}

		err = json.Unmarshal(body, &spotifyAuthDetails)
		if err != nil {
			http.Error(w, "Request does not contain Spotify auth details.", http.StatusBadRequest)
			return
		}

		if util.IsSpotifyAuthExpired(spotifyAuthDetails.TimeObtained, spotifyAuthDetails.ExpiresIn) {
			err = spotify.RefreshAuth(&spotifyAuthDetails)
			if err != nil {
				http.Error(w, "Could not refresh expired spotify auth details", http.StatusInternalServerError)
			}
		}

		// Copy the body back into the request
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		ctx := context.WithValue(r.Context(), AuthCxtKey, spotifyAuthDetails)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
