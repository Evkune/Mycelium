## Fonctions-Extraction 
	- Fonctions OpenFaas en Go 
	- Permet de récupérer les JSON décodés des capteurs (envoyé par Chirpstack sur MQTT). 
	- Récupère les données importantes, les stockent dans InfluxDB et va envoyer ces données au fonctions de traitement à lancer

## Fonctions-Traitement  
    - Fonctions OpenFaas en Go ou Python(downlink EM300)
	- Fonctions scénario
	- Envoie le résultat du scénario sur le topic MQTT notification

## Fonctions-Temporelles  
    - Fonctions OpenFaas en Go
	- Fonctions s'éxécutant à une heure ou un intervalle précis (tous les jours 19h, ou toutes les 5min, etc...)
	- Dans la Yaml de la fonction, dans schedule, on est en heure UTC+0. Pour configurer voir (https://www.openfaas.com/blog/schedule-your-functions/)

## Fonctions-Standard 
	- Fonctions Go ou Python
	- Ce ne sont pas des fonctions Openfaas, elle se lance à la main
	- Sont là pour l'exemple et pour tester sans avoir à modifier sans cesse des fonctions openfaas	

## Fonctions-VPS 
	- Fonctions Go
	- Fonctions OpenFaas qui tourne sur le VPS


## Serveur RSS
	- Contenue de ce qu'il y a dans l'image docker myceliumir/rssmycelium
