**This code is used to set the measurement interval for the Milesight EM300-TH sensor.**

## Principe

- Prend dans un premiers temps les variables environnements déclarées dans le fichier .env

- Va ensuite transformer la période (récupérée dans .env) dans un code hexadecimal correspondant au downlink du capteur EM300-TH (voir datasheet du capteur).

- Envoie ensutie ce code hexadecimal sur l'api de chirpstack, et ainsi dès le prochain envoie de données du capteur on va changer son intervalle de mesure.


## Request : 
	pip install python-dotenv
	pip install chirpstack-api


## Step 1 : 
	Create a .env file and configure your environment variables

## Step 2 : 
	Add your variables like this in the file
		
		# APi port of chirpstack
		SERVER=localhost:8082

		# The DevEUI for which you want to enqueue the downlink.
		DEV_EUI="24e124136d358401"

		# Token to retrieved  in the chirpstack web interface
		API_TOKEN="eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJjaGlycHN0YWNrIiwiaXNzIjoiY2hpcnBzdGFjayIsInN1YiI6ImMzZDY4Mzk2LWIyYTUtNDc2My05YmYwLTU1NWMyYTE2ZDJkMyIsInR5cCI6ImtleSJ9.fhHJOxcF0yI7kAmcVzDTKB0SFmBAEup-dFjiBv74RdQ"
		
		# Data capture period in seconds (min 60s, support only minutes. So only use multiples of 60)
		PERIOD_SEC="180


## Step 3 : 
	python3 main.py



## Note for other sensors : 
	line 7 to 17 aand line 29 to end will not be changed
	
	line 19 to 26 are used to transform seconds into hexadecimal, then to invert the 2 payload bytes, and add them to ff03. This is taken from the Milesight Em300-TH sensor documentation.
	For any other sensor, you'll need to check the code and make sure you're sending the right payload.
