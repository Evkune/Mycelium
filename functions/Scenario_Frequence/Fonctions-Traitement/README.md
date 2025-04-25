# chaleur
- À partir de la temperature et humdité qu'elle reçoit de extraction-milesight, cette fonction va calculer l'humidex, un indice qui permet de savoir la sensation de chaleur et le danger que ça peut représenter.

- La fonction renverra donc sur le topic notification du VPS un json avec la température, l'humidité, l'humidex et la sensation de chaleur correspondante.

- En fonction de la sensation de chaleur, si un danger est présent, on appelera la fonction downlink-em300th (par message mqtt avec la période) pour diminuer l'intervalle des mesures afin d'en faire plus


# compare-api
- À partir de la temperature et humdité qu'elle reçoit de extraction-milesight, cette fonction va récupérer la température et l'humidité de la station météo de Saint-Jaques (près de Rennes) grâce à l'API météo France et va comparer ces valeurs aux valeurs du capteurs.

- On enverra ensuite sur le topic notification du VPS si les valeurs sont cohérentes ou pas.

- **ATTENTION** au token de l'API qui peut expirer


# downlink-em300th
- Prend dans un premiers temps les variables environnements déclarées dans le fichier .yml de la fonction + la période reçu en JSON

- Va ensuite transformer la période (récupérée dans .env) dans un code hexadecimal correspondant au downlink du capteur EM300-TH (voir datasheet du capteur).

- Envoie ensutie ce code hexadecimal sur l'api de chirpstack, et ainsi dès le prochain envoie de données du capteur on va changer son intervalle de mesure.