package function

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
)

type Item struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PubDate     string `json:"pubDate"`
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Notification struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("running\n")

	var notif Notification

	// Decode the JSON message
	err := json.NewDecoder(r.Body).Decode(&notif)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Mocking JSON data
	item := Item{
		Title:       notif.Title,
		Description: notif.Description,
		PubDate:     "",
	}

	// Encode JSON
	jsonBytes, err := json.Marshal(item)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	// Send POST request
	_, err = http.Post("http://10.42.0.1:8082/add_post", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		http.Error(w, "Error sending POST request", http.StatusInternalServerError)
		return
	}
	// Send GET request
	response, err := http.Get("http://10.42.0.1:8082/rss")
	if err != nil {
		http.Error(w, "Error sending GET request", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	// Parse XML response
	var rss RSS
	if err := xml.NewDecoder(response.Body).Decode(&rss); err != nil {
		http.Error(w, "Error decoding XML", http.StatusInternalServerError)
		return
	}

	//fmt.Printf("%+v\n", rss)
	fmt.Printf("Réussite ! \n")
}
