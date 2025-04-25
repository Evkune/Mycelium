package function

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Fonction pour calculer l'indice humidex
func Humidex(temperature, humidity float64) float64 {
	// Calcul de la pression de vapeur d'eau (e)
	e := 6.112 * math.Exp((17.67*temperature)/(temperature+243.5)) * (humidity / 100.0)

	// Calcul de l'indice humidex
	return temperature + (5.0/9.0)*(e-10.0)
}

// Fonction pour envoyer un message MQTT à notification
func SendMQTT(messageJSON []byte, topic string, url string) {
	// Envoyer le message JSON via MQTT
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(url)
	mqttOpts.SetClientID("fonction_chaleur")

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

// Fonction pour determiner la sensation provoquée en fonction de l'humidex
func HumidexSensation(humidex float64) string {
	moreDataStr := os.Getenv("MORE_DATA")
	moreData, err := strconv.ParseBool(moreDataStr)
	if err != nil {
		panic(err)
	}
	switch {
	case humidex < 15:
		if moreData {
			changeSensorPeriode()
		}
		return "Sensation de frais ou de froid"
	case humidex >= 15 && humidex < 29:
		if moreData {
			changeSensorPeriode()
		}
		return "Sensation de confort"
	case humidex >= 29 && humidex < 34:
		if moreData {
			changeSensorPeriode()
		}
		return "Chaleur : sensation d'inconfort"
	case humidex >= 34 && humidex < 39:
		if !moreData {
			changeSensorPeriode()
		}
		return "Chaleur : sensation d'inconfort important"
	case humidex >= 39 && humidex < 45:
		if !moreData {
			changeSensorPeriode()
		}
		return "Forte Chaleur : Danger"
	case humidex >= 45 && humidex < 53:
		if !moreData {
			changeSensorPeriode()
		}
		return "Très forte chaleur : Danger extrême"
	case humidex > 54:
		if !moreData {
			changeSensorPeriode()
		}
		return "Coup de chaleur imminent (danger de mort)"
	}
	return "Erreur HumidexSensation"
}

// Fonction pour changer la période du capteur EM300 afin de faire plus ou moins de donner en fonction du cas (voir fonction humidex pour les cas)
func changeSensorPeriode() {
	moreDataStr := os.Getenv("MORE_DATA")
	moreData, err := strconv.ParseBool(moreDataStr)
	moreData = !moreData
	os.Setenv("MORE_DATA", strconv.FormatBool(moreData))

	var periode int
	if moreData {
		periode = 120
	} else {
		periode = 900
	}

	// Créer une structure de ChangePeriode
	changeP := ChangePeriode{
		Periode: periode,
	}

	// Convertir le ChangePeriode en JSON
	changePeriodeJSON, err := json.Marshal(changeP)
	if err != nil {
		log.Fatal(err)
	}

	//Envoyer le JSON
	url2 := "tcp://192.168.122.61:1883"
	topic2 := "EM300TH-changePeriode"
	SendMQTT(changePeriodeJSON, topic2, url2)
}

// Json que l'on reçoit en début
type Reception struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

// Json pour changer la période
type ChangePeriode struct {
	Periode int `json:"periode"`
}

// Json que l'on envoie au topic notification
type Notification struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func Handle(w http.ResponseWriter, r *http.Request) {
	var recep Reception

	// Decode the JSON message
	err := json.NewDecoder(r.Body).Decode(&recep)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Extract temperature and humidity from the message
	temperature := recep.Temperature
	humidity := recep.Humidity

	// Determine l'humidex et la sensation engendrée
	humidex := Humidex(temperature, humidity)
	sensation := HumidexSensation(humidex)

	//Resultat Envoyer Json a fonction notif
	fmt.Printf("Pour une température de %.1f°C et une humidité de %.1f%%, l'humidex est %.1f.\nSensation : %s\n", temperature, humidity, humidex, sensation)

	title := "humidex"
	msg := fmt.Sprintf("Temperature : %1.f°C  ,Humidité : %1.f%%  ,Humidex : %1.f , Sensation :  %s", temperature, humidity, humidex, sensation)

	// Créer une structure de notification
	notification := Notification{
		Title:       title,
		Description: msg,
	}

	// Convertir la notification en JSON
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		http.Error(w, "Failed to encode notification to JSON", http.StatusInternalServerError)
		return
	}

	//Envoyer le JSON
	url1 := "tcp://10.133.33.52:1883"
	topic1 := "notification"
	SendMQTT(notificationJSON, topic1, url1)

}
