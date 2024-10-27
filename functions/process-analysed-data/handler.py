import asyncio
import json
import os
from datetime import datetime, timedelta

import influxdb_client
import nats
from influxdb_client import Point
from influxdb_client.client.write_api import SYNCHRONOUS

client = None


def handle(req):
    """
    Fonction d'arrivée pour les programmes qui tournent avec nats

    :param req: un payload avec une date et un niveau d'eau
    :return: rien
    """
    asyncio.run(process(req))


async def export_to_database(measurement, json_input, date):
    """
    Permet d'enregistrer notre payload sur InfluxDB

    :param measurement: le nom de la base de données dans laquelle on veut enregistrer les données
    :param json_input: le payload sous format json
    :param date: la date associée au paylaod
    :return: rien
    """
    write_api = connect_to_database().write_api(write_options=SYNCHRONOUS)

    point = (Point(measurement)
             .field("waterLevel", json_input['waterLevel'])
             .time(date, write_precision="s"))

    write_api.write(bucket=os.environ.get('influxdb_bucket'), org=os.environ.get('influxdb_org'), record=point)


def is_there_alert(json_input):
    """
    Regarde si le payload reçu doit déclencher une alerte

    :param json_input: le payload sous format json
    :return: un json signalant une alerte
    """
    print(f"Water level is {json_input['waterLevel']}m")
    dictionnaire = {}
    if json_input['waterLevel'] >= 1.5:
        print(f"Water level is above threshold, triggering alert")
        dictionnaire.update({'alertType': 'floodPrediction'})
        dictionnaire.update({'date': json_input['date']})
    json_output = json.dumps(dictionnaire)
    return json_output


def connect_to_database():
    """
    Lance la connexion avec la base de données

    :return: l'objet qui permet de communiquer avec la base de données
    """
    global client

    if client is None:
        client = influxdb_client.InfluxDBClient(
            os.environ.get('influxdb_host'),
            token=os.environ.get('influxdb_token'),
            org=os.environ.get('influxdb_org')
        )

    return client


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

    :param req: un payload avec une date et un niveau d'eau
    :return: envoie une alerte sur le topic triggerAlert
    """
    nc = await nats.connect(servers=os.environ.get('nats_host'))
    json_input = json.loads(req)

    date = datetime.fromisoformat(json_input['date'])
    print(f"Received prediction for {date}")

    await export_to_database("predictions", json_input, date)

    alert = is_there_alert(json_input)
    if alert != '{}':
        await nc.publish('triggerAlert', f"{alert}".encode())
    await nc.flush()
    await nc.close()
    close_database_connection()
