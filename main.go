package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/conorbros/las-tools/conf"

	"github.com/conorbros/las-tools/chart"
	"github.com/conorbros/las-tools/middleware"
	"github.com/conorbros/las-tools/playlist"
	"github.com/conorbros/las-tools/spotify"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	var tpl = template.Must(template.ParseFiles("./web/template/index.html"))
	tpl.Execute(w, nil)
}

func main() {

	// Serve the static files from the static directory in web
	fs := http.FileServer(http.Dir("web/static"))

	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)

	// Playlist routes
	mux.HandleFunc("/playlist", playlist.PageHandler)

	finalPlaylistHandler := http.HandlerFunc(playlist.PortTopTracksHandler)
	mux.Handle("/port_toptracks", middleware.SpotifyAuthRequired(finalPlaylistHandler))

	// Chart routes
	mux.HandleFunc("/chart", chart.PageHandler)
	mux.HandleFunc("/generate_chart", chart.GenerateChartHandler)

	// Spotify auth routes
	mux.HandleFunc("/login", spotify.LoginHandler)
	mux.HandleFunc("/get_access_token", spotify.GetUserAccessTokenHandler)

	// Requests to /static should be handled by the file server
	mux.Handle("/static/", http.StripPrefix("/static", fs))

	fmt.Println("Listening on " + conf.Config.Port)
	log.Fatal(http.ListenAndServe(":"+conf.Config.Port, mux))
}
