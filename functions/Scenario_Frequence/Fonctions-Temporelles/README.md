# moyennes-to-vps
- Récupère sur la base influxDB les données (température et humidité de Em300-TH, CO2 et Pression de Em500-CO2) des 12 dernières heures.

- Fais ensuite la moyenne, et envoie ces moyennes au VPS sur le topic moyennestovps

- La fonction se lance tous les jours à 7h et 19h UTC+0