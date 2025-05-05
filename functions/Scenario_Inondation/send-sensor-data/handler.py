import asyncio
import os
import csv
import json
import aiomqtt as mqtt

from datetime import datetime


def handle(req=None):
    """
    Entry point for OpenFaaS or manual invocation. 
    - If no req provided or empty: uses default CSV row 10.
    - If req is integer string: uses that row index.
    - If req contains 'test-alert': publishes custom JSON with waterlevel=1.9 (will trigger the alert function).

    :param req: request from OpenFaaS or CLI
    :return: None
    """
    # Delegate to async processing
    asyncio.run(process(req))

async def process(req=None):
    """
    Orchestrates MQTT publishing of sensor data.

    - If req is "test-alert", publishes a custom alert payload.
    - If req is an integer string, fetches data from CSV files based on that index.
    - If req is None or empty, defaults to row index 10.

    :param req: request from OpenFaaS or CLI
    :return: None
    """
    # Get MQTT broker URL and port from environment variable
    mqtt_url = os.environ.get('MQTT_URL', 'tcp://10.0.2.15:1883').replace('tcp://', '')
    broker, port = mqtt_url.split(':')

    # Custom alert payload
    if isinstance(req, str) and req == "test-alert":
        raw = {"date": "2023-01-01T14:00:00.000Z", "rainfall": 0.8, 'waterLevel': 1.9}
        rawData = json.dumps(raw)
        print(f"[custom alert] payload: {rawData}")

        # Publish to MQTT
        async with mqtt.Client(broker, int(port)) as client:
            await client.publish('rawData', rawData.encode('utf-8'))
            print("Published to MQTT topic 'rawData'.")
        return
    
    # Analyse test: publish 11 consecutive records to trigger analysis
    if isinstance(req, str) and req == "test-analyse":
        # default start index if none provided
        start = 10
        print(f"[test-analyse] starting at index={start}")
        
        async with mqtt.Client(broker, int(port)) as client:
            for i in range(start, start + 11):
                rawData = accessDatabase(i)
                print(f"[analyse] row={i} payload: {rawData}")
                await client.publish('rawData', rawData.encode('utf-8'))
                print(f"Published row {i} to MQTT topic 'rawData'.")
        return
    
    # Else, decide row index for CSV lookup
    try:
        index = int(req)
    except (TypeError, ValueError):
        index = 10  # default row if no valid index provided (in req)

    # Fetch from CSVs
    rawData = accessDatabase(index)
    print(f"[csv lookup] row={index} payload: {rawData}")
    # Publish to MQTT
    async with mqtt.Client(broker, int(port)) as client:
        await client.publish('rawData', rawData.encode('utf-8'))
        print("Published to MQTT topic 'rawData'.")

def accessDatabase(req_index):
    """
    Synchronous CSV lookup combining rainfall and water level data.
    ATTENTION : Les données utilisées devraient être celles de l'OSUR pour l'utilisation du modèle,
    étant donné qu'elles sont inexploitables, nous utilisons de nouveaux les données de météoFrance et de VigiCrues.

    :param req_index: index of the row to fetch from CSV files
    :return: JSON string with rainfall and water level data
    """
    data = {}
    # Meteo France CSV
    with open('./function/meteofrance.csv', 'r', newline='') as f:
        reader = csv.reader(f, delimiter=',', quotechar='"')
        for i, row in enumerate(reader):
            if i == req_index:
                if len(row) >= 3:
                    data['rainfall'] = float(row[2].replace(',', '.'))
                break

    # Vigi Crues CSV
    with open('./function/VigiCrues2023-2024.csv', 'r', newline='') as f:
        reader = csv.reader(f, delimiter=',', quotechar='"')
        for i, row in enumerate(reader):
            if i == req_index:
                if len(row) >= 2:
                    data['date'] = row[0]
                    data['waterLevel'] = float(row[1].replace(',', '.'))
                break

    return json.dumps(data)

