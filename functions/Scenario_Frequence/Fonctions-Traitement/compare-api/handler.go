package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Reception struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

type MeteoFranceResponse struct {
	Temperature float64 `json:"t"`
	Humidite    float64 `json:"u"`
}

// Json que l'on envoie au topic notification
type Notification struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Fonction pour envoyer un message MQTT à notification
func SendMQTT(messageJSON []byte) {
	// Envoyer le message JSON via MQTT
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker("tcp://10.133.33.52:1883")
	mqttOpts.SetClientID("compareApiData")

	// Création du client MQTT
	mqttClient := mqtt.NewClient(mqttOpts)
	if mqttToken := mqttClient.Connect(); mqttToken.Wait() && mqttToken.Error() != nil {
		log.Fatal(mqttToken.Error())
	}

	// Publication du message sur le topic
	topic := "notification"
	mqttToken := mqttClient.Publish(topic, 0, false, string(messageJSON))
	mqttToken.Wait()

	fmt.Printf("Message publié sur le topic %s: %s\n", topic, messageJSON)

	// Déconnexion du client MQTT
	mqttClient.Disconnect(250)
}

func getAPI() (float64, float64) {
	/* Récupérer dans https://portail-api.meteofrance.fr, il y a moyen d'avoir la liste des stations. 35281001 ==> Rennes St-Jaques */
	idStation := "35281001"

	/* Récupérer dans https://portail-api.meteofrance.fr/web/fr/api/DonneesPubliquesObservation */
	url := fmt.Sprintf("https://public-api.meteofrance.fr/public/DPObs/v1/station/infrahoraire-6m?id_station=%s&format=json", idStation)

	/* Le token est censé durée 999 999 999 secondes. Si il périme, aller sur https://portail-api.meteofrance.fr/web/fr/api/DonneesPubliquesObservation
	puis configurer API (il faudra créer un compte) et regénérer un token */
	token := "eyJ4NXQiOiJZV0kxTTJZNE1qWTNOemsyTkRZeU5XTTRPV014TXpjek1UVmhNbU14T1RSa09ETXlOVEE0Tnc9PSIsImtpZCI6ImdhdGV3YXlfY2VydGlmaWNhdGVfYWxpYXMiLCJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJwaWVyb3V2ckBjYXJib24uc3VwZXIiLCJhcHBsaWNhdGlvbiI6eyJvd25lciI6InBpZXJvdXZyIiwidGllclF1b3RhVHlwZSI6bnVsbCwidGllciI6IlVubGltaXRlZCIsIm5hbWUiOiJEZWZhdWx0QXBwbGljYXRpb24iLCJpZCI6MTExODIsInV1aWQiOiJiOTk0NGEyOC1kZDI1LTRiODYtODU1Ny0xNTQ5ZTM2ODAyZDkifSwiaXNzIjoiaHR0cHM6XC9cL3BvcnRhaWwtYXBpLm1ldGVvZnJhbmNlLmZyOjQ0M1wvb2F1dGgyXC90b2tlbiIsInRpZXJJbmZvIjp7IjUwUGVyTWluIjp7InRpZXJRdW90YVR5cGUiOiJyZXF1ZXN0Q291bnQiLCJncmFwaFFMTWF4Q29tcGxleGl0eSI6MCwiZ3JhcGhRTE1heERlcHRoIjowLCJzdG9wT25RdW90YVJlYWNoIjp0cnVlLCJzcGlrZUFycmVzdExpbWl0IjowLCJzcGlrZUFycmVzdFVuaXQiOiJzZWMifX0sImtleXR5cGUiOiJQUk9EVUNUSU9OIiwic3Vic2NyaWJlZEFQSXMiOlt7InN1YnNjcmliZXJUZW5hbnREb21haW4iOiJjYXJib24uc3VwZXIiLCJuYW1lIjoiRG9ubmVlc1B1YmxpcXVlc09ic2VydmF0aW9uIiwiY29udGV4dCI6IlwvcHVibGljXC9EUE9ic1wvdjEiLCJwdWJsaXNoZXIiOiJiYXN0aWVuZyIsInZlcnNpb24iOiJ2MSIsInN1YnNjcmlwdGlvblRpZXIiOiI1MFBlck1pbiJ9LHsic3Vic2NyaWJlclRlbmFudERvbWFpbiI6ImNhcmJvbi5zdXBlciIsIm5hbWUiOiJEb25uZWVzUHVibGlxdWVzQ2xpbWF0b2xvZ2llIiwiY29udGV4dCI6IlwvcHVibGljXC9EUENsaW1cL3YxIiwicHVibGlzaGVyIjoiYWRtaW5fbWYiLCJ2ZXJzaW9uIjoidjEiLCJzdWJzY3JpcHRpb25UaWVyIjoiNTBQZXJNaW4ifV0sInRva2VuX3R5cGUiOiJhcGlLZXkiLCJpYXQiOjE3MTMxOTY3NDEsImp0aSI6IjJkNGE5MmJjLTkxMGQtNDU1Yy1iOTJkLTIyMjgyOTZkYTljNCJ9.MHWe_-0WH9IUKw-i9MncgRcI9CWUwvREhaqdZGb0aBLZ8tjjZx1LGoWMVlP1bP-sAVXZTzpF1BRXQGgml3nU0k0iCUpYhIEO_vhAO6He27K-30vnNDRRr7CR2AJXsNMEkhkgJmDSEGbz4SSuJUJzMXZ2eq7zQX58qyE7LCcqzjVUUUgWfp2CW-7TkWAYvM6Sjvr_-GBY4kP85xYLrIgHMovhVf1ASCGlKeV7-h1b_e2h_2L00EnTB2CEOWVV3bjLATaSN4z-i6HsrK4C5-Lc0kJWZfJGSfLwD-GtlgpGVTNF2cg2wchDEEtxhXfHjTtYlacf1CoPE_kblgKXzxXBBA==" // Remplacez "votre_token" par votre vrai token

	// Créer une nouvelle requête HTTP GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Erreur lors de la création de la requête HTTP:", err)
		return 0, 0
	}

	// Ajouter l'en-tête d'authentification avec le token
	req.Header.Set("apikey", token)

	// Effectuer la requête HTTP
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Erreur lors de l'envoi de la requête HTTP:", err)
		return 0, 0
	}
	defer resp.Body.Close()

	// Lire le corps de la réponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Erreur lors de la lecture du corps de la réponse:", err)
		return 0, 0
	}

	// Vérifier si la réponse est valide
	if resp.StatusCode != http.StatusOK {
		fmt.Println("La requête a échoué avec le code de statut:", resp.StatusCode)
		fmt.Println("Réponse:", string(body))
		return 0, 0
	}

	// Décoder le JSON de la réponse
	var response []MeteoFranceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Erreur lors du décodage du JSON de la réponse:", err)
		return 0, 0
	}

	// Afficher les données récupérées
	if len(response) == 0 {
		fmt.Println("Aucune donnée n'a été retournée pour cette station")
		return 0, 0
	}

	temperatureCelsius := response[0].Temperature - 273.15

	fmt.Printf("Température: %.2f°C\n", temperatureCelsius)
	fmt.Printf("Humidité: %.2f%%\n", response[0].Humidite)

	return temperatureCelsius, response[0].Humidite

}

func ComparerCapteurAPI(tempCap float64, humCap float64, tempApi float64, humApi float64) bool {
	if tempApi-5 < tempCap && tempCap < tempApi+5 && humApi-20 < humCap && humCap < humApi+20 {
		return true
	} else {
		return false
	}
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
	temperatureCap := recep.Temperature
	humidityCap := recep.Humidity

	// Récupérer donner temperature et humidité de l'API
	temperatureAPI, humiditeAPI := getAPI()

	// Comparer les valeurs
	var coherence_valeur string
	if ComparerCapteurAPI(temperatureCap, humidityCap, temperatureAPI, humiditeAPI) {
		coherence_valeur = "Les valeurs du capteur sont cohérentes avec les valeurs de l'API"
	} else {
		coherence_valeur = "Valeurs non cohérente ! Vérifiez capteur !"
	}

	// Créer une structure de notification
	title := "Cohérence données EM300-TH"
	notification := Notification{
		Title:       title,
		Description: coherence_valeur,
	}

	// Convertir la notification en JSON
	messageJSON, err := json.Marshal(notification)
	if err != nil {
		http.Error(w, "Failed to encode notification to JSON", http.StatusInternalServerError)
		return
	}

	SendMQTT(messageJSON)

	fmt.Printf("Message envoyé. Fin de fonction")

}
