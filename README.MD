# Guide d'utilisation de Mycelium 4.0

Ce document a pour objectif de centraliser et clarifier toutes les étapes nécessaires à la configuration, au déploiement et à l’exploitation de l’infrastructure Mycelium dans les environnements VM, VPS et Cluster.
Il couvre les sujets essentiels tels que la création de buckets InfluxDB, la configuration des fonctions OpenFaaS, l’utilisation du proxy MQTT, ainsi que les bonnes pratiques pour la gestion des images Docker et le déploiement dans différents environnements.

Un guide léger des commandes pratiques de la VM est dans le fichier <code>NIX.MD</code>, ce sont des raccourcis de commandes existantes présents dans le fichier <code>justfile</code> qui permettent de démarrer la VM, se connecter à la gateway, etc.

## Explication rapide des technologies utilisées :

- **Kubernetes** : Système d’orchestration de conteneurs permettant de déployer, gérer et mettre à l’échelle des applications dans des clusters.

- **Docker** : Plateforme de conteneurisation permettant d’empaqueter et exécuter des applications avec leurs dépendances.

- **Image Docker** : Une image docker contient l'essentiel de ce qui est nécessaire pour lancer un programme. Elle va venir ajouter toutes les dépendances nécessaires pour que lorsque vous la déployez (création d'un conteneur), ce conteneur puisse fonctionner sans se soucier des versions des logiciels installés (Python, Go, Ubuntu 24 ou 22 etc...)

- **OpenFaaS** : Framework pour déployer des fonctions serverless (sans serveur) sur Kubernetes ou Docker.

- **InfluxDB** : Base de données orientée séries temporelles, idéale pour stocker des métriques ou données de capteurs.

- **MQTT** : Protocole de messagerie léger conçu pour la communication machine-à-machine (IoT). Il fonctionne avec un principe d'abonnement et de publication (Métaphore : Similaire à un abonnement à un magasine, étant abonné vous recevrez tous les nouveaux journaux publiés).

- **RSS** : Format de flux permettant de syndiquer automatiquement du contenu (articles, podcasts, etc.).

- **Nix** : Est un gestionnaire de dépendance reproductible : Brièvement il permet de préparer la machine virtuelle et d'y injecter toutes les technologies nécessaires et de les configurer au lancement grâce à des fichiers de configuration, Ca permet d'avoir une machine identique entre vous. (Ne pas y toucher si pas besoin de modifier la VM)

## Ajouter un bucket sur InfluxDB :

### Ajout dans la VM

- Aller sur le fichier <code>kube.nix</code> dans le dossier <code>flake-modules/fog-node</code> 
- Rajouter la ligne suivante au niveau de la fin du Service de configuration d'InfluxDB : 
```nix
${pkgs.influxdb2-cli}/bin/influx bucket create --host http://${toString  influxSettings.http-bind-address} --org Mycelium --name <Nom_du_Bucket>
```

### Ajout pour utilisation dans vos fonctions

- Rajouter dans le fichier yaml de configuration de vos fonctions (fichier .yaml) qui utilisent InfluxDB les deux lignes suivantes dans la partie *environment* :
```yaml   
INFLUXDB_URL: http://10.42.0.1:8086
INFLUXDB_TOKEN: Uar6D5Kg0hmAeDjTN9r6q_YN3AhRbhVgLfjuSp243o4R4xHiQ0sEJFdkORZi-1hB57QTDr2VRQjd4Lg4rW1stg==
INFLUXDB_ORG: "Mycelium"
INFLUXDB_BUCKET: "<Nom_du_Bucket>"
```
- Faire appel dans vos fonctions ensuite à InfluxDB via *INFLUXDB_URL* et *INFLUXDB_TOKEN*

## Utiliser le Proxy :

- Si le scénario utilise une base de données plusieurs fois, s'assurer que toutes les fonctions qui ont besoin de la base de données sont déployées dans le même environnement (Uniquement cluster ou VPS)

### 0. Adaptation des fonctions pour utiliser le proxy

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

### 1. Déploiement de fonctions

*OPENFAAS_PORT* vaut 8080 si c'est l'environnement 1 (OPENFAAS) sinon si sur environnement 2 (OPENFAAS-2) alors 8082
*MQTT_PORT* vaut 1883 si environnement 1 (OPENFAAS) sinon si sur environnement 2 (OPENFAAS-2) alors 1884

- Pour déployer des fonctions sur le 1er environnement (OPENFAAS), exécuter les commandes suivantes dans un bash:
```bash
OPENFAAS_PORT=8080 MQTT_PORT=1883 just faas-login openfaas
OPENFAAS_PORT=8080 MQTT_PORT=1883 just faas-pub
```

- Pour déployer des fonctions sur le 2ème environnement (OPENFAAS-2), exécuter les commandes suivantes dans un bash:
```bash
OPENFAAS_PORT=8082 MQTT_PORT=1884 just faas-login openfaas-2
OPENFAAS_PORT=8082 MQTT_PORT=1884 just faas-pub
```

#### Sur Cluster ou VPS :

```bash
faas-cli deploy -f <FUNCTION.yml>
```

Si vous avez besoin de vous connecter à la gateway (Code HTTP 401 lorsque déploie fonction) alors tapez la commande suivante pour récupérer le mot de passe :

```bash
sudo kubectl get secret -n openfaas basic-auth -o jsonpath="{.data.basic-auth-password}" | base64 --decode
```

puis avec le résultat la commande suivante pour se connecter à la gateway :
```bash
faas-cli login --password <RES> -g <IP_GATEWAY>
```

### 2. Déploiement du proxy

Exécuter les commandes suivantes dans la VM dans le dossier <code>mycelium/proxy</code> :

```bash
kubectl apply -f monitor/DeploymentVM1.yaml
kubectl apply -f monitor/DeploymentVM2.yaml
kubectl apply -f router/DeploymentVM1.yaml
kubectl apply -f router/DeploymentVM2.yaml
kubectl apply -f invoker/DeploymentVM1.yaml
kubectl apply -f invoker/DeploymentVM2.yaml
```

#### Déploiement sur VPS ou Cluster

Copier via Tailscale (En fin de guide) les fichiers de déploiements du proxy pour le VPS sur le VPS et pour le Cluster sur le Cluster puis sur le cluster :

```bash
kubectl apply -f DeploymentClusterRouter.yaml
kubectl apply -f DeploymentClusterMonitor.yaml
kubectl apply -f DeploymentClusterInvoker.yaml
```

Sur le VPS :

```bash
kubectl apply -f DeploymentVPSRouter.yaml
kubectl apply -f DeploymentVPSMonitor.yaml
kubectl apply -f DeploymentVPSInvoker.yaml
```

### 3. Envoyer un message sur un topic

*MQTT_PORT* vaut 1883 si sur l'environnement 1 (OPENFAAS) sinon si sur l'environnement 2 (OPENFAAS-2) alors 1884

```bash
MQTT_PORT=1883 just mqtt-pub TOPIC MESSAGE
```

Sur le VPS ou sur le Cluster envoyer la requête suivante : 

```bash
mosquitto_pub -t TOPIC -m MESSAGE
```

# Informations sur les Brokers MQTT, InfluxDB et OpenFaas utiles pour les déploiements de fonctions:

## Broker MQTT :

- Cluster : tcp://192.168.2.11:1883
- VPS : tcp://10.133.33.52:1883
- VM : tcp://10.0.2.15:1883 pour le premier  tcp://10.0.2.15:1884 pour le deuxième

## InfluxDB : 

- VM : 
  - URL : tcp://10.42.0.1:8086
  - Org : Mycelium
  - Token : Uar6D5Kg0hmAeDjTN9r6q_YN3AhRbhVgLfjuSp243o4R4xHiQ0sEJFdkORZi-1hB57QTDr2VRQjd4Lg4rW1stg==
  - Username : admin
  - Password : adminfaasfog
  - Interface graphique via l'adresse : http://localhost:8086
- VPS : 
  - URL : tcp://10.133.33.52:8086 
  - Org : Mycelium
  - Token : 8w77xozaB8ZeWm3KeSYJmG99EvstFl3KCDmIXlGfCWBZJOO9jxkWCypBrfETMuuzPTwBCcCNHHdFGUJ3W4VS5g==
  - Username : Mycelium
  - MDP : MyceliumIR
  - Interface graphique via l'adresse IP tailscale du VPS au port 8086
- Cluster : Il n'y a pas d'influxDB je crois, à déployer sur le VPS pour l'instant

## Gateway : 

Pour récupérer l'adresse de la gateway si elle change (adresse utile pour les déploiements de fonctions donc), taper la commande suivante et récupérer l'adresse en Internal IP (le service doit être en ClusterIP)

```bash
kubectl get svc -n openfaas | grep gateway
```

Les nouvelles versions d'OpenFaas avaient supprimé les images de gateway, si nécessaire la redéployer via le fichier <code>custom_gateway.yaml</code>, elle se base sur une ancienne version conservée d'OpenFaas.

## Build et Push des fonctions openfaas

Pour chaque fonction effectuer dans le dossier functions les instructions suivantes avec dossier de la fonction et le yaml de la fonction dans le dossier functions:

```bash
faas-cli build -f <FONCTION>.yml
faas-cli push -f <FONCTION>.yml
```

Si les fonctions restent comme nous (*ttl.sh/<NOM_HOTE>/<FONCTION>:2h*) alors elles seront publiés pendant 2h sur ce registre, passé cette barre si la fonction doit redémarré cela échouera (Le pod sera en ImagePullBackOff sur Kubernetes). ttl.sh permet de ne pas avoir à s'identifier, ça peut être pratique, mais si c'est un frein : Utilisez un compte mycelium sur Docker et supprimer dans le fichier <code>justfile</code> :

```justfile
# Registry for storing images temporarily
REGISTRY := env_var_or_default('REGISTRY', "ttl.sh/" + `whoami`)

# Time that the image will be stored in the registry
TAG := env_var_or_default('TAG', "2h")
```

Tous les déploiements effectués sur le Cluster doivent être compatible armv7 sinon ça ne fonctionnera pas. (Proxy comme Fonctions)

### Construction d'une image de fonction armv7 pour le Cluster

Cet aspect n'est pas sûr de fonctionner à 100 % mais vous pouvez essayer de build la fonction en classique dans le répertoire *functions* puis passer dans le sous-répertoire *build* et taper la commande suivante : 

```bash
docker buildx build --platform linux/arm/v7 -t <TAG_FUNCTION> --push <DOSSIER_FUNCTION>
```

## Problèmes rencontrés au déploiement sur le Cluster / le VPS

- Le VPS n'as pas beaucoup d'espace disponible, si lorsque vous déployez des choses dessus tout crash d'un coup c'est peut être lié à un manque de mémoire, essayez de faire de la place.

- Les images sur le cluster doivent donc être compatible avec armv7.

- Les images sur le Cluster / le VPS sont parfois sauvegardées en local, si lorsque vous les redéployer elles restent identiques, il faut forcer l'actualisation à chaque redéploiement d'une fonction en ajoutant juste en dessous de votre image pour votre container dans votre yaml :

```yaml
imagePullPolicy : Always
```

- Kubernetes sur le VPS s'accède uniquement via le compte root du VPS. (sudo -i à effectuer)


## Tailscale

Pour l'utilisation du Cluster et du VPS, faîtes en sorte de configurer Tailscale, cela contournera l'utilisation de la passerelle fournie par la DSI.

### Se connecter à distance au Cluster ou au VPS

Cluster : (MDP : nikos)

```bash
ssh nikos@nikos0
```

VPS : (MDP : Votre mot de passe insa)

```
ssh LOGIN_INSA@vps-mycelium
```

### Envoyer des fichiers de votre machine sur nikos0 (Cluster) ou sur vps-mycelium (VPS)

```bash
sudo tailscale file cp <FILE1> <FILE2> ... nikos0:
```

### Récupérer les fichiers envoyés :

```bash
sudo tailscale file get .
```

# Espace Mémoire 

Ce projet peut prendre beaucoup de place sur votre machine, lorsqu'il prend trop de place pensez à supprimer vos images docker inutiles et à supprimer le cache nix via la commande suivante (Peut rapidement prendre 20 - 40 Go de libéré via la commande):

```bash
nix-collect-garbage
```

# Ancien Guide Fourni par Mycelium 3.0

Beaucoup d'informations sont présentes sur ce guide, il a été mis à jour pour les informations concernant le cluster utilisées en début d'année :

Les informations concernant les VMs ne sont pas utiles normalement puisque nous utilisons une VM faite avec Nix, reproductible, qui a déjà tout ce qui est recherché d'installé.

https://docs.google.com/document/d/1zeqyLGFVsIeQFenrWVIXsBdJN7fD8fSSPS8Tly6i5KA/edit?tab=t.0#heading=h.8l6zs9xhhd2y

Gitlab du projet Mycelium 3.0 contenant des informations intéressantes : https://gitlab.insa-rennes.fr/mycelium-3.0/mycelium-3.0/-/tree/main

# Contact

Si des informations manquent sur ce guide ou sur le guide Mycelium 3.0 ou pour toute question :  
**Arno Lécrivain** : arno.lecrivain@insa-rennes.fr