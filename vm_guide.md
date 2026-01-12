### 1. Prérequis

Avant de commencer, assurez-vous d'avoir installé Nix sur votre machine Linux.

Installation de Nix : Suivez les instructions sur le site officiel : nixos.org.

Configuration de Nix : Ajoutez les lignes suivantes à votre fichier /etc/nix/nix.conf pour activer les fonctionnalités requises.

extra-experimental-features = nix-command flakes
max-jobs = auto
cores = 0
log-lines = 50

### 2. Lancement de l'Environnement de Développement

Toutes les manipulations se feront depuis la racine du projet.

Étape 1 : Entrer dans le Shell de Développement
Cette commande prépare tous les outils nécessaires (just, kubectl, faas-cli, etc.) dans votre terminal.

nix develop

Étape 2 : Démarrer la Machine Virtuelle
La commande just est un lanceur de commandes pré-configurées dans le justfile. Celle-ci va construire (si c'est le premier lancement) et démarrer la VM.

just vm

Note : Le premier démarrage peut être long, car Nix doit télécharger et configurer tous les composants. Les lancements suivants seront quasi-instantanés.

Étape 3 : Se Connecter à la VM
Une fois la VM démarrée, ouvrez un nouveau terminal et lancez la commande suivante pour vous y connecter en SSH.

just ssh

Vous êtes maintenant à l'intérieur de la VM Mycelium ! Toutes les commandes qui suivent doivent être exécutées depuis cette session SSH.

### 3. Déployer et Gérer Vos Fonctions OpenFaaS

La VM simule l'architecture de production avec deux environnements OpenFaaS distincts, vous permettant de tester des déploiements complexes.

Environnement 1 : Gateway OpenFaaS sur http://127.0.0.1:8080

Environnement 2 : Gateway OpenFaaS sur http://127.0.0.1:8082

Étape 1 : Se Connecter à une Gateway OpenFaaS
Pour pouvoir déployer, vous devez d'abord vous authentifier.

Récupérer le mot de passe : La VM génère un mot de passe unique à chaque lancement. Utilisez cette commande pour l'obtenir :

kubectl -n openfaas get secret basic-auth -o jsonpath="{.data.basic-auth-password}" | base64 --decode

Copiez le mot de passe qui s'affiche.

Se connecter : Utilisez la commande faas-cli login. Pour vous connecter à l'Environnement 1 :

faas-cli login --gateway http://127.0.0.1:8080/ --username admin --password <MOT_DE_PASSE_COPIÉ>

Vous pouvez aussi utiliser le raccourci du justfile :

# Se connecter à l'environnement 1

OPENFAAS_PORT=8080 just faas-login openfaas

# Se connecter à l'environnement 2

OPENFAAS_PORT=8082 just faas-login openfaas-2

Étape 2 : Déployer une Fonction
Le justfile simplifie grandement le processus de déploiement. Il va automatiquement construire l'image Docker de votre fonction, la pousser sur un registre temporaire (ttl.sh), et la déployer sur l'environnement OpenFaaS auquel vous êtes connecté.

Pour déployer une seule fonction, utilisez la commande just faas-pub-single suivie du chemin vers son fichier de configuration .yml.

Exemple : Déployer la fonction send-sensor-data sur l'Environnement 1

just faas-pub-single functions/Scenario_Inondation/send-sensor-data.yml

Important : Les images sur le registre ttl.sh expirent après 2 heures. Si votre fonction redémarre après ce délai, elle ne pourra pas se lancer (erreur ImagePullBackOff). Il vous suffira de la redéployer avec la même commande pour rafraîchir l'image.

### 4. Se Connecter à la Base de Données (InfluxDB)

La VM embarque une base de données InfluxDB pour stocker les données temporelles de vos capteurs.

Accéder à l'Interface Web
Vous pouvez gérer la base de données via une interface graphique accessible depuis votre navigateur.

URL : http://localhost:8086

Identifiants :

Nom d'utilisateur : admin

Mot de passe : adminfaasfog

Depuis cette interface, vous pouvez explorer vos "buckets" (les conteneurs de données), visualiser les données et configurer des alertes.

Utiliser InfluxDB dans vos Fonctions
Pour que vos fonctions puissent communiquer avec la base de données, assurez-vous que leur fichier .yml contienne les variables d'environnement suivantes :

environment:
  INFLUXDB_URL: http://10.42.0.1:8086
  INFLUXDB_TOKEN: "Uar6D5Kg0hmAeDjTN9r6q_YN3AhRbhVgLfjuSp243o4R4xHiQ0sEJFdkORZi-1hB57QTDr2VRQjd4Lg4rW1stg=="
  INFLUXDB_ORG: "Mycelium"
  INFLUXDB_BUCKET: "<NOM_DE_VOTRE_BUCKET>"

Tester et Interagir avec les Services
Publier un Message sur un Topic MQTT
Pour déclencher une fonction abonnée à un topic MQTT, utilisez la commande just mqtt-pub.

# Envoyer le message "test" sur le topic "sample-topic" de l'environnement 1

just mqtt-pub sample-topic "test"

Écouter un Topic MQTT
Pour voir les messages publiés sur un topic (par exemple, les résultats de votre fonction), utilisez just mqtt-sub.

# Écouter le topic "sample-topic" de l'environnement 1

just mqtt-sub sample-topic
