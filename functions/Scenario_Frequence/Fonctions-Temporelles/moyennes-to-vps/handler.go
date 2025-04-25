package function

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Fonction pour envoyer un message MQTT
func SendMQTT(messageJSON []byte, topic string) {
	// Envoyer le message JSON via MQTT
	mqttOpts := mqtt.NewClientOptions()
	mqttUrl := os.Getenv("MQTT_URL")
	mqttClientID := os.Getenv("MQTT_CLIENTID")
	mqttOpts.AddBroker(mqttUrl)
	mqttOpts.SetClientID(mqttClientID)

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

//##############################################################################################

// Fonction pour récupérer la moyenne de la température  et de l'humidité pour la journée précédente pour le capteur Milesight EM300-TH
func getMoyenneMilesghtEM300TH() (float64, float64, error) {
	influxToken := os.Getenv("INFLUXDB_TOKEN")
	influxURL := os.Getenv("INFLUXDB_URL")
	client := influxdb2.NewClientWithOptions(influxURL, influxToken, influxdb2.DefaultOptions().SetBatchSize(1))

	// Définition des informations sur l'organisation et le bucket
	org := "Mycelium"
	bucket := "Mesure-Milesight-Sensor"

	// Création de la requête pour récupérer les données de température des 12 dernières heures
	queryAPI := client.QueryAPI(org)
	query_temp := fmt.Sprintf(`from(bucket: "%s")
	|> range(start: -12h)
	|> filter(fn: (r) => r["_measurement"] == "Temperature & Humidité")
	|> filter(fn: (r) => r["Capteurs"] == "MilesightSensor")
	|> filter(fn: (r) => r["_field"] == "temperature")
	|> aggregateWindow(every: 1m, fn: last, createEmpty: false)
	|> yield(name: "last")`, bucket)
	query_hum := fmt.Sprintf(`from(bucket: "%s")
	|> range(start: -12h)
	|> filter(fn: (r) => r["_measurement"] == "Temperature & Humidité")
	|> filter(fn: (r) => r["Capteurs"] == "MilesightSensor")
	|> filter(fn: (r) => r["_field"] == "humidite")
	|> aggregateWindow(every: 1m, fn: last, createEmpty: false)
	|> yield(name: "last")`, bucket)

	// Exécution des requêtes
	result_temp, err := queryAPI.Query(context.Background(), query_temp)
	if err != nil {
		return 0, 0, fmt.Errorf("Error querying data from InfluxDB: %v", err)
	}
	result_hum, err := queryAPI.Query(context.Background(), query_hum)
	if err != nil {
		return 0, 0, fmt.Errorf("Error querying data from InfluxDB: %v", err)
	}

	// Initialisation des variables pour le calcul de la moyenne
	var totalTemperature float64
	var totalRecordsTemp int
	var totalHumidite float64
	var totalRecordsHum int

	// Parcourir les résultats et calculer la somme des températures
	for result_temp.Next() {
		record := result_temp.Record()
		temperature := record.ValueByKey("_value")
		if temperature != nil {
			totalTemperature += temperature.(float64)
			totalRecordsTemp++
		}
	}
	// Parcourir les résultats et calculer la somme des humidités
	for result_hum.Next() {
		record := result_hum.Record()
		humidite := record.ValueByKey("_value")
		if humidite != nil {
			totalHumidite += humidite.(float64)
			totalRecordsHum++
		}
	}

	// Vérifier les erreurs de fin
	if (result_temp.Err() != nil) || (result_hum.Err() != nil) {
		return 0, 0, fmt.Errorf("Error processing query result: %v", result_temp.Err())
	}

	// Vérifier si des enregistrements ont été trouvés
	if totalRecordsTemp == 0 || totalRecordsHum == 0 {
		return 0, 0, fmt.Errorf("No data records found in the last 12 hours")
	}

	// Calculer les moyennes
	MTemperature := totalTemperature / float64(totalRecordsTemp)
	MHumidite := totalHumidite / float64(totalRecordsHum)

	return MTemperature, MHumidite, nil
}

//##############################################################################################

// Fonction pour récupérer la moyenne de la pression atmosphérique et du taux de CO2 pour la journée précédente pour le capteur Milesight EM500 CO2
func getMoyenneMilesghtEM500CO2() (float64, float64, error) {
	influxToken := os.Getenv("INFLUXDB_TOKEN")
	influxURL := os.Getenv("INFLUXDB_URL")
	client := influxdb2.NewClientWithOptions(influxURL, influxToken, influxdb2.DefaultOptions().SetBatchSize(1))

	// Définition des informations sur l'organisation et le bucket
	org := "Mycelium"
	bucket := "Mesure-Capteurs-OSUR"

	// Création de la requête pour récupérer les données de température des 12 dernières heures
	queryAPI := client.QueryAPI(org)
	query_co2 := fmt.Sprintf(`from(bucket: "%s")
			|> range(start: -12h)
			|> filter(fn: (r) => r["_measurement"] == "Data")
			|> filter(fn: (r) => r["Capteurs"] == "Milesight-EM500-CO2")
			|> filter(fn: (r) => r["_field"] == "co2")
			|> aggregateWindow(every: 1m, fn: last, createEmpty: false)
			|> yield(name: "last")`, bucket)
	query_pressure := fmt.Sprintf(`from(bucket: "%s")
			|> range(start: -12h)
			|> filter(fn: (r) => r["_measurement"] == "Data")
			|> filter(fn: (r) => r["Capteurs"] == "Milesight-EM500-CO2")
			|> filter(fn: (r) => r["_field"] == "pressure")
			|> aggregateWindow(every: 1m, fn: last, createEmpty: false)
			|> yield(name: "last")`, bucket)

	// Exécution des requêtes
	result_co2, err := queryAPI.Query(context.Background(), query_co2)
	if err != nil {
		return 0, 0, fmt.Errorf("Error querying data from InfluxDB: %v", err)
	}
	result_pressure, err := queryAPI.Query(context.Background(), query_pressure)
	if err != nil {
		return 0, 0, fmt.Errorf("Error querying data from InfluxDB: %v", err)
	}

	// Initialisation des variables pour le calcul de la moyenne
	var totalCO2 float64
	var totalRecordsCO2 int
	var totalPressure float64
	var totalRecordsPressure int

	// Parcourir les résultats et calculer la somme des températures
	for result_co2.Next() {
		record := result_co2.Record()
		co2 := record.ValueByKey("_value")
		if co2 != nil {
			totalCO2 += co2.(float64)
			totalRecordsCO2++
		}
	}
	// Parcourir les résultats et calculer la somme des humidités
	for result_pressure.Next() {
		record := result_pressure.Record()
		humidite := record.ValueByKey("_value")
		if humidite != nil {
			totalPressure += humidite.(float64)
			totalRecordsPressure++
		}
	}

	// Vérifier les erreurs de fin
	if (result_co2.Err() != nil) || (result_pressure.Err() != nil) {
		return 0, 0, fmt.Errorf("Error processing query result")
	}

	// Vérifier si des enregistrements ont été trouvés
	if totalRecordsCO2 == 0 || totalRecordsPressure == 0 {
		return 0, 0, fmt.Errorf("No data records found in the last 12 hours")
	}

	// Calculer les moyennes
	MCO2 := totalCO2 / float64(totalRecordsCO2)
	MPressure := totalPressure / float64(totalRecordsPressure)

	return MCO2, MPressure, nil
}

//##############################################################################################

// Json que l'on envoie
type Moyennes struct {
	Moyenne_Temp     float64 `json:"Moyennes_Temp"`
	Moyenne_Humi     float64 `json:"Moyennes_Humi"`
	Moyenne_CO2      float64 `json:"Moyennes_CO2"`
	Moyenne_Pression float64 `json:"Moyennes_Pression"`
}

// Fonction principale
func Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Fonction Lancée !!\n")

	// Recuperer la moyenne des temperatures et humidité de la journée
	MoyenneTemp, MoyenneHumi, err := getMoyenneMilesghtEM300TH()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	// Recuperer la moyenne du CO2 et pression de la journée
	MoyenneCO2, MoyennePressure, err := getMoyenneMilesghtEM500CO2()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	/*fmt.Printf("MoyenneT : %.1f\n", MoyenneTemp)
	fmt.Printf("MoyenneH : %.1f\n", MoyenneHumi)
	fmt.Printf("MoyenneT : %.1f\n", MoyenneCO2)
	fmt.Printf("MoyenneH : %.1f\n", MoyennePressure)*/

	// Créer une structure de Moyennes
	moy := Moyennes{
		Moyenne_Temp:     MoyenneTemp,
		Moyenne_Humi:     MoyenneHumi,
		Moyenne_CO2:      MoyenneCO2,
		Moyenne_Pression: MoyennePressure,
	}

	// Convertir structure en JSON
	moyJSON, err := json.Marshal(moy)
	if err != nil {
		http.Error(w, "Failed to encode notification to JSON", http.StatusInternalServerError)
		return
	}

	//Envoyer le JSON
	topic := "moyennestovps"
	SendMQTT(moyJSON, topic)
}
