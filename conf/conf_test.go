package conf

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	os.Setenv("SPOTIFY_REDIRECT_URL", "")

	config := New()

	if config.SpotifyRedirectURI != "localhost:8080" {
		t.Errorf("SpotifyRedirectURI = %s; want localhost:8080", config.SpotifyRedirectURI)
	}
}
