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
		Temperature float64 `json:"temperature"`
		Humidity    float64 `json:"humidity"`
	} `json:"object"`
}

// Fonction pour envoyer un message MQTT à notification
func SendMQTT(messageJSON []byte, topic string) {
	// Envoyer le message JSON via MQTT
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker("tcp://192.168.122.61:1883")
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

	// Write temperature and humidity in the response
	response := fmt.Sprintf("Temperature: %.1f°C, Humidity: %.1f%%", msg.Object.Temperature, msg.Object.Humidity)

	//Write in the InfluxDB Database
	org := "Mycelium"
	bucket := "Mesure-Milesight-Sensor"
	writeAPI := influxClient.WriteAPIBlocking(org, bucket)

	tags := map[string]string{
		"Capteurs": "MilesightSensor",
	}
	fields := map[string]interface{}{
		"temperature": msg.Object.Temperature,
		"humidite":    msg.Object.Humidity,
	}
	point := write.NewPoint("Temperature & Humidité", tags, fields, time.Now())

	if err := writeAPI.WritePoint(context.Background(), point); err != nil {
		http.Error(w, "WriteAPI.WritePoint Error", http.StatusInternalServerError)
		fmt.Printf("Error writing data to InfluxDB: %v\n", err)
		return
	}

	fmt.Printf("Data: %s\n", response)

	// Publication du message sur le topic
	messageJSON, err := json.Marshal(msg.Object)
	if err != nil {
		http.Error(w, "Failed to encode sensor data to JSON", http.StatusInternalServerError)
		return
	}

	topic1 := "fonction_chaleur"
	topic2 := "fonction_compare_api"

	SendMQTT(messageJSON, topic1)
	time.Sleep(2 * time.Second)
	SendMQTT(messageJSON, topic2)

	w.WriteHeader(http.StatusOK)
	return
}
