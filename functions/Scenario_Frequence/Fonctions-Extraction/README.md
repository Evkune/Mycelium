# extraction-em500-osur

- Extrait les données du capteur EM500-CO2 venant du broker de l'OSUR  (capte CO2, Pression, Humidité et Température)

- Stocke les données dans InfluxDB

- Envoie les données au topic notification sur le broker du VPS


# extraction-milesight

- Extrait les données du capteur EM300-TH (celui du projet) venant du broker de ma VM/Cluster (capte Humidité et Température)

- Stocke les données dans InfluxDB

- Envoie les données au topic fonction_chaleur et fonction_compare_api qui lance respectivement la fonction chaleur et la fonction compareAPI