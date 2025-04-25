import paho.mqtt.client as mqtt
import json
import random
import os

# Configuration MQTT
MQTT_BROKER = os.getenv("MQTT_BROKER", "mosquitto.default.svc.cluster.local")
MQTT_PORT = int(os.getenv("MQTT_PORT", 1883))
MQTT_TOPIC = os.getenv("MQTT_TOPIC", "sensor/random_data")

def handle(req):
    client = mqtt.Client()
    client.connect(MQTT_BROKER, MQTT_PORT)

    if random.choice([True, False]):
        data = {
            "temperature": round(random.uniform(15, 30), 2),
            "humidity": round(random.uniform(40, 80), 2)
        }
    else:
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
