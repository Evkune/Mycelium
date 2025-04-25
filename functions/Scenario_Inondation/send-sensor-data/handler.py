import asyncio
import csv
import datetime
import json
import os
import aiomqtt as mqtt
import nats

from datetime import datetime


def handle(req):
    """
    Fonction d'arrivée pour les programmes qui tournent avec nats

    :param req: "mqtt" ou "test-alert" ou un numéro de ligne pour choisir ce que l'on veut faire avec la fonction process
    :return: rien
    """
    asyncio.run(process(req))


async def process(req):
    """
    Fonction générale qui gère la connexion avec nats et mqtt

    :param req: "mqtt" ou "test-alert" ou un numéro de ligne
    :return: un payload envoyé sur le topic rawData
    """
    if req == "mqtt":
        # Connexion MQTT
        async with mqtt.Client("10.0.2.15", 1883) as client:
            rawData = accessDatabase(10)
            # Publier les données sur le topic rawData
            await client.publish('rawData', rawData.encode('utf-8'))
            # S'abonner au topic pour écouter les messages
            #await client.subscribe("application/+/device/+/event/+")
            #async for message in client.messages:
            #    await on_message(message)
    else:
        async with mqtt.Client("10.0.2.15", 1883) as client:
            print(f"Sending data #{req}")
            #rawData = accessDatabase(int(req))  # Fonction qui accède à la DB
            rawData = accessDatabase(15)
            # Publier les données sur le topic rawData
            await client.publish('rawData', rawData.encode('utf-8'))


def accessDatabase(req):
    """
    Permet de récupérer des valeurs de nos deux csv
    ATTENTION : Les données utilisées devraient être celles de l'OSUR pour l'utilisation du modèle,
    étant donné qu'elles sont inexploitables, nous utilisons de nouveaux les données de météoFrance et de VigiCrues.

    :param req: la ligne que l'on souhaite récupérer
    :return: un json avec les données associées
    """
    fileRainfall = open('./function/meteofrance.csv', 'r')
    fileWaterLevel = open('./function/VigiCrues2023-2024.csv', 'r')
    reader = csv.reader(fileRainfall)
    rowNb = 0
    dictionary = {}
    for row in reader:
        if rowNb == req:
            data = row[2].replace(",", ".")
            dictionary.update({'rainfall': float(data)})
            break
        else:
            rowNb += 1
    fileRainfall.close()
    reader = csv.reader(fileWaterLevel)
    rowNb = 0
    for row in reader:
        if rowNb == req:
            dictionary.update({'date': row[0]})
            dictionary.update({'waterLevel': float(row[1])})
            break
        else:
            rowNb += 1
    fileWaterLevel.close()
    json_object = json.dumps(dictionary)
    return json_object


async def on_message(msg):
    """
    Permet de décider quoi faire lors de la réception d'un message

    :param msg: le message reçu de mqtt
    :return: rien
    """
    if str(msg.topic).endswith("/event/up"):
        print("C'est un gentil PAYLOAD")
        print(msg.topic)
        byte_string = msg.payload
        json_string = byte_string.decode('utf-8')
        json_data = json.loads(json_string)
        print(json_data)
        await processMqtt(json_data)
    else:
        print("C'est un méchant PAYLOAD")
        print(str(msg.topic) + " -> " + str(msg.payload))
    print(
        "************************************************************************************************************************************************************************")


async def processMqtt(json_file):
    """
    Envoie les données pertinentes des capteurs de l'OSUR avec topic NATS

    :param json_file: le message de mqtt sous format json
    :return: un payload envoyé sur le topic rawData
    """
    # nc = await nats.connect(servers=os.environ.get('nats_host'))
    dictionary = {}

    if "pluviometre" in json_file["deviceInfo"]["applicationName"].lower():
        date = json_file["time"]
        parsed_date = datetime.strptime(date, "%Y-%m-%dT%H:%M:%S.%f%z")
        formatted_date = parsed_date.strftime('%Y-%m-%d %H:%M:%S')

        dictionary.update({'date': formatted_date})
        dictionary.update({'rainfall': float(json_file["object"]["rain_current"]["value"])})

        print("The device called " + json_file["deviceInfo"]["deviceName"])
        print("tells us that the current rain fall is " + str(json_file["object"]["rain_current"]["value"]) +
              json_file["object"]["rain_current"]["unit"])
        print("and the total amount of rain is " + str(json_file["object"]["rain_total"]["value"]) +
              json_file["object"]["rain_total"]["unit"] + ".")

    elif "RAK_CTD" in json_file["deviceInfo"]["applicationName"]:
        date = json_file["time"]
        parsed_date = datetime.strptime(date, "%Y-%m-%dT%H:%M:%S.%f%z")
        formatted_date = parsed_date.strftime('%Y-%m-%d %H:%M:%S')

        dictionary.update({'date': formatted_date})
        dictionary.update({'waterlevel': float(json_file["object"]["waterlevel"]["value"]) / 1000})

        print("The device called " + json_file["deviceInfo"]["deviceName"])
        print("tells us that the current water level is " + str(json_file["object"]["waterlevel"]["value"]) +
              json_file["object"]["waterlevel"]["unit"] + ".")
    else:
        return

    print(dictionary)
    print(json.dumps(dictionary))

    json_output = json.dumps(dictionary)
    # await nc.publish('rawData', f"{json_output}".encode())
    # await nc.flush()
    # await nc.close()


def alexandre(start):
    """
    Fonction de test qui a pour but de donner 10 heures de données consécutives

    :param start: la ligne de départ
    :return: rien
    """
    liste = []
    for i in range(start, start + 11):
        entree = json.loads(accessDatabase(i))
        liste.append(entree)
    dictionnaire = {}
    dictionnaire.update({'data': liste})
    json_object = json.dumps(dictionnaire)
    print(json_object)


if __name__ == "__main__":
    instruction = input("What is your request ? ")
    asyncio.run(process(instruction))
