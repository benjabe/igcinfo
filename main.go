package main

import (
	"encoding/json"
	"fmt"
	"github.com/marni/goigc"
	"net/http"
	"os"
	"strings"
	"time"
)

type Meta struct {
	Uptime  string `json:"uptime"`
	Info    string `json:"info"`
	Version string `json:"version"`
}

type URLRequest struct {
	URL string `json:"url"`
}

type IDContainer struct {
	ID string `json:"id"`
}

type Track struct {
	ID    IDContainer
	Track igc.Track
}

type TrackMeta struct {
	H_date       time.Time `json:"H_date"`
	Pilot        string    `json:"pilot"`
	Glider       string    `json:"glider"`
	Glider_id    string    `json:"glider_id"`
	Track_length float64   `json:"track_length"`
}

type IDArray struct {
	IDs []string `json:"ids"`
}

var tracks map[string]igc.Track
var ids IDArray

var start time.Time
var meta Meta

var lastID int

func handlerAPI(w http.ResponseWriter, r *http.Request) {
	// return metadata for api
	http.Header.Add(w.Header(), "content-type", "application/json")
	elapsed := time.Since(start)
	meta.Uptime = fmt.Sprintf(
		"P%dDT%dH%dM%dS",
		int(elapsed.Seconds()/86400),
		int(elapsed.Hours())%24,
		int(elapsed.Minutes())%60,
		int(elapsed.Seconds())%60,
	)
	json.NewEncoder(w).Encode(meta)
}

func handlerIGC(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	switch r.Method {
	case "POST":
		// get igc from url and return id
		http.Header.Add(w.Header(), "content-type", "application/json")

		var urlReq URLRequest
		decoder := json.NewDecoder(r.Body)
		decoder.Decode(&urlReq)

		var track igc.Track
		var err error
		track, err = igc.ParseLocation(urlReq.URL)
		if err == nil {
			id := fmt.Sprintf("track%d", lastID)
			ids.IDs = append(ids.IDs, id)
			lastID++
			tracks[id] = track
			json.NewEncoder(w).Encode(IDContainer{id})
		} else {
			// malformed content
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "GET":
		if len(parts) > 5 {
			// get track by id
			track, ok := tracks[parts[4]]
			if !ok {
				// no such track
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			track.Task.Start = track.Points[0]
			track.Task.Finish = track.Points[len(track.Points)-1]
			track.Task.Turnpoints = track.Points[1 : len(track.Points)-2]

			if len(parts) == 6 {
				// return all metadata
				http.Header.Add(w.Header(), "content-type", "application/json")
				trackMeta := TrackMeta{
					track.Header.Date,
					track.Header.Pilot,
					track.Header.GliderType,
					track.Header.GliderID,
					track.Task.Distance(),
				}
				json.NewEncoder(w).Encode(trackMeta)
			} else if len(parts) == 7 {
				// return specific metadata
				switch parts[5] {
				case "pilot":
					fmt.Fprintf(w, track.Header.Pilot)
				case "glider":
					fmt.Fprintf(w, track.Header.GliderType)
				case "glider_id":
					fmt.Fprintf(w, track.Header.GliderID)
				case "track_length":
					fmt.Fprintf(w, "%f", track.Task.Distance())
				case "H_date":
					fmt.Fprintf(w, "%v", track.Header.Date)
				default:
					http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				}
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		} else {
			// return all ids
			http.Header.Add(w.Header(), "content-type", "application/json")
			json.NewEncoder(w).Encode(ids)
		}
	}
}

func determineListenAddress() string {
	port := os.Getenv("PORT")
	return ":" + port
}

func main() {
	tracks = make(map[string]igc.Track)
	ids = IDArray{make([]string, 0)}
	lastID = 0
	start = time.Now()

	meta.Info = "Service for IGC tracks."
	meta.Version = "v1"

	http.HandleFunc("/igcinfo/api/", handlerAPI)
	http.HandleFunc("/igcinfo/api/igc/", handlerIGC)
	http.ListenAndServe(determineListenAddress(), nil)
}
