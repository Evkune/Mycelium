package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type MeteoFranceResponse struct {
	Temperature float64 `json:"t"`
	Humidite    float64 `json:"u"`
}

func main() {
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
		return
	}

	// Ajouter l'en-tête d'authentification avec le token
	req.Header.Set("apikey", token)

	// Effectuer la requête HTTP
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Erreur lors de l'envoi de la requête HTTP:", err)
		return
	}
	defer resp.Body.Close()

	// Lire le corps de la réponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Erreur lors de la lecture du corps de la réponse:", err)
		return
	}

	// Vérifier si la réponse est valide
	if resp.StatusCode != http.StatusOK {
		fmt.Println("La requête a échoué avec le code de statut:", resp.StatusCode)
		fmt.Println("Réponse:", string(body))
		return
	}

	// Décoder le JSON de la réponse
	var response []MeteoFranceResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Erreur lors du décodage du JSON de la réponse:", err)
		return
	}

	// Afficher les données récupérées
	if len(response) == 0 {
		fmt.Println("Aucune donnée n'a été retournée pour cette station")
		return
	}

	temperatureCelsius := response[0].Temperature - 273.15

	fmt.Printf("Température: %.2f°C\n", temperatureCelsius)
	fmt.Printf("Humidité: %.2f%%\n", response[0].Humidite)
}
