import grpc
import math
import os
import json  # Importez le module json

from chirpstack_api import api  # type: ignore

# Récupérez les variables d'environnement
server = os.getenv("SERVER")
api_token = os.getenv("API_TOKEN")
dev_eui = os.getenv("DEV_EUI")

def handle(req):
    # Analyser le message MQTT JSON
    mqtt_message = json.loads(req)

    # Récupérer la période du champ "période" du JSON
    periode = mqtt_message.get("periode")

    # Effectuer les opérations nécessaires sur la période (conversions, calculs, etc.)
    hex_period = format(periode, 'x')
    len_payload = math.ceil(len(hex_period) / 2)
    hex_period = int(periode).to_bytes(len_payload, byteorder='big')
    hex_period = hex_period.hex()[2:] + hex_period.hex()[:2]
    payload = "ff03" + hex_period
    payload = bytes.fromhex(payload)

    # Connecter sans utiliser TLS.
    channel = grpc.insecure_channel(server)

    # Client de l'API de la file d'attente du périphérique.
    client = api.DeviceServiceStub(channel)

    # Définir les métadonnées de la clé API.
    auth_token = [("authorization", "Bearer %s" % api_token)]

    # Construire la demande.
    req = api.EnqueueDeviceQueueItemRequest()
    req.queue_item.confirmed = False
    req.queue_item.data = payload
    req.queue_item.dev_eui = dev_eui
    req.queue_item.f_port = 85

    # Envoyer la demande et obtenir la réponse
    resp = client.Enqueue(req, metadata=auth_token)

    # Imprimer l'identifiant de la liaison descendante
    print(resp.id)
