import paho.mqtt.client as mqtt
import json
import random
import os

# Configuration MQTT
MQTT_BROKER = os.getenv("MQTT_BROKER", "tcp://10.0.2.15")
MQTT_PORT = int(os.getenv("MQTT_PORT", 1883))
MQTT_TOPIC = "analyzedData"

def handle(req):
    client = mqtt.Client()
    client.connect(MQTT_BROKER, MQTT_PORT)

    # Générer des valeurs aléatoires
    data = {
        "temperature": round(random.uniform(10, 35), 2),
        "pressure": round(random.uniform(900, 1100), 2)
    }

    payload = json.dumps(data)
    client.publish(MQTT_TOPIC, payload)
    client.disconnect()

    return {
        "status": "Message published",
        "topic": MQTT_TOPIC,
        "data": data
    }