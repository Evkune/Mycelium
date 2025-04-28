# Ajouter un bucket sur InfluxDB :

## Ajout dans la VM

- Aller sur le fichier <code>kube.nix</code> dans le dossier <code>flake-modules/fog-node</code> 
- Rajouter la ligne suivante au niveau de la fin du Service de configuration d'InfluxDB : 
```nix
${pkgs.influxdb2-cli}/bin/influx bucket create --host http://${toString  influxSettings.http-bind-address} --org Mycelium --name <Nom_du_Bucket>
```

## Ajout pour utilisation dans vos fonctions

- Rajouter dans le fichier yaml de configuration de vos fonctions (fichier .yaml) qui utilisent InfluxDB les deux lignes suivantes dans la partie *environment* :
```yaml   
INFLUXDB_URL: http://10.42.0.1:8086
INFLUXDB_TOKEN: Uar6D5Kg0hmAeDjTN9r6q_YN3AhRbhVgLfjuSp243o4R4xHiQ0sEJFdkORZi-1hB57QTDr2VRQjd4Lg4rW1stg==
INFLUXDB_ORG: "Mycelium"
INFLUXDB_BUCKET: "<Nom_du_Bucket>"
```
- Faire appel dans vos fonctions ensuite à InfluxDB via *INFLUXDB_URL* et *INFLUXDB_TOKEN*

# Utiliser le Proxy :

## 0. Adaptation des fonctions pour utiliser le proxy sur la VM

- Si le scénario utilise une base de données plusieurs fois, s'assurer que toutes les fonctions qui ont besoin de la base de données sont déployées dans le même environnement (Uniquement cluster ou VPS)

- Aller dans le fichier yaml de configuration de votre fonction et y rajouter à la fin si non présents la partie *annotations*:
```yaml
annotations:
  topic: <TOPIC>
  functionId: <FUNCTION_ID>
  tag: 1.0
```
Il faut également ajouter cette partie *environment* dans le yaml si la fonction publie sur MQTT:
```yaml
environment:
  MQTT_CLIENTID: <NOM_FONCTION>
  MQTT_URL: tcp://10.0.2.15:1883
```

## 1. Déploiement de fonctions

*OPENFAAS_PORT* vaut 8080 si cluster (OPENFAAS) sinon si sur VPS (OPENFAAS-2) alors 8082
*MQTT_PORT* vaut 1883 si cluster (OPENFAAS) sinon si sur VPS (OPENFAAS-2) alors 1884

- Pour déployer des fonctions sur le cluster, exécuter les commandes suivantes dans un bash:
```bash
OPENFAAS_PORT=8080 MQTT_PORT=1883 just faas-login openfaas
OPENFAAS_PORT=8080 MQTT_PORT=1883 just faas-pub
```

- Pour déployer des fonctions sur le VPS (2ème environnement), exécuter les commandes suivantes dans un bash:
```bash
OPENFAAS_PORT=8082 MQTT_PORT=1884 just faas-login openfaas-2
OPENFAAS_PORT=8082 MQTT_PORT=1884 just faas-pub
```

## 2. Déploiement du proxy

Exécuter les commandes suivantes dans la VM dans le dossier mycelium/proxy :

```bash
kubectl apply -f monitor/DeploymentVM1.yaml
kubectl apply -f monitor/DeploymentVM2.yaml
kubectl apply -f router/DeploymentVM1.yaml
kubectl apply -f router/DeploymentVM2.yaml
kubectl apply -f invoker/DeploymentVM1.yaml
kubectl apply -f invoker/DeploymentVM2.yaml
```

## 3. Envoyer un message sur un topic

*MQTT_PORT* vaut 1883 si cluster (OPENFAAS) sinon si sur VPS (OPENFAAS-2) alors 1884

```bash
MQTT_PORT=1883 just mqtt-pub TOPIC MESSAGE
```

## URL des différents brokers MQTT et des Bases de données

- Broker MQTT Cluster : tcp://192.168.122.61:1883
- Broker MQTT VPS : tcp://10.133.33.52:1883
- Broker MQTT VM : tcp://10.0.2.15:1883 pour le premier  tcp://10.0.2.15:1883 pour le deuxième

- InfluxDB Cluster /  VM : tcp://10.42.0.1:8086
- InfluxDB VPS : tcp://10.133.33.52:8086

## Build et Push des fonctions openfaas

Pour chaque fonction effectuer :

```bash
faas-cli build -f <FONCTION>.yml
faas-cli push -f <FONCTION>.yml
```