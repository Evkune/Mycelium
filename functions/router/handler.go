package function

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"log"
	"net/http"
	"os"
)

//"strings"

var (
	mqttBroker = os.Getenv("MQTT_URL")
	clientID   = os.Getenv("MQTT_CLIENTID")
)

// Fonction pour envoyer un message MQTT
func SendMQTT(messageJSON []byte, topic string) {
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(mqttBroker)
	mqttOpts.SetClientID(clientID)

	client := mqtt.NewClient(mqttOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	token := client.Publish(topic, 0, false, string(messageJSON))
	token.Wait()

	fmt.Printf("Message publié sur le topic %s: %s\n", topic, messageJSON)

	client.Disconnect(250)
}

// Middleware pour intercepter la requête HTTP
func Handle(w http.ResponseWriter, r *http.Request) {
	// Vous pouvez récupérer le message et le topic depuis la requête HTTP
	messageJSON := []byte(r.URL.Query().Get("message"))
	topic := r.URL.Query().Get("topic")

	if messageJSON == nil || topic == "" {
		http.Error(w, "Message ou topic manquant", http.StatusBadRequest)
		return
	}

	// Appeler la fonction pour envoyer le message MQTT
	SendMQTT(messageJSON, topic)

	// Répondre à la requête HTTP
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := fmt.Sprintf(`{"status": "Message envoyé sur le topic %s"}`, topic)
	w.Write([]byte(response))
}
