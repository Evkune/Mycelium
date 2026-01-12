import os
import json
import asyncio
import aiomqtt

async def process(req):
    # Récupération dynamique du broker (Cluster ou VPS simulé)
    mqtt_url = os.environ.get('MQTT_URL', 'tcp://10.0.2.15:1883').replace('tcp://', '')
    broker, port = mqtt_url.split(':')

    # Données simulées (Cas "Risque Incendie")
    data = {
        "Temperature": 35.0,
        "RH": 40,
        "Ws": 15,
        "Rain": 0.0,
        "FFMC": 88.0,
        "DMC": 20.0,
        "DC": 50.0,
        "ISI": 8.0,
        "BUI": 25.0,
        "FWI": 12.0,
        "date": "2025-06-15T14:00:00.000Z"
    }
    
    # Cas "Safe" si demandé
    if req == "safe":
        data["Temperature"] = 20.0
        data["Rain"] = 5.0

    payload = json.dumps(data)
    
    try:
        async with aiomqtt.Client(broker, int(port)) as client:
            await client.publish("fireData", payload.encode('utf-8'))
            print(f"Published to fireData: {payload}")
    except Exception as e:
        print(f"Error publishing to MQTT: {e}")

def handle(req):
    asyncio.run(process(req))
