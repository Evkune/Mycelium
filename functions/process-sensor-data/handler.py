import asyncio
import json
import os
from datetime import datetime, timedelta
from re import split

import influxdb_client
import aiomqtt as mqtt
from influxdb_client import Point
from influxdb_client.client.write_api import SYNCHRONOUS

client = None
date_format = '%Y-%m-%dT%H:%M:%S.000Z'

def handle(req):
    """
    Fonction d'arrivée pour les programmes qui tournent avec nats

    :param req: un payload avec une date, un niveau d'eau de pluie cumulé
    (rainfall par météoFrance) et le niveau d'eau (waterLevel rainFall)
    :return: rien
    """
    asyncio.run(process(req))


def is_there_alert(json_input):
    """
    Regarde si le payload reçu doit déclencher une alerte

    :param json_input: le payload sous format json
    :return: un json signalant une alerte
    """
    print(f"Water level is {json_input['waterLevel']}m")
    dicti = {}
    if json_input['waterLevel'] > 1.8:
        print(f"Water level is above threshold, triggering alert")
        dicti.update({'alertType': 'floodUnderway'})
        dicti.update({'date': json_input['date']})
    json_output = json.dumps(dicti)
    return json_output


def is_there_analyse(query):
    """
    Regarde si les conditions pour lancer une analyse sont atteintes

    :param query: une liste de données venant d'InfluxDB
    :return: un json avec les données s'il doit y avoir une analyse
    """
    if len(query) != 0:
        measures = [{} for _ in query[0]]

        for e in query:
            i = 0
            for r in e.records:
                measures[i][r.get_field()] = r.get_value()

                if "date" not in measures[i]:
                    measures[i]["date"] = r.get_time().isoformat()

                i = i + 1

        print(f"There are {len(measures)} measures in the last 11 hours")
        if len(measures) > 10:
            print(f"There are enough measures in then last 11 hours, triggering analyse")
            return json.dumps({"data": measures})
        else:
            return "{}"
    else:
        print(f"There is no measure in the last 11 hours")
        return "{}"


def connect_to_database():
    """
    Lance la connexion avec la base de données

    :return: l'objet qui permet de communiquer avec la base de données
    """
    global client

    if client is None:
        client = influxdb_client.InfluxDBClient(
            os.environ.get('INFLUXDB_URL'),
            token=os.environ.get('INFLUXDB_TOKEN'),
            org=os.environ.get('INFLUXDB_ORG')
        )

    return client


def export_to_database(measurement, json_input, date):
    """
    Permet d'enregistrer notre payload sur InfluxDB

    :param measurement: le nom de la base de données dans laquelle on veut enregistrer les données
    :param json_input: le payload sous format json
    :param date: la date associée au paylaod
    :return: rien
    """
    write_api = connect_to_database().write_api(write_options=SYNCHRONOUS)

    point = (Point(measurement)
             .field("rainfall", json_input['rainfall'])
             .field("waterLevel", json_input['waterLevel'])
             .time(date, write_precision="s"))

    write_api.write(bucket=os.environ.get('INFLUXDB_BUCKET'), org=os.environ.get('influxdb_org'), record=point)
    write_api.flush()
    write_api.close()


def query_from_database(measurement, stop_time):
    """
    Permet de récupérer des données enregistrées sur InfluxDB sur un intervalle de 11 heures

    :param measurement: le nom de la base de données de laquelle on veut récupérer les données
    :param stop_time: date de fin de la requête
    :return: une liste avec les données correspondantes
    """
    query_api = connect_to_database().query_api()

    start_time = stop_time - timedelta(hours=11)

    query = f"""from(bucket: "{os.environ.get('INFLUXDB_BUCKET')}")
     |> range(start: time(v: "{str(start_time).replace(" ", "T")}Z"), stop: time(v: "{str(stop_time).replace(" ", "T")}Z"))
     |> filter(fn: (r) => r._measurement == "{measurement}")
     """

    return query_api.query(query, org="Mycelium")


def close_database_connection():
    """
    Ferme la connexion avec la base de données

    :return: rien
    """
    global client
    if client is not None:
        client.close()
        client = None


async def process(req):
    """
    Fonction principale qui gère la connexion avec nats et appelle les fonctions pour enregistrer les données

    :param req: un payload avec une date, un niveau d'eau de pluie (rainfall) et le niveau d'eau (waterlevel)
    :return: envoie deux payloads distincts, un sur le topic triggerAnalyse et l'autre sur triggerAlert
    """
    # Connection au serveur MQTT
    async with mqtt.Client("10.0.2.15", 1883) as mqtt:
        json_input = json.loads(req)
        date = datetime.strptime(json_input["date"], date_format)
        print(f"Receiving data collected on {date}")

        export_to_database("measures", json_input, date)
        alert = is_there_alert(json_input)
        if alert != '{}':
            await mqtt.publish('triggerAlert', f"{alert}".encode())

        query = query_from_database("measures", date)
        analyse = is_there_analyse(query)
        if analyse != '{}':
            await mqtt.publish('triggerAnalyse', f"{analyse}".encode())
        close_database_connection()