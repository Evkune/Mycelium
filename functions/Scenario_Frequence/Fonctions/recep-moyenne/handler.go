package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type Message struct {
	Moyenne_Temp     float64 `json:"Moyennes_Temp"`
	Moyenne_Humi     float64 `json:"Moyennes_Humi"`
	Moyenne_CO2      float64 `json:"Moyennes_CO2"`
	Moyenne_Pression float64 `json:"Moyennes_Pression"`
}

func Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Fonction Lancée !!\n")
	var input []byte

	//Initialize InfluxDB
	influxToken := os.Getenv("INFLUXDB_TOKEN")
	influxURL := os.Getenv("INFLUXDB_URL")
	influxClient := influxdb2.NewClient(influxURL, influxToken)

	if r.Body != nil {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		input = body
	}

	// Decode the JSON message
	var msg Message
	err := json.Unmarshal(input, &msg)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Write temperature and humidity in the response
	response := fmt.Sprintf("MTemperature: %.1f°C, MHumidity: %.1f%%, MPressure : %.1fhPa, MCO2 : %.1fppm", msg.Moyenne_Temp, msg.Moyenne_Humi, msg.Moyenne_Pression, msg.Moyenne_CO2)

	//Write in the InfluxDB Database
	org := "MyceliumVPS"
	bucket := "Moyennes-VM"
	writeAPI := influxClient.WriteAPIBlocking(org, bucket)

	tags1 := map[string]string{
		"Capteurs": "Milesight-EM500-CO2",
	}
	fields1 := map[string]interface{}{
		"co2":      msg.Moyenne_CO2,
		"pressure": msg.Moyenne_Pression,
	}
	point1 := write.NewPoint("Data", tags1, fields1, time.Now())

	tags2 := map[string]string{
		"Capteurs": "Milesight-EM300-TH",
	}
	fields2 := map[string]interface{}{
		"temperature": msg.Moyenne_Temp,
		"humidite":    msg.Moyenne_Humi,
	}
	point2 := write.NewPoint("Data", tags2, fields2, time.Now())

	if err := writeAPI.WritePoint(context.Background(), point1); err != nil {
		http.Error(w, "WriteAPI.WritePoint Error", http.StatusInternalServerError)
		fmt.Printf("Error writing data to InfluxDB: %v\n", err)
		return
	}
	if err := writeAPI.WritePoint(context.Background(), point2); err != nil {
		http.Error(w, "WriteAPI.WritePoint Error", http.StatusInternalServerError)
		fmt.Printf("Error writing data to InfluxDB: %v\n", err)
		return
	}

	fmt.Printf("Data: %s\n", response)

	w.WriteHeader(http.StatusOK)
	return

}
