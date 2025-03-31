import paho.mqtt.client as mqtt
import json
import random
import os

# Configuration MQTT
MQTT_BROKER = os.getenv("MQTT_BROKER")
MQTT_PORT = int(os.getenv("MQTT_PORT", 1883))
MQTT_TOPIC = "processedData"

def handle(req):
    client = mqtt.Client()
    print("Connecting to MQTT broker: {}:{}".format(MQTT_BROKER, MQTT_PORT))
    client.connect(MQTT_BROKER, MQTT_PORT)

    # Générer des valeurs aléatoires
    data = {
        "temperature": round(random.uniform(15, 30), 2),
        "humidity": round(random.uniform(40, 80), 2)
    }

    payload = json.dumps(data)
    client.publish(MQTT_TOPIC, payload)
    client.disconnect()

    return {
        "status": "Message published",
        "topic": MQTT_TOPIC,
        "data": data
    }