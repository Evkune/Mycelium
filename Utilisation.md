# Ajouter un bucket sur InfluxDB :

- Aller sur le fichier <code>kube.nix</code> dans le dossier <code>flake-modules/fog-node</code> 
- Rajouter la ligne suivante au niveau de la fin du Service de configuration d'InfluxDB : 
```nix
${pkgs.influxdb2-cli}/bin/influx bucket create --host http://${toString  influxSettings.http-bind-address} --org Mycelium --name <Nom_du_Bucket>
```
- Rajouter dans le fichier yaml de configuration de vos fonctions (fichier .yaml) qui utilisent InfluxDB les deux lignes suivantes dans la partie *environment* :
```yaml   
INFLUXDB_URL: http://10.42.0.1:8086
INFLUXDB_TOKEN: Uar6D5Kg0hmAeDjTN9r6q_YN3AhRbhVgLfjuSp243o4R4xHiQ0sEJFdkORZi-1hB57QTDr2VRQjd4Lg4rW1stg==
INFLUXDB_ORG: "Mycelium"
INFLUXDB_BUCKET: "<Nom_du_Bucket>"
```
- Faire appel dans vos fonctions ensuite à InfluxDB via INFLUXDB_URL et INFLUXDB_TOKEN

# Ajouter un topic MQTT :

- Aller sur le fichier <code>kube.nix</code> dans le dossier <code>flake-modules/fog-node</code> 
- Rajouter les lignes de code suivantes à l'intérieur de *kubernetes.helm.releases*: 
```nix
mqtt-<NOM_FONCTION> = {
  namespace = lib.mkForce "openfaas";
  overrideNamespace = false;
  chart = pkgs.stdenvNoCC.mkDerivation {
    name = "mqtt-connector";
    src = openfaas;
    
    buildCommand = ''
      ls $src
      cp -r $src/chart/mqtt-connector/ $out
    '';
  };
  values = {
    broker = "tcp://10.0.2.15:1883";
    topic = ""<NOM_FONCTION>"";
    clientID = "m<NUMERO>";
    };
};
```

- Aller dans le fichier yaml de configuration de votre fonction et y rajouter à la fin si non présents la partie environment, et annotations avec topic:
```yaml
environment:
  MQTT_CLIENTID: <NOM_FONCTION>
  MQTT_URL: tcp://10.0.2.15:1883
```
```yaml
annotations:
  topic: <NOM_FONCTION>
```

- Faire appel dans vos fonctions ensuite à InfluxDB via INFLUXDB_URL et INFLUXDB_TOKEN