package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Message struct {
	Object struct {
		Pressure    float64 `json:"pressure"`
		Temperature float64 `json:"temperature"`
		Humidity    float64 `json:"humidity"`
		CO2         float64 `json:"co2"`
	} `json:"object"`
}

// Json que l'on envoie au topic notification
type Notification struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Fonction pour envoyer un message MQTT à notification
func SendMQTT(messageJSON []byte, topic string, url string) {
	// Envoyer le message JSON via MQTT
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(url)
	mqttOpts.SetClientID("extraction-milesight-fct")

	// Création du client MQTT
	mqttClient := mqtt.NewClient(mqttOpts)
	if mqttToken := mqttClient.Connect(); mqttToken.Wait() && mqttToken.Error() != nil {
		log.Fatal(mqttToken.Error())
	}

	// Publication du message sur le topic
	mqttToken := mqttClient.Publish(topic, 0, false, string(messageJSON))
	mqttToken.Wait()

	fmt.Printf("Message publié sur le topic %s: %s\n", topic, messageJSON)

	time.Sleep(2 * time.Second)
	// Déconnexion du client MQTT
	mqttClient.Disconnect(250)
}

func Handle(w http.ResponseWriter, r *http.Request) {
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

	//Write in the InfluxDB Database
	org := "Mycelium"
	bucket := "Mesure-Capteurs-OSUR"
	writeAPI := influxClient.WriteAPIBlocking(org, bucket)

	tags := map[string]string{
		"Capteurs": "Milesight-EM500-CO2",
	}
	fields := map[string]interface{}{
		"temperature": msg.Object.Temperature,
		"humidite":    msg.Object.Humidity,
		"co2":         msg.Object.CO2,
		"pressure":    msg.Object.Pressure,
	}
	point := write.NewPoint("Data", tags, fields, time.Now())

	if err := writeAPI.WritePoint(context.Background(), point); err != nil {
		http.Error(w, "WriteAPI.WritePoint Error", http.StatusInternalServerError)
		fmt.Printf("Error writing data to InfluxDB: %v\n", err)
		return
	}

	// Write temperature and humidity in the response
	response := fmt.Sprintf("Temperature: %.1f°C, Humidity: %.1f%%, Pressure : %.1fhPa, CO2 : %.1fppm", msg.Object.Temperature, msg.Object.Humidity, msg.Object.Pressure, msg.Object.CO2)
	fmt.Printf("Data: %s\n", response)

	// Créer une structure de notification
	title := "Données EM500-CO2 OSUR"
	notification := Notification{
		Title:       title,
		Description: response,
	}

	// Convertir la notification en JSON
	messageJSON, err := json.Marshal(notification)
	if err != nil {
		http.Error(w, "Failed to encode notification to JSON", http.StatusInternalServerError)
		return
	}

	url1 := "tcp://10.133.33.52:1883"
	topic1 := "notification"
	SendMQTT(messageJSON, topic1, url1)

	w.WriteHeader(http.StatusOK)
	return

}
