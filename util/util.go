package util

import (
	"time"
)

// EpochUTC returns milliseconds since epoch UTC. Golang equivalent of JS Date.now()
func EpochUTC() int64 {
	return time.Now().UTC().UnixNano() / 1e6
}

// IsSpotifyAuthExpired calculates if a spotify auth token has expired using the timeObtained and expiresIn values
func IsSpotifyAuthExpired(timeObtained int64, expiresIn int) bool {
	now := EpochUTC()
	expiresInMS := int64(int(expiresIn)) * 1000
	return now > timeObtained+expiresInMS
}
