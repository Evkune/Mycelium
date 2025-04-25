# downlink_EM300-TH_chirpstack
- Prend dans un premiers temps les variables environnements déclarées dans le fichier .env

- Va ensuite transformer la période (récupérée dans .env) dans un code hexadecimal correspondant au downlink du capteur EM300-TH (voir datasheet du capteur).

- Envoie ensutie ce code hexadecimal sur l'api de chirpstack, et ainsi dès le prochain envoie de données du capteur on va changer son intervalle de mesure.

- Principe réutilisé dans la fonction de traitement downlink-em300th


# GetAPIMeteoFrance

- Fonction qui va accéder à l'API de météo France et récupérer un JSON complet des données de la station météo de Saint-Jaques (près de Rennes)

- Va ensuite récupérer la température et l'humidité, et va les afficher

- Principe réutilisé dans la fonction de traitement CompareAPI

- **ATTENTION** au token de l'API qui peut expirer