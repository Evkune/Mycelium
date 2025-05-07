# Introduction

Ce fichier contient les requêtes *curl* à effectuer pour aller simuler un envoi de donnée des 2 capteurs (Milesight et EM500), ici c'est simulé en envoyant sur la gateway Openfaas du VPS.

Si vous souhaitez simuler sur la VM : changer les IP par <code>127.0.0.1:8080</code>

# Requêtes Capteur Milesight

curl -X POST http://10.43.35.16:8080/function/extraction-milesight/ -H "Content-Type: application/json" -d '{
  "Object": {
	  "Temperature": 55,
	  "Humidity": 50
	  }
}'

curl -X POST http://10.43.35.16:8080/function/extraction-milesight/ -H "Content-Type: application/json" -d '{
  "Object": {
	  "Temperature": 25,
	  "Humidity": 60
	  }
}'


# Requêtes Capteur EM500 de l'OSUR 

curl -X POST http://10.43.35.16:8080/function/extraction-em500-osur/ -H "Content-Type: application/json" -d '{
  "Object": {
	  "Temperature": 30,
	  "Humidity": 30,
	  "CO2": 10,
	  "Pressure": 1
	  }
}'

curl -X POST http://10.43.35.16:8080/function/extraction-em500-osur/ -H "Content-Type: application/json" -d '{
  "Object": {
	  "Temperature": 20,
	  "Humidity": 20,
	  "CO2": 15,
	  "Pressure": 2
	  }
}'


# Requêtes pour effectuer la fonction de moyenne du VPS

curl -X POST http://10.43.35.16:8080/function/moyennes-to-vps/
