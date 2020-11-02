package chart

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/jpeg"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/conorbros/las-tools/conf"
	gim "github.com/ozankasikci/go-image-merge"
)

type imageResponse struct {
	Size string `json:"size"`
	Text string `json:"#text"`
}

type topAlbumsResponse struct {
	Topalbums struct {
		Album []struct {
			Artist struct {
				Name string `json:"name"`
			} `json:"artist"`
			Image     []imageResponse `json:"image"`
			Playcount string          `json:"playcount"`
			Name      string          `json:"name"`
		} `json:"album"`
	} `json:"topalbums"`
}

type albumImagesURL struct {
	ExtraLarge string
	Large      string
	Medium     string
	Small      string
}

type albumColor struct {
	Hue   float64
	Sat   float64
	Value float64
}

// Album represents an album to be added to the chart
type album struct {
	Artist    string
	Playcount uint64
	ImageURLS albumImagesURL
	Title     string
	Image     *image.Image
	Color     albumColor
}

// PageHandler loads the chart page for the requesting user
func PageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	var tpl = template.Must(template.ParseFiles("./web/template/chart.html"))
	tpl.Execute(w, nil)
}

func extractQuery(r *http.Request) (username string, x, y int, err error) {
	usernameQ, ok := r.URL.Query()["username"]
	if !ok || len(usernameQ) == 0 || usernameQ[0] == "" {
		err = errors.New("Missing username")
		return
	}
	xQ, ok := r.URL.Query()["x"]
	if !ok {
		err = errors.New("Missing X")
		return
	}
	yQ, ok := r.URL.Query()["y"]
	if !ok {
		err = errors.New("Missing y")
		return
	}

	username = usernameQ[0]

	if x, err = strconv.Atoi(xQ[0]); err != nil {

		errors.New("X is not an int")
		return
	}
	if y, err = strconv.Atoi(yQ[0]); err != nil {
		errors.New("Y is not an int")
		return
	}

	return
}

func writeToLogFile(username string, x int, y int) {
	f, err := os.OpenFile("./log.txt", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	defer f.Close()
	if err != nil {
		log.Print(err)
		return
	}
	_, err = fmt.Fprintln(f, username, x, y, time.Now().String())

	if err != nil {
		log.Print(err)
	}
}

// GenerateChartHandler generates a chart for the user and returns that chart
func GenerateChartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, x, y, err := extractQuery(r)
	if err != nil {
		http.Error(w, "Bad request. Try reloading the page.", http.StatusBadRequest)
		return
	}

	albums, err := getLastFmTopAlbums(username, x*y)
	if err != nil {
		http.Error(w, "There was an error getting the albums. Try again or contact me.", http.StatusInternalServerError)
		return
	}

	if len(albums) <= 0 {
		http.Error(w, "No albums were found. Check the Last.fm username", http.StatusBadRequest)
		return
	}

	if len(albums) < x*y {
		http.Error(w, "Not enough albums to generate a chart. Try choosing a smaller size.", http.StatusBadRequest)
		return
	}

	var size string
	if x >= 30 || y >= 30 {
		size = "Medium"
	} else {
		size = "Large"
	}

	albums, err = getAlbumCovers(albums, x*y, size)
	if err != nil {
		http.Error(w, "Download to failed images. Try again or contact me.", http.StatusInternalServerError)
		return
	}

	err = sortAlbumsByHsv(albums)
	if err != nil {
		http.Error(w, "There was an error generating the chart. Try again or contact me.", http.StatusInternalServerError)
	}

	rearrangeAlbums(albums, x, y)

	var grids = make([]*gim.Grid, len(albums))

	for i, a := range albums {
		if a.Image != nil {
			grids[i] = &gim.Grid{
				Image: a.Image,
			}
		}
	}

	chart, err := gim.New(grids, x, y).Merge()
	if err != nil {
		http.Error(w, "There was an error generating the image. Try again or contact me.", http.StatusInternalServerError)
	}

	buffer := new(bytes.Buffer)
	jpeg.Encode(buffer, chart, &jpeg.Options{Quality: 80})

	w.Header().Set("Content-type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	w.Write(buffer.Bytes())
}

func getLastFmTopAlbums(username string, count int) ([]album, error) {
	// add 50 to the count as a buffer against downloads that fail
	urlParams := fmt.Sprintf("&user=%s&api_key=%s&format=json&period=%s&limit=%d", username, conf.Config.LastFm.APIKey, "overall", count+50)
	url := conf.Config.LastFm.UserTopAlbumsEndpoint + urlParams

	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: time.Duration(60 * time.Second),
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

	var response topAlbumsResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	var albums []album
	for _, a := range response.Topalbums.Album {
		images, err := makeAlbumImagesURL(a.Image)
		if err != nil {
			continue
		}
		playcount, err := strconv.ParseUint(a.Playcount, 10, 64)
		if err != nil {
			playcount = 0
		}
		album := album{
			Artist:    a.Artist.Name,
			ImageURLS: images,
			Playcount: playcount,
			Title:     a.Name,
		}
		albums = append(albums, album)
	}
	return albums, nil
}

func makeAlbumImagesURL(images []imageResponse) (albumImagesURL, error) {
	var albumImages albumImagesURL
	for _, img := range images {

		size := img.Size
		urlExists := img.Text != ""

		switch true {
		case size == "small" && urlExists:
			albumImages.Small = img.Text

		case size == "medium" && urlExists:
			albumImages.Medium = img.Text

		case size == "large" && urlExists:
			albumImages.Large = img.Text

		case size == "extralarge" && urlExists:
			albumImages.ExtraLarge = img.Text

		default:
			return albumImages, errors.New("No album art detected")
		}
	}

	return albumImages, nil
}

func getAlbumCovers(albums []album, count int, size string) ([]album, error) {
	errIndexes := downloadImages(albums, size)

	var newAlbums []album
	for i := 0; i < len(albums); i++ {
		if !find(errIndexes, i) {
			newAlbums = append(newAlbums, albums[i])
		}
		if len(newAlbums) >= count {
			break
		}
	}

	return newAlbums, nil
}

func downloadImages(albums []album, size string) []int {
	errs := make(chan int)
	success := make(chan int)
	wgDone := make(chan bool)

	var wg sync.WaitGroup

	for i := 0; i < len(albums); i++ {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()

			var url string
			if size == "Large" {
				url = albums[i].ImageURLS.Large
			} else {
				url = albums[i].ImageURLS.Medium
			}

			res, err := http.Get(url)
			if err != nil {
				errs <- i
				log.Print(err)
				return
			}

			defer res.Body.Close()
			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				errs <- i
				log.Print(err)
				return
			}

			img, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				errs <- i
			} else {
				albums[i].Image = &img
				success <- i
			}
		}(i, &wg)
	}

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	count := 0
	var errIndexes []int
	for {
		select {
		case <-success:
			count++
		case i := <-errs:
			errIndexes = append(errIndexes, i)
		case <-wgDone:
			return errIndexes
		}
	}
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func find(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func sortAlbumsByHsv(albums []album) error {

	var wg sync.WaitGroup

	for i := range albums {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			albums[i].Color = getAlbumColor(albums[i].Image, albums[i].Title)
		}(i, &wg)
	}

	wg.Wait()

	sort.Slice(albums, func(i int, j int) bool {
		if albums[i].Color.Hue == albums[j].Color.Hue {
			if albums[i].Color.Sat == albums[j].Color.Sat {
				return albums[i].Color.Value < albums[j].Color.Value
			}
			return albums[i].Color.Sat < albums[j].Color.Sat
		}
		return albums[i].Color.Hue < albums[j].Color.Hue
	})
	return nil
}

// getAlbumColor gets the
func getAlbumColor(i *image.Image, t string) albumColor {
	color := averageImageColor(*i)
	h, s, v := hsv(color)
	return albumColor{
		Hue:   h * 60,
		Sat:   s,
		Value: v,
	}
}

// hsv converts a go color type to HSV format (Hue, Saturation, Value)
func hsv(c color.Color) (h, s, v float64) {
	fR, fG, fB := normalize(c)

	max := math.Max(math.Max(fR, fG), fB)
	min := math.Min(math.Min(fR, fG), fB)
	d := max - min
	s, v = 0, max
	if max > 0 {
		s = d / max
	}
	if max == min {
		// Achromatic.
		h = 0
	} else {
		// Chromatic.
		switch max {
		case fR:
			h = (fG - fB) / d
			if fG < fB {
				h += 6
			}
		case fG:
			h = (fB-fR)/d + 2
		case fB:
			h = (fR-fG)/d + 4
		}
		h /= 6
	}
	return
}

// Convert Go colors to RGB values in range [0,1).
func normalize(col color.Color) (r, g, b float64) {
	ri, gi, bi, _ := col.RGBA()
	r = float64(ri) / float64(0x10000)
	g = float64(gi) / float64(0x10000)
	b = float64(bi) / float64(0x10000)
	return
}

// averageImageColor gets the average RGB color from an image type by inspecting all pixels and determining the average
func averageImageColor(i image.Image) color.Color {
	var r, g, b uint32

	bounds := i.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pr, pg, pb, _ := i.At(x, y).RGBA()

			r += pr
			g += pg
			b += pb
		}
	}

	d := uint32(bounds.Dy() * bounds.Dx())

	r /= d
	g /= d
	b /= d

	return color.NRGBA{uint8(r / 0x101), uint8(g / 0x101), uint8(b / 0x101), 255}
}

func rearrangeAlbums(albums []album, x int, y int) {

	matrix := make([][]album, y)
	for i := 0; i < y; i++ {
		matrix[i] = make([]album, x)
	}

	row, col := 0, 0
	var dir = true
	l := 0

	for row < y && col < x {
		matrix[row][col] = albums[l]
		l++

		var newRow int
		var newCol int

		if dir {
			newRow = row + -1
			newCol = col + 1
		} else {
			newRow = row + 1
			newCol = col + -1
		}

		if newRow < 0 || newRow == y || newCol < 0 || newCol == x {
			if dir {
				if col == x-1 {
					row++
				}
				if col < x-1 {
					col++
				}
			} else {
				if row == y-1 {
					col++
				}
				if row < y-1 {
					row++
				}
			}
			dir = !dir
		} else {
			row = newRow
			col = newCol
		}

	}

	l = 0

	for i := 0; i < y; i++ {
		for j := 0; j < x; j++ {
			albums[l] = matrix[i][j]
			l++
		}
	}
}
